package upstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"owas-api/internal/domain"
)

const maxUpstreamBodyBytes = 16 << 20 // 16 MiB safety cap

// newDefaultHTTPClient constructs a pooled HTTP client with the given timeout.
func newDefaultHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

type queryParam struct {
	key   string
	value string
}

// encodeOrderedQuery encodes query pairs deterministically with uppercase %XX escaping.
func encodeOrderedQuery(params []queryParam) string {
	if len(params) == 0 {
		return ""
	}
	var b strings.Builder
	for i, p := range params {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.key))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.value))
	}
	return b.String()
}

// applyHeaders writes headers preserving exact wire casing (e.g., lowercase
// nonce/appid/device-id/platCode), which case-sensitive HTTP/1.1 gateways require.
func applyHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header[k] = []string{v}
	}
}

func readBodyLimited(r io.Reader) ([]byte, error) {
	lr := io.LimitReader(r, maxUpstreamBodyBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if len(data) > maxUpstreamBodyBytes {
		return nil, fmt.Errorf("upstream response exceeded %d bytes", maxUpstreamBodyBytes)
	}
	return data, nil
}

func mapTransportError(op string, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.NewUpstreamTimeoutError(fmt.Sprintf("%s timed out", op), err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return domain.NewUpstreamTimeoutError(fmt.Sprintf("%s timed out", op), err)
	}
	return domain.NewUpstreamError(fmt.Sprintf("%s failed", op), err, nil)
}

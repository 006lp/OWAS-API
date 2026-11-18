package upstream

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"owas-api/internal/domain"
	"owas-api/internal/protocol/crypto"
)

var _ domain.StoreGateway = (*StoreClient)(nil)

// StoreClient implements domain.StoreGateway against https://watch-cn.store.heytapmobi.com.
type StoreClient struct {
	baseURL string
	http    *http.Client
}

// NewStoreClient creates a Watch Store upstream HTTP client.
func NewStoreClient(baseURL string, timeout time.Duration, customClient *http.Client) *StoreClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = domain.AppStoreBaseURL
	}
	client := customClient
	if client == nil {
		client = newDefaultHTTPClient(timeout)
	}
	return &StoreClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    client,
	}
}

func doProtostuffRequest[T any](
	ctx context.Context,
	c *StoreClient,
	op string,
	method string,
	path string,
	params []queryParam,
	body []byte,
	profile domain.DeviceProfile,
	decoder func([]byte) (*T, error),
) (*T, error) {
	encodedQuery := encodeOrderedQuery(params)
	fullURL := c.baseURL + path
	if encodedQuery != "" {
		fullURL += "?" + encodedQuery
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, domain.NewUpstreamError(fmt.Sprintf("%s: build request", op), err, nil)
	}

	// Sign using the exact path and raw query transmitted on the wire.
	encodedPath := req.URL.EscapedPath()
	rawQuery := req.URL.RawQuery
	signedHeaders := crypto.BuildAppStoreHeaders(encodedPath, rawQuery, profile, "")
	applyHeaders(req, signedHeaders)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-protostuff; charset=UTF-8")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, mapTransportError(op, err)
	}
	defer resp.Body.Close()

	raw, err := readBodyLimited(resp.Body)
	if err != nil {
		return nil, domain.NewUpstreamError(fmt.Sprintf("%s: read body", op), err, nil)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		decoded, err := decoder(raw)
		if err != nil {
			return nil, domain.NewUpstreamError(fmt.Sprintf("%s: decode protostuff response", op), err, nil)
		}
		return decoded, nil
	case http.StatusNoContent:
		return nil, domain.NewNotFoundError(fmt.Sprintf("%s: resource not found", op))
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, domain.NewUnauthorizedError(
			fmt.Sprintf("%s: upstream authentication rejected", op),
			map[string]any{"upstreamStatus": resp.StatusCode},
		)
	default:
		return nil, domain.NewUpstreamError(
			fmt.Sprintf("%s: unexpected upstream HTTP %d", op, resp.StatusCode),
			nil,
			map[string]any{"upstreamStatus": resp.StatusCode},
		)
	}
}

// 1. GET /watch/search/list
func (c *StoreClient) Search(ctx context.Context, profile domain.DeviceProfile, params domain.SearchParams) (*domain.SearchResult, error) {
	return doProtostuffRequest(
		ctx, c, "store.search", http.MethodGet, "/watch/search/list",
		[]queryParam{
			{key: "word", value: params.Word},
			{key: "type", value: strconv.Itoa(params.SearchType)},
		},
		nil, profile, decodeSearchResult,
	)
}

// 2. GET /watch/pages/v2/home
func (c *StoreClient) GetHome(ctx context.Context, profile domain.DeviceProfile, params domain.HomeParams) (*domain.ViewLayer, error) {
	return doProtostuffRequest(
		ctx, c, "store.home", http.MethodGet, "/watch/pages/v2/home",
		[]queryParam{
			{key: "start", value: strconv.Itoa(params.Start)},
			{key: "size", value: strconv.Itoa(params.Size)},
		},
		nil, profile, decodeViewLayer,
	)
}

// 3. GET /watch/card/store/v2/subjects/{id}
func (c *StoreClient) GetSubject(ctx context.Context, profile domain.DeviceProfile, subjectID int64) (*domain.ViewLayer, error) {
	path := fmt.Sprintf("/watch/card/store/v2/subjects/%d", subjectID)
	return doProtostuffRequest(
		ctx, c, "store.subject", http.MethodGet, path,
		nil, nil, profile, decodeViewLayer,
	)
}

// 4. GET /watch/apps/{appId}
func (c *StoreClient) GetAppDetailByID(ctx context.Context, profile domain.DeviceProfile, params domain.AppDetailByIDParams) (*domain.AppDetail, error) {
	path := fmt.Sprintf("/watch/apps/%d", params.AppID)
	return doProtostuffRequest(
		ctx, c, "store.appDetailByID", http.MethodGet, path,
		[]queryParam{
			{key: "appType", value: strconv.Itoa(params.AppType)},
			{key: "query", value: strconv.Itoa(params.Query)},
			{key: "module", value: params.Module},
		},
		nil, profile, decodeAppDetail,
	)
}

// 5. GET /watch/apps?pkg=...
func (c *StoreClient) GetAppDetailByPkg(ctx context.Context, profile domain.DeviceProfile, params domain.AppDetailByPkgParams) (*domain.AppDetail, error) {
	return doProtostuffRequest(
		ctx, c, "store.appDetailByPkg", http.MethodGet, "/watch/apps",
		[]queryParam{
			{key: "pkg", value: params.PkgName},
			{key: "query", value: strconv.Itoa(params.Query)},
			{key: "module", value: params.Module},
		},
		nil, profile, decodeAppDetail,
	)
}

// 6. GET /watch/apps/{appId}/dl-infos/{verId}
func (c *StoreClient) GetDownloadInfo(ctx context.Context, profile domain.DeviceProfile, params domain.DownloadInfoParams) (*domain.AppDownload, error) {
	path := fmt.Sprintf("/watch/apps/%d/dl-infos/%d", params.AppID, params.VerID)
	return doProtostuffRequest(
		ctx, c, "store.downloadInfo", http.MethodGet, path,
		[]queryParam{
			{key: "type", value: params.DownloadType},
			{key: "retry", value: strconv.Itoa(params.Retry)},
		},
		nil, profile, decodeAppDownload,
	)
}

// 7. POST /watch/update-info
func (c *StoreClient) CheckUpdates(ctx context.Context, profile domain.DeviceProfile, apps []domain.InstalledApp) (*domain.UpgradeResult, error) {
	body := encodeUpgradeWrapReq(apps)
	return doProtostuffRequest(
		ctx, c, "store.checkUpdates", http.MethodPost, "/watch/update-info",
		nil, body, profile, decodeUpgradeResult,
	)
}

// 8. GET /watch/protocol
func (c *StoreClient) GetProtocol(ctx context.Context, profile domain.DeviceProfile, language string) (*domain.Protocol, error) {
	return doProtostuffRequest(
		ctx, c, "store.protocol", http.MethodGet, "/watch/protocol",
		[]queryParam{{key: "language", value: language}},
		nil, profile, decodeProtocol,
	)
}

// 9. POST /watch/createOrder
func (c *StoreClient) CreateOrder(ctx context.Context, profile domain.DeviceProfile, input domain.CreateOrderInput) (*domain.PayResponse[domain.OrderPay], error) {
	if input.Token == "" {
		input.Token = profile.Token
	}
	body := encodeOrderPayParam(input)
	return doProtostuffRequest(
		ctx, c, "store.createOrder", http.MethodPost, "/watch/createOrder",
		nil, body, profile,
		func(raw []byte) (*domain.PayResponse[domain.OrderPay], error) {
			return decodePayResponse(raw, decodeOrderPay)
		},
	)
}

// 10. POST /watch/queryOrder
func (c *StoreClient) QueryOrder(ctx context.Context, profile domain.DeviceProfile, input domain.QueryOrderInput) (*domain.PayResponse[domain.OrderStatus], error) {
	body := encodeOrderParam(input)
	return doProtostuffRequest(
		ctx, c, "store.queryOrder", http.MethodPost, "/watch/queryOrder",
		nil, body, profile,
		func(raw []byte) (*domain.PayResponse[domain.OrderStatus], error) {
			return decodePayResponse(raw, decodeOrderStatus)
		},
	)
}

// 11. GET /watch/queryPayStatus
func (c *StoreClient) QueryPayStatus(ctx context.Context, profile domain.DeviceProfile, appIDs []int64) (*domain.PayResponse[domain.OrderStatusList], error) {
	parts := make([]string, len(appIDs))
	for i, id := range appIDs {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return doProtostuffRequest(
		ctx, c, "store.queryPayStatus", http.MethodGet, "/watch/queryPayStatus",
		[]queryParam{{key: "appIds", value: strings.Join(parts, ",")}},
		nil, profile,
		func(raw []byte) (*domain.PayResponse[domain.OrderStatusList], error) {
			return decodePayResponse(raw, decodeOrderStatusList)
		},
	)
}

// Sanity helper ensuring url package stays clean if path escaping is ever extended.
var _ = url.PathEscape

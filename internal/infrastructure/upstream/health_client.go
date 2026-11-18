package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"owas-api/internal/domain"
	"owas-api/internal/protocol/crypto"
)

var _ domain.HealthGateway = (*HealthClient)(nil)

// HealthClient implements domain.HealthGateway against https://sport.health.heytapmobi.com.
type HealthClient struct {
	baseURL string
	http    *http.Client
}

// NewHealthClient creates a Health Cloud upstream HTTP client.
func NewHealthClient(baseURL string, timeout time.Duration, customClient *http.Client) *HealthClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = domain.HealthCloudBaseURL
	}
	client := customClient
	if client == nil {
		client = newDefaultHTTPClient(timeout)
	}
	return &HealthClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    client,
	}
}

// 12. POST /v1/c2s/switch/querySwitchStatus
func (c *HealthClient) QuerySwitchStatus(ctx context.Context, profile domain.DeviceProfile, switchType int) (*domain.SwitchStatusResponse, error) {
	const path = "/v1/c2s/switch/querySwitchStatus"
	fullURL := c.baseURL + path

	bodyBytes, err := json.Marshal(struct {
		SwitchType int `json:"switchType"`
	}{SwitchType: switchType})
	if err != nil {
		return nil, domain.NewUpstreamError("health.querySwitchStatus: marshal body", err, nil)
	}

	headers, err := crypto.BuildHealthCloudHeaders(fullURL, string(bodyBytes), profile)
	if err != nil {
		return nil, domain.NewUpstreamError("health.querySwitchStatus: build headers", err, nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, domain.NewUpstreamError("health.querySwitchStatus: build request", err, nil)
	}
	applyHeaders(req, headers)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, mapTransportError("health.querySwitchStatus", err)
	}
	defer resp.Body.Close()

	raw, err := readBodyLimited(resp.Body)
	if err != nil {
		return nil, domain.NewUpstreamError("health.querySwitchStatus: read body", err, nil)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, domain.NewUnauthorizedError(
			"health.querySwitchStatus: upstream authentication rejected",
			map[string]any{"upstreamStatus": resp.StatusCode, "upstreamBody": truncateForDetails(raw)},
		)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, domain.NewUpstreamError(
			fmt.Sprintf("health.querySwitchStatus: unexpected upstream HTTP %d", resp.StatusCode),
			nil,
			map[string]any{"upstreamStatus": resp.StatusCode},
		)
	}

	var out domain.SwitchStatusResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, domain.NewUpstreamError("health.querySwitchStatus: decode response json", err, nil)
	}
	if out.Body != nil && out.Body.Config != "" {
		var parsed map[string]any
		if json.Unmarshal([]byte(out.Body.Config), &parsed) == nil {
			out.Body.ParsedConfig = parsed
		}
	}
	return &out, nil
}

// 13. POST /v1/c2s/appStore/checkPay
func (c *HealthClient) CheckPay(ctx context.Context, profile domain.DeviceProfile, input domain.CheckPayInput) (*domain.CheckPayResponse, error) {
	const path = "/v1/c2s/appStore/checkPay"
	fullURL := c.baseURL + path

	cipherText := input.Cipher
	var generatedAESKey string
	if cipherText == "" && strings.TrimSpace(input.PublicKey) != "" {
		aesKey, err := crypto.GenerateAES128KeyBase64()
		if err != nil {
			return nil, domain.NewUpstreamError("health.checkPay: generate AES key", err, nil)
		}
		enc, err := crypto.RSAEncryptPKCS1(aesKey, input.PublicKey)
		if err != nil {
			return nil, domain.NewInvalidInputError("invalid RSA publicKey", map[string]any{"reason": err.Error()})
		}
		cipherText = enc
		generatedAESKey = aesKey
	}

	bodyBytes, err := json.Marshal(struct {
		AppID  int64  `json:"appid"`
		Cipher string `json:"cipher"`
	}{
		AppID:  input.AppID,
		Cipher: cipherText,
	})
	if err != nil {
		return nil, domain.NewUpstreamError("health.checkPay: marshal body", err, nil)
	}

	headers, err := crypto.BuildHealthCloudHeaders(fullURL, string(bodyBytes), profile)
	if err != nil {
		return nil, domain.NewUpstreamError("health.checkPay: build headers", err, nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, domain.NewUpstreamError("health.checkPay: build request", err, nil)
	}
	applyHeaders(req, headers)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, mapTransportError("health.checkPay", err)
	}
	defer resp.Body.Close()

	raw, err := readBodyLimited(resp.Body)
	if err != nil {
		return nil, domain.NewUpstreamError("health.checkPay: read body", err, nil)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, domain.NewUnauthorizedError(
			"health.checkPay: valid HeyTap token required",
			map[string]any{"upstreamStatus": resp.StatusCode, "upstreamBody": truncateForDetails(raw)},
		)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, domain.NewUpstreamError(
			fmt.Sprintf("health.checkPay: unexpected upstream HTTP %d", resp.StatusCode),
			nil,
			map[string]any{"upstreamStatus": resp.StatusCode},
		)
	}

	var out domain.CheckPayResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, domain.NewUpstreamError("health.checkPay: decode response json", err, nil)
	}
	if generatedAESKey != "" && out.Body != "" {
		decrypted, err := crypto.AESGCMDecryptText(generatedAESKey, out.Body)
		if err == nil {
			var items []domain.CheckPayItem
			if json.Unmarshal([]byte(decrypted), &items) == nil {
				out.DecryptedBody = items
			}
		}
	}
	return &out, nil
}

// 14. POST /sporthealth/v1/c2s/encrypt/exchangeSymmetricKey
func (c *HealthClient) ExchangeSymmetricKey(ctx context.Context, profile domain.DeviceProfile, encryptionAlgorithm int, firstExchangeKey int) (*domain.ExchangeKeyResponse, error) {
	const path = "/sporthealth/v1/c2s/encrypt/exchangeSymmetricKey"
	fullURL := c.baseURL + path

	clientKey, err := crypto.GenerateUUIDHex()
	if err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: generate client key", err, nil)
	}

	plaintextBytes, err := json.Marshal(struct {
		EncryptionAlgorithm int    `json:"encryptionAlgorithm"`
		FirstExchangeKey    int    `json:"firstExchangeKey"`
		MobileUniqueID      string `json:"mobileUniqueId"`
		SymmetricKey        string `json:"symmetricKey"`
	}{
		EncryptionAlgorithm: encryptionAlgorithm,
		FirstExchangeKey:    firstExchangeKey,
		MobileUniqueID:      domain.AppStoreDeviceID,
		SymmetricKey:        clientKey,
	})
	if err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: marshal request", err, nil)
	}

	encryptedBody, err := crypto.RSAEncryptPKCS1(string(plaintextBytes), "")
	if err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: RSA encrypt", err, nil)
	}

	headers, err := crypto.BuildExchangeKeyHeaders(encryptedBody, profile)
	if err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: build headers", err, nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, strings.NewReader(encryptedBody))
	if err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: build request", err, nil)
	}
	applyHeaders(req, headers)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, mapTransportError("health.exchangeSymmetricKey", err)
	}
	defer resp.Body.Close()

	raw, err := readBodyLimited(resp.Body)
	if err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: read body", err, nil)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, domain.NewUpstreamError(
			fmt.Sprintf("health.exchangeSymmetricKey: unexpected upstream HTTP %d", resp.StatusCode),
			nil,
			map[string]any{"upstreamStatus": resp.StatusCode, "upstreamBody": truncateForDetails(raw)},
		)
	}

	var out domain.ExchangeKeyResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: decode response json", err, nil)
	}
	if out.Body != "" {
		decrypted, err := crypto.AESGCMDecryptText(clientKey, out.Body)
		if err != nil {
			return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: AES-GCM decrypt body", err, nil)
		}
		var dec domain.ExchangeKeyDecryptedBody
		if err := json.Unmarshal([]byte(decrypted), &dec); err != nil {
			return nil, domain.NewUpstreamError("health.exchangeSymmetricKey: decode decrypted body", err, nil)
		}
		out.DecryptedBody = &dec
	}
	return &out, nil
}

func truncateForDetails(b []byte) string {
	const limit = 256
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "..."
}

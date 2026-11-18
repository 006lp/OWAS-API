//go:build live

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
	"owas-api/internal/infrastructure/upstream"
	"owas-api/internal/service"
	httptransport "owas-api/internal/transport/http"
)

type envelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	RequestID string          `json:"requestId"`
	Data      json.RawMessage `json:"data"`
	Details   map[string]any  `json:"details"`
}

func hasUnknownFields(raw []byte) bool {
	return strings.Contains(string(raw), `"_unknown_fields"`)
}

func extractSubjectIDs(cards []domain.Card) []int64 {
	var ids []int64
	seen := map[int64]bool{}
	for _, card := range cards {
		candidates := []string{card.ActionParam}
		for _, b := range card.Banners {
			candidates = append(candidates, b.ActionParam)
		}
		for _, ap := range candidates {
			if !strings.Contains(ap, "pk=6000") {
				continue
			}
			parts := strings.Split(ap, "?")
			query := parts[len(parts)-1]
			for _, kv := range strings.Split(query, "&") {
				if strings.HasPrefix(kv, "pk=6000") {
					idStr := strings.TrimPrefix(kv, "pk=6000")
					if id, err := strconv.ParseInt(idStr, 10, 64); err == nil && !seen[id] {
						seen[id] = true
						ids = append(ids, id)
					}
				}
			}
		}
	}
	return ids
}

// TestLivePublicAPIsAndAuthBoundaries exercises all 14 OPPO Watch Store & Health Cloud
// endpoints through the full Gin RESTful HTTP stack against live upstream services.
//
// Run with: go test -v -tags=live ./tests/...
func TestLivePublicAPIsAndAuthBoundaries(t *testing.T) {
	storeGW := upstream.NewStoreClient(domain.AppStoreBaseURL, 20*time.Second, nil)
	healthGW := upstream.NewHealthClient(domain.HealthCloudBaseURL, 20*time.Second, nil)
	router := httptransport.NewRouter(httptransport.RouterDeps{
		StoreService:   service.NewStoreService(storeGW),
		HealthService:  service.NewHealthService(healthGW),
		DefaultProfile: domain.DefaultDeviceProfile(),
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:        "live-test",
		GinMode:        gin.TestMode,
	})

	doReq := func(method, path string, body string, headers map[string]string) (*httptest.ResponseRecorder, envelope) {
		var rdr io.Reader
		if body != "" {
			rdr = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, path, rdr)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var env envelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		return w, env
	}

	var (
		wechatAppID   int64 = 31617863
		wechatVerID   int64 = 25869985
		wechatVerCode int64 = 65543
		wechatSize    int64
		subjectIDs    []int64
	)

	t.Run("#1a Search fuzzy", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/apps?word=%E5%BE%AE%E4%BF%A1&type=0", "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.SearchResult
		if err := json.Unmarshal(env.Data, &res); err != nil || len(res.Apps) == 0 {
			t.Fatalf("unexpected search data: err=%v count=%d", err, len(res.Apps))
		}
		wechatAppID = res.Apps[0].AppID
		wechatVerID = res.Apps[0].VerID
		wechatVerCode = res.Apps[0].VerCode
		wechatSize = res.Apps[0].Size
		t.Logf("fuzzy search count=%d top1=%s (appId=%d)", len(res.Apps), res.Apps[0].AppName, wechatAppID)
	})

	t.Run("#1b Search exact", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/apps?word=%E5%BE%AE%E4%BF%A1&type=1", "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.SearchResult
		_ = json.Unmarshal(env.Data, &res)
		if len(res.Apps) != 1 || res.Apps[0].PkgName != "com.tencent.watchwechat" {
			t.Fatalf("expected 1 exact match com.tencent.watchwechat, got %+v", res.Apps)
		}
	})

	t.Run("#1c Search device header override (square watch)", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/apps?word=%E5%BE%AE%E4%BF%A1&type=0", "", map[string]string{
			"X-Watch-Type":    "1",
			"X-Watch-Support": "ios",
		})
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.SearchResult
		_ = json.Unmarshal(env.Data, &res)
		t.Logf("square+ios search count=%d", len(res.Apps))
	})

	t.Run("#2 Home cards", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/home?start=0&size=10", "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.ViewLayer
		_ = json.Unmarshal(env.Data, &res)
		if res.PageKey != 100 || len(res.Cards) == 0 {
			t.Fatalf("unexpected home response: pageKey=%d cards=%d", res.PageKey, len(res.Cards))
		}
		subjectIDs = extractSubjectIDs(res.Cards)
		t.Logf("home cards=%d isEnd=%d extractedSubjects=%v", len(res.Cards), res.IsEnd, subjectIDs)
	})

	t.Run("#3 Subject detail", func(t *testing.T) {
		sid := int64(60)
		if len(subjectIDs) > 0 {
			sid = subjectIDs[0]
		}
		w, env := doReq(http.MethodGet, fmt.Sprintf("/api/v1/subjects/%d", sid), "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.ViewLayer
		_ = json.Unmarshal(env.Data, &res)
		if res.PageKey != 101 || len(res.Cards) == 0 {
			t.Fatalf("unexpected subject response: pageKey=%d cards=%d", res.PageKey, len(res.Cards))
		}
	})

	t.Run("#4a App detail by ID", func(t *testing.T) {
		w, env := doReq(http.MethodGet, fmt.Sprintf("/api/v1/apps/%d?appType=1&module=search", wechatAppID), "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.AppDetail
		_ = json.Unmarshal(env.Data, &res)
		if res.Base == nil || res.Base.AppID != wechatAppID || res.Developer == nil || res.Developer.Developer == "" {
			t.Fatalf("unexpected app detail: %+v", res)
		}
	})

	t.Run("#4b App detail nonexistent -> 404", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/apps/999999999", "", nil)
		if w.Code != http.StatusNotFound || env.Code != "NOT_FOUND" {
			t.Fatalf("expected 404 NOT_FOUND, got status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("#5a App detail by package", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/packages/com.tencent.watchwechat", "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.AppDetail
		_ = json.Unmarshal(env.Data, &res)
		if res.Base == nil || res.Base.AppID != wechatAppID {
			t.Fatalf("unexpected pkg detail: %+v", res)
		}
	})

	t.Run("#5b App detail nonexistent package -> 404", func(t *testing.T) {
		w, _ := doReq(http.MethodGet, "/api/v1/packages/com.nonexistent.watch.fake.xyz", "", nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("#6 Download info + CDN HEAD", func(t *testing.T) {
		w, env := doReq(http.MethodGet, fmt.Sprintf("/api/v1/apps/%d/versions/%d/download?type=1&retry=2", wechatAppID, wechatVerID), "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.AppDownload
		_ = json.Unmarshal(env.Data, &res)
		if !strings.HasPrefix(res.DownURL, "https://") || len(res.MD5) != 32 || len(res.HeaderMD5) != 32 {
			t.Fatalf("unexpected download info: %+v", res)
		}
		headResp, err := (&http.Client{Timeout: 15 * time.Second}).Head(res.DownURL)
		if err != nil {
			t.Fatalf("CDN HEAD failed: %v", err)
		}
		defer headResp.Body.Close()
		if headResp.StatusCode != http.StatusOK || (wechatSize > 0 && headResp.ContentLength != wechatSize) {
			t.Fatalf("CDN HEAD mismatch: status=%d contentLength=%d expected=%d", headResp.StatusCode, headResp.ContentLength, wechatSize)
		}
		t.Logf("CDN HEAD OK (%d bytes)", headResp.ContentLength)
	})

	t.Run("#7 Check updates", func(t *testing.T) {
		body := fmt.Sprintf(`{"apps":[{"pkgName":"com.tencent.watchwechat","verCode":"1"},{"pkgName":"com.tencent.watchwechat","verCode":"%d"}]}`, wechatVerCode)
		w, env := doReq(http.MethodPost, "/api/v1/updates/check", body, nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.UpgradeResult
		_ = json.Unmarshal(env.Data, &res)
		if len(res.Upgrades) == 0 {
			t.Fatalf("expected upgrade entries, got %+v", res)
		}
	})

	t.Run("#8 Protocol", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/protocols?language=zh_CN", "", nil)
		if w.Code != http.StatusOK || hasUnknownFields(env.Data) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.Protocol
		_ = json.Unmarshal(env.Data, &res)
		if !strings.HasPrefix(res.Privacy, "https://") || !strings.HasPrefix(res.Agreement, "https://") {
			t.Fatalf("unexpected protocol: %+v", res)
		}
	})

	t.Run("#12 Health Cloud switch status", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/health-cloud/switches/116", "", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.SwitchStatusResponse
		_ = json.Unmarshal(env.Data, &res)
		if res.ErrorCode != 0 || res.Body == nil || res.Body.ParsedConfig == nil {
			t.Fatalf("unexpected switch status: %+v", res)
		}
		t.Logf("switchStatus=%d parsedConfig=%v", res.Body.SwitchStatus, res.Body.ParsedConfig)
	})

	t.Run("#14 Health Cloud exchangeSymmetricKey", func(t *testing.T) {
		w, env := doReq(http.MethodPost, "/api/v1/health-cloud/symmetric-keys/exchange", "", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var res domain.ExchangeKeyResponse
		_ = json.Unmarshal(env.Data, &res)
		if res.ErrorCode != 0 || res.DecryptedBody == nil || len(res.DecryptedBody.SymmetricKey) < 16 || res.DecryptedBody.EncryptionAlgorithm != 4 {
			t.Fatalf("unexpected exchangeSymmetricKey response: %+v", res)
		}
		t.Logf("exchangeKey OK: alg=%d keyLen=%d symKeyLen=%d", res.DecryptedBody.EncryptionAlgorithm, res.DecryptedBody.KeyLength, len(res.DecryptedBody.SymmetricKey))
	})

	// Token-gated auth boundaries (#9, #10, #11, #13)
	t.Run("#9 CreateOrder without token -> 401", func(t *testing.T) {
		w, env := doReq(http.MethodPost, "/api/v1/orders", `{"appId":37627586,"price":600,"productName":"test","productDesc":"test"}`, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d (%+v)", w.Code, env)
		}
	})

	t.Run("#10 QueryOrder without token -> 401 or 502(code=303)", func(t *testing.T) {
		w, env := doReq(http.MethodGet, "/api/v1/orders/0?userId=test&appId=31617863", "", nil)
		upstreamCode, _ := env.Details["upstreamCode"].(string)
		if w.Code != http.StatusUnauthorized && !(w.Code == http.StatusBadGateway && upstreamCode == "303") {
			t.Fatalf("expected 401 or 502 with upstreamCode=303, got %d (%+v)", w.Code, env)
		}
	})

	t.Run("#11 QueryPayStatus without token -> 401", func(t *testing.T) {
		w, env := doReq(http.MethodGet, fmt.Sprintf("/api/v1/payments/status?appIds=%d", wechatAppID), "", nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d (%+v)", w.Code, env)
		}
	})

	t.Run("#13 Health Cloud checkPay without token -> 401", func(t *testing.T) {
		w, env := doReq(http.MethodPost, "/api/v1/health-cloud/pay-checks", fmt.Sprintf(`{"appId":%d,"cipher":""}`, wechatAppID), nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d (%+v)", w.Code, env)
		}
	})
}

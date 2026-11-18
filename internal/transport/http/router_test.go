package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
	"owas-api/internal/service"
)

type mockStoreService struct {
	lastProfile domain.DeviceProfile
	searchErr   error
	detailErr   error
	orderErr    error
}

func (m *mockStoreService) Search(_ context.Context, p domain.DeviceProfile, params domain.SearchParams) (*domain.SearchResult, error) {
	m.lastProfile = p
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return &domain.SearchResult{
		Apps:  []domain.Resource{{AppID: 31617863, AppName: params.Word, PkgName: "com.tencent.watchwechat"}},
		IsEnd: 0,
	}, nil
}
func (m *mockStoreService) GetHome(_ context.Context, p domain.DeviceProfile, _ domain.HomeParams) (*domain.ViewLayer, error) {
	m.lastProfile = p
	return &domain.ViewLayer{PageKey: 100, Cards: []domain.Card{{Code: 1001}}}, nil
}
func (m *mockStoreService) GetSubject(_ context.Context, p domain.DeviceProfile, _ int64) (*domain.ViewLayer, error) {
	m.lastProfile = p
	return &domain.ViewLayer{PageKey: 101, Cards: []domain.Card{}}, nil
}
func (m *mockStoreService) GetAppDetailByID(_ context.Context, p domain.DeviceProfile, _ domain.AppDetailByIDParams) (*domain.AppDetail, error) {
	m.lastProfile = p
	if m.detailErr != nil {
		return nil, m.detailErr
	}
	return &domain.AppDetail{Base: &domain.Resource{AppID: 31617863, AppName: "微信"}}, nil
}
func (m *mockStoreService) GetAppDetailByPkg(_ context.Context, p domain.DeviceProfile, params domain.AppDetailByPkgParams) (*domain.AppDetail, error) {
	m.lastProfile = p
	return &domain.AppDetail{Base: &domain.Resource{PkgName: params.PkgName}}, nil
}
func (m *mockStoreService) GetDownloadInfo(_ context.Context, p domain.DeviceProfile, params domain.DownloadInfoParams) (*domain.AppDownload, error) {
	m.lastProfile = p
	return &domain.AppDownload{AppID: params.AppID, VerID: params.VerID, DownURL: "https://cdn/app.apk"}, nil
}
func (m *mockStoreService) CheckUpdates(_ context.Context, p domain.DeviceProfile, apps []domain.InstalledApp) (*domain.UpgradeResult, error) {
	m.lastProfile = p
	return &domain.UpgradeResult{Upgrades: []domain.UpgradeItem{{Resource: domain.Resource{PkgName: apps[0].PkgName}, UpgradeFlag: 1}}}, nil
}
func (m *mockStoreService) GetProtocol(_ context.Context, p domain.DeviceProfile, _ string) (*domain.Protocol, error) {
	m.lastProfile = p
	return &domain.Protocol{Privacy: "https://p", Agreement: "https://a", PrivacyVersion: 1}, nil
}
func (m *mockStoreService) CreateOrder(_ context.Context, p domain.DeviceProfile, _ domain.CreateOrderInput) (*domain.PayResponse[domain.OrderPay], error) {
	m.lastProfile = p
	if m.orderErr != nil {
		return nil, m.orderErr
	}
	return &domain.PayResponse[domain.OrderPay]{Code: "200", Data: &domain.OrderPay{OrderID: "ord-9"}}, nil
}
func (m *mockStoreService) QueryOrder(_ context.Context, p domain.DeviceProfile, in domain.QueryOrderInput) (*domain.PayResponse[domain.OrderStatus], error) {
	m.lastProfile = p
	return &domain.PayResponse[domain.OrderStatus]{Code: "200", Data: &domain.OrderStatus{AppID: in.AppID, PayStatus: 1}}, nil
}
func (m *mockStoreService) QueryPayStatus(_ context.Context, p domain.DeviceProfile, appIDs []int64) (*domain.PayResponse[domain.OrderStatusList], error) {
	m.lastProfile = p
	return &domain.PayResponse[domain.OrderStatusList]{
		Code: "200",
		Data: &domain.OrderStatusList{StatusVOList: []domain.OrderStatus{{AppID: appIDs[0], PayStatus: 0}}},
	}, nil
}

type mockHealthService struct{}

func (m *mockHealthService) QuerySwitchStatus(_ context.Context, _ domain.DeviceProfile, _ int) (*domain.SwitchStatusResponse, error) {
	return &domain.SwitchStatusResponse{ErrorCode: 0, Message: "ok", Body: &domain.SwitchBody{SwitchStatus: 1}}, nil
}
func (m *mockHealthService) CheckPay(_ context.Context, _ domain.DeviceProfile, _ domain.CheckPayInput) (*domain.CheckPayResponse, error) {
	return &domain.CheckPayResponse{ErrorCode: 0, Message: "ok"}, nil
}
func (m *mockHealthService) ExchangeSymmetricKey(_ context.Context, _ domain.DeviceProfile, _ domain.ExchangeKeyInput) (*domain.ExchangeKeyResponse, error) {
	return &domain.ExchangeKeyResponse{
		ErrorCode:     0,
		Message:       "ok",
		DecryptedBody: &domain.ExchangeKeyDecryptedBody{EncryptionAlgorithm: 4, KeyLength: 256, SymmetricKey: "0123456789abcdef0123456789abcdef"},
	}, nil
}

func newTestRouter(storeSvc service.StoreUseCase, healthSvc service.HealthUseCase) *gin.Engine {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(RouterDeps{
		StoreService:   storeSvc,
		HealthService:  healthSvc,
		DefaultProfile: domain.DefaultDeviceProfile(),
		Logger:         discardLogger,
		Version:        "test",
		GinMode:        gin.TestMode,
	})
}

func TestRouterEndpointsAndRESTSemantics(t *testing.T) {
	storeSvc := &mockStoreService{}
	r := newTestRouter(storeSvc, &mockHealthService{})

	cases := []struct {
		name       string
		method     string
		target     string
		body       string
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{"healthz", http.MethodGet, "/healthz", "", nil, http.StatusOK, "OK"},
		{"search apps with device headers", http.MethodGet, "/api/v1/apps?word=微信&type=1", "", map[string]string{"X-Watch-Type": "1", "X-Watch-Support": "ios", "Authorization": "Bearer abc-token"}, http.StatusOK, "OK"},
		{"invalid device header", http.MethodGet, "/api/v1/apps?word=微信", "", map[string]string{"X-Watch-Type": "9"}, http.StatusBadRequest, "INVALID_ARGUMENT"},
		{"home", http.MethodGet, "/api/v1/home?start=0&size=10", "", nil, http.StatusOK, "OK"},
		{"subject", http.MethodGet, "/api/v1/subjects/87", "", nil, http.StatusOK, "OK"},
		{"app detail by id", http.MethodGet, "/api/v1/apps/31617863?appType=1&module=search", "", nil, http.StatusOK, "OK"},
		{"app detail invalid id", http.MethodGet, "/api/v1/apps/notanint", "", nil, http.StatusBadRequest, "INVALID_ARGUMENT"},
		{"app detail by pkg", http.MethodGet, "/api/v1/packages/com.tencent.watchwechat", "", nil, http.StatusOK, "OK"},
		{"download info", http.MethodGet, "/api/v1/apps/31617863/versions/25869985/download?type=1&retry=2", "", nil, http.StatusOK, "OK"},
		{"check updates", http.MethodPost, "/api/v1/updates/check", `{"apps":[{"pkgName":"com.tencent.watchwechat","verCode":"1"}]}`, nil, http.StatusOK, "OK"},
		{"check updates empty body", http.MethodPost, "/api/v1/updates/check", `{"apps":[]}`, nil, http.StatusBadRequest, "INVALID_ARGUMENT"},
		{"protocol", http.MethodGet, "/api/v1/protocols?language=zh_CN", "", nil, http.StatusOK, "OK"},
		{"create order 201", http.MethodPost, "/api/v1/orders", `{"appId":37627586,"price":600,"productName":"Lab","productDesc":"com.lab"}`, nil, http.StatusCreated, "OK"},
		{"query order", http.MethodGet, "/api/v1/orders/ord-9?userId=u1&appId=37627586", "", nil, http.StatusOK, "OK"},
		{"query pay status", http.MethodGet, "/api/v1/payments/status?appIds=31617863,37627586", "", nil, http.StatusOK, "OK"},
		{"health switch", http.MethodGet, "/api/v1/health-cloud/switches/116", "", nil, http.StatusOK, "OK"},
		{"health checkPay", http.MethodPost, "/api/v1/health-cloud/pay-checks", `{"appId":37627586}`, nil, http.StatusOK, "OK"},
		{"health exchangeKey empty body", http.MethodPost, "/api/v1/health-cloud/symmetric-keys/exchange", "", nil, http.StatusOK, "OK"},
		{"unknown route 404", http.MethodGet, "/api/v1/does-not-exist", "", nil, http.StatusNotFound, "ROUTE_NOT_FOUND"},
		{"wrong method 405", http.MethodDelete, "/api/v1/home", "", nil, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = bytes.NewBufferString(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.target, bodyReader)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d; body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
			if w.Header().Get("X-Request-ID") == "" {
				t.Fatal("missing X-Request-ID response header")
			}
			var env struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
				t.Fatalf("invalid json response: %v (%s)", err, w.Body.String())
			}
			if env.Code != tc.wantCode {
				t.Fatalf("code=%q, want %q; body=%s", env.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestRouterErrorStatusMapping(t *testing.T) {
	storeSvc := &mockStoreService{
		detailErr: domain.NewNotFoundError("app not found"),
		orderErr:  domain.NewUnauthorizedError("token required", map[string]any{"upstreamCode": "302"}),
		searchErr: domain.NewUpstreamTimeoutError("upstream timed out", nil),
	}
	r := newTestRouter(storeSvc, &mockHealthService{})

	// 404
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/apps/999", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// 401
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"appId":1,"price":100,"productName":"A","productDesc":"B"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// 504
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/apps?word=test", nil))
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", w.Code)
	}
}

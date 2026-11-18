package service

import (
	"context"
	"errors"
	"testing"

	"owas-api/internal/domain"
)

type stubStoreGateway struct {
	lastProfile  domain.DeviceProfile
	lastSearch   domain.SearchParams
	lastHome     domain.HomeParams
	lastOrderIn  domain.CreateOrderInput
	payRespCode  string
	payRespMsg   string
	searchResult *domain.SearchResult
}

func (g *stubStoreGateway) Search(_ context.Context, p domain.DeviceProfile, params domain.SearchParams) (*domain.SearchResult, error) {
	g.lastProfile = p
	g.lastSearch = params
	return g.searchResult, nil
}
func (g *stubStoreGateway) GetHome(_ context.Context, p domain.DeviceProfile, params domain.HomeParams) (*domain.ViewLayer, error) {
	g.lastProfile = p
	g.lastHome = params
	return &domain.ViewLayer{PageKey: 100}, nil
}
func (g *stubStoreGateway) GetSubject(_ context.Context, _ domain.DeviceProfile, _ int64) (*domain.ViewLayer, error) {
	return &domain.ViewLayer{PageKey: 101}, nil
}
func (g *stubStoreGateway) GetAppDetailByID(_ context.Context, _ domain.DeviceProfile, _ domain.AppDetailByIDParams) (*domain.AppDetail, error) {
	return &domain.AppDetail{}, nil
}
func (g *stubStoreGateway) GetAppDetailByPkg(_ context.Context, _ domain.DeviceProfile, _ domain.AppDetailByPkgParams) (*domain.AppDetail, error) {
	return &domain.AppDetail{}, nil
}
func (g *stubStoreGateway) GetDownloadInfo(_ context.Context, _ domain.DeviceProfile, _ domain.DownloadInfoParams) (*domain.AppDownload, error) {
	return &domain.AppDownload{}, nil
}
func (g *stubStoreGateway) CheckUpdates(_ context.Context, _ domain.DeviceProfile, _ []domain.InstalledApp) (*domain.UpgradeResult, error) {
	return &domain.UpgradeResult{}, nil
}
func (g *stubStoreGateway) GetProtocol(_ context.Context, _ domain.DeviceProfile, _ string) (*domain.Protocol, error) {
	return &domain.Protocol{}, nil
}
func (g *stubStoreGateway) CreateOrder(_ context.Context, p domain.DeviceProfile, in domain.CreateOrderInput) (*domain.PayResponse[domain.OrderPay], error) {
	g.lastProfile = p
	g.lastOrderIn = in
	return &domain.PayResponse[domain.OrderPay]{
		Code: g.payRespCode,
		Msg:  g.payRespMsg,
		Data: &domain.OrderPay{OrderID: "ord-1"},
	}, nil
}
func (g *stubStoreGateway) QueryOrder(_ context.Context, _ domain.DeviceProfile, _ domain.QueryOrderInput) (*domain.PayResponse[domain.OrderStatus], error) {
	return &domain.PayResponse[domain.OrderStatus]{Code: g.payRespCode, Msg: g.payRespMsg}, nil
}
func (g *stubStoreGateway) QueryPayStatus(_ context.Context, _ domain.DeviceProfile, _ []int64) (*domain.PayResponse[domain.OrderStatusList], error) {
	return &domain.PayResponse[domain.OrderStatusList]{Code: g.payRespCode, Msg: g.payRespMsg}, nil
}

type stubHealthGateway struct {
	lastAlg   int
	lastFirst int
}

func (g *stubHealthGateway) QuerySwitchStatus(_ context.Context, _ domain.DeviceProfile, _ int) (*domain.SwitchStatusResponse, error) {
	return &domain.SwitchStatusResponse{ErrorCode: 0}, nil
}
func (g *stubHealthGateway) CheckPay(_ context.Context, _ domain.DeviceProfile, _ domain.CheckPayInput) (*domain.CheckPayResponse, error) {
	return &domain.CheckPayResponse{ErrorCode: 0}, nil
}
func (g *stubHealthGateway) ExchangeSymmetricKey(_ context.Context, _ domain.DeviceProfile, alg int, first int) (*domain.ExchangeKeyResponse, error) {
	g.lastAlg = alg
	g.lastFirst = first
	return &domain.ExchangeKeyResponse{ErrorCode: 0}, nil
}

func TestStoreServiceValidationAndDefaults(t *testing.T) {
	gw := &stubStoreGateway{searchResult: &domain.SearchResult{}}
	svc := NewStoreService(gw)

	// Empty word rejected.
	if _, err := svc.Search(context.Background(), domain.DefaultDeviceProfile(), domain.SearchParams{Word: "   "}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for blank word, got %v", err)
	}

	// Default size=10 when omitted.
	if _, err := svc.GetHome(context.Background(), domain.DeviceProfile{}, domain.HomeParams{Start: 0, Size: 0}); err != nil {
		t.Fatalf("unexpected GetHome error: %v", err)
	}
	if gw.lastHome.Size != 10 {
		t.Fatalf("expected default size 10, got %d", gw.lastHome.Size)
	}
}

func TestStoreServicePayResponseMapping(t *testing.T) {
	gw := &stubStoreGateway{payRespCode: "302", payRespMsg: "token校验未通过"}
	svc := NewStoreService(gw)

	_, err := svc.CreateOrder(context.Background(), domain.DefaultDeviceProfile(), domain.CreateOrderInput{
		AppID:       123,
		Price:       600,
		ProductName: "App",
		ProductDesc: "com.app",
		Token:       "body-tok",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for code=302, got %v", err)
	}
	if gw.lastProfile.Token != "body-tok" {
		t.Fatalf("expected body token synced to profile header, got %q", gw.lastProfile.Token)
	}

	gw.payRespCode = "303"
	gw.payRespMsg = "unknown error"
	if _, err := svc.QueryOrder(context.Background(), domain.DefaultDeviceProfile(), domain.QueryOrderInput{
		OrderID: "1", UserID: "u1", AppID: 123,
	}); !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("expected ErrUpstream for code=303, got %v", err)
	}

	gw.payRespCode = "200"
	resp, err := svc.CreateOrder(context.Background(), domain.DefaultDeviceProfile(), domain.CreateOrderInput{
		AppID: 123, Price: 600, ProductName: "App", ProductDesc: "com.app",
	})
	if err != nil || resp.Data.OrderID != "ord-1" {
		t.Fatalf("expected successful order creation, got resp=%+v err=%v", resp, err)
	}
}

func TestHealthServiceDefaults(t *testing.T) {
	gw := &stubHealthGateway{}
	svc := NewHealthService(gw)

	if _, err := svc.ExchangeSymmetricKey(context.Background(), domain.DefaultDeviceProfile(), domain.ExchangeKeyInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.lastAlg != 4 || gw.lastFirst != 0 {
		t.Fatalf("expected defaults alg=4 first=0, got alg=%d first=%d", gw.lastAlg, gw.lastFirst)
	}
}

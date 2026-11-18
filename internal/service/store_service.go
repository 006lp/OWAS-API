package service

import (
	"context"
	"fmt"
	"strings"

	"owas-api/internal/domain"
)

// StoreUseCase defines the application-layer contract for Watch Store operations.
type StoreUseCase interface {
	Search(ctx context.Context, profile domain.DeviceProfile, params domain.SearchParams) (*domain.SearchResult, error)
	GetHome(ctx context.Context, profile domain.DeviceProfile, params domain.HomeParams) (*domain.ViewLayer, error)
	GetSubject(ctx context.Context, profile domain.DeviceProfile, subjectID int64) (*domain.ViewLayer, error)
	GetAppDetailByID(ctx context.Context, profile domain.DeviceProfile, params domain.AppDetailByIDParams) (*domain.AppDetail, error)
	GetAppDetailByPkg(ctx context.Context, profile domain.DeviceProfile, params domain.AppDetailByPkgParams) (*domain.AppDetail, error)
	GetDownloadInfo(ctx context.Context, profile domain.DeviceProfile, params domain.DownloadInfoParams) (*domain.AppDownload, error)
	CheckUpdates(ctx context.Context, profile domain.DeviceProfile, apps []domain.InstalledApp) (*domain.UpgradeResult, error)
	GetProtocol(ctx context.Context, profile domain.DeviceProfile, language string) (*domain.Protocol, error)
	CreateOrder(ctx context.Context, profile domain.DeviceProfile, input domain.CreateOrderInput) (*domain.PayResponse[domain.OrderPay], error)
	QueryOrder(ctx context.Context, profile domain.DeviceProfile, input domain.QueryOrderInput) (*domain.PayResponse[domain.OrderStatus], error)
	QueryPayStatus(ctx context.Context, profile domain.DeviceProfile, appIDs []int64) (*domain.PayResponse[domain.OrderStatusList], error)
}

var _ StoreUseCase = (*StoreService)(nil)

// StoreService orchestrates validation, default normalization, and gateway calls for Watch Store APIs.
type StoreService struct {
	gateway domain.StoreGateway
}

// NewStoreService constructs a StoreService with the injected StoreGateway.
func NewStoreService(gateway domain.StoreGateway) *StoreService {
	return &StoreService{gateway: gateway}
}

func normalizeProfile(profile domain.DeviceProfile) (domain.DeviceProfile, error) {
	dev := profile.WithDefaults()
	if err := dev.Validate(); err != nil {
		return domain.DeviceProfile{}, err
	}
	return dev, nil
}

func (s *StoreService) Search(ctx context.Context, profile domain.DeviceProfile, params domain.SearchParams) (*domain.SearchResult, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	params.Word = strings.TrimSpace(params.Word)
	if params.Word == "" {
		return nil, domain.NewInvalidInputError("query parameter 'word' is required", map[string]any{"field": "word"})
	}
	if params.SearchType < 0 || params.SearchType > 2 {
		return nil, domain.NewInvalidInputError(
			fmt.Sprintf("invalid search type %d: expected 0 (fuzzy), 1 (exact), or 2", params.SearchType),
			map[string]any{"field": "type", "value": params.SearchType},
		)
	}
	return s.gateway.Search(ctx, dev, params)
}

func (s *StoreService) GetHome(ctx context.Context, profile domain.DeviceProfile, params domain.HomeParams) (*domain.ViewLayer, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if params.Start < 0 {
		return nil, domain.NewInvalidInputError("query parameter 'start' must be >= 0", map[string]any{"field": "start"})
	}
	if params.Size == 0 {
		params.Size = 10
	}
	if params.Size < 1 || params.Size > 100 {
		return nil, domain.NewInvalidInputError("query parameter 'size' must be between 1 and 100", map[string]any{"field": "size"})
	}
	return s.gateway.GetHome(ctx, dev, params)
}

func (s *StoreService) GetSubject(ctx context.Context, profile domain.DeviceProfile, subjectID int64) (*domain.ViewLayer, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if subjectID <= 0 {
		return nil, domain.NewInvalidInputError("path parameter 'subjectId' must be > 0", map[string]any{"field": "subjectId"})
	}
	return s.gateway.GetSubject(ctx, dev, subjectID)
}

func (s *StoreService) GetAppDetailByID(ctx context.Context, profile domain.DeviceProfile, params domain.AppDetailByIDParams) (*domain.AppDetail, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if params.AppID <= 0 {
		return nil, domain.NewInvalidInputError("path parameter 'appId' must be > 0", map[string]any{"field": "appId"})
	}
	if params.AppType == 0 {
		params.AppType = 1
	}
	if params.AppType != 1 && params.AppType != 2 {
		return nil, domain.NewInvalidInputError("query parameter 'appType' must be 1 (APK) or 2 (Instant)", map[string]any{"field": "appType"})
	}
	if params.Query < 0 {
		return nil, domain.NewInvalidInputError("query parameter 'query' must be >= 0", map[string]any{"field": "query"})
	}
	if params.Module == "" {
		params.Module = "home"
	}
	return s.gateway.GetAppDetailByID(ctx, dev, params)
}

func (s *StoreService) GetAppDetailByPkg(ctx context.Context, profile domain.DeviceProfile, params domain.AppDetailByPkgParams) (*domain.AppDetail, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	params.PkgName = strings.TrimSpace(params.PkgName)
	if params.PkgName == "" {
		return nil, domain.NewInvalidInputError("path parameter 'pkgName' is required", map[string]any{"field": "pkgName"})
	}
	if params.Query < 0 {
		return nil, domain.NewInvalidInputError("query parameter 'query' must be >= 0", map[string]any{"field": "query"})
	}
	if params.Module == "" {
		params.Module = "action"
	}
	return s.gateway.GetAppDetailByPkg(ctx, dev, params)
}

func (s *StoreService) GetDownloadInfo(ctx context.Context, profile domain.DeviceProfile, params domain.DownloadInfoParams) (*domain.AppDownload, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if params.AppID <= 0 {
		return nil, domain.NewInvalidInputError("path parameter 'appId' must be > 0", map[string]any{"field": "appId"})
	}
	if params.VerID <= 0 {
		return nil, domain.NewInvalidInputError("path parameter 'verId' must be > 0", map[string]any{"field": "verId"})
	}
	if params.DownloadType == "" {
		params.DownloadType = "1"
	}
	if params.DownloadType != "1" && params.DownloadType != "2" && params.DownloadType != "3" {
		return nil, domain.NewInvalidInputError("query parameter 'type' must be \"1\" (normal), \"2\" (upgrade), or \"3\" (auto)", map[string]any{"field": "type"})
	}
	if params.Retry == 0 {
		params.Retry = 2
	}
	if params.Retry < 1 || params.Retry > 10 {
		return nil, domain.NewInvalidInputError("query parameter 'retry' must be between 1 and 10", map[string]any{"field": "retry"})
	}
	return s.gateway.GetDownloadInfo(ctx, dev, params)
}

func (s *StoreService) CheckUpdates(ctx context.Context, profile domain.DeviceProfile, apps []domain.InstalledApp) (*domain.UpgradeResult, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return nil, domain.NewInvalidInputError("'apps' must contain at least one item", map[string]any{"field": "apps"})
	}
	if len(apps) > 200 {
		return nil, domain.NewInvalidInputError("'apps' batch size must not exceed 200", map[string]any{"field": "apps"})
	}
	cleaned := make([]domain.InstalledApp, len(apps))
	for i, item := range apps {
		pkg := strings.TrimSpace(item.PkgName)
		vc := strings.TrimSpace(item.VerCode)
		if pkg == "" || vc == "" {
			return nil, domain.NewInvalidInputError(
				fmt.Sprintf("apps[%d] requires non-empty pkgName and verCode", i),
				map[string]any{"index": i},
			)
		}
		cleaned[i] = domain.InstalledApp{PkgName: pkg, VerCode: vc}
	}
	return s.gateway.CheckUpdates(ctx, dev, cleaned)
}

func (s *StoreService) GetProtocol(ctx context.Context, profile domain.DeviceProfile, language string) (*domain.Protocol, error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	lang := strings.TrimSpace(language)
	if lang == "" {
		lang = "zh_CN"
	}
	return s.gateway.GetProtocol(ctx, dev, lang)
}

func (s *StoreService) CreateOrder(ctx context.Context, profile domain.DeviceProfile, input domain.CreateOrderInput) (*domain.PayResponse[domain.OrderPay], error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if input.AppID <= 0 {
		return nil, domain.NewInvalidInputError("'appId' must be > 0", map[string]any{"field": "appId"})
	}
	if input.Price < 0 {
		return nil, domain.NewInvalidInputError("'price' must be >= 0", map[string]any{"field": "price"})
	}
	if strings.TrimSpace(input.ProductName) == "" || strings.TrimSpace(input.ProductDesc) == "" {
		return nil, domain.NewInvalidInputError("'productName' and 'productDesc' are required", nil)
	}
	// Keep header token and body token synchronized as required by /watch/createOrder.
	if input.Token == "" && dev.Token != "" {
		input.Token = dev.Token
	} else if dev.Token == "" && input.Token != "" {
		dev.Token = input.Token
	}

	resp, err := s.gateway.CreateOrder(ctx, dev, input)
	if err != nil {
		return nil, err
	}
	return evaluatePayResponse("createOrder", resp)
}

func (s *StoreService) QueryOrder(ctx context.Context, profile domain.DeviceProfile, input domain.QueryOrderInput) (*domain.PayResponse[domain.OrderStatus], error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	input.OrderID = strings.TrimSpace(input.OrderID)
	input.UserID = strings.TrimSpace(input.UserID)
	if input.OrderID == "" {
		return nil, domain.NewInvalidInputError("path parameter 'orderId' is required", map[string]any{"field": "orderId"})
	}
	if input.UserID == "" {
		return nil, domain.NewInvalidInputError("query parameter 'userId' is required", map[string]any{"field": "userId"})
	}
	if input.AppID <= 0 {
		return nil, domain.NewInvalidInputError("query parameter 'appId' must be > 0", map[string]any{"field": "appId"})
	}
	if dev.Token == "" && input.Token != "" {
		dev.Token = input.Token
	}

	resp, err := s.gateway.QueryOrder(ctx, dev, input)
	if err != nil {
		return nil, err
	}
	return evaluatePayResponse("queryOrder", resp)
}

func (s *StoreService) QueryPayStatus(ctx context.Context, profile domain.DeviceProfile, appIDs []int64) (*domain.PayResponse[domain.OrderStatusList], error) {
	dev, err := normalizeProfile(profile)
	if err != nil {
		return nil, err
	}
	if len(appIDs) == 0 {
		return nil, domain.NewInvalidInputError("query parameter 'appIds' is required", map[string]any{"field": "appIds"})
	}
	if len(appIDs) > 100 {
		return nil, domain.NewInvalidInputError("at most 100 appIds may be queried per request", map[string]any{"field": "appIds"})
	}
	for _, id := range appIDs {
		if id <= 0 {
			return nil, domain.NewInvalidInputError("all appIds must be positive integers", map[string]any{"field": "appIds", "value": id})
		}
	}

	resp, err := s.gateway.QueryPayStatus(ctx, dev, appIDs)
	if err != nil {
		return nil, err
	}
	return evaluatePayResponse("queryPayStatus", resp)
}

func evaluatePayResponse[T any](op string, resp *domain.PayResponse[T]) (*domain.PayResponse[T], error) {
	if resp == nil {
		return nil, domain.NewUpstreamError(fmt.Sprintf("%s: empty upstream pay response", op), nil, nil)
	}
	switch resp.Code {
	case "200":
		return resp, nil
	case "302":
		msg := resp.Msg
		if msg == "" {
			msg = "HeyTap token validation failed"
		}
		return nil, domain.NewUnauthorizedError(msg, map[string]any{
			"upstreamCode":    resp.Code,
			"upstreamMessage": resp.Msg,
		})
	default:
		msg := resp.Msg
		if msg == "" {
			msg = fmt.Sprintf("%s returned upstream code %q", op, resp.Code)
		}
		return nil, domain.NewUpstreamError(msg, nil, map[string]any{
			"upstreamCode":    resp.Code,
			"upstreamMessage": resp.Msg,
		})
	}
}

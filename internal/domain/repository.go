package domain

import "context"

// StoreGateway abstracts upstream calls to OPPO Watch Store (watch-cn.store.heytapmobi.com).
type StoreGateway interface {
	Search(ctx context.Context, profile DeviceProfile, params SearchParams) (*SearchResult, error)
	GetHome(ctx context.Context, profile DeviceProfile, params HomeParams) (*ViewLayer, error)
	GetSubject(ctx context.Context, profile DeviceProfile, subjectID int64) (*ViewLayer, error)
	GetAppDetailByID(ctx context.Context, profile DeviceProfile, params AppDetailByIDParams) (*AppDetail, error)
	GetAppDetailByPkg(ctx context.Context, profile DeviceProfile, params AppDetailByPkgParams) (*AppDetail, error)
	GetDownloadInfo(ctx context.Context, profile DeviceProfile, params DownloadInfoParams) (*AppDownload, error)
	CheckUpdates(ctx context.Context, profile DeviceProfile, apps []InstalledApp) (*UpgradeResult, error)
	GetProtocol(ctx context.Context, profile DeviceProfile, language string) (*Protocol, error)
	CreateOrder(ctx context.Context, profile DeviceProfile, input CreateOrderInput) (*PayResponse[OrderPay], error)
	QueryOrder(ctx context.Context, profile DeviceProfile, input QueryOrderInput) (*PayResponse[OrderStatus], error)
	QueryPayStatus(ctx context.Context, profile DeviceProfile, appIDs []int64) (*PayResponse[OrderStatusList], error)
}

// HealthGateway abstracts upstream calls to OPPO Health Cloud (sport.health.heytapmobi.com).
type HealthGateway interface {
	QuerySwitchStatus(ctx context.Context, profile DeviceProfile, switchType int) (*SwitchStatusResponse, error)
	CheckPay(ctx context.Context, profile DeviceProfile, input CheckPayInput) (*CheckPayResponse, error)
	ExchangeSymmetricKey(ctx context.Context, profile DeviceProfile, encryptionAlgorithm int, firstExchangeKey int) (*ExchangeKeyResponse, error)
}

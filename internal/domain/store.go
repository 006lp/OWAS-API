package domain

// UnknownField captures an unmapped Protostuff wire field for forward compatibility.
type UnknownField struct {
	Tag      uint64 `json:"tag"`
	WireType uint8  `json:"wire_type"`
	Value    any    `json:"value"`
}

// Resource corresponds to com.heytap.appstore.watch.domain.ResourceDto / BaseDetailDto.
type Resource struct {
	AppID            int64          `json:"appId"`
	VerID            int64          `json:"verId"`
	AppName          string         `json:"appName"`
	PkgName          string         `json:"pkgName"`
	VerName          string         `json:"verName"`
	VerCode          int64          `json:"verCode"`
	IconURL          string         `json:"iconUrl,omitempty"`
	Size             int64          `json:"size"`
	SizeDesc         string         `json:"sizeDesc,omitempty"`
	DlCount          int64          `json:"dlCount"`
	DlDesc           string         `json:"dlDesc,omitempty"`
	URL              string         `json:"url,omitempty"`
	Desc             string         `json:"desc,omitempty"`
	Screenshots      []string       `json:"screenshots,omitempty"`
	MD5              string         `json:"md5,omitempty"`
	Checksum         string         `json:"checksum,omitempty"`
	ShortDesc        string         `json:"shortDesc,omitempty"`
	Ext              map[string]any `json:"ext,omitempty"`
	Stat             map[string]any `json:"stat,omitempty"`
	OnlineTime       int64          `json:"onlineTime,omitempty"`
	Type             int32          `json:"type"`
	EnglishName      string         `json:"englishName,omitempty"`
	PrivacyPolicy    string         `json:"privacyPolicy,omitempty"`
	PrivacyRights    string         `json:"privacyRights,omitempty"`
	VersionExplain   string         `json:"versionExplain,omitempty"`
	IPhoneLink       int32          `json:"iphoneLink,omitempty"`
	IPhoneLinkDetail string         `json:"iphoneLinkDetail,omitempty"`
	QuickAppURL      string         `json:"quickAppUrl,omitempty"`
	AgeRating        string         `json:"ageRating,omitempty"`
	Permission       string         `json:"permission,omitempty"`
	Payment          int32          `json:"payment"`
	Price            int32          `json:"price"`
	IsBought         int32          `json:"isBought"`
	UnknownFields    []UnknownField `json:"_unknown_fields,omitempty"`
}

// SearchResult corresponds to com.heytap.appstore.watch.domain.SearchResultDto.
type SearchResult struct {
	Apps          []Resource     `json:"apps"`
	AllMatchApp   string         `json:"allMatchApp,omitempty"`
	IsEnd         int32          `json:"isEnd"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// Developer corresponds to com.heytap.appstore.watch.domain.DeveloperDto.
type Developer struct {
	ReleaseTime   int64          `json:"releaseTime,omitempty"`
	PkgPermiss    string         `json:"pkgPermiss,omitempty"`
	Developer     string         `json:"developer,omitempty"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// CategoryPermission corresponds to com.heytap.appstore.watch.domain.CategoryPermissionDto.
type CategoryPermission struct {
	BasePermission    []string       `json:"basePermission"`
	PrivacyPermission []string       `json:"privacyPermission"`
	UnknownFields     []UnknownField `json:"_unknown_fields,omitempty"`
}

// AppDetail corresponds to com.heytap.appstore.watch.domain.AppDetailDto.
type AppDetail struct {
	Base          *Resource           `json:"base,omitempty"`
	Developer     *Developer          `json:"developer,omitempty"`
	PermissionDto *CategoryPermission `json:"permissionDto,omitempty"`
	UnknownFields []UnknownField      `json:"_unknown_fields,omitempty"`
}

// AppDownload corresponds to com.heytap.appstore.watch.domain.AppDownloadDto.
type AppDownload struct {
	AppID         int64          `json:"appId"`
	VerID         int64          `json:"verId"`
	DownURL       string         `json:"downUrl"`
	MD5           string         `json:"md5"`
	HeaderMD5     string         `json:"headerMd5"`
	PkgName       string         `json:"pkgName"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// Banner corresponds to com.heytap.appstore.watch.domain.v2.BannerDto.
type Banner struct {
	ID            int64          `json:"id"`
	Image         string         `json:"image,omitempty"`
	ImageWidth    int32          `json:"imageWidth,omitempty"`
	ImageHeight   int32          `json:"imageHeight,omitempty"`
	ActionParam   string         `json:"actionParam,omitempty"`
	Title         string         `json:"title,omitempty"`
	Desc          string         `json:"desc,omitempty"`
	StartTime     int64          `json:"startTime,omitempty"`
	EndTime       int64          `json:"endTime,omitempty"`
	Sort          int32          `json:"sort,omitempty"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// Card represents polymorphic com.heytap.appstore.watch.domain.v2.CardDto subclasses
// (NavCardDto, BannerCardDto, ResourceListCardDto, CardDto).
type Card struct {
	Type          string         `json:"_type,omitempty"`
	Code          int32          `json:"code"`
	Key           int64          `json:"key"`
	ActionParam   string         `json:"actionParam,omitempty"`
	CornerLabel   int32          `json:"cornerLabel,omitempty"`
	Ext           map[string]any `json:"ext,omitempty"`
	Stat          map[string]any `json:"stat,omitempty"`
	Title         string         `json:"title,omitempty"`
	Desc          string         `json:"desc,omitempty"`
	BgImageURL    string         `json:"bgImageUrl,omitempty"`
	Banners       []Banner       `json:"banners,omitempty"`
	Items         []Banner       `json:"items,omitempty"`
	Apps          []Resource     `json:"apps,omitempty"`
	Resources     []Resource     `json:"resources,omitempty"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// ViewLayer corresponds to com.heytap.appstore.watch.domain.v2.ViewLayerDto.
type ViewLayer struct {
	Title         string         `json:"title,omitempty"`
	PageKey       int32          `json:"pageKey"`
	Cards         []Card         `json:"cards"`
	IsEnd         int32          `json:"isEnd"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// InstalledApp identifies an installed package and versionCode for update checking.
type InstalledApp struct {
	PkgName string `json:"pkgName" binding:"required"`
	VerCode string `json:"verCode" binding:"required"`
}

// UpgradeItem corresponds to com.heytap.appstore.watch.domain.UpgradeDto.
type UpgradeItem struct {
	Resource
	UpdateDesc  string `json:"updateDesc,omitempty"`
	UpgradeFlag int32  `json:"upgradeFlag"`
}

// UpgradeResult corresponds to com.heytap.appstore.watch.domain.UrgradeWrapDto.
type UpgradeResult struct {
	Upgrades      []UpgradeItem  `json:"upgrades"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// Protocol corresponds to ProtocolResult (GET /watch/protocol).
type Protocol struct {
	Privacy        string         `json:"privacy"`
	Agreement      string         `json:"agreement"`
	PrivacyVersion int32          `json:"privacyVersion"`
	UnknownFields  []UnknownField `json:"_unknown_fields,omitempty"`
}

// CreateOrderInput represents parameters for POST /watch/createOrder.
type CreateOrderInput struct {
	AppID       int64  `json:"appId" binding:"required,gt=0"`
	Price       int32  `json:"price" binding:"gte=0"`
	ProductName string `json:"productName" binding:"required"`
	ProductDesc string `json:"productDesc" binding:"required"`
	Token       string `json:"token,omitempty"`
	Model       string `json:"model,omitempty"`
	IP          string `json:"ip,omitempty"`
}

// OrderPay corresponds to OrderPayVO.
type OrderPay struct {
	OrderID       string         `json:"orderId"`
	PayURL        string         `json:"payUrl"`
	ExpireTime    string         `json:"expireTime"`
	PrePayToken   string         `json:"prePayToken"`
	PartnerCode   string         `json:"partnerCode"`
	UserID        string         `json:"userId"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// QueryOrderInput represents parameters for POST /watch/queryOrder.
type QueryOrderInput struct {
	OrderID string `json:"orderId"`
	UserID  string `json:"userId"`
	AppID   int64  `json:"appId"`
	Token   string `json:"token,omitempty"`
	Model   string `json:"model,omitempty"`
	IP      string `json:"ip,omitempty"`
}

// OrderStatus corresponds to OrderStatusVO.
type OrderStatus struct {
	AppID         int64          `json:"appId"`
	PayStatus     int32          `json:"payStatus"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// OrderStatusList corresponds to OrderStatusList.
type OrderStatusList struct {
	StatusVOList  []OrderStatus  `json:"statusVOList"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// PayResponse corresponds to generic PayCommonResponse<T> (p177z2.C3153h).
type PayResponse[T any] struct {
	Code          string         `json:"code"`
	Msg           string         `json:"msg"`
	Data          *T             `json:"data,omitempty"`
	UnknownFields []UnknownField `json:"_unknown_fields,omitempty"`
}

// SearchParams encapsulates parameters for searching watch apps.
type SearchParams struct {
	Word       string
	SearchType int
}

// HomeParams encapsulates pagination parameters for store home cards.
type HomeParams struct {
	Start int
	Size  int
}

// AppDetailByIDParams encapsulates parameters for fetching app detail by appId.
type AppDetailByIDParams struct {
	AppID   int64
	AppType int
	Query   int
	Module  string
}

// AppDetailByPkgParams encapsulates parameters for fetching app detail by package name.
type AppDetailByPkgParams struct {
	PkgName string
	Query   int
	Module  string
}

// DownloadInfoParams encapsulates parameters for fetching APK download info.
type DownloadInfoParams struct {
	AppID        int64
	VerID        int64
	DownloadType string
	Retry        int
}

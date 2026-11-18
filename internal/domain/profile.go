package domain

import "fmt"

// AppStore protocol constants extracted from com.heytap.wearable.market (8.0.0.23).
const (
	AppStoreBaseURL     = "https://watch-cn.store.heytapmobi.com"
	HealthCloudBaseURL  = "https://sport.health.heytapmobi.com"
	AppStoreDeviceID    = "000000000000000"
	AppStoreChannel     = "2230"
	AppStoreVersionCode = "8000023"
)

// WatchType values controlling upstream screen-shape filtering.
const (
	WatchTypeSquare = "1"
	WatchTypeRound  = "2"
)

// Support values controlling upstream phone-pairing filtering.
const (
	SupportAndroidOPPO   = "android-oppo"
	SupportAndroidOthers = "android-others"
	SupportIOS           = "ios"
)

// DeviceProfile represents simulated OPPO Watch parameters sent in upstream headers.
type DeviceProfile struct {
	Brand          string
	Product        string
	Model          string
	SDKInt         int
	AndroidRelease string
	HeyRelease     string
	ColorOSVer     string
	SysVer         string
	Locale         string
	Support        string // android-oppo | android-others | ios
	Linkage        string // 0=family_device_pair_type==1, 1=normal
	PlatCode       int    // com.heytap.wearable.instant versionCode
	InstantSwClose string // 0=instant enabled, 1=closed
	WatchType      string // 2=round screen, 1=square screen
	Token          string // optional HeyTap account accessToken
}

// DefaultDeviceProfile returns the standard round-watch android-oppo device profile.
func DefaultDeviceProfile() DeviceProfile {
	return DeviceProfile{
		Brand:          "OPPO",
		Product:        "OWW251",
		Model:          "OWW251",
		SDKInt:         30,
		AndroidRelease: "11",
		HeyRelease:     "4.0",
		ColorOSVer:     "16",
		SysVer:         "OWW251_11_A.319",
		Locale:         "zh-CN;CN",
		Support:        SupportAndroidOPPO,
		Linkage:        "1",
		PlatCode:       0,
		InstantSwClose: "0",
		WatchType:      WatchTypeRound,
		Token:          "",
	}
}

// WithDefaults returns a copy of p with any empty/zero fields populated from defaults.
func (p DeviceProfile) WithDefaults() DeviceProfile {
	def := DefaultDeviceProfile()
	if p.Brand == "" {
		p.Brand = def.Brand
	}
	if p.Product == "" {
		p.Product = def.Product
	}
	if p.Model == "" {
		p.Model = def.Model
	}
	if p.SDKInt == 0 {
		p.SDKInt = def.SDKInt
	}
	if p.AndroidRelease == "" {
		p.AndroidRelease = def.AndroidRelease
	}
	if p.HeyRelease == "" {
		p.HeyRelease = def.HeyRelease
	}
	if p.ColorOSVer == "" {
		p.ColorOSVer = def.ColorOSVer
	}
	if p.SysVer == "" {
		p.SysVer = def.SysVer
	}
	if p.Locale == "" {
		p.Locale = def.Locale
	}
	if p.Support == "" {
		p.Support = def.Support
	}
	if p.Linkage == "" {
		p.Linkage = def.Linkage
	}
	if p.InstantSwClose == "" {
		p.InstantSwClose = def.InstantSwClose
	}
	if p.WatchType == "" {
		p.WatchType = def.WatchType
	}
	return p
}

// Validate checks device profile invariants before calling upstream APIs.
func (p DeviceProfile) Validate() error {
	if p.WatchType != WatchTypeSquare && p.WatchType != WatchTypeRound {
		return NewInvalidInputError(
			fmt.Sprintf("invalid watch-type %q: expected %q (square) or %q (round)", p.WatchType, WatchTypeSquare, WatchTypeRound),
			map[string]any{"field": "watchType", "value": p.WatchType},
		)
	}
	switch p.Support {
	case SupportAndroidOPPO, SupportAndroidOthers, SupportIOS:
	default:
		return NewInvalidInputError(
			fmt.Sprintf("invalid support %q: expected android-oppo, android-others, or ios", p.Support),
			map[string]any{"field": "support", "value": p.Support},
		)
	}
	if p.Linkage != "0" && p.Linkage != "1" {
		return NewInvalidInputError(
			fmt.Sprintf("invalid linkage %q: expected \"0\" or \"1\"", p.Linkage),
			map[string]any{"field": "linkage", "value": p.Linkage},
		)
	}
	if p.InstantSwClose != "0" && p.InstantSwClose != "1" {
		return NewInvalidInputError(
			fmt.Sprintf("invalid instantSwClose %q: expected \"0\" or \"1\"", p.InstantSwClose),
			map[string]any{"field": "instantSwClose", "value": p.InstantSwClose},
		)
	}
	if p.PlatCode < 0 {
		return NewInvalidInputError(
			"invalid platCode: must be non-negative",
			map[string]any{"field": "platCode", "value": p.PlatCode},
		)
	}
	if p.SDKInt <= 0 {
		return NewInvalidInputError(
			"invalid sdkInt: must be positive",
			map[string]any{"field": "sdkInt", "value": p.SDKInt},
		)
	}
	return nil
}

// UserAgent constructs the User-Agent string matching com.heytap.wearable.market.C1148a.m2973d().
func (p DeviceProfile) UserAgent() string {
	dev := p.WithDefaults()
	return fmt.Sprintf(
		"%s/%s/%d/%s/%s/20192/%s/%s",
		dev.Brand,
		dev.Product,
		dev.SDKInt,
		dev.AndroidRelease,
		dev.HeyRelease,
		AppStoreChannel,
		AppStoreVersionCode,
	)
}

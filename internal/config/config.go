package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"owas-api/internal/domain"
)

// Config holds all runtime settings for the OWAS-API service.
type Config struct {
	Server         ServerConfig
	Upstream       UpstreamConfig
	DefaultProfile domain.DeviceProfile
}

// ServerConfig controls the HTTP listener and Gin runtime behavior.
type ServerConfig struct {
	Host            string
	Port            int
	GinMode         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Addr returns the host:port bind address.
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// UpstreamConfig controls upstream OPPO Watch Store & Health Cloud HTTP clients.
type UpstreamConfig struct {
	AppStoreBaseURL    string
	HealthCloudBaseURL string
	Timeout            time.Duration
}

// Load reads configuration from environment variables with sane defaults.
func Load() (Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

// LoadFromLookup reads configuration using the provided env lookup function (for testability).
func LoadFromLookup(lookup func(string) (string, bool)) (Config, error) {
	port, err := getIntEnv(lookup, "OWAS_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := getDurationEnv(lookup, "OWAS_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := getDurationEnv(lookup, "OWAS_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := getDurationEnv(lookup, "OWAS_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := getDurationEnv(lookup, "OWAS_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	upstreamTimeout, err := getDurationEnv(lookup, "OWAS_UPSTREAM_TIMEOUT", 20*time.Second)
	if err != nil {
		return Config{}, err
	}

	def := domain.DefaultDeviceProfile()
	sdkInt, err := getIntEnv(lookup, "OWAS_DEFAULT_SDK_INT", def.SDKInt)
	if err != nil {
		return Config{}, err
	}
	platCode, err := getIntEnv(lookup, "OWAS_DEFAULT_PLAT_CODE", def.PlatCode)
	if err != nil {
		return Config{}, err
	}

	profile := domain.DeviceProfile{
		Brand:          getStringEnv(lookup, "OWAS_DEFAULT_BRAND", def.Brand),
		Product:        getStringEnv(lookup, "OWAS_DEFAULT_PRODUCT", def.Product),
		Model:          getStringEnv(lookup, "OWAS_DEFAULT_MODEL", def.Model),
		SDKInt:         sdkInt,
		AndroidRelease: getStringEnv(lookup, "OWAS_DEFAULT_ANDROID_RELEASE", def.AndroidRelease),
		HeyRelease:     getStringEnv(lookup, "OWAS_DEFAULT_HEY_RELEASE", def.HeyRelease),
		ColorOSVer:     getStringEnv(lookup, "OWAS_DEFAULT_COLOR_OS_VER", def.ColorOSVer),
		SysVer:         getStringEnv(lookup, "OWAS_DEFAULT_SYS_VER", def.SysVer),
		Locale:         getStringEnv(lookup, "OWAS_DEFAULT_LOCALE", def.Locale),
		Support:        getStringEnv(lookup, "OWAS_DEFAULT_SUPPORT", def.Support),
		Linkage:        getStringEnv(lookup, "OWAS_DEFAULT_LINKAGE", def.Linkage),
		PlatCode:       platCode,
		InstantSwClose: getStringEnv(lookup, "OWAS_DEFAULT_INSTANT_SW_CLOSE", def.InstantSwClose),
		WatchType:      getStringEnv(lookup, "OWAS_DEFAULT_WATCH_TYPE", def.WatchType),
		Token:          getStringEnv(lookup, "OWAS_DEFAULT_TOKEN", def.Token),
	}.WithDefaults()

	if err := profile.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid default device profile: %w", err)
	}

	cfg := Config{
		Server: ServerConfig{
			Host:            getStringEnv(lookup, "OWAS_HOST", "0.0.0.0"),
			Port:            port,
			GinMode:         getStringEnv(lookup, "GIN_MODE", "release"),
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			IdleTimeout:     idleTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Upstream: UpstreamConfig{
			AppStoreBaseURL:    strings.TrimRight(getStringEnv(lookup, "OWAS_APPSTORE_BASE_URL", domain.AppStoreBaseURL), "/"),
			HealthCloudBaseURL: strings.TrimRight(getStringEnv(lookup, "OWAS_HEALTH_CLOUD_BASE_URL", domain.HealthCloudBaseURL), "/"),
			Timeout:            upstreamTimeout,
		},
		DefaultProfile: profile,
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return Config{}, fmt.Errorf("OWAS_PORT must be between 1 and 65535, got %d", cfg.Server.Port)
	}
	if cfg.Upstream.Timeout <= 0 {
		return Config{}, fmt.Errorf("OWAS_UPSTREAM_TIMEOUT must be positive, got %v", cfg.Upstream.Timeout)
	}

	return cfg, nil
}

func getStringEnv(lookup func(string) (string, bool), key, fallback string) string {
	if v, ok := lookup(key); ok {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func getIntEnv(lookup func(string) (string, bool), key string, fallback int) (int, error) {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s=%q: %w", key, v, err)
	}
	return parsed, nil
}

func getDurationEnv(lookup func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s=%q: %w", key, v, err)
	}
	return parsed, nil
}

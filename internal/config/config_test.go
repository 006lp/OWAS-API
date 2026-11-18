package config

import (
	"testing"
	"time"
)

func mapLookup(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func TestLoadFromLookupDefaults(t *testing.T) {
	cfg, err := LoadFromLookup(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Addr() != "0.0.0.0:8080" {
		t.Fatalf("expected addr 0.0.0.0:8080, got %s", cfg.Server.Addr())
	}
	if cfg.Upstream.Timeout != 20*time.Second {
		t.Fatalf("expected 20s upstream timeout, got %v", cfg.Upstream.Timeout)
	}
	if cfg.DefaultProfile.WatchType != "2" || cfg.DefaultProfile.Support != "android-oppo" {
		t.Fatalf("unexpected default profile: %+v", cfg.DefaultProfile)
	}
}

func TestLoadFromLookupOverrides(t *testing.T) {
	cfg, err := LoadFromLookup(mapLookup(map[string]string{
		"OWAS_HOST":               "127.0.0.1",
		"OWAS_PORT":               "9090",
		"GIN_MODE":                "debug",
		"OWAS_UPSTREAM_TIMEOUT":   "12s",
		"OWAS_DEFAULT_WATCH_TYPE": "1",
		"OWAS_DEFAULT_SUPPORT":    "ios",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Addr() != "127.0.0.1:9090" {
		t.Fatalf("expected 127.0.0.1:9090, got %s", cfg.Server.Addr())
	}
	if cfg.Server.GinMode != "debug" {
		t.Fatalf("expected debug mode, got %s", cfg.Server.GinMode)
	}
	if cfg.Upstream.Timeout != 12*time.Second {
		t.Fatalf("expected 12s timeout, got %v", cfg.Upstream.Timeout)
	}
	if cfg.DefaultProfile.WatchType != "1" || cfg.DefaultProfile.Support != "ios" {
		t.Fatalf("unexpected overridden profile: %+v", cfg.DefaultProfile)
	}
}

func TestLoadFromLookupInvalid(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{"bad port", map[string]string{"OWAS_PORT": "99999"}},
		{"non-int port", map[string]string{"OWAS_PORT": "abc"}},
		{"bad duration", map[string]string{"OWAS_UPSTREAM_TIMEOUT": "not-a-duration"}},
		{"bad watch type", map[string]string{"OWAS_DEFAULT_WATCH_TYPE": "3"}},
		{"bad support", map[string]string{"OWAS_DEFAULT_SUPPORT": "windows"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := LoadFromLookup(mapLookup(tc.env)); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

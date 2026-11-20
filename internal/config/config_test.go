package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOptionalWithoutFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := LoadOptional()
	if err != nil || cfg.APIKey != "" {
		t.Fatalf("LoadOptional = %+v, %v", cfg, err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("Load without a token must fail")
	}
}

func TestUnknownKeysIgnoredAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "octa", configFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := "api_key: octa_ak_test\nunknown_key: 42\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg.APIKey != "octa_ak_test" {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
	cfg.APIURL = "https://example.test"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	again, err := Load()
	if err != nil || *again != *cfg {
		t.Fatalf("round trip = %+v, %v; want %+v", again, err, cfg)
	}
}

func TestBaseURLPrecedence(t *testing.T) {
	t.Setenv(EnvAPIURL, "")
	if got := (&Config{}).BaseURL(); got != DefaultAPIURL {
		t.Errorf("default URL = %q", got)
	}
	cfg := &Config{APIURL: " https://from-config.test "}
	if got := cfg.BaseURL(); got != "https://from-config.test" {
		t.Errorf("config URL = %q", got)
	}
	t.Setenv(EnvAPIURL, "https://from-env.test")
	if got := cfg.BaseURL(); got != "https://from-env.test" {
		t.Errorf("env URL = %q", got)
	}
	if _, source := cfg.BaseURLSource(); source != "$OCTA_API_URL" {
		t.Errorf("env source = %q", source)
	}
	t.Setenv(EnvAPIURL, "")
	if _, source := cfg.BaseURLSource(); source != "config" {
		t.Errorf("config source = %q", source)
	}
	if _, source := (&Config{}).BaseURLSource(); source != "default" {
		t.Errorf("default source = %q", source)
	}
}

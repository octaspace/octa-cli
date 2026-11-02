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
	cfg := &Config{APIURL: " https://from-config.test "}
	t.Setenv(EnvAPIURL, "")
	if got := cfg.BaseURL(); got != "https://from-config.test" {
		t.Errorf("config URL = %q", got)
	}
	t.Setenv(EnvAPIURL, "https://from-env.test")
	if got := cfg.BaseURL(); got != "https://from-env.test" {
		t.Errorf("env URL = %q", got)
	}
}

func TestLoadRejectsLegacyToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "octa", configFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := "api_key: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err != ErrLegacyToken {
		t.Fatalf("Load with a legacy token = %v, want ErrLegacyToken", err)
	}
}

func TestLegacyConfigFileNotRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "octa", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("api_key: octa_ak_main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadOptional()
	if err != nil || cfg.APIKey != "" {
		t.Fatalf("the legacy config.yaml must not be read: %+v, %v", cfg, err)
	}
}

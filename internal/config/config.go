package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// configFile is the name of the config file in $XDG_CONFIG_HOME/octa.
const configFile = "next.yaml"

// EnvAPIURL overrides the API base URL (config key api_url). Empty uses the
// SDK default.
const EnvAPIURL = "OCTA_API_URL"

// APIKeyPrefix starts every API key the API accepts: keys issued in
// OctaSpace under Profile → Credentials.
const APIKeyPrefix = "octa_ak_"

// errNoToken is returned by Load when no API key is configured.
var errNoToken = errors.New("run 'octa auth <key>' first")

// ErrLegacyToken is returned when the saved token is not an API key: the CLI
// never sends it.
var ErrLegacyToken = errors.New("the saved token is not an API key and is not used: " +
	"create an API key (octa_ak_…) in OctaSpace under Profile → Credentials and run 'octa auth <key>'")

// IsAPIKey reports whether key is an API key (octa_ak_…).
func IsAPIKey(key string) bool { return strings.HasPrefix(key, APIKeyPrefix) }

// Config holds the application configuration.
type Config struct {
	APIKey string `yaml:"api_key"`
	// APIURL overrides the API base URL; empty uses the SDK default.
	APIURL string `yaml:"api_url,omitempty"`
}

// BaseURL returns the API base URL to use: $OCTA_API_URL, then api_url from
// the config file, then "" for the SDK default.
func (c *Config) BaseURL() string {
	if v := strings.TrimSpace(os.Getenv(EnvAPIURL)); v != "" {
		return v
	}
	return strings.TrimSpace(c.APIURL)
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	configBase := os.Getenv("XDG_CONFIG_HOME")
	if configBase == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		configBase = filepath.Join(home, ".config")
	}
	return filepath.Join(configBase, "octa", configFile), nil
}

// Load reads the config file and returns the Config. It fails when no API key
// is configured, or when the saved token is not an API key, so that such a
// token is never sent. Commands that also work anonymously use LoadOptional.
func Load() (*Config, error) {
	cfg, err := LoadOptional()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, errNoToken
	}
	if !IsAPIKey(cfg.APIKey) {
		return nil, ErrLegacyToken
	}
	return cfg, nil
}

// LoadOptional reads the config file like Load, but a missing file or API key
// is not an error: it returns a Config with an empty APIKey, for the endpoints
// that accept anonymous requests (the compute and render listings).
func LoadOptional() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to the config file, creating the directory if needed.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

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
const configFile = "config.yaml"

// EnvAPIURL overrides the API base URL (config key api_url).
const EnvAPIURL = "OCTA_API_URL"

// DefaultAPIURL is the API base URL used when neither $OCTA_API_URL nor
// api_url is set.
const DefaultAPIURL = "https://api.octa.space"

// APIKeyPrefix starts every API key the new API accepts: keys issued in
// OctaSpace under Profile → Credentials.
const APIKeyPrefix = "octa_ak_"

// errNoToken is returned by Load when no API key is configured.
var errNoToken = errors.New("run 'octa auth <key>' first")

// IsAPIKey reports whether key looks like an API key (octa_ak_…).
func IsAPIKey(key string) bool { return strings.HasPrefix(key, APIKeyPrefix) }

// Config holds the application configuration.
type Config struct {
	APIKey string `yaml:"api_key"`
	// APIURL overrides the API base URL; empty uses DefaultAPIURL.
	APIURL string `yaml:"api_url,omitempty"`
}

// BaseURL returns the API base URL to use: $OCTA_API_URL, then api_url from
// the config file, then DefaultAPIURL.
func (c *Config) BaseURL() string {
	url, _ := c.BaseURLSource()
	return url
}

// BaseURLSource returns the API base URL like BaseURL, and where it comes
// from: "$OCTA_API_URL", "config" or "default".
func (c *Config) BaseURLSource() (url, source string) {
	if v := strings.TrimSpace(os.Getenv(EnvAPIURL)); v != "" {
		return v, "$" + EnvAPIURL
	}
	if v := strings.TrimSpace(c.APIURL); v != "" {
		return v, "config"
	}
	return DefaultAPIURL, "default"
}

// Path returns the full path to the config file.
func Path() (string, error) {
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
// is configured. Commands that also work anonymously use LoadOptional.
func Load() (*Config, error) {
	cfg, err := LoadOptional()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, errNoToken
	}
	return cfg, nil
}

// LoadOptional reads the config file like Load, but a missing file or API key
// is not an error: it returns a Config with an empty APIKey, for the endpoints
// that accept anonymous requests (the compute listing).
func LoadOptional() (*Config, error) {
	path, err := Path()
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
	path, err := Path()
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

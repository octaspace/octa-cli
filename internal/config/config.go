package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFile = "config.yaml"

// Config holds the application configuration.
type Config struct {
	APIKey          string `yaml:"api_key"`
	VPNRelayNode    int    `yaml:"vpn_relay_node,omitempty"`
	VPNRelayCountry string `yaml:"vpn_relay_country,omitempty"`
	VPNRelayCity    string `yaml:"vpn_relay_city,omitempty"`
	VPNSessionUUID  string `yaml:"vpn_session_uuid,omitempty"`
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

// Load reads the config file and returns the Config.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("run 'octa auth <token>' first")
		}
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("run 'octa auth <token>' first")
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

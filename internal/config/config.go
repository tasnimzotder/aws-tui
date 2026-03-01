package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds optional defaults loaded from ~/.config/aws-tui/config.yaml.
type Config struct {
	DefaultProfile      string `yaml:"default_profile"`
	DefaultRegion       string `yaml:"default_region"`
	AutoRefreshInterval int    `yaml:"auto_refresh_interval"`
}

// RefreshInterval returns the auto-refresh duration, with a minimum of 5s and default of 15s.
func (c *Config) RefreshInterval() time.Duration {
	s := c.AutoRefreshInterval
	if s <= 0 {
		s = 15
	}
	if s < 5 {
		s = 5
	}
	return time.Duration(s) * time.Second
}

// Load reads the config file. Returns zero-value Config if the file doesn't exist.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Config{}, nil
	}

	path := filepath.Join(home, ".config", "aws-tui", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Merge applies CLI flag overrides. Flags take precedence over config defaults.
func (c *Config) Merge(profile, region string) (string, string) {
	p := c.DefaultProfile
	if profile != "" {
		p = profile
	}
	r := c.DefaultRegion
	if region != "" {
		r = region
	}
	return p, r
}

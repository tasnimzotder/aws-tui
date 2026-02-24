package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds optional defaults loaded from ~/.config/aws-tui/config.yaml.
type Config struct {
	DefaultProfile string `yaml:"default_profile"`
	DefaultRegion  string `yaml:"default_region"`
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

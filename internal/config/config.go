package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultProfile      string `yaml:"default_profile"`
	DefaultRegion       string `yaml:"default_region"`
	AutoRefreshInterval int    `yaml:"auto_refresh_interval"`
	LastRegion          string `yaml:"last_region,omitempty"`
	LastProfile         string `yaml:"last_profile,omitempty"`

	path string `yaml:"-"`
}

func Load(path string) (Config, error) {
	cfg := Config{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return applyDefaults(cfg), nil
		}
		return cfg, err
	}

	if len(data) == 0 {
		return applyDefaults(cfg), nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	cfg.path = path
	return applyDefaults(cfg), nil
}

func applyDefaults(cfg Config) Config {
	if cfg.AutoRefreshInterval == 0 {
		cfg.AutoRefreshInterval = 15
	}
	return cfg
}

// Save writes the config back to disk.
func (c *Config) Save() error {
	if c.path == "" {
		return nil
	}
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o644)
}

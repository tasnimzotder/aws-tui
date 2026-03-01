package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "", cfg.DefaultProfile)
	assert.Equal(t, "", cfg.DefaultRegion)
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("default_profile: my-profile\ndefault_region: eu-west-1\n"), 0644)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	assert.Equal(t, "my-profile", cfg.DefaultProfile)
	assert.Equal(t, "eu-west-1", cfg.DefaultRegion)
}

func TestConfig_AutoRefreshInterval(t *testing.T) {
	data := []byte("auto_refresh_interval: 30\n")
	var cfg Config
	err := yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.AutoRefreshInterval)
	assert.Equal(t, 30*time.Second, cfg.RefreshInterval())
}

func TestConfig_DefaultAutoRefreshInterval(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, 15*time.Second, cfg.RefreshInterval())
}

func TestConfig_MinAutoRefreshInterval(t *testing.T) {
	cfg := &Config{AutoRefreshInterval: 2}
	assert.Equal(t, 5*time.Second, cfg.RefreshInterval())
}

func TestMerge_CLIFlagsTakePrecedence(t *testing.T) {
	cfg := &Config{DefaultProfile: "config-profile", DefaultRegion: "us-east-1"}

	// CLI flags override
	p, r := cfg.Merge("cli-profile", "ap-south-1")
	assert.Equal(t, "cli-profile", p)
	assert.Equal(t, "ap-south-1", r)

	// Empty flags fall back to config
	p, r = cfg.Merge("", "")
	assert.Equal(t, "config-profile", p)
	assert.Equal(t, "us-east-1", r)

	// Partial override
	p, r = cfg.Merge("other", "")
	assert.Equal(t, "other", p)
	assert.Equal(t, "us-east-1", r)
}

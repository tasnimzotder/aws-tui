package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string // returns path to config file
		expected Config
	}{
		{
			name: "full config parses correctly",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				content := []byte("default_profile: production\ndefault_region: eu-west-1\nauto_refresh_interval: 30\n")
				require.NoError(t, os.WriteFile(path, content, 0644))
				return path
			},
			expected: Config{
				DefaultProfile:      "production",
				DefaultRegion:       "eu-west-1",
				AutoRefreshInterval: 30,
			},
		},
		{
			name: "empty file returns defaults",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte(""), 0644))
				return path
			},
			expected: Config{
				AutoRefreshInterval: 15,
			},
		},
		{
			name: "missing file returns defaults not error",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				return filepath.Join(dir, "nonexistent.yaml")
			},
			expected: Config{
				AutoRefreshInterval: 15,
			},
		},
		{
			name: "zero auto_refresh_interval treated as default",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				content := []byte("default_profile: dev\nauto_refresh_interval: 0\n")
				require.NoError(t, os.WriteFile(path, content, 0644))
				return path
			},
			expected: Config{
				DefaultProfile:      "dev",
				AutoRefreshInterval: 15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			cfg, err := Load(path)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.DefaultProfile, cfg.DefaultProfile)
			assert.Equal(t, tt.expected.DefaultRegion, cfg.DefaultRegion)
			assert.Equal(t, tt.expected.AutoRefreshInterval, cfg.AutoRefreshInterval)
			assert.Equal(t, tt.expected.LastRegion, cfg.LastRegion)
			assert.Equal(t, tt.expected.LastProfile, cfg.LastProfile)
		})
	}
}

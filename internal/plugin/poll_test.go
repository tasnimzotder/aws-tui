package plugin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveInterval(t *testing.T) {
	tests := []struct {
		name     string
		cfg      PollConfig
		expected time.Duration
	}{
		{
			name: "not active returns idle interval",
			cfg: PollConfig{
				IdleInterval:   30 * time.Second,
				ActiveInterval: 5 * time.Second,
				IsActive:       func() bool { return false },
			},
			expected: 30 * time.Second,
		},
		{
			name: "active returns active interval",
			cfg: PollConfig{
				IdleInterval:   30 * time.Second,
				ActiveInterval: 5 * time.Second,
				IsActive:       func() bool { return true },
			},
			expected: 5 * time.Second,
		},
		{
			name: "IsActive nil returns idle interval",
			cfg: PollConfig{
				IdleInterval:   30 * time.Second,
				ActiveInterval: 5 * time.Second,
				IsActive:       nil,
			},
			expected: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveInterval(tt.cfg)
			assert.Equal(t, tt.expected, got)
		})
	}
}

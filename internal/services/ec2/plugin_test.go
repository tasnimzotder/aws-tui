package ec2

import (
	"testing"
	"time"

	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummary_AllRunning(t *testing.T) {
	summary := awsec2.EC2Summary{Total: 2, Running: 2, Stopped: 0}
	result := mapSummary(summary)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Status["running"])
	assert.Equal(t, 0, result.Status["stopped"])
	assert.Equal(t, plugin.HealthHealthy, result.Health)
	assert.Equal(t, "instances", result.Label)
}

func TestSummary_Mixed(t *testing.T) {
	summary := awsec2.EC2Summary{Total: 5, Running: 3, Stopped: 2}
	result := mapSummary(summary)

	assert.Equal(t, 5, result.Total)
	assert.Equal(t, 3, result.Status["running"])
	assert.Equal(t, 2, result.Status["stopped"])
	assert.Equal(t, plugin.HealthWarning, result.Health)
}

func TestSummary_NoneRunning(t *testing.T) {
	summary := awsec2.EC2Summary{Total: 3, Running: 0, Stopped: 3}
	result := mapSummary(summary)

	assert.Equal(t, 3, result.Total)
	assert.Equal(t, plugin.HealthCritical, result.Health)
}

func TestSummary_Empty(t *testing.T) {
	summary := awsec2.EC2Summary{Total: 0, Running: 0, Stopped: 0}
	result := mapSummary(summary)

	assert.Equal(t, 0, result.Total)
	assert.Equal(t, plugin.HealthHealthy, result.Health)
}

func TestSummary_OtherStates(t *testing.T) {
	summary := awsec2.EC2Summary{Total: 4, Running: 2, Stopped: 1}
	result := mapSummary(summary)

	assert.Equal(t, 4, result.Total)
	assert.Equal(t, 1, result.Status["other"])
}

func TestCommands(t *testing.T) {
	p := NewPlugin(nil, "", "")
	cmds := p.Commands()

	require.Len(t, cmds, 1)
	assert.Equal(t, "List EC2 Instances", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "ec2")
	assert.Contains(t, cmds[0].Keywords, "instances")
}

func TestPollConfig(t *testing.T) {
	p := NewPlugin(nil, "", "")
	cfg := p.PollConfig()

	assert.Equal(t, 60*time.Second, cfg.IdleInterval)
	assert.Equal(t, 5*time.Second, cfg.ActiveInterval)
	assert.NotNil(t, cfg.IsActive)
}

func TestPollConfig_IsActive(t *testing.T) {
	tests := []struct {
		name      string
		instances []awsec2.EC2Instance
		want      bool
	}{
		{
			name:      "no instances",
			instances: nil,
			want:      false,
		},
		{
			name: "all running",
			instances: []awsec2.EC2Instance{
				{State: "running"},
				{State: "running"},
			},
			want: false,
		},
		{
			name: "has pending",
			instances: []awsec2.EC2Instance{
				{State: "running"},
				{State: "pending"},
			},
			want: true,
		},
		{
			name: "has stopping",
			instances: []awsec2.EC2Instance{
				{State: "stopping"},
			},
			want: true,
		},
		{
			name: "has shutting-down",
			instances: []awsec2.EC2Instance{
				{State: "shutting-down"},
			},
			want: true,
		},
		{
			name: "all stopped",
			instances: []awsec2.EC2Instance{
				{State: "stopped"},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plugin{instances: tc.instances}
			cfg := p.PollConfig()
			assert.Equal(t, tc.want, cfg.IsActive())
		})
	}
}

func TestPluginMetadata(t *testing.T) {
	p := NewPlugin(nil, "", "")
	assert.Equal(t, "ec2", p.ID())
	assert.Equal(t, "EC2", p.Name())
}

package eks

import (
	"testing"
	"time"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapSummary_AllActive(t *testing.T) {
	clusters := []awseks.EKSCluster{
		{Name: "cluster-1", Status: "ACTIVE"},
		{Name: "cluster-2", Status: "ACTIVE"},
	}
	result := mapSummary(clusters)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Status["ACTIVE"])
	assert.Equal(t, plugin.HealthHealthy, result.Health)
	assert.Equal(t, "clusters", result.Label)
}

func TestMapSummary_Mixed(t *testing.T) {
	clusters := []awseks.EKSCluster{
		{Name: "cluster-1", Status: "ACTIVE"},
		{Name: "cluster-2", Status: "CREATING"},
	}
	result := mapSummary(clusters)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 1, result.Status["ACTIVE"])
	assert.Equal(t, 1, result.Status["CREATING"])
	assert.Equal(t, plugin.HealthWarning, result.Health)
}

func TestMapSummary_Empty(t *testing.T) {
	result := mapSummary(nil)

	assert.Equal(t, 0, result.Total)
	assert.Equal(t, plugin.HealthHealthy, result.Health)
	assert.Equal(t, "clusters", result.Label)
}

func TestCommands(t *testing.T) {
	p := NewPlugin(nil, "", "")
	cmds := p.Commands()

	require.Len(t, cmds, 1)
	assert.Equal(t, "EKS Clusters", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "eks")
	assert.Contains(t, cmds[0].Keywords, "kubernetes")
	assert.Contains(t, cmds[0].Keywords, "clusters")
	assert.Contains(t, cmds[0].Keywords, "k8s")
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
		name     string
		clusters []awseks.EKSCluster
		want     bool
	}{
		{
			name:     "no clusters",
			clusters: nil,
			want:     false,
		},
		{
			name: "all active",
			clusters: []awseks.EKSCluster{
				{Name: "c1", Status: "ACTIVE"},
				{Name: "c2", Status: "ACTIVE"},
			},
			want: false,
		},
		{
			name: "has creating",
			clusters: []awseks.EKSCluster{
				{Name: "c1", Status: "ACTIVE"},
				{Name: "c2", Status: "CREATING"},
			},
			want: true,
		},
		{
			name: "has updating",
			clusters: []awseks.EKSCluster{
				{Name: "c1", Status: "UPDATING"},
			},
			want: true,
		},
		{
			name: "has deleting",
			clusters: []awseks.EKSCluster{
				{Name: "c1", Status: "DELETING"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plugin{clusters: tc.clusters}
			cfg := p.PollConfig()
			assert.Equal(t, tc.want, cfg.IsActive())
		})
	}
}

func TestPluginMetadata(t *testing.T) {
	p := NewPlugin(nil, "", "")
	assert.Equal(t, "eks", p.ID())
	assert.Equal(t, "EKS", p.Name())
}

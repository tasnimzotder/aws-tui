package ecs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/plugin"
)

// --- mock ECS client ---

type mockClient struct {
	listClustersFunc    func(ctx context.Context) ([]ecs.ECSCluster, error)
	listServicesFunc    func(ctx context.Context, cluster string) ([]ecs.ECSService, error)
	listTasksFunc       func(ctx context.Context, cluster, service string) ([]ecs.ECSTask, error)
	describeServiceFunc func(ctx context.Context, cluster, service string) (*ecs.ECSServiceDetail, error)
	describeTaskFunc    func(ctx context.Context, cluster, taskARN string) (*ecs.ECSTaskDetail, error)
}

func (m *mockClient) ListClusters(ctx context.Context) ([]ecs.ECSCluster, error) {
	return m.listClustersFunc(ctx)
}
func (m *mockClient) ListServices(ctx context.Context, cluster string) ([]ecs.ECSService, error) {
	return m.listServicesFunc(ctx, cluster)
}
func (m *mockClient) ListTasks(ctx context.Context, cluster, service string) ([]ecs.ECSTask, error) {
	return m.listTasksFunc(ctx, cluster, service)
}
func (m *mockClient) DescribeService(ctx context.Context, cluster, service string) (*ecs.ECSServiceDetail, error) {
	return m.describeServiceFunc(ctx, cluster, service)
}
func (m *mockClient) DescribeTask(ctx context.Context, cluster, taskARN string) (*ecs.ECSTaskDetail, error) {
	return m.describeTaskFunc(ctx, cluster, taskARN)
}

// --- mock router ---

type mockRouter struct{}

func (m mockRouter) Push(_ plugin.View)                  {}
func (m mockRouter) Pop()                                {}
func (m mockRouter) Navigate(_ string)                   {}
func (m mockRouter) NavigateDetail(_ string, _ string)   {}
func (m mockRouter) Toast(_ plugin.ToastLevel, _ string) {}

// --- tests ---

func TestPlugin_IDNameIcon(t *testing.T) {
	p := NewPlugin(&mockClient{}, "", "")
	assert.Equal(t, "ecs", p.ID())
	assert.Equal(t, "ECS", p.Name())
	assert.NotEmpty(t, p.Icon())
}

func TestPlugin_Summary(t *testing.T) {
	tests := []struct {
		name           string
		clusters       []ecs.ECSCluster
		err            error
		expectedTotal  int
		expectedHealth plugin.HealthLevel
		expectedLabel  string
	}{
		{
			name:           "no clusters returns unknown health",
			clusters:       nil,
			expectedTotal:  0,
			expectedHealth: plugin.HealthUnknown,
			expectedLabel:  "clusters",
		},
		{
			name: "all active clusters returns healthy",
			clusters: []ecs.ECSCluster{
				{Name: "prod", Status: "ACTIVE", ServiceCount: 3, RunningTaskCount: 10},
				{Name: "staging", Status: "ACTIVE", ServiceCount: 2, RunningTaskCount: 5},
			},
			expectedTotal:  2,
			expectedHealth: plugin.HealthHealthy,
			expectedLabel:  "clusters",
		},
		{
			name: "inactive cluster returns warning",
			clusters: []ecs.ECSCluster{
				{Name: "prod", Status: "ACTIVE", ServiceCount: 3, RunningTaskCount: 10},
				{Name: "old", Status: "INACTIVE", ServiceCount: 0, RunningTaskCount: 0},
			},
			expectedTotal:  2,
			expectedHealth: plugin.HealthWarning,
			expectedLabel:  "clusters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{
				listClustersFunc: func(_ context.Context) ([]ecs.ECSCluster, error) {
					return tt.clusters, tt.err
				},
			}
			p := NewPlugin(client, "", "")
			summary, err := p.Summary(context.Background())

			if tt.err != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedTotal, summary.Total)
			assert.Equal(t, tt.expectedHealth, summary.Health)
			assert.Equal(t, tt.expectedLabel, summary.Label)
		})
	}
}

func TestPlugin_Commands(t *testing.T) {
	p := NewPlugin(&mockClient{}, "", "")
	cmds := p.Commands()
	require.NotEmpty(t, cmds)
	assert.Equal(t, "List ECS Clusters", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "ecs")
}

func TestPlugin_PollConfig(t *testing.T) {
	client := &mockClient{
		listClustersFunc: func(_ context.Context) ([]ecs.ECSCluster, error) {
			return []ecs.ECSCluster{
				{Name: "prod", Status: "ACTIVE", ServiceCount: 2, RunningTaskCount: 0},
			}, nil
		},
	}
	p := NewPlugin(client, "", "")

	cfg := p.PollConfig()
	assert.Equal(t, 30_000_000_000, int(cfg.IdleInterval))  // 30s in ns
	assert.Equal(t, 5_000_000_000, int(cfg.ActiveInterval))  // 5s in ns
	require.NotNil(t, cfg.IsActive)

	// Before Summary is called, IsActive should be false
	assert.False(t, cfg.IsActive())

	// After Summary with a cluster that has services but 0 running tasks, IsActive should be true
	_, err := p.Summary(context.Background())
	require.NoError(t, err)
	assert.True(t, cfg.IsActive())
}

func TestPlugin_ListView(t *testing.T) {
	p := NewPlugin(&mockClient{}, "", "")
	view := p.ListView(mockRouter{})
	require.NotNil(t, view)
	assert.Equal(t, "ECS Clusters", view.Title())
}

func TestPlugin_DetailView(t *testing.T) {
	p := NewPlugin(&mockClient{}, "", "")
	view := p.DetailView(mockRouter{}, "prod/my-service")
	require.NotNil(t, view)
	assert.Equal(t, "ECS Detail", view.Title())
}

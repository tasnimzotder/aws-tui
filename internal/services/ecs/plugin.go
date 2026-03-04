package ecs

import (
	"context"
	"sync"
	"time"

	"tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/plugin"
)

// ECSClient defines the subset of ecs.Client methods used by the plugin.
type ECSClient interface {
	ListClusters(ctx context.Context) ([]ecs.ECSCluster, error)
	ListServices(ctx context.Context, clusterName string) ([]ecs.ECSService, error)
	ListTasks(ctx context.Context, clusterName, serviceName string) ([]ecs.ECSTask, error)
	DescribeService(ctx context.Context, clusterName, serviceName string) (*ecs.ECSServiceDetail, error)
	DescribeTask(ctx context.Context, clusterName, taskARN string) (*ecs.ECSTaskDetail, error)
}

// Plugin implements plugin.ServicePlugin for Amazon ECS.
type Plugin struct {
	client ECSClient

	mu             sync.Mutex
	hasPendingTask bool
	region         string
	profile        string
}

// NewPlugin creates a new ECS service plugin.
func NewPlugin(client ECSClient, region, profile string) *Plugin {
	return &Plugin{client: client, region: region, profile: profile}
}

func (p *Plugin) ID() string   { return "ecs" }
func (p *Plugin) Name() string { return "ECS" }
func (p *Plugin) Icon() string { return "\U000F01A7" } // nf-mdi-cloud

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	clusters, err := p.client.ListClusters(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}

	status := map[string]int{}
	totalRunning := 0
	totalServices := 0
	hasPending := false

	for _, c := range clusters {
		status[c.Status]++
		totalRunning += c.RunningTaskCount
		totalServices += c.ServiceCount
		if c.RunningTaskCount == 0 && c.ServiceCount > 0 {
			hasPending = true
		}
	}

	p.mu.Lock()
	p.hasPendingTask = hasPending
	p.mu.Unlock()

	health := plugin.HealthHealthy
	if len(clusters) == 0 {
		health = plugin.HealthUnknown
	} else {
		for _, c := range clusters {
			if c.Status != "ACTIVE" {
				health = plugin.HealthWarning
				break
			}
		}
	}

	return plugin.ServiceSummary{
		Total:  len(clusters),
		Status: status,
		Health: health,
		Label:  "clusters",
	}, nil
}

func (p *Plugin) ListView(router plugin.Router) plugin.View {
	return NewClusterListView(p.client, router, p.region, p.profile)
}

func (p *Plugin) DetailView(router plugin.Router, id string) plugin.View {
	return NewDetailView(p.client, router, id, p.region, p.profile)
}


func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Title:    "List ECS Clusters",
			Keywords: []string{"ecs", "clusters", "containers"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   30 * time.Second,
		ActiveInterval: 5 * time.Second,
		IsActive: func() bool {
			p.mu.Lock()
			defer p.mu.Unlock()
			return p.hasPendingTask
		},
	}
}

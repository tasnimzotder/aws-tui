package eks

import (
	"context"
	"time"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/plugin"
)

// Plugin implements plugin.ServicePlugin for AWS EKS clusters.
type Plugin struct {
	client   *awseks.Client
	clusters []awseks.EKSCluster
	region   string
	profile  string
}

// NewPlugin creates a new EKS ServicePlugin.
func NewPlugin(client *awseks.Client, region, profile string) *Plugin {
	return &Plugin{client: client, region: region, profile: profile}
}

func (p *Plugin) ID() string   { return "eks" }
func (p *Plugin) Name() string { return "EKS" }
func (p *Plugin) Icon() string { return "\U000F10FE" } // nf-mdi-kubernetes

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	clusters, err := p.client.ListClusters(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}
	p.clusters = clusters
	return mapSummary(clusters), nil
}

// mapSummary converts a list of EKS clusters into a plugin.ServiceSummary.
func mapSummary(clusters []awseks.EKSCluster) plugin.ServiceSummary {
	status := make(map[string]int)
	for _, c := range clusters {
		status[c.Status]++
	}

	health := plugin.HealthHealthy
	for _, c := range clusters {
		if c.Status != "ACTIVE" {
			health = plugin.HealthWarning
			break
		}
	}

	return plugin.ServiceSummary{
		Total:  len(clusters),
		Status: status,
		Health: health,
		Label:  "clusters",
	}
}

func (p *Plugin) ListView(router plugin.Router) plugin.View {
	return NewListView(p.client, router, p.region, p.profile)
}

func (p *Plugin) DetailView(router plugin.Router, id string) plugin.View {
	return NewDetailView(p.client, router, id, p.region, p.profile)
}

func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Title:    "EKS Clusters",
			Keywords: []string{"eks", "kubernetes", "clusters", "k8s"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   60 * time.Second,
		ActiveInterval: 5 * time.Second,
		IsActive: func() bool {
			for _, c := range p.clusters {
				if c.Status != "ACTIVE" {
					return true
				}
			}
			return false
		},
	}
}

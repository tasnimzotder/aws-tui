package ec2

import (
	"context"
	"time"

	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/plugin"
)

// EC2Client defines the subset of ec2.Client methods used by the plugin.
type EC2Client interface {
	ListInstances(ctx context.Context) ([]awsec2.EC2Instance, awsec2.EC2Summary, error)
	GetInstanceVolumes(ctx context.Context, volumeIDs []string) ([]awsec2.EBSVolume, error)
}

// Plugin implements plugin.ServicePlugin for AWS EC2 instances.
type Plugin struct {
	client    EC2Client
	instances []awsec2.EC2Instance
	region    string
	profile   string
}

// NewPlugin creates a new EC2 ServicePlugin.
func NewPlugin(client EC2Client, region, profile string) *Plugin {
	return &Plugin{client: client, region: region, profile: profile}
}

func (p *Plugin) ID() string   { return "ec2" }
func (p *Plugin) Name() string { return "EC2" }
func (p *Plugin) Icon() string { return "\U000F01C4" } // nf-mdi-desktop-tower

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	instances, summary, err := p.client.ListInstances(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}
	p.instances = instances
	return mapSummary(summary), nil
}

// mapSummary converts an EC2Summary into a plugin.ServiceSummary.
func mapSummary(summary awsec2.EC2Summary) plugin.ServiceSummary {
	status := map[string]int{
		"running": summary.Running,
		"stopped": summary.Stopped,
	}

	other := summary.Total - summary.Running - summary.Stopped
	if other > 0 {
		status["other"] = other
	}

	health := plugin.HealthHealthy
	if summary.Stopped > 0 {
		health = plugin.HealthWarning
	}
	if summary.Running == 0 && summary.Total > 0 {
		health = plugin.HealthCritical
	}

	return plugin.ServiceSummary{
		Total:  summary.Total,
		Status: status,
		Health: health,
		Label:  "instances",
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
			Title:    "List EC2 Instances",
			Keywords: []string{"ec2", "instances", "servers", "compute"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   60 * time.Second,
		ActiveInterval: 5 * time.Second,
		IsActive: func() bool {
			for _, inst := range p.instances {
				switch inst.State {
				case "pending", "stopping", "shutting-down":
					return true
				}
			}
			return false
		},
	}
}

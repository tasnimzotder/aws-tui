package elb

import (
	"context"
	"time"

	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	"tasnim.dev/aws-tui/internal/plugin"
)

// Plugin implements plugin.ServicePlugin for AWS Elastic Load Balancers.
type Plugin struct {
	client       *awselb.Client
	loadBalancers []awselb.ELBLoadBalancer
	hasUnhealthy bool
}

// NewPlugin creates a new ELB ServicePlugin.
func NewPlugin(client *awselb.Client) *Plugin {
	return &Plugin{client: client}
}

func (p *Plugin) ID() string   { return "elb" }
func (p *Plugin) Name() string { return "ELB" }
func (p *Plugin) Icon() string { return "\U000F04E7" } // nf-mdi-scale-balance

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	lbs, err := p.client.ListLoadBalancers(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}
	p.loadBalancers = lbs

	return mapSummary(ctx, p.client, lbs)
}

// mapSummary converts load balancers into a plugin.ServiceSummary, checking target health.
func mapSummary(ctx context.Context, client *awselb.Client, lbs []awselb.ELBLoadBalancer) (plugin.ServiceSummary, error) {
	status := make(map[string]int)
	for _, lb := range lbs {
		status[lb.State]++
	}

	health := plugin.HealthHealthy
	hasUnhealthy := false

	for _, lb := range lbs {
		tgs, err := client.ListTargetGroups(ctx, lb.ARN)
		if err != nil {
			continue
		}
		for _, tg := range tgs {
			if tg.UnhealthyCount > 0 {
				hasUnhealthy = true
				break
			}
		}
		if hasUnhealthy {
			break
		}
	}

	if hasUnhealthy {
		health = plugin.HealthWarning
	}

	return plugin.ServiceSummary{
		Total:  len(lbs),
		Status: status,
		Health: health,
		Label:  "load balancers",
	}, nil
}

func (p *Plugin) ListView(router plugin.Router) plugin.View {
	return NewListView(p.client, router)
}

func (p *Plugin) DetailView(router plugin.Router, id string) plugin.View {
	return NewDetailView(p.client, router, id)
}

func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Title:    "List Load Balancers",
			Keywords: []string{"elb", "load balancer", "alb", "nlb", "gateway"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   60 * time.Second,
		ActiveInterval: 10 * time.Second,
		IsActive: func() bool {
			return p.hasUnhealthy
		},
	}
}

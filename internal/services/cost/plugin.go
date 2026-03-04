package cost

import (
	"context"
	"fmt"
	"time"

	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	"tasnim.dev/aws-tui/internal/plugin"
)

// CostClient defines the interface for fetching cost data.
type CostClient interface {
	FetchCostData(ctx context.Context) (*awscost.CostData, error)
	FetchCostDataForMonth(ctx context.Context, target time.Time) (*awscost.CostData, error)
}

// Plugin implements plugin.ServicePlugin for AWS Cost Explorer.
type Plugin struct {
	client CostClient
}

// NewPlugin creates a new Cost Explorer ServicePlugin.
func NewPlugin(client CostClient) *Plugin {
	return &Plugin{client: client}
}

func (p *Plugin) ID() string   { return "cost" }
func (p *Plugin) Name() string { return "Cost Explorer" }
func (p *Plugin) Icon() string { return "\U000F011F" } // nf-mdi-cash

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	data, err := p.client.FetchCostData(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}
	return mapSummary(data), nil
}

// mapSummary converts CostData into a plugin.ServiceSummary.
func mapSummary(data *awscost.CostData) plugin.ServiceSummary {
	currency := data.Currency
	if currency == "" {
		currency = "USD"
	}

	label := fmt.Sprintf("$%.2f this month", data.MTDSpend)

	health := plugin.HealthHealthy
	if data.MoMChangePercent > 20 {
		health = plugin.HealthWarning
	}
	if data.MoMChangePercent > 50 {
		health = plugin.HealthCritical
	}

	return plugin.ServiceSummary{
		Total:  len(data.TopServices),
		Status: map[string]int{"services": len(data.TopServices)},
		Health: health,
		Label:  label,
	}
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
			Title:    "View Cost Explorer",
			Keywords: []string{"cost", "billing", "spend", "money", "budget"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   1 * time.Hour,
		ActiveInterval: 0,
		IsActive:       func() bool { return false },
	}
}

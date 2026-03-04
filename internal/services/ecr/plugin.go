package ecr

import (
	"context"
	"time"

	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	"tasnim.dev/aws-tui/internal/plugin"
)

// ECRClient defines the subset of ecr.Client methods used by the plugin.
type ECRClient interface {
	ListRepositories(ctx context.Context) ([]awsecr.ECRRepo, error)
	ListImages(ctx context.Context, repoName string) ([]awsecr.ECRImage, error)
}

// Plugin implements plugin.ServicePlugin for Amazon ECR.
type Plugin struct {
	client ECRClient
}

// NewPlugin creates a new ECR service plugin.
func NewPlugin(client ECRClient) *Plugin {
	return &Plugin{client: client}
}

func (p *Plugin) ID() string   { return "ecr" }
func (p *Plugin) Name() string { return "ECR" }
func (p *Plugin) Icon() string { return "\U000F0342" } // nf-mdi-docker

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	repos, err := p.client.ListRepositories(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}

	totalImages := 0
	for _, r := range repos {
		totalImages += r.ImageCount
	}

	return plugin.ServiceSummary{
		Total:  len(repos),
		Status: map[string]int{"images": totalImages},
		Health: plugin.HealthHealthy,
		Label:  "repositories",
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
			Title:    "ECR Repositories",
			Keywords: []string{"ecr", "repositories", "images", "containers"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval: 2 * time.Minute,
	}
}

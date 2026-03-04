package s3

import (
	"context"
	"fmt"
	"time"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/plugin"
)

// S3Client defines the subset of s3.Client methods used by the plugin.
type S3Client interface {
	ListBuckets(ctx context.Context) ([]awss3.S3Bucket, error)
	ListObjects(ctx context.Context, bucket, prefix, continuationToken, region string) (awss3.ListObjectsResult, error)
	GetObject(ctx context.Context, bucket, key, region string) ([]byte, error)
}

// Plugin implements plugin.ServicePlugin for Amazon S3.
type Plugin struct {
	client S3Client
}

// NewPlugin creates a new S3 ServicePlugin.
func NewPlugin(client S3Client) *Plugin {
	return &Plugin{client: client}
}

func (p *Plugin) ID() string   { return "s3" }
func (p *Plugin) Name() string { return "S3" }
func (p *Plugin) Icon() string { return "\U000F01BC" } // nf-mdi-database

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	buckets, err := p.client.ListBuckets(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}

	// Count buckets by region
	status := map[string]int{}
	for _, b := range buckets {
		region := b.Region
		if region == "" {
			region = "unknown"
		}
		status[region]++
	}

	health := plugin.HealthHealthy
	if len(buckets) == 0 {
		health = plugin.HealthUnknown
	}

	return plugin.ServiceSummary{
		Total:  len(buckets),
		Status: status,
		Health: health,
		Label:  fmt.Sprintf("%d buckets", len(buckets)),
	}, nil
}

func (p *Plugin) ListView(router plugin.Router) plugin.View {
	return NewListView(p.client, router)
}

func (p *Plugin) DetailView(router plugin.Router, id string) plugin.View {
	return NewDetailView(p.client, router, id, "")
}

func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Title:    "S3 Buckets",
			Keywords: []string{"s3", "buckets", "storage"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   5 * time.Minute,
		ActiveInterval: 0,
		IsActive:       func() bool { return false },
	}
}

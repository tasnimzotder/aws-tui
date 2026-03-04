package iam

import (
	"context"
	"time"

	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
	"tasnim.dev/aws-tui/internal/plugin"
)

// IAMClient defines the subset of iam.Client methods used by the plugin.
type IAMClient interface {
	ListUsers(ctx context.Context) ([]awsiam.IAMUser, error)
	ListRoles(ctx context.Context) ([]awsiam.IAMRole, error)
	ListPolicies(ctx context.Context) ([]awsiam.IAMPolicy, error)
	ListAttachedUserPolicies(ctx context.Context, userName string) ([]awsiam.IAMAttachedPolicy, error)
	ListGroupsForUser(ctx context.Context, userName string) ([]awsiam.IAMGroup, error)
	ListAttachedRolePolicies(ctx context.Context, roleName string) ([]awsiam.IAMAttachedPolicy, error)
	ListEntitiesForPolicy(ctx context.Context, policyARN string) ([]awsiam.IAMPolicyEntity, error)
	GetPolicyDocument(ctx context.Context, policyARN, versionID string) (string, error)
	ListInlineUserPolicies(ctx context.Context, userName string) ([]awsiam.IAMInlinePolicy, error)
	ListInlineRolePolicies(ctx context.Context, roleName string) ([]awsiam.IAMInlinePolicy, error)
}

// Plugin implements plugin.ServicePlugin for AWS IAM.
type Plugin struct {
	client IAMClient
}

// NewPlugin creates a new IAM ServicePlugin.
func NewPlugin(client IAMClient) *Plugin {
	return &Plugin{client: client}
}

func (p *Plugin) ID() string   { return "iam" }
func (p *Plugin) Name() string { return "IAM" }
func (p *Plugin) Icon() string { return "\U000F0343" } // nf-mdi-key

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	users, err := p.client.ListUsers(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}

	return plugin.ServiceSummary{
		Total:  len(users),
		Status: map[string]int{"users": len(users)},
		Health: plugin.HealthHealthy,
		Label:  "users",
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
			Title:    "IAM",
			Keywords: []string{"iam", "users", "roles", "policies"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   10 * time.Minute,
		ActiveInterval: 0,
		IsActive:       func() bool { return false },
	}
}

package vpc

import (
	"context"
	"fmt"
	"time"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/plugin"
)

// VPCClient defines the subset of awsvpc.Client methods used by the plugin.
type VPCClient interface {
	ListVPCs(ctx context.Context) ([]awsvpc.VPCInfo, error)
	ListSubnets(ctx context.Context, vpcID string) ([]awsvpc.SubnetInfo, error)
	ListSecurityGroups(ctx context.Context, vpcID string) ([]awsvpc.SecurityGroupInfo, error)
	ListSecurityGroupRules(ctx context.Context, groupID string) ([]awsvpc.SecurityGroupRule, error)
	ListRouteTables(ctx context.Context, vpcID string) ([]awsvpc.RouteTableInfo, error)
	ListNATGateways(ctx context.Context, vpcID string) ([]awsvpc.NATGatewayInfo, error)
	ListVPCEndpoints(ctx context.Context, vpcID string) ([]awsvpc.VPCEndpointInfo, error)
	ListVPCPeering(ctx context.Context, vpcID string) ([]awsvpc.VPCPeeringInfo, error)
	ListNetworkACLs(ctx context.Context, vpcID string) ([]awsvpc.NetworkACLInfo, error)
	ListNetworkACLEntries(ctx context.Context, naclID string) ([]awsvpc.NetworkACLEntry, error)
	ListFlowLogs(ctx context.Context, vpcID string) ([]awsvpc.FlowLogInfo, error)
	ListInternetGateways(ctx context.Context, vpcID string) ([]awsvpc.InternetGatewayInfo, error)
	GetVPCTags(ctx context.Context, vpcID string) (map[string]string, error)
}

// Plugin implements plugin.ServicePlugin for AWS VPC.
type Plugin struct {
	client VPCClient
}

// NewPlugin creates a new VPC ServicePlugin.
func NewPlugin(client VPCClient) *Plugin {
	return &Plugin{client: client}
}

func (p *Plugin) ID() string   { return "vpc" }
func (p *Plugin) Name() string { return "VPC" }
func (p *Plugin) Icon() string { return "\U000F0317" } // nf-mdi-sitemap

func (p *Plugin) Summary(ctx context.Context) (plugin.ServiceSummary, error) {
	vpcs, err := p.client.ListVPCs(ctx)
	if err != nil {
		return plugin.ServiceSummary{}, err
	}

	status := map[string]int{}
	for _, v := range vpcs {
		status[v.State]++
	}

	health := plugin.HealthHealthy
	if len(vpcs) == 0 {
		health = plugin.HealthUnknown
	}

	return plugin.ServiceSummary{
		Total:  len(vpcs),
		Status: status,
		Health: health,
		Label:  fmt.Sprintf("%d VPCs", len(vpcs)),
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
			Title:    "VPC",
			Keywords: []string{"vpc", "network", "subnets"},
		},
	}
}

func (p *Plugin) PollConfig() plugin.PollConfig {
	return plugin.PollConfig{
		IdleInterval:   5 * time.Minute,
		ActiveInterval: 0, // VPC is static; no active polling
		IsActive:       func() bool { return false },
	}
}

package vpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
)

// mockVPCClient implements VPCClient for tests.
type mockVPCClient struct {
	vpcs           []awsvpc.VPCInfo
	subnets        []awsvpc.SubnetInfo
	securityGroups []awsvpc.SecurityGroupInfo
	routeTables    []awsvpc.RouteTableInfo
	natGateways    []awsvpc.NATGatewayInfo
	endpoints      []awsvpc.VPCEndpointInfo
	peering        []awsvpc.VPCPeeringInfo
	nacls          []awsvpc.NetworkACLInfo
	flowLogs       []awsvpc.FlowLogInfo
	igws           []awsvpc.InternetGatewayInfo
	tags           map[string]string
	err            error
}

func (m *mockVPCClient) ListVPCs(_ context.Context) ([]awsvpc.VPCInfo, error) {
	return m.vpcs, m.err
}
func (m *mockVPCClient) ListSubnets(_ context.Context, _ string) ([]awsvpc.SubnetInfo, error) {
	return m.subnets, m.err
}
func (m *mockVPCClient) ListSecurityGroups(_ context.Context, _ string) ([]awsvpc.SecurityGroupInfo, error) {
	return m.securityGroups, m.err
}
func (m *mockVPCClient) ListRouteTables(_ context.Context, _ string) ([]awsvpc.RouteTableInfo, error) {
	return m.routeTables, m.err
}
func (m *mockVPCClient) ListNATGateways(_ context.Context, _ string) ([]awsvpc.NATGatewayInfo, error) {
	return m.natGateways, m.err
}
func (m *mockVPCClient) ListVPCEndpoints(_ context.Context, _ string) ([]awsvpc.VPCEndpointInfo, error) {
	return m.endpoints, m.err
}
func (m *mockVPCClient) ListVPCPeering(_ context.Context, _ string) ([]awsvpc.VPCPeeringInfo, error) {
	return m.peering, m.err
}
func (m *mockVPCClient) ListNetworkACLs(_ context.Context, _ string) ([]awsvpc.NetworkACLInfo, error) {
	return m.nacls, m.err
}
func (m *mockVPCClient) ListFlowLogs(_ context.Context, _ string) ([]awsvpc.FlowLogInfo, error) {
	return m.flowLogs, m.err
}
func (m *mockVPCClient) ListInternetGateways(_ context.Context, _ string) ([]awsvpc.InternetGatewayInfo, error) {
	return m.igws, m.err
}
func (m *mockVPCClient) ListSecurityGroupRules(_ context.Context, _ string) ([]awsvpc.SecurityGroupRule, error) {
	return nil, m.err
}
func (m *mockVPCClient) ListNetworkACLEntries(_ context.Context, _ string) ([]awsvpc.NetworkACLEntry, error) {
	return nil, m.err
}
func (m *mockVPCClient) GetVPCTags(_ context.Context, _ string) (map[string]string, error) {
	return m.tags, m.err
}

func TestPluginIdentity(t *testing.T) {
	p := NewPlugin(&mockVPCClient{})
	assert.Equal(t, "vpc", p.ID())
	assert.Equal(t, "VPC", p.Name())
}

func TestSummary(t *testing.T) {
	client := &mockVPCClient{
		vpcs: []awsvpc.VPCInfo{
			{VPCID: "vpc-111", State: "available"},
			{VPCID: "vpc-222", State: "available"},
			{VPCID: "vpc-333", State: "pending"},
		},
	}
	p := NewPlugin(client)

	summary, err := p.Summary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 2, summary.Status["available"])
	assert.Equal(t, 1, summary.Status["pending"])
	assert.Contains(t, summary.Label, "3")
}

func TestSummaryEmpty(t *testing.T) {
	client := &mockVPCClient{vpcs: nil}
	p := NewPlugin(client)

	summary, err := p.Summary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, summary.Total)
}

func TestCommands(t *testing.T) {
	p := NewPlugin(&mockVPCClient{})
	cmds := p.Commands()
	require.Len(t, cmds, 1)
	assert.Equal(t, "VPC", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "vpc")
	assert.Contains(t, cmds[0].Keywords, "network")
	assert.Contains(t, cmds[0].Keywords, "subnets")
}

func TestPollConfig(t *testing.T) {
	p := NewPlugin(&mockVPCClient{})
	cfg := p.PollConfig()
	assert.Equal(t, 5*time.Minute, cfg.IdleInterval)
	assert.Equal(t, time.Duration(0), cfg.ActiveInterval)
	assert.False(t, cfg.IsActive())
}

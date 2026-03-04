package elb

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	"tasnim.dev/aws-tui/internal/plugin"
)

// mockELBAPI implements awselb.ELBAPI for testing.
type mockELBAPI struct {
	lbs           []elbtypes.LoadBalancer
	listeners     []elbtypes.Listener
	targetGroups  []elbtypes.TargetGroup
	targetHealths map[string][]elbtypes.TargetHealthDescription
	err           error
}

func (m *mockELBAPI) DescribeLoadBalancers(_ context.Context, _ *elbv2.DescribeLoadBalancersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &elbv2.DescribeLoadBalancersOutput{LoadBalancers: m.lbs}, nil
}

func (m *mockELBAPI) DescribeListeners(_ context.Context, _ *elbv2.DescribeListenersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &elbv2.DescribeListenersOutput{Listeners: m.listeners}, nil
}

func (m *mockELBAPI) DescribeTargetGroups(_ context.Context, _ *elbv2.DescribeTargetGroupsInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &elbv2.DescribeTargetGroupsOutput{TargetGroups: m.targetGroups}, nil
}

func (m *mockELBAPI) DescribeTargetHealth(_ context.Context, params *elbv2.DescribeTargetHealthInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	arn := aws.ToString(params.TargetGroupArn)
	descs := m.targetHealths[arn]
	return &elbv2.DescribeTargetHealthOutput{TargetHealthDescriptions: descs}, nil
}

func (m *mockELBAPI) DescribeRules(_ context.Context, _ *elbv2.DescribeRulesInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error) {
	return &elbv2.DescribeRulesOutput{}, nil
}

func (m *mockELBAPI) DescribeLoadBalancerAttributes(_ context.Context, _ *elbv2.DescribeLoadBalancerAttributesInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancerAttributesOutput, error) {
	return &elbv2.DescribeLoadBalancerAttributesOutput{}, nil
}

func (m *mockELBAPI) DescribeTags(_ context.Context, _ *elbv2.DescribeTagsInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error) {
	return &elbv2.DescribeTagsOutput{}, nil
}

func newMockClient(api *mockELBAPI) *awselb.Client {
	return awselb.NewClient(api)
}

func TestPluginMetadata(t *testing.T) {
	p := NewPlugin(nil)
	assert.Equal(t, "elb", p.ID())
	assert.Equal(t, "ELB", p.Name())
	// Icon may be empty; just ensure it doesn't panic.
	_ = p.Icon()
}

func TestCommands(t *testing.T) {
	p := NewPlugin(nil)
	cmds := p.Commands()

	require.Len(t, cmds, 1)
	assert.Equal(t, "List Load Balancers", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "elb")
	assert.Contains(t, cmds[0].Keywords, "load balancer")
	assert.Contains(t, cmds[0].Keywords, "alb")
}

func TestPollConfig(t *testing.T) {
	p := NewPlugin(nil)
	cfg := p.PollConfig()

	assert.Equal(t, 60*time.Second, cfg.IdleInterval)
	assert.Equal(t, 10*time.Second, cfg.ActiveInterval)
	assert.NotNil(t, cfg.IsActive)
}

func TestPollConfig_IsActive(t *testing.T) {
	tests := []struct {
		name         string
		hasUnhealthy bool
		want         bool
	}{
		{
			name:         "no unhealthy targets",
			hasUnhealthy: false,
			want:         false,
		},
		{
			name:         "has unhealthy targets",
			hasUnhealthy: true,
			want:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plugin{hasUnhealthy: tc.hasUnhealthy}
			cfg := p.PollConfig()
			assert.Equal(t, tc.want, cfg.IsActive())
		})
	}
}

func TestSummary_AllHealthy(t *testing.T) {
	api := &mockELBAPI{
		lbs: []elbtypes.LoadBalancer{
			{
				LoadBalancerName: aws.String("my-alb"),
				LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc"),
				Type:             elbtypes.LoadBalancerTypeEnumApplication,
				State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
				Scheme:           elbtypes.LoadBalancerSchemeEnumInternetFacing,
			},
		},
		targetGroups: []elbtypes.TargetGroup{
			{
				TargetGroupName: aws.String("tg-1"),
				TargetGroupArn:  aws.String("arn:tg-1"),
			},
		},
		targetHealths: map[string][]elbtypes.TargetHealthDescription{
			"arn:tg-1": {
				{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumHealthy}},
				{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumHealthy}},
			},
		},
	}

	client := newMockClient(api)
	summary, err := mapSummary(context.Background(), client, []awselb.ELBLoadBalancer{
		{Name: "my-alb", ARN: "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc", State: "active"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, summary.Total)
	assert.Equal(t, 1, summary.Status["active"])
	assert.Equal(t, plugin.HealthHealthy, summary.Health)
	assert.Equal(t, "load balancers", summary.Label)
}

func TestSummary_WithUnhealthyTargets(t *testing.T) {
	api := &mockELBAPI{
		lbs: []elbtypes.LoadBalancer{
			{
				LoadBalancerName: aws.String("my-alb"),
				LoadBalancerArn:  aws.String("arn:lb-1"),
				Type:             elbtypes.LoadBalancerTypeEnumApplication,
				State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
			},
		},
		targetGroups: []elbtypes.TargetGroup{
			{
				TargetGroupName: aws.String("tg-1"),
				TargetGroupArn:  aws.String("arn:tg-1"),
			},
		},
		targetHealths: map[string][]elbtypes.TargetHealthDescription{
			"arn:tg-1": {
				{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumHealthy}},
				{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumUnhealthy}},
			},
		},
	}

	client := newMockClient(api)
	summary, err := mapSummary(context.Background(), client, []awselb.ELBLoadBalancer{
		{Name: "my-alb", ARN: "arn:lb-1", State: "active"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, summary.Total)
	assert.Equal(t, plugin.HealthWarning, summary.Health)
}

func TestSummary_Empty(t *testing.T) {
	api := &mockELBAPI{}
	client := newMockClient(api)
	summary, err := mapSummary(context.Background(), client, nil)

	require.NoError(t, err)
	assert.Equal(t, 0, summary.Total)
	assert.Equal(t, plugin.HealthHealthy, summary.Health)
}

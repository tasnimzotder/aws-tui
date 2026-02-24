package elb

import (
	"context"
	"fmt"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockELBAPI struct {
	describeLoadBalancersFunc func(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	describeListenersFunc     func(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error)
	describeTargetGroupsFunc  func(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
	describeTargetHealthFunc  func(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error)
	describeRulesFunc         func(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
}

func (m *mockELBAPI) DescribeLoadBalancers(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
	return m.describeLoadBalancersFunc(ctx, params, optFns...)
}
func (m *mockELBAPI) DescribeListeners(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
	return m.describeListenersFunc(ctx, params, optFns...)
}
func (m *mockELBAPI) DescribeTargetGroups(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	return m.describeTargetGroupsFunc(ctx, params, optFns...)
}
func (m *mockELBAPI) DescribeTargetHealth(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
	return m.describeTargetHealthFunc(ctx, params, optFns...)
}
func (m *mockELBAPI) DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error) {
	return m.describeRulesFunc(ctx, params, optFns...)
}

func TestListLoadBalancers(t *testing.T) {
	created := time.Date(2025, 8, 10, 12, 0, 0, 0, time.UTC)
	mock := &mockELBAPI{
		describeLoadBalancersFunc: func(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
			return &elbv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{
					{
						LoadBalancerName: awssdk.String("my-alb"),
						LoadBalancerArn:  awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:loadbalancer/app/my-alb/abc123"),
						Type:             elbtypes.LoadBalancerTypeEnumApplication,
						State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
						Scheme:           elbtypes.LoadBalancerSchemeEnumInternetFacing,
						DNSName:          awssdk.String("my-alb-123.us-east-1.elb.amazonaws.com"),
						VpcId:            awssdk.String("vpc-abc123"),
						CreatedTime:      &created,
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	lbs, err := client.ListLoadBalancers(context.Background())
	require.NoError(t, err)
	require.Len(t, lbs, 1)

	lb := lbs[0]
	assert.Equal(t, "my-alb", lb.Name)
	assert.Equal(t, "application", lb.Type)
	assert.Equal(t, "active", lb.State)
	assert.Equal(t, "internet-facing", lb.Scheme)
	assert.Equal(t, "my-alb-123.us-east-1.elb.amazonaws.com", lb.DNSName)
	assert.Equal(t, "vpc-abc123", lb.VPCID)
}

func TestListLoadBalancers_Pagination(t *testing.T) {
	calls := 0
	mock := &mockELBAPI{
		describeLoadBalancersFunc: func(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
			calls++
			if calls == 1 {
				return &elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []elbtypes.LoadBalancer{
						{LoadBalancerName: awssdk.String("alb-1"), LoadBalancerArn: awssdk.String("arn:1")},
					},
					NextMarker: awssdk.String("page2"),
				}, nil
			}
			return &elbv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{
					{LoadBalancerName: awssdk.String("alb-2"), LoadBalancerArn: awssdk.String("arn:2")},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	lbs, err := client.ListLoadBalancers(context.Background())
	require.NoError(t, err)
	require.Len(t, lbs, 2)
	assert.Equal(t, "alb-1", lbs[0].Name)
	assert.Equal(t, "alb-2", lbs[1].Name)
	assert.Equal(t, 2, calls)
}

func TestListListeners(t *testing.T) {
	mock := &mockELBAPI{
		describeListenersFunc: func(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
			return &elbv2.DescribeListenersOutput{
				Listeners: []elbtypes.Listener{
					{
						ListenerArn: awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:listener/app/my-alb/abc123/def456"),
						Port:        awssdk.Int32(443),
						Protocol:    elbtypes.ProtocolEnumHttps,
						DefaultActions: []elbtypes.Action{
							{
								Type:           elbtypes.ActionTypeEnumForward,
								TargetGroupArn: awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup/my-tg/abc123"),
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	listeners, err := client.ListListeners(context.Background(), "arn:lb")
	require.NoError(t, err)
	require.Len(t, listeners, 1)

	l := listeners[0]
	assert.Equal(t, 443, l.Port)
	assert.Equal(t, "HTTPS", l.Protocol)
	assert.Equal(t, "forward → my-tg", l.DefaultAction)
}

func TestListTargetGroups(t *testing.T) {
	mock := &mockELBAPI{
		describeTargetGroupsFunc: func(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
			return &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []elbtypes.TargetGroup{
					{
						TargetGroupName: awssdk.String("my-tg"),
						TargetGroupArn:  awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup/my-tg/abc123"),
						Protocol:        elbtypes.ProtocolEnumHttps,
						Port:            awssdk.Int32(8080),
						TargetType:      elbtypes.TargetTypeEnumInstance,
					},
				},
			}, nil
		},
		describeTargetHealthFunc: func(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
			return &elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbtypes.TargetHealthDescription{
					{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumHealthy}},
					{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumHealthy}},
					{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumUnhealthy}},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	tgs, err := client.ListTargetGroups(context.Background(), "arn:lb")
	require.NoError(t, err)
	require.Len(t, tgs, 1)

	tg := tgs[0]
	assert.Equal(t, "my-tg", tg.Name)
	assert.Equal(t, 8080, tg.Port)
	assert.Equal(t, "instance", tg.TargetType)
	assert.Equal(t, 2, tg.HealthyCount)
	assert.Equal(t, 1, tg.UnhealthyCount)
}

func TestListTargetGroups_HealthError(t *testing.T) {
	mock := &mockELBAPI{
		describeTargetGroupsFunc: func(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
			return &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []elbtypes.TargetGroup{
					{
						TargetGroupName: awssdk.String("my-tg"),
						TargetGroupArn:  awssdk.String("arn:tg"),
						Protocol:        elbtypes.ProtocolEnumHttps,
						Port:            awssdk.Int32(8080),
						TargetType:      elbtypes.TargetTypeEnumInstance,
					},
				},
			}, nil
		},
		describeTargetHealthFunc: func(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	client := NewClient(mock)
	_, err := client.ListTargetGroups(context.Background(), "arn:lb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DescribeTargetHealth")
}

func TestListListenerTargetGroups(t *testing.T) {
	mock := &mockELBAPI{
		describeRulesFunc: func(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error) {
			return &elbv2.DescribeRulesOutput{
				Rules: []elbtypes.Rule{
					{
						Actions: []elbtypes.Action{
							{
								Type:           elbtypes.ActionTypeEnumForward,
								TargetGroupArn: awssdk.String("arn:tg-1"),
							},
						},
					},
					{
						Actions: []elbtypes.Action{
							{
								Type:           elbtypes.ActionTypeEnumForward,
								TargetGroupArn: awssdk.String("arn:tg-1"), // duplicate — should be deduped
							},
						},
					},
				},
			}, nil
		},
		describeTargetGroupsFunc: func(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
			assert.Nil(t, params.LoadBalancerArn, "should query by TG ARNs, not LB ARN")
			assert.Equal(t, []string{"arn:tg-1"}, params.TargetGroupArns)
			return &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []elbtypes.TargetGroup{
					{
						TargetGroupName: awssdk.String("my-tg"),
						TargetGroupArn:  awssdk.String("arn:tg-1"),
						Protocol:        elbtypes.ProtocolEnumHttps,
						Port:            awssdk.Int32(8080),
						TargetType:      elbtypes.TargetTypeEnumInstance,
					},
				},
			}, nil
		},
		describeTargetHealthFunc: func(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
			return &elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbtypes.TargetHealthDescription{
					{TargetHealth: &elbtypes.TargetHealth{State: elbtypes.TargetHealthStateEnumHealthy}},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	tgs, err := client.ListListenerTargetGroups(context.Background(), "arn:listener")
	require.NoError(t, err)
	require.Len(t, tgs, 1)
	assert.Equal(t, "my-tg", tgs[0].Name)
	assert.Equal(t, 1, tgs[0].HealthyCount)
}

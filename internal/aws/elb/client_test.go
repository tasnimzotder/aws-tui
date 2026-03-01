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
	describeLoadBalancersFunc          func(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	describeListenersFunc              func(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error)
	describeTargetGroupsFunc           func(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
	describeTargetHealthFunc           func(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error)
	describeRulesFunc                  func(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
	describeLoadBalancerAttributesFunc func(ctx context.Context, params *elbv2.DescribeLoadBalancerAttributesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancerAttributesOutput, error)
	describeTagsFunc                   func(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error)
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
func (m *mockELBAPI) DescribeLoadBalancerAttributes(ctx context.Context, params *elbv2.DescribeLoadBalancerAttributesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancerAttributesOutput, error) {
	return m.describeLoadBalancerAttributesFunc(ctx, params, optFns...)
}
func (m *mockELBAPI) DescribeTags(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error) {
	return m.describeTagsFunc(ctx, params, optFns...)
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
						SslPolicy:   awssdk.String("ELBSecurityPolicy-TLS13-1-2-2021-06"),
						Certificates: []elbtypes.Certificate{
							{CertificateArn: awssdk.String("arn:aws:acm:us-east-1:123456:certificate/abc-123")},
						},
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
	assert.Equal(t, "ELBSecurityPolicy-TLS13-1-2-2021-06", l.SSLPolicy)
	assert.Equal(t, []string{"arn:aws:acm:us-east-1:123456:certificate/abc-123"}, l.Certificates)
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

func TestListListenerRules(t *testing.T) {
	isDefault := true
	mock := &mockELBAPI{
		describeRulesFunc: func(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error) {
			return &elbv2.DescribeRulesOutput{
				Rules: []elbtypes.Rule{
					{
						RuleArn:   awssdk.String("arn:rule-1"),
						Priority:  awssdk.String("1"),
						IsDefault: nil,
						Conditions: []elbtypes.RuleCondition{
							{
								Field:            awssdk.String("host-header"),
								HostHeaderConfig: &elbtypes.HostHeaderConditionConfig{Values: []string{"example.com"}},
							},
							{
								Field:             awssdk.String("path-pattern"),
								PathPatternConfig: &elbtypes.PathPatternConditionConfig{Values: []string{"/api/*"}},
							},
						},
						Actions: []elbtypes.Action{
							{
								Type:           elbtypes.ActionTypeEnumForward,
								TargetGroupArn: awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup/my-tg/abc123"),
							},
						},
					},
					{
						RuleArn:   awssdk.String("arn:rule-default"),
						Priority:  awssdk.String("default"),
						IsDefault: &isDefault,
						Actions: []elbtypes.Action{
							{
								Type: elbtypes.ActionTypeEnumFixedResponse,
								FixedResponseConfig: &elbtypes.FixedResponseActionConfig{
									StatusCode: awssdk.String("404"),
								},
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	rules, err := client.ListListenerRules(context.Background(), "arn:listener")
	require.NoError(t, err)
	require.Len(t, rules, 2)

	r := rules[0]
	assert.Equal(t, "1", r.Priority)
	assert.False(t, r.IsDefault)
	assert.Equal(t, []string{"host: example.com", "path: /api/*"}, r.Conditions)
	assert.Equal(t, []string{"forward → my-tg"}, r.Actions)

	rd := rules[1]
	assert.Equal(t, "default", rd.Priority)
	assert.True(t, rd.IsDefault)
	assert.Equal(t, []string{"fixed-response 404"}, rd.Actions)
}

func TestListTargets(t *testing.T) {
	mock := &mockELBAPI{
		describeTargetHealthFunc: func(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
			return &elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbtypes.TargetHealthDescription{
					{
						Target: &elbtypes.TargetDescription{
							Id:               awssdk.String("i-abc123"),
							Port:             awssdk.Int32(8080),
							AvailabilityZone: awssdk.String("us-east-1a"),
						},
						TargetHealth: &elbtypes.TargetHealth{
							State:       elbtypes.TargetHealthStateEnumHealthy,
							Description: awssdk.String("Health checks passed"),
						},
					},
					{
						Target: &elbtypes.TargetDescription{
							Id:               awssdk.String("i-def456"),
							Port:             awssdk.Int32(8080),
							AvailabilityZone: awssdk.String("us-east-1b"),
						},
						TargetHealth: &elbtypes.TargetHealth{
							State:       elbtypes.TargetHealthStateEnumUnhealthy,
							Reason:      elbtypes.TargetHealthReasonEnumTimeout,
							Description: awssdk.String("Request timed out"),
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	targets, err := client.ListTargets(context.Background(), "arn:tg")
	require.NoError(t, err)
	require.Len(t, targets, 2)

	assert.Equal(t, "i-abc123", targets[0].ID)
	assert.Equal(t, 8080, targets[0].Port)
	assert.Equal(t, "us-east-1a", targets[0].AZ)
	assert.Equal(t, "healthy", targets[0].HealthState)

	assert.Equal(t, "i-def456", targets[1].ID)
	assert.Equal(t, "unhealthy", targets[1].HealthState)
	assert.Equal(t, "Target.Timeout", targets[1].HealthReason)
	assert.Equal(t, "Request timed out", targets[1].HealthDesc)
}

func TestGetLoadBalancerAttributes(t *testing.T) {
	mock := &mockELBAPI{
		describeLoadBalancerAttributesFunc: func(ctx context.Context, params *elbv2.DescribeLoadBalancerAttributesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancerAttributesOutput, error) {
			return &elbv2.DescribeLoadBalancerAttributesOutput{
				Attributes: []elbtypes.LoadBalancerAttribute{
					{Key: awssdk.String("idle_timeout.timeout_seconds"), Value: awssdk.String("60")},
					{Key: awssdk.String("deletion_protection.enabled"), Value: awssdk.String("true")},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	attrs, err := client.GetLoadBalancerAttributes(context.Background(), "arn:lb")
	require.NoError(t, err)
	require.Len(t, attrs, 2)
	assert.Equal(t, "idle_timeout.timeout_seconds", attrs[0].Key)
	assert.Equal(t, "60", attrs[0].Value)
	assert.Equal(t, "deletion_protection.enabled", attrs[1].Key)
	assert.Equal(t, "true", attrs[1].Value)
}

func TestListLoadBalancersPage(t *testing.T) {
	created := time.Date(2025, 11, 5, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		inputMarker    *string
		lbs            []elbtypes.LoadBalancer
		nextMarker     *string
		wantCount      int
		wantNextMarker *string
	}{
		{
			name:        "single page nil marker in nil marker out",
			inputMarker: nil,
			lbs: []elbtypes.LoadBalancer{
				{
					LoadBalancerName: awssdk.String("prod-alb"),
					LoadBalancerArn:  awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:loadbalancer/app/prod-alb/abc"),
					Type:             elbtypes.LoadBalancerTypeEnumApplication,
					State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
					Scheme:           elbtypes.LoadBalancerSchemeEnumInternetFacing,
					DNSName:          awssdk.String("prod-alb.us-east-1.elb.amazonaws.com"),
					VpcId:            awssdk.String("vpc-prod"),
					CreatedTime:      &created,
				},
			},
			nextMarker:     nil,
			wantCount:      1,
			wantNextMarker: nil,
		},
		{
			name:        "first page with more pages marker returned",
			inputMarker: nil,
			lbs: []elbtypes.LoadBalancer{
				{
					LoadBalancerName: awssdk.String("alb-page1"),
					LoadBalancerArn:  awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:loadbalancer/app/alb-page1/001"),
					Type:             elbtypes.LoadBalancerTypeEnumApplication,
					State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
					DNSName:          awssdk.String("alb-page1.us-east-1.elb.amazonaws.com"),
				},
			},
			nextMarker:     awssdk.String("marker-page2"),
			wantCount:      1,
			wantNextMarker: awssdk.String("marker-page2"),
		},
		{
			name:        "subsequent page marker in nil marker out",
			inputMarker: awssdk.String("marker-page2"),
			lbs: []elbtypes.LoadBalancer{
				{
					LoadBalancerName: awssdk.String("alb-page2"),
					LoadBalancerArn:  awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:loadbalancer/app/alb-page2/002"),
					Type:             elbtypes.LoadBalancerTypeEnumNetwork,
					State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
					DNSName:          awssdk.String("alb-page2.us-east-1.elb.amazonaws.com"),
				},
			},
			nextMarker:     nil,
			wantCount:      1,
			wantNextMarker: nil,
		},
		{
			name:           "empty results",
			inputMarker:    nil,
			lbs:            []elbtypes.LoadBalancer{},
			nextMarker:     nil,
			wantCount:      0,
			wantNextMarker: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockELBAPI{
				describeLoadBalancersFunc: func(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
					return &elbv2.DescribeLoadBalancersOutput{
						LoadBalancers: tt.lbs,
						NextMarker:    tt.nextMarker,
					}, nil
				},
			}

			client := NewClient(mock)
			lbs, nextMarker, err := client.ListLoadBalancersPage(context.Background(), tt.inputMarker)
			require.NoError(t, err)
			assert.Len(t, lbs, tt.wantCount)
			if tt.wantNextMarker == nil {
				assert.Nil(t, nextMarker)
			} else {
				require.NotNil(t, nextMarker)
				assert.Equal(t, *tt.wantNextMarker, *nextMarker)
			}
		})
	}
}

func TestGetResourceTags(t *testing.T) {
	mock := &mockELBAPI{
		describeTagsFunc: func(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error) {
			return &elbv2.DescribeTagsOutput{
				TagDescriptions: []elbtypes.TagDescription{
					{
						ResourceArn: awssdk.String("arn:lb-1"),
						Tags: []elbtypes.Tag{
							{Key: awssdk.String("Environment"), Value: awssdk.String("prod")},
							{Key: awssdk.String("Team"), Value: awssdk.String("platform")},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	tags, err := client.GetResourceTags(context.Background(), []string{"arn:lb-1"})
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, "prod", tags["arn:lb-1"]["Environment"])
	assert.Equal(t, "platform", tags["arn:lb-1"]["Team"])
}

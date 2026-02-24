package ec2

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2API struct {
	describeInstancesFunc func(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error)
}

func (m *mockEC2API) DescribeInstances(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
	return m.describeInstancesFunc(ctx, params, optFns...)
}

func TestListInstances(t *testing.T) {
	mock := &mockEC2API{
		describeInstancesFunc: func(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
			return &awsec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{{
					Instances: []types.Instance{
						{
							InstanceId: awssdk.String("i-abc123"), InstanceType: types.InstanceTypeT3Medium,
							State: &types.InstanceState{Name: types.InstanceStateNameRunning},
							PrivateIpAddress: awssdk.String("10.0.1.50"), PublicIpAddress: awssdk.String("54.21.3.100"),
							Tags: []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("web-server")}},
						},
						{
							InstanceId: awssdk.String("i-def456"), InstanceType: types.InstanceTypeT3Large,
							State: &types.InstanceState{Name: types.InstanceStateNameStopped},
							PrivateIpAddress: awssdk.String("10.0.2.30"),
							Tags: []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("api-server")}},
						},
					},
				}},
			}, nil
		},
	}

	client := NewClient(mock)
	instances, summary, err := client.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}
	if instances[0].Name != "web-server" {
		t.Errorf("instances[0].Name = %s, want web-server", instances[0].Name)
	}
	if instances[0].PublicIP != "54.21.3.100" {
		t.Errorf("instances[0].PublicIP = %s, want 54.21.3.100", instances[0].PublicIP)
	}
	if instances[1].PublicIP != "—" {
		t.Errorf("instances[1].PublicIP = %s, want —", instances[1].PublicIP)
	}
	if summary.Total != 2 || summary.Running != 1 || summary.Stopped != 1 {
		t.Errorf("unexpected summary: %+v", summary)
	}
}

func TestListInstances_Pagination(t *testing.T) {
	callCount := 0
	mock := &mockEC2API{
		describeInstancesFunc: func(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
			callCount++
			if callCount == 1 {
				return &awsec2.DescribeInstancesOutput{
					Reservations: []types.Reservation{{Instances: []types.Instance{{
						InstanceId: awssdk.String("i-page1"), InstanceType: types.InstanceTypeT3Medium,
						State: &types.InstanceState{Name: types.InstanceStateNameRunning}, PrivateIpAddress: awssdk.String("10.0.1.1"),
					}}}},
					NextToken: awssdk.String("token2"),
				}, nil
			}
			return &awsec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{{Instances: []types.Instance{{
					InstanceId: awssdk.String("i-page2"), InstanceType: types.InstanceTypeT3Large,
					State: &types.InstanceState{Name: types.InstanceStateNameStopped}, PrivateIpAddress: awssdk.String("10.0.1.2"),
				}}}},
			}, nil
		},
	}

	client := NewClient(mock)
	instances, summary, err := client.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}
	if summary.Total != 2 || summary.Running != 1 || summary.Stopped != 1 {
		t.Errorf("unexpected summary: %+v", summary)
	}
}

package autoscaling

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
)

type mockAutoScalingAPI struct {
	describeScalableTargetsFunc func(ctx context.Context, params *applicationautoscaling.DescribeScalableTargetsInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalableTargetsOutput, error)
	describeScalingPoliciesFunc func(ctx context.Context, params *applicationautoscaling.DescribeScalingPoliciesInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalingPoliciesOutput, error)
}

func (m *mockAutoScalingAPI) DescribeScalableTargets(ctx context.Context, params *applicationautoscaling.DescribeScalableTargetsInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalableTargetsOutput, error) {
	return m.describeScalableTargetsFunc(ctx, params, optFns...)
}

func (m *mockAutoScalingAPI) DescribeScalingPolicies(ctx context.Context, params *applicationautoscaling.DescribeScalingPoliciesInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalingPoliciesOutput, error) {
	return m.describeScalingPoliciesFunc(ctx, params, optFns...)
}

func TestGetECSScalingTargets(t *testing.T) {
	mock := &mockAutoScalingAPI{
		describeScalableTargetsFunc: func(ctx context.Context, params *applicationautoscaling.DescribeScalableTargetsInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalableTargetsOutput, error) {
			if len(params.ResourceIds) != 1 || params.ResourceIds[0] != "service/prod/web" {
				t.Errorf("ResourceIds = %v, want [service/prod/web]", params.ResourceIds)
			}
			return &applicationautoscaling.DescribeScalableTargetsOutput{
				ScalableTargets: []astypes.ScalableTarget{
					{
						MinCapacity: awssdk.Int32(2),
						MaxCapacity: awssdk.Int32(10),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	targets, err := client.GetECSScalingTargets(context.Background(), "prod", "web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].MinCapacity != 2 {
		t.Errorf("MinCapacity = %d, want 2", targets[0].MinCapacity)
	}
	if targets[0].MaxCapacity != 10 {
		t.Errorf("MaxCapacity = %d, want 10", targets[0].MaxCapacity)
	}
	if targets[0].ResourceID != "service/prod/web" {
		t.Errorf("ResourceID = %s, want service/prod/web", targets[0].ResourceID)
	}
}

func TestGetECSScalingPolicies(t *testing.T) {
	mock := &mockAutoScalingAPI{
		describeScalingPoliciesFunc: func(ctx context.Context, params *applicationautoscaling.DescribeScalingPoliciesInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalingPoliciesOutput, error) {
			return &applicationautoscaling.DescribeScalingPoliciesOutput{
				ScalingPolicies: []astypes.ScalingPolicy{
					{
						PolicyName: awssdk.String("cpu-tracking"),
						PolicyType: astypes.PolicyTypeTargetTrackingScaling,
						TargetTrackingScalingPolicyConfiguration: &astypes.TargetTrackingScalingPolicyConfiguration{
							TargetValue: awssdk.Float64(70.0),
							PredefinedMetricSpecification: &astypes.PredefinedMetricSpecification{
								PredefinedMetricType: astypes.MetricTypeECSServiceAverageCPUUtilization,
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	policies, err := client.GetECSScalingPolicies(context.Background(), "prod", "web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].PolicyName != "cpu-tracking" {
		t.Errorf("PolicyName = %s, want cpu-tracking", policies[0].PolicyName)
	}
	if policies[0].TargetValue != 70.0 {
		t.Errorf("TargetValue = %f, want 70.0", policies[0].TargetValue)
	}
	if policies[0].MetricName != "ECSServiceAverageCPUUtilization" {
		t.Errorf("MetricName = %s, want ECSServiceAverageCPUUtilization", policies[0].MetricName)
	}
}

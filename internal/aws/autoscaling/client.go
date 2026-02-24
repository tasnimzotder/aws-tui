package autoscaling

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
)

type ApplicationAutoScalingAPI interface {
	DescribeScalableTargets(ctx context.Context, params *applicationautoscaling.DescribeScalableTargetsInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalableTargetsOutput, error)
	DescribeScalingPolicies(ctx context.Context, params *applicationautoscaling.DescribeScalingPoliciesInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalingPoliciesOutput, error)
}

type Client struct {
	api ApplicationAutoScalingAPI
}

func NewClient(api ApplicationAutoScalingAPI) *Client {
	return &Client{api: api}
}

func (c *Client) GetECSScalingTargets(ctx context.Context, clusterName, serviceName string) ([]AutoScalingTarget, error) {
	resourceID := fmt.Sprintf("service/%s/%s", clusterName, serviceName)
	out, err := c.api.DescribeScalableTargets(ctx, &applicationautoscaling.DescribeScalableTargetsInput{
		ServiceNamespace:  astypes.ServiceNamespaceEcs,
		ResourceIds:       []string{resourceID},
		ScalableDimension: astypes.ScalableDimensionECSServiceDesiredCount,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeScalableTargets: %w", err)
	}

	var targets []AutoScalingTarget
	for _, t := range out.ScalableTargets {
		var minCap, maxCap int
		if t.MinCapacity != nil {
			minCap = int(*t.MinCapacity)
		}
		if t.MaxCapacity != nil {
			maxCap = int(*t.MaxCapacity)
		}
		targets = append(targets, AutoScalingTarget{
			MinCapacity: minCap,
			MaxCapacity: maxCap,
			ResourceID:  resourceID,
		})
	}
	return targets, nil
}

func (c *Client) GetECSScalingPolicies(ctx context.Context, clusterName, serviceName string) ([]AutoScalingPolicy, error) {
	resourceID := fmt.Sprintf("service/%s/%s", clusterName, serviceName)
	out, err := c.api.DescribeScalingPolicies(ctx, &applicationautoscaling.DescribeScalingPoliciesInput{
		ServiceNamespace:  astypes.ServiceNamespaceEcs,
		ResourceId:        &resourceID,
		ScalableDimension: astypes.ScalableDimensionECSServiceDesiredCount,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeScalingPolicies: %w", err)
	}

	var policies []AutoScalingPolicy
	for _, p := range out.ScalingPolicies {
		policy := AutoScalingPolicy{
			PolicyName: derefStr(p.PolicyName),
			PolicyType: string(p.PolicyType),
		}
		if p.TargetTrackingScalingPolicyConfiguration != nil {
			policy.TargetValue = *p.TargetTrackingScalingPolicyConfiguration.TargetValue
			if p.TargetTrackingScalingPolicyConfiguration.PredefinedMetricSpecification != nil {
				policy.MetricName = string(p.TargetTrackingScalingPolicyConfiguration.PredefinedMetricSpecification.PredefinedMetricType)
			}
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

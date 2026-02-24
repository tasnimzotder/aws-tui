package ecs

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type mockECSAPI struct {
	listClustersFunc           func(ctx context.Context, params *awsecs.ListClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.ListClustersOutput, error)
	describeClustersFunc       func(ctx context.Context, params *awsecs.DescribeClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeClustersOutput, error)
	listServicesFunc           func(ctx context.Context, params *awsecs.ListServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.ListServicesOutput, error)
	describeServicesFunc       func(ctx context.Context, params *awsecs.DescribeServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeServicesOutput, error)
	listTasksFunc              func(ctx context.Context, params *awsecs.ListTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.ListTasksOutput, error)
	describeTasksFunc          func(ctx context.Context, params *awsecs.DescribeTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTasksOutput, error)
	describeTaskDefinitionFunc func(ctx context.Context, params *awsecs.DescribeTaskDefinitionInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTaskDefinitionOutput, error)
}

func (m *mockECSAPI) ListClusters(ctx context.Context, params *awsecs.ListClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.ListClustersOutput, error) {
	return m.listClustersFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeClusters(ctx context.Context, params *awsecs.DescribeClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeClustersOutput, error) {
	return m.describeClustersFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) ListServices(ctx context.Context, params *awsecs.ListServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.ListServicesOutput, error) {
	return m.listServicesFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeServices(ctx context.Context, params *awsecs.DescribeServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeServicesOutput, error) {
	return m.describeServicesFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) ListTasks(ctx context.Context, params *awsecs.ListTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.ListTasksOutput, error) {
	return m.listTasksFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeTasks(ctx context.Context, params *awsecs.DescribeTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTasksOutput, error) {
	return m.describeTasksFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeTaskDefinition(ctx context.Context, params *awsecs.DescribeTaskDefinitionInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTaskDefinitionOutput, error) {
	return m.describeTaskDefinitionFunc(ctx, params, optFns...)
}

func TestListClusters(t *testing.T) {
	mock := &mockECSAPI{
		listClustersFunc: func(ctx context.Context, params *awsecs.ListClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.ListClustersOutput, error) {
			return &awsecs.ListClustersOutput{
				ClusterArns: []string{"arn:aws:ecs:us-east-1:123456:cluster/prod"},
			}, nil
		},
		describeClustersFunc: func(ctx context.Context, params *awsecs.DescribeClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeClustersOutput, error) {
			return &awsecs.DescribeClustersOutput{
				Clusters: []ecstypes.Cluster{
					{
						ClusterName:         awssdk.String("prod"),
						ClusterArn:          awssdk.String("arn:aws:ecs:us-east-1:123456:cluster/prod"),
						Status:              awssdk.String("ACTIVE"),
						RunningTasksCount:   12,
						ActiveServicesCount: 5,
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Name != "prod" {
		t.Errorf("Name = %s, want prod", clusters[0].Name)
	}
	if clusters[0].RunningTaskCount != 12 {
		t.Errorf("RunningTaskCount = %d, want 12", clusters[0].RunningTaskCount)
	}
	if clusters[0].ServiceCount != 5 {
		t.Errorf("ServiceCount = %d, want 5", clusters[0].ServiceCount)
	}
	if clusters[0].ARN != "arn:aws:ecs:us-east-1:123456:cluster/prod" {
		t.Errorf("ARN = %s, want arn:aws:ecs:us-east-1:123456:cluster/prod", clusters[0].ARN)
	}
}

func TestListClusters_Pagination(t *testing.T) {
	listCalls := 0
	mock := &mockECSAPI{
		listClustersFunc: func(ctx context.Context, params *awsecs.ListClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.ListClustersOutput, error) {
			listCalls++
			if listCalls == 1 {
				return &awsecs.ListClustersOutput{
					ClusterArns: []string{"arn:aws:ecs:us-east-1:123456:cluster/c1"},
					NextToken:   awssdk.String("page2"),
				}, nil
			}
			return &awsecs.ListClustersOutput{
				ClusterArns: []string{"arn:aws:ecs:us-east-1:123456:cluster/c2"},
			}, nil
		},
		describeClustersFunc: func(ctx context.Context, params *awsecs.DescribeClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeClustersOutput, error) {
			var clusters []ecstypes.Cluster
			for _, arn := range params.Clusters {
				name := arn[len(arn)-2:]
				clusters = append(clusters, ecstypes.Cluster{
					ClusterName: awssdk.String(name),
					ClusterArn:  awssdk.String(arn),
					Status:      awssdk.String("ACTIVE"),
				})
			}
			return &awsecs.DescribeClustersOutput{Clusters: clusters}, nil
		},
	}

	client := NewClient(mock)
	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCalls != 2 {
		t.Errorf("expected 2 ListClusters calls, got %d", listCalls)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
}

func TestListServices_PendingCount(t *testing.T) {
	mock := &mockECSAPI{
		listServicesFunc: func(ctx context.Context, params *awsecs.ListServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.ListServicesOutput, error) {
			return &awsecs.ListServicesOutput{
				ServiceArns: []string{"arn:aws:ecs:us-east-1:123456:service/prod/web"},
			}, nil
		},
		describeServicesFunc: func(ctx context.Context, params *awsecs.DescribeServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeServicesOutput, error) {
			return &awsecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						ServiceName:    awssdk.String("web"),
						ServiceArn:     awssdk.String("arn:aws:ecs:us-east-1:123456:service/prod/web"),
						Status:         awssdk.String("ACTIVE"),
						DesiredCount:   3,
						RunningCount:   2,
						PendingCount:   1,
						TaskDefinition: awssdk.String("arn:aws:ecs:us-east-1:123456:task-definition/web:5"),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	services, err := client.ListServices(context.Background(), "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].PendingCount != 1 {
		t.Errorf("PendingCount = %d, want 1", services[0].PendingCount)
	}
}

func TestDescribeService(t *testing.T) {
	eventTime := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	deployTime := time.Date(2026, 2, 19, 8, 0, 0, 0, time.UTC)

	mock := &mockECSAPI{
		describeServicesFunc: func(ctx context.Context, params *awsecs.DescribeServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeServicesOutput, error) {
			return &awsecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						ServiceName:          awssdk.String("web"),
						ServiceArn:           awssdk.String("arn:aws:ecs:us-east-1:123456:service/prod/web"),
						Status:               awssdk.String("ACTIVE"),
						DesiredCount:         3,
						RunningCount:         3,
						PendingCount:         0,
						TaskDefinition:       awssdk.String("arn:aws:ecs:us-east-1:123456:task-definition/web:5"),
						LaunchType:           ecstypes.LaunchTypeFargate,
						EnableExecuteCommand: true,
						Events: []ecstypes.ServiceEvent{
							{
								Id:        awssdk.String("evt-1"),
								CreatedAt: &eventTime,
								Message:   awssdk.String("service web has reached a steady state."),
							},
						},
						Deployments: []ecstypes.Deployment{
							{
								Id:             awssdk.String("dep-1"),
								Status:         awssdk.String("PRIMARY"),
								TaskDefinition: awssdk.String("arn:aws:ecs:us-east-1:123456:task-definition/web:5"),
								DesiredCount:   3,
								RunningCount:   3,
								PendingCount:   0,
								RolloutState:   ecstypes.DeploymentRolloutStateCompleted,
								CreatedAt:      &deployTime,
							},
						},
						LoadBalancers: []ecstypes.LoadBalancer{
							{
								TargetGroupArn: awssdk.String("arn:aws:elasticloadbalancing:us-east-1:123456:targetgroup/web-tg/abc123"),
								ContainerName:  awssdk.String("web"),
								ContainerPort:  awssdk.Int32(8080),
							},
						},
						PlacementConstraints: []ecstypes.PlacementConstraint{
							{
								Type:       ecstypes.PlacementConstraintTypeDistinctInstance,
								Expression: awssdk.String(""),
							},
						},
						PlacementStrategy: []ecstypes.PlacementStrategy{
							{
								Type:  ecstypes.PlacementStrategyTypeSpread,
								Field: awssdk.String("attribute:ecs.availability-zone"),
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	detail, err := client.DescribeService(context.Background(), "prod", "web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.Name != "web" {
		t.Errorf("Name = %s, want web", detail.Name)
	}
	if detail.LaunchType != "FARGATE" {
		t.Errorf("LaunchType = %s, want FARGATE", detail.LaunchType)
	}
	if !detail.EnableExecuteCommand {
		t.Error("EnableExecuteCommand = false, want true")
	}
	if detail.TaskDef != "web:5" {
		t.Errorf("TaskDef = %s, want web:5", detail.TaskDef)
	}
	if len(detail.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(detail.Events))
	}
	if detail.Events[0].ID != "evt-1" {
		t.Errorf("Event ID = %s, want evt-1", detail.Events[0].ID)
	}
	if len(detail.Deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(detail.Deployments))
	}
	if detail.Deployments[0].RolloutState != "COMPLETED" {
		t.Errorf("RolloutState = %s, want COMPLETED", detail.Deployments[0].RolloutState)
	}
	if len(detail.LoadBalancers) != 1 {
		t.Fatalf("expected 1 load balancer, got %d", len(detail.LoadBalancers))
	}
	if detail.LoadBalancers[0].ContainerPort != 8080 {
		t.Errorf("ContainerPort = %d, want 8080", detail.LoadBalancers[0].ContainerPort)
	}
	if len(detail.PlacementConstraints) != 1 {
		t.Fatalf("expected 1 placement constraint, got %d", len(detail.PlacementConstraints))
	}
	if len(detail.PlacementStrategy) != 1 {
		t.Fatalf("expected 1 placement strategy, got %d", len(detail.PlacementStrategy))
	}
	if detail.PlacementStrategy[0].Field != "attribute:ecs.availability-zone" {
		t.Errorf("PlacementStrategy Field = %s, want attribute:ecs.availability-zone", detail.PlacementStrategy[0].Field)
	}
}

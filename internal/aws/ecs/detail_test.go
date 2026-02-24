package ecs

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestDescribeTask(t *testing.T) {
	started := time.Date(2026, 2, 24, 10, 30, 0, 0, time.UTC)
	mock := &mockECSAPI{
		describeTasksFunc: func(ctx context.Context, params *awsecs.DescribeTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTasksOutput, error) {
			return &awsecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{
					{
						TaskArn:           awssdk.String("arn:aws:ecs:us-east-1:123456:task/prod/abc123def"),
						LastStatus:        awssdk.String("RUNNING"),
						TaskDefinitionArn: awssdk.String("arn:aws:ecs:us-east-1:123456:task-definition/web-api:42"),
						StartedAt:         &started,
						Cpu:               awssdk.String("256"),
						Memory:            awssdk.String("512"),
						Attachments: []ecstypes.Attachment{
							{
								Type: awssdk.String("ElasticNetworkInterface"),
								Details: []ecstypes.KeyValuePair{
									{Name: awssdk.String("privateIPv4Address"), Value: awssdk.String("10.0.1.55")},
									{Name: awssdk.String("subnetId"), Value: awssdk.String("subnet-abc123")},
									{Name: awssdk.String("networkInterfaceId"), Value: awssdk.String("eni-xyz")},
								},
							},
						},
						Containers: []ecstypes.Container{
							{
								Name:         awssdk.String("app"),
								Image:        awssdk.String("123456.dkr.ecr.us-east-1.amazonaws.com/web:latest"),
								LastStatus:   awssdk.String("RUNNING"),
								HealthStatus: ecstypes.HealthStatusHealthy,
							},
						},
					},
				},
			}, nil
		},
		describeTaskDefinitionFunc: func(ctx context.Context, params *awsecs.DescribeTaskDefinitionInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTaskDefinitionOutput, error) {
			return &awsecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name: awssdk.String("app"),
							LogConfiguration: &ecstypes.LogConfiguration{
								LogDriver: ecstypes.LogDriverAwslogs,
								Options: map[string]string{
									"awslogs-group":         "/ecs/web-api",
									"awslogs-stream-prefix": "ecs",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	detail, err := client.DescribeTask(context.Background(), "prod", "arn:aws:ecs:us-east-1:123456:task/prod/abc123def")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.TaskID != "abc123def" {
		t.Errorf("TaskID = %s, want abc123def", detail.TaskID)
	}
	if detail.CPU != "256" {
		t.Errorf("CPU = %s, want 256", detail.CPU)
	}
	if detail.PrivateIP != "10.0.1.55" {
		t.Errorf("PrivateIP = %s, want 10.0.1.55", detail.PrivateIP)
	}
	if detail.NetworkMode != "awsvpc" {
		t.Errorf("NetworkMode = %s, want awsvpc", detail.NetworkMode)
	}
	if len(detail.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(detail.Containers))
	}
	if detail.Containers[0].Name != "app" {
		t.Errorf("Container.Name = %s, want app", detail.Containers[0].Name)
	}
	if detail.Containers[0].LogGroup != "/ecs/web-api" {
		t.Errorf("Container.LogGroup = %s, want /ecs/web-api", detail.Containers[0].LogGroup)
	}
	if detail.Containers[0].LogStream != "ecs/app/abc123def" {
		t.Errorf("Container.LogStream = %s, want ecs/app/abc123def", detail.Containers[0].LogStream)
	}
}

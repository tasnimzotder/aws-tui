package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	awss3sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	awsautoscaling "tasnim.dev/aws-tui/internal/aws/autoscaling"
	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	awsecs "tasnim.dev/aws-tui/internal/aws/ecs"
	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	awslogs "tasnim.dev/aws-tui/internal/aws/logs"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
)

type ServiceClient struct {
	EC2         *awsec2.Client
	ECS         *awsecs.Client
	VPC         *awsvpc.Client
	ECR         *awsecr.Client
	ELB         *awselb.Client
	Logs        *awslogs.Client
	AutoScaling *awsautoscaling.Client
	S3          *awss3.Client
}

func NewServiceClient(ctx context.Context, profile, region string) (*ServiceClient, error) {
	cfg, err := LoadConfig(ctx, profile, region)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	return &ServiceClient{
		EC2:         awsec2.NewClient(ec2Client),
		ECS:         awsecs.NewClient(ecs.NewFromConfig(cfg)),
		VPC:         awsvpc.NewClient(ec2Client),
		ECR:         awsecr.NewClient(ecr.NewFromConfig(cfg)),
		ELB:         awselb.NewClient(elbv2.NewFromConfig(cfg)),
		Logs:        awslogs.NewClient(cloudwatchlogs.NewFromConfig(cfg)),
		AutoScaling: awsautoscaling.NewClient(applicationautoscaling.NewFromConfig(cfg)),
		S3:          awss3.NewClient(awss3sdk.NewFromConfig(cfg)),
	}, nil
}

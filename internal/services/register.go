package services

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2sdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsecrsdk "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecssdk "github.com/aws/aws-sdk-go-v2/service/ecs"
	awsekssdk "github.com/aws/aws-sdk-go-v2/service/eks"
	awselbsdk "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	awsiamsdk "github.com/aws/aws-sdk-go-v2/service/iam"
	awss3sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	awscost "tasnim.dev/aws-tui/internal/aws/cost"
	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	awsecs "tasnim.dev/aws-tui/internal/aws/ecs"
	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/plugin"
	svccost "tasnim.dev/aws-tui/internal/services/cost"
	svcec2 "tasnim.dev/aws-tui/internal/services/ec2"
	svcecr "tasnim.dev/aws-tui/internal/services/ecr"
	svcecs "tasnim.dev/aws-tui/internal/services/ecs"
	svceks "tasnim.dev/aws-tui/internal/services/eks"
	svcelb "tasnim.dev/aws-tui/internal/services/elb"
	svciam "tasnim.dev/aws-tui/internal/services/iam"
	svcs3 "tasnim.dev/aws-tui/internal/services/s3"
	svcvpc "tasnim.dev/aws-tui/internal/services/vpc"
)

// Register creates all AWS service clients from the given config and registers
// their corresponding service plugins with the registry.
func Register(reg *plugin.Registry, cfg aws.Config, region, profile string) {
	ec2api := awsec2sdk.NewFromConfig(cfg)

	reg.Add(svcec2.NewPlugin(awsec2.NewClient(ec2api), region, profile))
	reg.Add(svcecs.NewPlugin(awsecs.NewClient(awsecssdk.NewFromConfig(cfg)), region, profile))
	reg.Add(svceks.NewPlugin(awseks.NewClient(awsekssdk.NewFromConfig(cfg)), region, profile))
	reg.Add(svcvpc.NewPlugin(awsvpc.NewClient(ec2api)))
	reg.Add(svcs3.NewPlugin(awss3.NewClient(awss3sdk.NewFromConfig(cfg))))
	reg.Add(svciam.NewPlugin(awsiam.NewClient(awsiamsdk.NewFromConfig(cfg))))
	reg.Add(svcecr.NewPlugin(awsecr.NewClient(awsecrsdk.NewFromConfig(cfg))))
	reg.Add(svcelb.NewPlugin(awselb.NewClient(awselbsdk.NewFromConfig(cfg))))
	reg.Add(svccost.NewPlugin(awscost.NewClient(cfg)))
}

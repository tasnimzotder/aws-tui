package ec2

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2API struct {
	describeInstancesFunc func(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error)
	describeVolumesFunc   func(ctx context.Context, params *awsec2.DescribeVolumesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVolumesOutput, error)
}

func (m *mockEC2API) DescribeInstances(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
	return m.describeInstancesFunc(ctx, params, optFns...)
}

func (m *mockEC2API) DescribeVolumes(ctx context.Context, params *awsec2.DescribeVolumesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVolumesOutput, error) {
	return m.describeVolumesFunc(ctx, params, optFns...)
}

func TestListInstances(t *testing.T) {
	launchTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	deleteOnTerm := true
	mock := &mockEC2API{
		describeInstancesFunc: func(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
			return &awsec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{{
					Instances: []types.Instance{
						{
							InstanceId:       awssdk.String("i-abc123"),
							InstanceType:     types.InstanceTypeT3Medium,
							State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
							PrivateIpAddress: awssdk.String("10.0.1.50"),
							PublicIpAddress:   awssdk.String("54.21.3.100"),
							Tags:             []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("web-server")}},
							LaunchTime:       &launchTime,
							Placement:        &types.Placement{AvailabilityZone: awssdk.String("us-east-1a")},
							Architecture:     types.ArchitectureValuesX8664,
							ImageId:          awssdk.String("ami-0abc"),
							KeyName:          awssdk.String("my-key"),
							VpcId:            awssdk.String("vpc-abc123"),
							SubnetId:         awssdk.String("subnet-def456"),
							PlatformDetails:  awssdk.String("Linux/UNIX"),
							SecurityGroups: []types.GroupIdentifier{
								{GroupId: awssdk.String("sg-111"), GroupName: awssdk.String("web-sg")},
							},
							BlockDeviceMappings: []types.InstanceBlockDeviceMapping{
								{
									DeviceName: awssdk.String("/dev/xvda"),
									Ebs: &types.EbsInstanceBlockDevice{
										VolumeId:            awssdk.String("vol-aaa"),
										Status:              types.AttachmentStatusAttached,
										DeleteOnTermination: &deleteOnTerm,
									},
								},
							},
						},
						{
							InstanceId:       awssdk.String("i-def456"),
							InstanceType:     types.InstanceTypeT3Large,
							State:            &types.InstanceState{Name: types.InstanceStateNameStopped},
							PrivateIpAddress: awssdk.String("10.0.2.30"),
							Tags:             []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("api-server")}},
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

	// Original assertions
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

	// New field assertions
	inst := instances[0]
	if inst.AZ != "us-east-1a" {
		t.Errorf("AZ = %s, want us-east-1a", inst.AZ)
	}
	if inst.Architecture != "x86_64" {
		t.Errorf("Architecture = %s, want x86_64", inst.Architecture)
	}
	if inst.ImageID != "ami-0abc" {
		t.Errorf("ImageID = %s, want ami-0abc", inst.ImageID)
	}
	if inst.KeyName != "my-key" {
		t.Errorf("KeyName = %s, want my-key", inst.KeyName)
	}
	if inst.VpcID != "vpc-abc123" {
		t.Errorf("VpcID = %s, want vpc-abc123", inst.VpcID)
	}
	if inst.SubnetID != "subnet-def456" {
		t.Errorf("SubnetID = %s, want subnet-def456", inst.SubnetID)
	}
	if inst.Platform != "Linux/UNIX" {
		t.Errorf("Platform = %s, want Linux/UNIX", inst.Platform)
	}
	if !inst.LaunchTime.Equal(launchTime) {
		t.Errorf("LaunchTime = %v, want %v", inst.LaunchTime, launchTime)
	}
	if len(inst.SecurityGroups) != 1 || inst.SecurityGroups[0].GroupID != "sg-111" {
		t.Errorf("SecurityGroups = %+v, want [{sg-111 web-sg}]", inst.SecurityGroups)
	}
	if len(inst.Volumes) != 1 || inst.Volumes[0].VolumeID != "vol-aaa" {
		t.Errorf("Volumes = %+v, want [{/dev/xvda vol-aaa attached true}]", inst.Volumes)
	}
	if inst.Tags["Name"] != "web-server" {
		t.Errorf("Tags[Name] = %s, want web-server", inst.Tags["Name"])
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

func TestGetInstanceVolumes(t *testing.T) {
	mock := &mockEC2API{
		describeVolumesFunc: func(ctx context.Context, params *awsec2.DescribeVolumesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVolumesOutput, error) {
			encrypted := true
			return &awsec2.DescribeVolumesOutput{
				Volumes: []types.Volume{
					{
						VolumeId:         awssdk.String("vol-aaa"),
						Size:             awssdk.Int32(100),
						VolumeType:       types.VolumeTypeGp3,
						State:            types.VolumeStateInUse,
						Iops:             awssdk.Int32(3000),
						Encrypted:        &encrypted,
						AvailabilityZone: awssdk.String("us-east-1a"),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	vols, err := client.GetInstanceVolumes(context.Background(), []string{"vol-aaa"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vols) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(vols))
	}
	v := vols[0]
	if v.VolumeID != "vol-aaa" {
		t.Errorf("VolumeID = %s, want vol-aaa", v.VolumeID)
	}
	if v.Size != 100 {
		t.Errorf("Size = %d, want 100", v.Size)
	}
	if v.VolumeType != "gp3" {
		t.Errorf("VolumeType = %s, want gp3", v.VolumeType)
	}
	if !v.Encrypted {
		t.Error("expected Encrypted=true")
	}
}

func TestGetInstanceVolumes_Empty(t *testing.T) {
	client := NewClient(&mockEC2API{})
	vols, err := client.GetInstanceVolumes(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vols) != 0 {
		t.Errorf("expected 0 volumes, got %d", len(vols))
	}
}

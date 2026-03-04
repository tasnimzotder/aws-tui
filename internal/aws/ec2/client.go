package ec2

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2API defines the subset of the EC2 SDK client used by this package.
type EC2API interface {
	DescribeInstances(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error)
	DescribeVolumes(ctx context.Context, params *awsec2.DescribeVolumesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVolumesOutput, error)
}

// Client wraps an EC2API for higher-level operations.
type Client struct {
	api EC2API
}

// NewClient creates a new EC2 Client.
func NewClient(api EC2API) *Client {
	return &Client{api: api}
}

// EC2Instance represents a single EC2 instance.
type EC2Instance struct {
	Name       string
	InstanceID string
	Type       string
	State      string
	PrivateIP  string
	PublicIP   string

	LaunchTime     time.Time
	AZ             string
	Architecture   string
	ImageID        string
	KeyName        string
	IAMProfile     string
	VpcID          string
	SubnetID       string
	SecurityGroups []EC2SecurityGroup
	Volumes        []EC2BlockDevice
	Tags           map[string]string
	Platform       string
}

// EC2SecurityGroup is a minimal SG reference attached to an instance.
type EC2SecurityGroup struct {
	GroupID   string
	GroupName string
}

// EC2BlockDevice represents a block device mapping on an instance.
type EC2BlockDevice struct {
	DeviceName          string
	VolumeID            string
	Status              string
	DeleteOnTermination bool
}

// EBSVolume holds details for a single EBS volume.
type EBSVolume struct {
	VolumeID   string
	Size       int32
	VolumeType string
	State      string
	IOPS       int32
	Encrypted  bool
	AZ         string
}

// EC2Summary holds aggregate instance counts.
type EC2Summary struct {
	Total   int
	Running int
	Stopped int
}

// parseInstance converts an SDK Instance into an EC2Instance.
func parseInstance(inst types.Instance) EC2Instance {
	name := ""
	tags := make(map[string]string)
	for _, tag := range inst.Tags {
		k := aws.ToString(tag.Key)
		v := aws.ToString(tag.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}

	publicIP := "\u2014"
	if inst.PublicIpAddress != nil {
		publicIP = aws.ToString(inst.PublicIpAddress)
	}

	sgs := make([]EC2SecurityGroup, len(inst.SecurityGroups))
	for i, sg := range inst.SecurityGroups {
		sgs[i] = EC2SecurityGroup{
			GroupID:   aws.ToString(sg.GroupId),
			GroupName: aws.ToString(sg.GroupName),
		}
	}

	var volumes []EC2BlockDevice
	for _, bdm := range inst.BlockDeviceMappings {
		bd := EC2BlockDevice{
			DeviceName: aws.ToString(bdm.DeviceName),
		}
		if bdm.Ebs != nil {
			bd.VolumeID = aws.ToString(bdm.Ebs.VolumeId)
			bd.Status = string(bdm.Ebs.Status)
			if bdm.Ebs.DeleteOnTermination != nil {
				bd.DeleteOnTermination = *bdm.Ebs.DeleteOnTermination
			}
		}
		volumes = append(volumes, bd)
	}

	iamProfile := ""
	if inst.IamInstanceProfile != nil {
		iamProfile = aws.ToString(inst.IamInstanceProfile.Arn)
	}

	az := ""
	if inst.Placement != nil {
		az = aws.ToString(inst.Placement.AvailabilityZone)
	}

	return EC2Instance{
		Name:           name,
		InstanceID:     aws.ToString(inst.InstanceId),
		Type:           string(inst.InstanceType),
		State:          string(inst.State.Name),
		PrivateIP:      aws.ToString(inst.PrivateIpAddress),
		PublicIP:       publicIP,
		LaunchTime:     aws.ToTime(inst.LaunchTime),
		AZ:             az,
		Architecture:   string(inst.Architecture),
		ImageID:        aws.ToString(inst.ImageId),
		KeyName:        aws.ToString(inst.KeyName),
		IAMProfile:     iamProfile,
		VpcID:          aws.ToString(inst.VpcId),
		SubnetID:       aws.ToString(inst.SubnetId),
		SecurityGroups: sgs,
		Volumes:        volumes,
		Tags:           tags,
		Platform:       aws.ToString(inst.PlatformDetails),
	}
}

// updateSummary increments summary counters based on instance state.
func updateSummary(summary *EC2Summary, state types.InstanceStateName) {
	summary.Total++
	switch state {
	case types.InstanceStateNameRunning:
		summary.Running++
	case types.InstanceStateNameStopped:
		summary.Stopped++
	}
}

// ListInstances fetches all EC2 instances across all pages.
func (c *Client) ListInstances(ctx context.Context) ([]EC2Instance, EC2Summary, error) {
	var instances []EC2Instance
	var summary EC2Summary
	var nextToken *string

	for {
		out, err := c.api.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, EC2Summary{}, fmt.Errorf("DescribeInstances: %w", err)
		}

		for _, reservation := range out.Reservations {
			for _, inst := range reservation.Instances {
				instances = append(instances, parseInstance(inst))
				updateSummary(&summary, inst.State.Name)
			}
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return instances, summary, nil
}

// GetInstanceVolumes fetches EBS volume details for the given volume IDs.
func (c *Client) GetInstanceVolumes(ctx context.Context, volumeIDs []string) ([]EBSVolume, error) {
	if len(volumeIDs) == 0 {
		return nil, nil
	}

	out, err := c.api.DescribeVolumes(ctx, &awsec2.DescribeVolumesInput{
		VolumeIds: volumeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeVolumes: %w", err)
	}

	volumes := make([]EBSVolume, len(out.Volumes))
	for i, vol := range out.Volumes {
		volumes[i] = EBSVolume{
			VolumeID:   aws.ToString(vol.VolumeId),
			Size:       aws.ToInt32(vol.Size),
			VolumeType: string(vol.VolumeType),
			State:      string(vol.State),
			IOPS:       aws.ToInt32(vol.Iops),
			Encrypted:  aws.ToBool(vol.Encrypted),
			AZ:         aws.ToString(vol.AvailabilityZone),
		}
	}
	return volumes, nil
}

// ListInstancesPage fetches a single page of EC2 instances with summary.
func (c *Client) ListInstancesPage(ctx context.Context, token *string) ([]EC2Instance, EC2Summary, *string, error) {
	out, err := c.api.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		NextToken: token,
	})
	if err != nil {
		return nil, EC2Summary{}, nil, fmt.Errorf("DescribeInstances: %w", err)
	}

	var instances []EC2Instance
	var summary EC2Summary
	for _, reservation := range out.Reservations {
		for _, inst := range reservation.Instances {
			instances = append(instances, parseInstance(inst))
			updateSummary(&summary, inst.State.Name)
		}
	}

	return instances, summary, out.NextToken, nil
}

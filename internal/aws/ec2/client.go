package ec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type EC2API interface {
	DescribeInstances(ctx context.Context, params *awsec2.DescribeInstancesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error)
	DescribeVolumes(ctx context.Context, params *awsec2.DescribeVolumesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVolumesOutput, error)
}

type Client struct {
	api EC2API
}

func NewClient(api EC2API) *Client {
	return &Client{api: api}
}

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

				publicIP := "—"
				if inst.PublicIpAddress != nil {
					publicIP = aws.ToString(inst.PublicIpAddress)
				}

				// Security groups
				sgs := make([]EC2SecurityGroup, len(inst.SecurityGroups))
				for i, sg := range inst.SecurityGroups {
					sgs[i] = EC2SecurityGroup{
						GroupID:   aws.ToString(sg.GroupId),
						GroupName: aws.ToString(sg.GroupName),
					}
				}

				// Block device mappings
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

				// IAM instance profile ARN
				iamProfile := ""
				if inst.IamInstanceProfile != nil {
					iamProfile = aws.ToString(inst.IamInstanceProfile.Arn)
				}

				// AZ
				az := ""
				if inst.Placement != nil {
					az = aws.ToString(inst.Placement.AvailabilityZone)
				}

				// Launch time
				var launchTime = inst.LaunchTime

				state := string(inst.State.Name)
				instances = append(instances, EC2Instance{
					Name:           name,
					InstanceID:     aws.ToString(inst.InstanceId),
					Type:           string(inst.InstanceType),
					State:          state,
					PrivateIP:      aws.ToString(inst.PrivateIpAddress),
					PublicIP:       publicIP,
					LaunchTime:     aws.ToTime(launchTime),
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
				})

				summary.Total++
				switch inst.State.Name {
				case types.InstanceStateNameRunning:
					summary.Running++
				case types.InstanceStateNameStopped:
					summary.Stopped++
				}
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

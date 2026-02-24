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
				for _, tag := range inst.Tags {
					if aws.ToString(tag.Key) == "Name" {
						name = aws.ToString(tag.Value)
						break
					}
				}

				publicIP := "—"
				if inst.PublicIpAddress != nil {
					publicIP = aws.ToString(inst.PublicIpAddress)
				}

				state := string(inst.State.Name)
				instances = append(instances, EC2Instance{
					Name:       name,
					InstanceID: aws.ToString(inst.InstanceId),
					Type:       string(inst.InstanceType),
					State:      state,
					PrivateIP:  aws.ToString(inst.PrivateIpAddress),
					PublicIP:   publicIP,
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

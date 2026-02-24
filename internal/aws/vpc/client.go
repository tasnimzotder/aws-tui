package vpc

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type VPCAPI interface {
	DescribeVpcs(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error)
	DescribeInternetGateways(ctx context.Context, params *awsec2.DescribeInternetGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInternetGatewaysOutput, error)
}

type Client struct {
	api VPCAPI
}

func NewClient(api VPCAPI) *Client {
	return &Client{api: api}
}

func nameFromTags(tags []types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func (c *Client) ListVPCs(ctx context.Context) ([]VPCInfo, error) {
	var vpcs []VPCInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeVpcs(ctx, &awsec2.DescribeVpcsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeVpcs: %w", err)
		}

		for _, v := range out.Vpcs {
			vpcs = append(vpcs, VPCInfo{
				VPCID:     aws.ToString(v.VpcId),
				Name:      nameFromTags(v.Tags),
				CIDR:      aws.ToString(v.CidrBlock),
				IsDefault: aws.ToBool(v.IsDefault),
				State:     string(v.State),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return vpcs, nil
}

func (c *Client) ListSubnets(ctx context.Context, vpcID string) ([]SubnetInfo, error) {
	var subnets []SubnetInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeSubnets(ctx, &awsec2.DescribeSubnetsInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeSubnets: %w", err)
		}

		for _, s := range out.Subnets {
			subnets = append(subnets, SubnetInfo{
				SubnetID:     aws.ToString(s.SubnetId),
				Name:         nameFromTags(s.Tags),
				CIDR:         aws.ToString(s.CidrBlock),
				AZ:           aws.ToString(s.AvailabilityZone),
				AvailableIPs: int(aws.ToInt32(s.AvailableIpAddressCount)),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return subnets, nil
}

func (c *Client) ListSecurityGroups(ctx context.Context, vpcID string) ([]SecurityGroupInfo, error) {
	var sgs []SecurityGroupInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeSecurityGroups(ctx, &awsec2.DescribeSecurityGroupsInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeSecurityGroups: %w", err)
		}

		for _, sg := range out.SecurityGroups {
			sgs = append(sgs, SecurityGroupInfo{
				GroupID:       aws.ToString(sg.GroupId),
				Name:          aws.ToString(sg.GroupName),
				Description:   aws.ToString(sg.Description),
				InboundRules:  len(sg.IpPermissions),
				OutboundRules: len(sg.IpPermissionsEgress),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return sgs, nil
}

func (c *Client) ListInternetGateways(ctx context.Context, vpcID string) ([]InternetGatewayInfo, error) {
	var igws []InternetGatewayInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeInternetGateways(ctx, &awsec2.DescribeInternetGatewaysInput{
			Filters: []types.Filter{
				{Name: aws.String("attachment.vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeInternetGateways: %w", err)
		}

		for _, igw := range out.InternetGateways {
			state := "detached"
			for _, att := range igw.Attachments {
				if aws.ToString(att.VpcId) == vpcID {
					state = string(att.State)
					break
				}
			}
			igws = append(igws, InternetGatewayInfo{
				GatewayID: aws.ToString(igw.InternetGatewayId),
				Name:      nameFromTags(igw.Tags),
				State:     state,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return igws, nil
}

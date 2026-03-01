package vpc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type VPCAPI interface {
	DescribeVpcs(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error)
	DescribeInternetGateways(ctx context.Context, params *awsec2.DescribeInternetGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInternetGatewaysOutput, error)
	DescribeRouteTables(ctx context.Context, params *awsec2.DescribeRouteTablesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeRouteTablesOutput, error)
	DescribeNatGateways(ctx context.Context, params *awsec2.DescribeNatGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNatGatewaysOutput, error)
	DescribeSecurityGroupRules(ctx context.Context, params *awsec2.DescribeSecurityGroupRulesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupRulesOutput, error)
	DescribeVpcEndpoints(ctx context.Context, params *awsec2.DescribeVpcEndpointsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcEndpointsOutput, error)
	DescribeVpcPeeringConnections(ctx context.Context, params *awsec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcPeeringConnectionsOutput, error)
	DescribeNetworkAcls(ctx context.Context, params *awsec2.DescribeNetworkAclsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNetworkAclsOutput, error)
	DescribeFlowLogs(ctx context.Context, params *awsec2.DescribeFlowLogsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeFlowLogsOutput, error)
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

func (c *Client) ListRouteTables(ctx context.Context, vpcID string) ([]RouteTableInfo, error) {
	var rts []RouteTableInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeRouteTables(ctx, &awsec2.DescribeRouteTablesInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeRouteTables: %w", err)
		}

		for _, rt := range out.RouteTables {
			info := RouteTableInfo{
				RouteTableID: aws.ToString(rt.RouteTableId),
				Name:         nameFromTags(rt.Tags),
			}

			for _, r := range rt.Routes {
				dest := aws.ToString(r.DestinationCidrBlock)
				if dest == "" {
					dest = aws.ToString(r.DestinationIpv6CidrBlock)
				}
				if dest == "" {
					dest = aws.ToString(r.DestinationPrefixListId)
				}

				target := "local"
				if gw := aws.ToString(r.GatewayId); gw != "" && gw != "local" {
					target = gw
				} else if nat := aws.ToString(r.NatGatewayId); nat != "" {
					target = nat
				} else if pcx := aws.ToString(r.VpcPeeringConnectionId); pcx != "" {
					target = pcx
				} else if tgw := aws.ToString(r.TransitGatewayId); tgw != "" {
					target = tgw
				} else if eni := aws.ToString(r.NetworkInterfaceId); eni != "" {
					target = eni
				}

				info.Routes = append(info.Routes, RouteEntry{
					Destination: dest,
					Target:      target,
					Status:      string(r.State),
					Origin:      string(r.Origin),
				})
			}

			for _, assoc := range rt.Associations {
				isMain := aws.ToBool(assoc.Main)
				if isMain {
					info.IsMain = true
				}
				subnetID := aws.ToString(assoc.SubnetId)
				if subnetID == "" && isMain {
					continue // skip main association with no subnet
				}
				info.Associations = append(info.Associations, RouteTableAssociation{
					SubnetID: subnetID,
					IsMain:   isMain,
				})
			}

			rts = append(rts, info)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return rts, nil
}

func (c *Client) ListNATGateways(ctx context.Context, vpcID string) ([]NATGatewayInfo, error) {
	var nats []NATGatewayInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeNatGateways(ctx, &awsec2.DescribeNatGatewaysInput{
			Filter: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
				{Name: aws.String("state"), Values: []string{"pending", "available", "deleting", "failed"}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeNatGateways: %w", err)
		}

		for _, ng := range out.NatGateways {
			info := NATGatewayInfo{
				GatewayID: aws.ToString(ng.NatGatewayId),
				Name:      nameFromTags(ng.Tags),
				State:     string(ng.State),
				Type:      string(ng.ConnectivityType),
				SubnetID:  aws.ToString(ng.SubnetId),
			}

			if len(ng.NatGatewayAddresses) > 0 {
				info.ElasticIP = aws.ToString(ng.NatGatewayAddresses[0].PublicIp)
				info.PrivateIP = aws.ToString(ng.NatGatewayAddresses[0].PrivateIp)
			}

			nats = append(nats, info)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return nats, nil
}

func (c *Client) ListSecurityGroupRules(ctx context.Context, groupID string) ([]SecurityGroupRule, error) {
	var rules []SecurityGroupRule
	var nextToken *string

	for {
		out, err := c.api.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
			Filters: []types.Filter{
				{Name: aws.String("group-id"), Values: []string{groupID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeSecurityGroupRules: %w", err)
		}

		for _, r := range out.SecurityGroupRules {
			direction := "inbound"
			if aws.ToBool(r.IsEgress) {
				direction = "outbound"
			}

			protocol := NormalizeProtocol(aws.ToString(r.IpProtocol))

			portRange := "All"
			fromPort := aws.ToInt32(r.FromPort)
			toPort := aws.ToInt32(r.ToPort)
			if fromPort != -1 {
				if fromPort == toPort {
					portRange = strconv.Itoa(int(fromPort))
				} else {
					portRange = strconv.Itoa(int(fromPort)) + "-" + strconv.Itoa(int(toPort))
				}
			}

			source := ""
			if cidr := aws.ToString(r.CidrIpv4); cidr != "" {
				source = cidr
			} else if cidr6 := aws.ToString(r.CidrIpv6); cidr6 != "" {
				source = cidr6
			} else if r.ReferencedGroupInfo != nil {
				source = aws.ToString(r.ReferencedGroupInfo.GroupId)
			} else if pl := aws.ToString(r.PrefixListId); pl != "" {
				source = pl
			}

			rules = append(rules, SecurityGroupRule{
				Direction:   direction,
				Protocol:    protocol,
				PortRange:   portRange,
				Source:      source,
				Description: aws.ToString(r.Description),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return rules, nil
}

func (c *Client) ListVPCEndpoints(ctx context.Context, vpcID string) ([]VPCEndpointInfo, error) {
	var endpoints []VPCEndpointInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeVpcEndpoints(ctx, &awsec2.DescribeVpcEndpointsInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeVpcEndpoints: %w", err)
		}

		for _, ep := range out.VpcEndpoints {
			var subnetIDs []string
			for _, sn := range ep.SubnetIds {
				subnetIDs = append(subnetIDs, sn)
			}
			var rtIDs []string
			for _, rt := range ep.RouteTableIds {
				rtIDs = append(rtIDs, rt)
			}
			endpoints = append(endpoints, VPCEndpointInfo{
				EndpointID:    aws.ToString(ep.VpcEndpointId),
				ServiceName:   aws.ToString(ep.ServiceName),
				Type:          string(ep.VpcEndpointType),
				State:         string(ep.State),
				SubnetIDs:     subnetIDs,
				RouteTableIDs: rtIDs,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return endpoints, nil
}

func (c *Client) ListVPCPeering(ctx context.Context, vpcID string) ([]VPCPeeringInfo, error) {
	var peerings []VPCPeeringInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeVpcPeeringConnections(ctx, &awsec2.DescribeVpcPeeringConnectionsInput{
			Filters: []types.Filter{
				{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeVpcPeeringConnections (requester): %w", err)
		}

		for _, pcx := range out.VpcPeeringConnections {
			peerings = append(peerings, vpcPeeringFromSDK(pcx))
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	// Also fetch where this VPC is the accepter
	nextToken = nil
	for {
		out, err := c.api.DescribeVpcPeeringConnections(ctx, &awsec2.DescribeVpcPeeringConnectionsInput{
			Filters: []types.Filter{
				{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeVpcPeeringConnections (accepter): %w", err)
		}

		for _, pcx := range out.VpcPeeringConnections {
			// Deduplicate: skip if already seen from requester query
			dup := false
			id := aws.ToString(pcx.VpcPeeringConnectionId)
			for _, existing := range peerings {
				if existing.PeeringID == id {
					dup = true
					break
				}
			}
			if !dup {
				peerings = append(peerings, vpcPeeringFromSDK(pcx))
			}
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return peerings, nil
}

func vpcPeeringFromSDK(pcx types.VpcPeeringConnection) VPCPeeringInfo {
	info := VPCPeeringInfo{
		PeeringID: aws.ToString(pcx.VpcPeeringConnectionId),
		Name:      nameFromTags(pcx.Tags),
	}
	if pcx.Status != nil {
		info.Status = string(pcx.Status.Code)
	}
	if pcx.RequesterVpcInfo != nil {
		info.RequesterVPC = aws.ToString(pcx.RequesterVpcInfo.VpcId)
		info.RequesterCIDR = aws.ToString(pcx.RequesterVpcInfo.CidrBlock)
	}
	if pcx.AccepterVpcInfo != nil {
		info.AccepterVPC = aws.ToString(pcx.AccepterVpcInfo.VpcId)
		info.AccepterCIDR = aws.ToString(pcx.AccepterVpcInfo.CidrBlock)
	}
	return info
}

func (c *Client) ListNetworkACLs(ctx context.Context, vpcID string) ([]NetworkACLInfo, error) {
	var nacls []NetworkACLInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeNetworkAcls(ctx, &awsec2.DescribeNetworkAclsInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeNetworkAcls: %w", err)
		}

		for _, acl := range out.NetworkAcls {
			inbound := 0
			outbound := 0
			for _, entry := range acl.Entries {
				if aws.ToBool(entry.Egress) {
					outbound++
				} else {
					inbound++
				}
			}
			nacls = append(nacls, NetworkACLInfo{
				NACLID:    aws.ToString(acl.NetworkAclId),
				Name:      nameFromTags(acl.Tags),
				IsDefault: aws.ToBool(acl.IsDefault),
				Inbound:   inbound,
				Outbound:  outbound,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return nacls, nil
}

func (c *Client) ListNetworkACLEntries(ctx context.Context, naclID string) ([]NetworkACLEntry, error) {
	out, err := c.api.DescribeNetworkAcls(ctx, &awsec2.DescribeNetworkAclsInput{
		NetworkAclIds: []string{naclID},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeNetworkAcls: %w", err)
	}
	if len(out.NetworkAcls) == 0 {
		return nil, nil
	}

	var entries []NetworkACLEntry
	for _, e := range out.NetworkAcls[0].Entries {
		direction := "inbound"
		if aws.ToBool(e.Egress) {
			direction = "outbound"
		}

		protocol := NormalizeProtocol(aws.ToString(e.Protocol))

		portRange := "All"
		if e.PortRange != nil {
			from := aws.ToInt32(e.PortRange.From)
			to := aws.ToInt32(e.PortRange.To)
			if from == to {
				portRange = strconv.Itoa(int(from))
			} else {
				portRange = strconv.Itoa(int(from)) + "-" + strconv.Itoa(int(to))
			}
		}

		cidr := aws.ToString(e.CidrBlock)
		if cidr == "" {
			cidr = aws.ToString(e.Ipv6CidrBlock)
		}

		action := string(e.RuleAction)

		entries = append(entries, NetworkACLEntry{
			RuleNumber: int(aws.ToInt32(e.RuleNumber)),
			Direction:  direction,
			Protocol:   protocol,
			PortRange:  portRange,
			CIDRBlock:  cidr,
			Action:     action,
		})
	}
	return entries, nil
}

func (c *Client) ListFlowLogs(ctx context.Context, vpcID string) ([]FlowLogInfo, error) {
	var flowLogs []FlowLogInfo
	var nextToken *string

	for {
		out, err := c.api.DescribeFlowLogs(ctx, &awsec2.DescribeFlowLogsInput{
			Filter: []types.Filter{
				{Name: aws.String("resource-id"), Values: []string{vpcID}},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeFlowLogs: %w", err)
		}

		for _, fl := range out.FlowLogs {
			flowLogs = append(flowLogs, FlowLogInfo{
				FlowLogID:      aws.ToString(fl.FlowLogId),
				Status:         aws.ToString(fl.FlowLogStatus),
				TrafficType:    string(fl.TrafficType),
				LogDestination: aws.ToString(fl.LogDestination),
				LogFormat:      aws.ToString(fl.LogFormat),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return flowLogs, nil
}

func (c *Client) GetVPCTags(ctx context.Context, vpcID string) (map[string]string, error) {
	out, err := c.api.DescribeVpcs(ctx, &awsec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeVpcs: %w", err)
	}
	if len(out.Vpcs) == 0 {
		return nil, nil
	}
	tags := make(map[string]string, len(out.Vpcs[0].Tags))
	for _, t := range out.Vpcs[0].Tags {
		tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return tags, nil
}

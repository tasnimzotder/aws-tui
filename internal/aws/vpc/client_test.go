package vpc

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockVPCAPI struct {
	describeVpcsFunc               func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error)
	describeSubnetsFunc            func(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error)
	describeSecurityGroupsFunc     func(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error)
	describeInternetGatewaysFunc   func(ctx context.Context, params *awsec2.DescribeInternetGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInternetGatewaysOutput, error)
	describeRouteTablesFunc        func(ctx context.Context, params *awsec2.DescribeRouteTablesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeRouteTablesOutput, error)
	describeNatGatewaysFunc        func(ctx context.Context, params *awsec2.DescribeNatGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNatGatewaysOutput, error)
	describeSecurityGroupRulesFunc func(ctx context.Context, params *awsec2.DescribeSecurityGroupRulesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupRulesOutput, error)
}

func (m *mockVPCAPI) DescribeVpcs(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error) {
	return m.describeVpcsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeSubnets(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error) {
	return m.describeSubnetsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeSecurityGroups(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error) {
	return m.describeSecurityGroupsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeInternetGateways(ctx context.Context, params *awsec2.DescribeInternetGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInternetGatewaysOutput, error) {
	return m.describeInternetGatewaysFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeRouteTables(ctx context.Context, params *awsec2.DescribeRouteTablesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeRouteTablesOutput, error) {
	return m.describeRouteTablesFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeNatGateways(ctx context.Context, params *awsec2.DescribeNatGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNatGatewaysOutput, error) {
	return m.describeNatGatewaysFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeSecurityGroupRules(ctx context.Context, params *awsec2.DescribeSecurityGroupRulesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupRulesOutput, error) {
	return m.describeSecurityGroupRulesFunc(ctx, params, optFns...)
}

func TestListVPCs(t *testing.T) {
	mock := &mockVPCAPI{
		describeVpcsFunc: func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error) {
			return &awsec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{
					{
						VpcId:     awssdk.String("vpc-abc123"),
						CidrBlock: awssdk.String("10.0.0.0/16"),
						IsDefault: awssdk.Bool(false),
						State:     types.VpcStateAvailable,
						Tags: []types.Tag{
							{Key: awssdk.String("Name"), Value: awssdk.String("prod-vpc")},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	vpcs, err := client.ListVPCs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vpcs) != 1 {
		t.Fatalf("expected 1 VPC, got %d", len(vpcs))
	}
	if vpcs[0].Name != "prod-vpc" {
		t.Errorf("Name = %s, want prod-vpc", vpcs[0].Name)
	}
	if vpcs[0].CIDR != "10.0.0.0/16" {
		t.Errorf("CIDR = %s, want 10.0.0.0/16", vpcs[0].CIDR)
	}
}

func TestListVPCs_Pagination(t *testing.T) {
	callCount := 0
	mock := &mockVPCAPI{
		describeVpcsFunc: func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error) {
			callCount++
			if callCount == 1 {
				return &awsec2.DescribeVpcsOutput{
					Vpcs: []types.Vpc{{
						VpcId:     awssdk.String("vpc-1"),
						CidrBlock: awssdk.String("10.0.0.0/16"),
						State:     types.VpcStateAvailable,
					}},
					NextToken: awssdk.String("page2"),
				}, nil
			}
			return &awsec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{{
					VpcId:     awssdk.String("vpc-2"),
					CidrBlock: awssdk.String("10.1.0.0/16"),
					State:     types.VpcStateAvailable,
				}},
			}, nil
		},
	}

	client := NewClient(mock)
	vpcs, err := client.ListVPCs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
	if len(vpcs) != 2 {
		t.Fatalf("expected 2 VPCs, got %d", len(vpcs))
	}
	if vpcs[0].VPCID != "vpc-1" || vpcs[1].VPCID != "vpc-2" {
		t.Errorf("unexpected VPC IDs: %s, %s", vpcs[0].VPCID, vpcs[1].VPCID)
	}
}

func TestListRouteTables(t *testing.T) {
	mock := &mockVPCAPI{
		describeRouteTablesFunc: func(ctx context.Context, params *awsec2.DescribeRouteTablesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeRouteTablesOutput, error) {
			return &awsec2.DescribeRouteTablesOutput{
				RouteTables: []types.RouteTable{
					{
						RouteTableId: awssdk.String("rtb-main"),
						Tags: []types.Tag{
							{Key: awssdk.String("Name"), Value: awssdk.String("main-rt")},
						},
						Routes: []types.Route{
							{
								DestinationCidrBlock: awssdk.String("10.0.0.0/16"),
								GatewayId:            awssdk.String("local"),
								State:                types.RouteStateActive,
								Origin:               types.RouteOriginCreateRouteTable,
							},
							{
								DestinationCidrBlock: awssdk.String("0.0.0.0/0"),
								GatewayId:            awssdk.String("igw-abc123"),
								State:                types.RouteStateActive,
								Origin:               types.RouteOriginCreateRoute,
							},
							{
								DestinationCidrBlock: awssdk.String("172.16.0.0/16"),
								NatGatewayId:         awssdk.String("nat-xyz789"),
								State:                types.RouteStateBlackhole,
								Origin:               types.RouteOriginCreateRoute,
							},
							{
								DestinationPrefixListId:  awssdk.String("pl-abc"),
								VpcPeeringConnectionId:   awssdk.String("pcx-111"),
								State:                    types.RouteStateActive,
								Origin:                   types.RouteOriginCreateRoute,
							},
							{
								DestinationCidrBlock: awssdk.String("192.168.0.0/16"),
								TransitGatewayId:     awssdk.String("tgw-222"),
								State:                types.RouteStateActive,
								Origin:               types.RouteOriginEnableVgwRoutePropagation,
							},
						},
						Associations: []types.RouteTableAssociation{
							{
								Main:     awssdk.Bool(true),
								SubnetId: nil,
							},
							{
								Main:     awssdk.Bool(false),
								SubnetId: awssdk.String("subnet-aaa"),
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	rts, err := client.ListRouteTables(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rts) != 1 {
		t.Fatalf("expected 1 route table, got %d", len(rts))
	}

	rt := rts[0]
	if rt.RouteTableID != "rtb-main" {
		t.Errorf("RouteTableID = %s, want rtb-main", rt.RouteTableID)
	}
	if rt.Name != "main-rt" {
		t.Errorf("Name = %s, want main-rt", rt.Name)
	}
	if !rt.IsMain {
		t.Error("expected IsMain to be true")
	}

	// Verify routes
	if len(rt.Routes) != 5 {
		t.Fatalf("expected 5 routes, got %d", len(rt.Routes))
	}

	// Route 0: local route
	if rt.Routes[0].Destination != "10.0.0.0/16" {
		t.Errorf("route[0].Destination = %s, want 10.0.0.0/16", rt.Routes[0].Destination)
	}
	if rt.Routes[0].Target != "local" {
		t.Errorf("route[0].Target = %s, want local", rt.Routes[0].Target)
	}
	if rt.Routes[0].Status != "active" {
		t.Errorf("route[0].Status = %s, want active", rt.Routes[0].Status)
	}
	if rt.Routes[0].Origin != "CreateRouteTable" {
		t.Errorf("route[0].Origin = %s, want CreateRouteTable", rt.Routes[0].Origin)
	}

	// Route 1: IGW
	if rt.Routes[1].Target != "igw-abc123" {
		t.Errorf("route[1].Target = %s, want igw-abc123", rt.Routes[1].Target)
	}

	// Route 2: NAT gateway (blackhole)
	if rt.Routes[2].Target != "nat-xyz789" {
		t.Errorf("route[2].Target = %s, want nat-xyz789", rt.Routes[2].Target)
	}
	if rt.Routes[2].Status != "blackhole" {
		t.Errorf("route[2].Status = %s, want blackhole", rt.Routes[2].Status)
	}

	// Route 3: prefix list destination, VPC peering target
	if rt.Routes[3].Destination != "pl-abc" {
		t.Errorf("route[3].Destination = %s, want pl-abc", rt.Routes[3].Destination)
	}
	if rt.Routes[3].Target != "pcx-111" {
		t.Errorf("route[3].Target = %s, want pcx-111", rt.Routes[3].Target)
	}

	// Route 4: transit gateway
	if rt.Routes[4].Target != "tgw-222" {
		t.Errorf("route[4].Target = %s, want tgw-222", rt.Routes[4].Target)
	}
	if rt.Routes[4].Origin != "EnableVgwRoutePropagation" {
		t.Errorf("route[4].Origin = %s, want EnableVgwRoutePropagation", rt.Routes[4].Origin)
	}

	// Verify associations (main with no subnet is skipped)
	if len(rt.Associations) != 1 {
		t.Fatalf("expected 1 association (main skipped), got %d", len(rt.Associations))
	}
	if rt.Associations[0].SubnetID != "subnet-aaa" {
		t.Errorf("assoc[0].SubnetID = %s, want subnet-aaa", rt.Associations[0].SubnetID)
	}
	if rt.Associations[0].IsMain {
		t.Error("expected assoc[0].IsMain to be false")
	}
}

func TestListNATGateways(t *testing.T) {
	mock := &mockVPCAPI{
		describeNatGatewaysFunc: func(ctx context.Context, params *awsec2.DescribeNatGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNatGatewaysOutput, error) {
			return &awsec2.DescribeNatGatewaysOutput{
				NatGateways: []types.NatGateway{
					{
						NatGatewayId:     awssdk.String("nat-pub123"),
						ConnectivityType: types.ConnectivityTypePublic,
						State:            types.NatGatewayStateAvailable,
						SubnetId:         awssdk.String("subnet-pub"),
						Tags: []types.Tag{
							{Key: awssdk.String("Name"), Value: awssdk.String("public-nat")},
						},
						NatGatewayAddresses: []types.NatGatewayAddress{
							{
								PublicIp:  awssdk.String("54.100.200.1"),
								PrivateIp: awssdk.String("10.0.1.50"),
							},
						},
					},
					{
						NatGatewayId:     awssdk.String("nat-priv456"),
						ConnectivityType: types.ConnectivityTypePrivate,
						State:            types.NatGatewayStatePending,
						SubnetId:         awssdk.String("subnet-priv"),
						Tags:             []types.Tag{},
						NatGatewayAddresses: []types.NatGatewayAddress{
							{
								PrivateIp: awssdk.String("10.0.2.100"),
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	nats, err := client.ListNATGateways(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nats) != 2 {
		t.Fatalf("expected 2 NAT gateways, got %d", len(nats))
	}

	// Public NAT gateway
	pub := nats[0]
	if pub.GatewayID != "nat-pub123" {
		t.Errorf("GatewayID = %s, want nat-pub123", pub.GatewayID)
	}
	if pub.Name != "public-nat" {
		t.Errorf("Name = %s, want public-nat", pub.Name)
	}
	if pub.State != "available" {
		t.Errorf("State = %s, want available", pub.State)
	}
	if pub.Type != "public" {
		t.Errorf("Type = %s, want public", pub.Type)
	}
	if pub.SubnetID != "subnet-pub" {
		t.Errorf("SubnetID = %s, want subnet-pub", pub.SubnetID)
	}
	if pub.ElasticIP != "54.100.200.1" {
		t.Errorf("ElasticIP = %s, want 54.100.200.1", pub.ElasticIP)
	}
	if pub.PrivateIP != "10.0.1.50" {
		t.Errorf("PrivateIP = %s, want 10.0.1.50", pub.PrivateIP)
	}

	// Private NAT gateway
	priv := nats[1]
	if priv.GatewayID != "nat-priv456" {
		t.Errorf("GatewayID = %s, want nat-priv456", priv.GatewayID)
	}
	if priv.Type != "private" {
		t.Errorf("Type = %s, want private", priv.Type)
	}
	if priv.State != "pending" {
		t.Errorf("State = %s, want pending", priv.State)
	}
	if priv.ElasticIP != "" {
		t.Errorf("ElasticIP = %s, want empty for private NAT", priv.ElasticIP)
	}
	if priv.PrivateIP != "10.0.2.100" {
		t.Errorf("PrivateIP = %s, want 10.0.2.100", priv.PrivateIP)
	}
}

func TestListSecurityGroupRules(t *testing.T) {
	mock := &mockVPCAPI{
		describeSecurityGroupRulesFunc: func(ctx context.Context, params *awsec2.DescribeSecurityGroupRulesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupRulesOutput, error) {
			return &awsec2.DescribeSecurityGroupRulesOutput{
				SecurityGroupRules: []types.SecurityGroupRule{
					{
						IsEgress:    awssdk.Bool(false),
						IpProtocol:  awssdk.String("6"),
						FromPort:    awssdk.Int32(443),
						ToPort:      awssdk.Int32(443),
						CidrIpv4:    awssdk.String("0.0.0.0/0"),
						Description: awssdk.String("HTTPS from anywhere"),
					},
					{
						IsEgress:   awssdk.Bool(false),
						IpProtocol: awssdk.String("6"),
						FromPort:   awssdk.Int32(8080),
						ToPort:     awssdk.Int32(8090),
						CidrIpv6:   awssdk.String("::/0"),
					},
					{
						IsEgress:   awssdk.Bool(true),
						IpProtocol: awssdk.String("-1"),
						FromPort:   awssdk.Int32(-1),
						ToPort:     awssdk.Int32(-1),
						CidrIpv4:   awssdk.String("0.0.0.0/0"),
					},
					{
						IsEgress:   awssdk.Bool(false),
						IpProtocol: awssdk.String("17"),
						FromPort:   awssdk.Int32(53),
						ToPort:     awssdk.Int32(53),
						ReferencedGroupInfo: &types.ReferencedSecurityGroup{
							GroupId: awssdk.String("sg-dns"),
						},
					},
					{
						IsEgress:    awssdk.Bool(false),
						IpProtocol:  awssdk.String("1"),
						FromPort:    awssdk.Int32(-1),
						ToPort:      awssdk.Int32(-1),
						PrefixListId: awssdk.String("pl-vpce"),
						Description:  awssdk.String("ICMP from prefix list"),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	rules, err := client.ListSecurityGroupRules(context.Background(), "sg-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 5 {
		t.Fatalf("expected 5 rules, got %d", len(rules))
	}

	// Rule 0: inbound TCP 443 from CIDR
	r0 := rules[0]
	if r0.Direction != "inbound" {
		t.Errorf("rule[0].Direction = %s, want inbound", r0.Direction)
	}
	if r0.Protocol != "TCP" {
		t.Errorf("rule[0].Protocol = %s, want TCP", r0.Protocol)
	}
	if r0.PortRange != "443" {
		t.Errorf("rule[0].PortRange = %s, want 443", r0.PortRange)
	}
	if r0.Source != "0.0.0.0/0" {
		t.Errorf("rule[0].Source = %s, want 0.0.0.0/0", r0.Source)
	}
	if r0.Description != "HTTPS from anywhere" {
		t.Errorf("rule[0].Description = %s, want HTTPS from anywhere", r0.Description)
	}

	// Rule 1: inbound TCP port range from IPv6 CIDR
	r1 := rules[1]
	if r1.PortRange != "8080-8090" {
		t.Errorf("rule[1].PortRange = %s, want 8080-8090", r1.PortRange)
	}
	if r1.Source != "::/0" {
		t.Errorf("rule[1].Source = %s, want ::/0", r1.Source)
	}

	// Rule 2: outbound All traffic
	r2 := rules[2]
	if r2.Direction != "outbound" {
		t.Errorf("rule[2].Direction = %s, want outbound", r2.Direction)
	}
	if r2.Protocol != "All" {
		t.Errorf("rule[2].Protocol = %s, want All", r2.Protocol)
	}
	if r2.PortRange != "All" {
		t.Errorf("rule[2].PortRange = %s, want All", r2.PortRange)
	}

	// Rule 3: inbound UDP 53 from referenced security group
	r3 := rules[3]
	if r3.Protocol != "UDP" {
		t.Errorf("rule[3].Protocol = %s, want UDP", r3.Protocol)
	}
	if r3.PortRange != "53" {
		t.Errorf("rule[3].PortRange = %s, want 53", r3.PortRange)
	}
	if r3.Source != "sg-dns" {
		t.Errorf("rule[3].Source = %s, want sg-dns", r3.Source)
	}

	// Rule 4: inbound ICMP from prefix list
	r4 := rules[4]
	if r4.Protocol != "ICMP" {
		t.Errorf("rule[4].Protocol = %s, want ICMP", r4.Protocol)
	}
	if r4.PortRange != "All" {
		t.Errorf("rule[4].PortRange = %s, want All", r4.PortRange)
	}
	if r4.Source != "pl-vpce" {
		t.Errorf("rule[4].Source = %s, want pl-vpce", r4.Source)
	}
	if r4.Description != "ICMP from prefix list" {
		t.Errorf("rule[4].Description = %s, want ICMP from prefix list", r4.Description)
	}
}

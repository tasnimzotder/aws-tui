package vpc

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockVPCAPI struct {
	describeVpcsFunc                     func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error)
	describeSubnetsFunc                  func(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error)
	describeSecurityGroupsFunc           func(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error)
	describeInternetGatewaysFunc         func(ctx context.Context, params *awsec2.DescribeInternetGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInternetGatewaysOutput, error)
	describeRouteTablesFunc              func(ctx context.Context, params *awsec2.DescribeRouteTablesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeRouteTablesOutput, error)
	describeNatGatewaysFunc              func(ctx context.Context, params *awsec2.DescribeNatGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNatGatewaysOutput, error)
	describeSecurityGroupRulesFunc       func(ctx context.Context, params *awsec2.DescribeSecurityGroupRulesInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupRulesOutput, error)
	describeVpcEndpointsFunc             func(ctx context.Context, params *awsec2.DescribeVpcEndpointsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcEndpointsOutput, error)
	describeVpcPeeringConnectionsFunc    func(ctx context.Context, params *awsec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcPeeringConnectionsOutput, error)
	describeNetworkAclsFunc              func(ctx context.Context, params *awsec2.DescribeNetworkAclsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNetworkAclsOutput, error)
	describeFlowLogsFunc                 func(ctx context.Context, params *awsec2.DescribeFlowLogsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeFlowLogsOutput, error)
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
func (m *mockVPCAPI) DescribeVpcEndpoints(ctx context.Context, params *awsec2.DescribeVpcEndpointsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcEndpointsOutput, error) {
	return m.describeVpcEndpointsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeVpcPeeringConnections(ctx context.Context, params *awsec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcPeeringConnectionsOutput, error) {
	return m.describeVpcPeeringConnectionsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeNetworkAcls(ctx context.Context, params *awsec2.DescribeNetworkAclsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNetworkAclsOutput, error) {
	return m.describeNetworkAclsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeFlowLogs(ctx context.Context, params *awsec2.DescribeFlowLogsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeFlowLogsOutput, error) {
	return m.describeFlowLogsFunc(ctx, params, optFns...)
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

func TestListVPCEndpoints(t *testing.T) {
	mock := &mockVPCAPI{
		describeVpcEndpointsFunc: func(ctx context.Context, params *awsec2.DescribeVpcEndpointsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcEndpointsOutput, error) {
			return &awsec2.DescribeVpcEndpointsOutput{
				VpcEndpoints: []types.VpcEndpoint{
					{
						VpcEndpointId:   awssdk.String("vpce-abc123"),
						ServiceName:     awssdk.String("com.amazonaws.us-east-1.s3"),
						VpcEndpointType: types.VpcEndpointTypeGateway,
						State:           types.StateAvailable,
						RouteTableIds:   []string{"rtb-111"},
					},
					{
						VpcEndpointId:   awssdk.String("vpce-def456"),
						ServiceName:     awssdk.String("com.amazonaws.us-east-1.ec2"),
						VpcEndpointType: types.VpcEndpointTypeInterface,
						State:           types.StateAvailable,
						SubnetIds:       []string{"subnet-aaa", "subnet-bbb"},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	endpoints, err := client.ListVPCEndpoints(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints[0].EndpointID != "vpce-abc123" {
		t.Errorf("EndpointID = %s, want vpce-abc123", endpoints[0].EndpointID)
	}
	if endpoints[0].Type != "Gateway" {
		t.Errorf("Type = %s, want Gateway", endpoints[0].Type)
	}
	if len(endpoints[0].RouteTableIDs) != 1 {
		t.Errorf("expected 1 route table ID, got %d", len(endpoints[0].RouteTableIDs))
	}
	if len(endpoints[1].SubnetIDs) != 2 {
		t.Errorf("expected 2 subnet IDs, got %d", len(endpoints[1].SubnetIDs))
	}
}

func TestListVPCPeering(t *testing.T) {
	mock := &mockVPCAPI{
		describeVpcPeeringConnectionsFunc: func(ctx context.Context, params *awsec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcPeeringConnectionsOutput, error) {
			// Return data only for requester query (first call)
			for _, f := range params.Filters {
				if awssdk.ToString(f.Name) == "requester-vpc-info.vpc-id" {
					return &awsec2.DescribeVpcPeeringConnectionsOutput{
						VpcPeeringConnections: []types.VpcPeeringConnection{
							{
								VpcPeeringConnectionId: awssdk.String("pcx-111"),
								Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
								Tags:                   []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("my-peering")}},
								RequesterVpcInfo:        &types.VpcPeeringConnectionVpcInfo{VpcId: awssdk.String("vpc-123"), CidrBlock: awssdk.String("10.0.0.0/16")},
								AccepterVpcInfo:         &types.VpcPeeringConnectionVpcInfo{VpcId: awssdk.String("vpc-456"), CidrBlock: awssdk.String("10.1.0.0/16")},
							},
						},
					}, nil
				}
			}
			return &awsec2.DescribeVpcPeeringConnectionsOutput{}, nil
		},
	}

	client := NewClient(mock)
	peerings, err := client.ListVPCPeering(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peerings) != 1 {
		t.Fatalf("expected 1 peering, got %d", len(peerings))
	}
	p := peerings[0]
	if p.PeeringID != "pcx-111" {
		t.Errorf("PeeringID = %s, want pcx-111", p.PeeringID)
	}
	if p.Name != "my-peering" {
		t.Errorf("Name = %s, want my-peering", p.Name)
	}
	if p.Status != "active" {
		t.Errorf("Status = %s, want active", p.Status)
	}
	if p.RequesterCIDR != "10.0.0.0/16" {
		t.Errorf("RequesterCIDR = %s, want 10.0.0.0/16", p.RequesterCIDR)
	}
	if p.AccepterVPC != "vpc-456" {
		t.Errorf("AccepterVPC = %s, want vpc-456", p.AccepterVPC)
	}
}

func TestListNetworkACLs(t *testing.T) {
	mock := &mockVPCAPI{
		describeNetworkAclsFunc: func(ctx context.Context, params *awsec2.DescribeNetworkAclsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNetworkAclsOutput, error) {
			return &awsec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{
					{
						NetworkAclId: awssdk.String("acl-123"),
						IsDefault:    awssdk.Bool(true),
						Tags:         []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("default-acl")}},
						Entries: []types.NetworkAclEntry{
							{Egress: awssdk.Bool(false), RuleNumber: awssdk.Int32(100)},
							{Egress: awssdk.Bool(false), RuleNumber: awssdk.Int32(200)},
							{Egress: awssdk.Bool(true), RuleNumber: awssdk.Int32(100)},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	nacls, err := client.ListNetworkACLs(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nacls) != 1 {
		t.Fatalf("expected 1 NACL, got %d", len(nacls))
	}
	if nacls[0].NACLID != "acl-123" {
		t.Errorf("NACLID = %s, want acl-123", nacls[0].NACLID)
	}
	if !nacls[0].IsDefault {
		t.Error("expected IsDefault to be true")
	}
	if nacls[0].Inbound != 2 {
		t.Errorf("Inbound = %d, want 2", nacls[0].Inbound)
	}
	if nacls[0].Outbound != 1 {
		t.Errorf("Outbound = %d, want 1", nacls[0].Outbound)
	}
}

func TestListNetworkACLEntries(t *testing.T) {
	mock := &mockVPCAPI{
		describeNetworkAclsFunc: func(ctx context.Context, params *awsec2.DescribeNetworkAclsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeNetworkAclsOutput, error) {
			return &awsec2.DescribeNetworkAclsOutput{
				NetworkAcls: []types.NetworkAcl{
					{
						NetworkAclId: awssdk.String("acl-123"),
						Entries: []types.NetworkAclEntry{
							{
								Egress:     awssdk.Bool(false),
								RuleNumber: awssdk.Int32(100),
								Protocol:   awssdk.String("6"),
								PortRange:  &types.PortRange{From: awssdk.Int32(443), To: awssdk.Int32(443)},
								CidrBlock:  awssdk.String("0.0.0.0/0"),
								RuleAction: types.RuleActionAllow,
							},
							{
								Egress:     awssdk.Bool(true),
								RuleNumber: awssdk.Int32(100),
								Protocol:   awssdk.String("-1"),
								CidrBlock:  awssdk.String("0.0.0.0/0"),
								RuleAction: types.RuleActionAllow,
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	entries, err := client.ListNetworkACLEntries(context.Background(), "acl-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Direction != "inbound" {
		t.Errorf("entries[0].Direction = %s, want inbound", entries[0].Direction)
	}
	if entries[0].Protocol != "TCP" {
		t.Errorf("entries[0].Protocol = %s, want TCP", entries[0].Protocol)
	}
	if entries[0].PortRange != "443" {
		t.Errorf("entries[0].PortRange = %s, want 443", entries[0].PortRange)
	}
	if entries[0].Action != "allow" {
		t.Errorf("entries[0].Action = %s, want allow", entries[0].Action)
	}
	if entries[1].Direction != "outbound" {
		t.Errorf("entries[1].Direction = %s, want outbound", entries[1].Direction)
	}
	if entries[1].Protocol != "All" {
		t.Errorf("entries[1].Protocol = %s, want All", entries[1].Protocol)
	}
}

func TestListFlowLogs(t *testing.T) {
	mock := &mockVPCAPI{
		describeFlowLogsFunc: func(ctx context.Context, params *awsec2.DescribeFlowLogsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeFlowLogsOutput, error) {
			return &awsec2.DescribeFlowLogsOutput{
				FlowLogs: []types.FlowLog{
					{
						FlowLogId:      awssdk.String("fl-abc123"),
						FlowLogStatus:  awssdk.String("ACTIVE"),
						TrafficType:    types.TrafficTypeAll,
						LogDestination: awssdk.String("arn:aws:s3:::my-bucket"),
						LogFormat:      awssdk.String("${version} ${account-id} ${interface-id}"),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	flowLogs, err := client.ListFlowLogs(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flowLogs) != 1 {
		t.Fatalf("expected 1 flow log, got %d", len(flowLogs))
	}
	if flowLogs[0].FlowLogID != "fl-abc123" {
		t.Errorf("FlowLogID = %s, want fl-abc123", flowLogs[0].FlowLogID)
	}
	if flowLogs[0].Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", flowLogs[0].Status)
	}
	if flowLogs[0].TrafficType != "ALL" {
		t.Errorf("TrafficType = %s, want ALL", flowLogs[0].TrafficType)
	}
}

func TestGetVPCTags_ReturnsMap(t *testing.T) {
	mock := &mockVPCAPI{
		describeVpcsFunc: func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error) {
			return &awsec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{
					{
						VpcId: awssdk.String("vpc-123"),
						Tags: []types.Tag{
							{Key: awssdk.String("Name"), Value: awssdk.String("prod-vpc")},
							{Key: awssdk.String("Environment"), Value: awssdk.String("production")},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	tags, err := client.GetVPCTags(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags["Name"] != "prod-vpc" {
		t.Errorf("tags[Name] = %s, want prod-vpc", tags["Name"])
	}
	if tags["Environment"] != "production" {
		t.Errorf("tags[Environment] = %s, want production", tags["Environment"])
	}
}

func TestListSubnets(t *testing.T) {
	mock := &mockVPCAPI{
		describeSubnetsFunc: func(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error) {
			return &awsec2.DescribeSubnetsOutput{
				Subnets: []types.Subnet{
					{
						SubnetId:                awssdk.String("subnet-abc123"),
						CidrBlock:               awssdk.String("10.0.1.0/24"),
						AvailabilityZone:        awssdk.String("us-east-1a"),
						AvailableIpAddressCount: awssdk.Int32(250),
						Tags: []types.Tag{
							{Key: awssdk.String("Name"), Value: awssdk.String("public-a")},
						},
					},
					{
						SubnetId:                awssdk.String("subnet-def456"),
						CidrBlock:               awssdk.String("10.0.2.0/24"),
						AvailabilityZone:        awssdk.String("us-east-1b"),
						AvailableIpAddressCount: awssdk.Int32(100),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	subnets, err := client.ListSubnets(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subnets) != 2 {
		t.Fatalf("expected 2 subnets, got %d", len(subnets))
	}
	if subnets[0].SubnetID != "subnet-abc123" {
		t.Errorf("SubnetID = %s, want subnet-abc123", subnets[0].SubnetID)
	}
	if subnets[0].Name != "public-a" {
		t.Errorf("Name = %s, want public-a", subnets[0].Name)
	}
	if subnets[0].CIDR != "10.0.1.0/24" {
		t.Errorf("CIDR = %s, want 10.0.1.0/24", subnets[0].CIDR)
	}
	if subnets[0].AZ != "us-east-1a" {
		t.Errorf("AZ = %s, want us-east-1a", subnets[0].AZ)
	}
	if subnets[0].AvailableIPs != 250 {
		t.Errorf("AvailableIPs = %d, want 250", subnets[0].AvailableIPs)
	}
	if subnets[1].Name != "" {
		t.Errorf("expected empty name for subnet without Name tag, got %s", subnets[1].Name)
	}
}

func TestListSecurityGroups(t *testing.T) {
	mock := &mockVPCAPI{
		describeSecurityGroupsFunc: func(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error) {
			return &awsec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{
						GroupId:     awssdk.String("sg-abc123"),
						GroupName:   awssdk.String("web-sg"),
						Description: awssdk.String("Web security group"),
						IpPermissions: []types.IpPermission{
							{IpProtocol: awssdk.String("tcp"), FromPort: awssdk.Int32(80), ToPort: awssdk.Int32(80)},
							{IpProtocol: awssdk.String("tcp"), FromPort: awssdk.Int32(443), ToPort: awssdk.Int32(443)},
						},
						IpPermissionsEgress: []types.IpPermission{
							{IpProtocol: awssdk.String("-1")},
						},
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	sgs, err := client.ListSecurityGroups(context.Background(), "vpc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sgs) != 1 {
		t.Fatalf("expected 1 security group, got %d", len(sgs))
	}
	sg := sgs[0]
	if sg.GroupID != "sg-abc123" {
		t.Errorf("GroupID = %s, want sg-abc123", sg.GroupID)
	}
	if sg.Name != "web-sg" {
		t.Errorf("Name = %s, want web-sg", sg.Name)
	}
	if sg.Description != "Web security group" {
		t.Errorf("Description = %s, want Web security group", sg.Description)
	}
	if sg.InboundRules != 2 {
		t.Errorf("InboundRules = %d, want 2", sg.InboundRules)
	}
	if sg.OutboundRules != 1 {
		t.Errorf("OutboundRules = %d, want 1", sg.OutboundRules)
	}
}

func TestGetVPCTags_Empty(t *testing.T) {
	mock := &mockVPCAPI{
		describeVpcsFunc: func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error) {
			return &awsec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{
					{VpcId: awssdk.String("vpc-empty")},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	tags, err := client.GetVPCTags(context.Background(), "vpc-empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected 0 tags, got %d", len(tags))
	}
}

func TestListVPCsPage(t *testing.T) {
	tt := []struct {
		name          string
		inputToken    *string
		mockVpcs      []types.Vpc
		mockNextToken *string
		wantLen       int
		wantNextToken *string
		wantVPCID     string
	}{
		{
			name:       "single page no next token",
			inputToken: nil,
			mockVpcs: []types.Vpc{
				{
					VpcId:     awssdk.String("vpc-single"),
					CidrBlock: awssdk.String("10.0.0.0/16"),
					IsDefault: awssdk.Bool(false),
					State:     types.VpcStateAvailable,
					Tags:      []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("prod")}},
				},
			},
			mockNextToken: nil,
			wantLen:       1,
			wantNextToken: nil,
			wantVPCID:     "vpc-single",
		},
		{
			name:       "first page with more results",
			inputToken: nil,
			mockVpcs: []types.Vpc{
				{
					VpcId:     awssdk.String("vpc-first"),
					CidrBlock: awssdk.String("10.1.0.0/16"),
					IsDefault: awssdk.Bool(false),
					State:     types.VpcStateAvailable,
				},
			},
			mockNextToken: awssdk.String("page2-token"),
			wantLen:       1,
			wantNextToken: awssdk.String("page2-token"),
			wantVPCID:     "vpc-first",
		},
		{
			name:          "empty results",
			inputToken:    nil,
			mockVpcs:      []types.Vpc{},
			mockNextToken: nil,
			wantLen:       0,
			wantNextToken: nil,
			wantVPCID:     "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockVPCAPI{
				describeVpcsFunc: func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error) {
					return &awsec2.DescribeVpcsOutput{
						Vpcs:      tc.mockVpcs,
						NextToken: tc.mockNextToken,
					}, nil
				},
			}
			client := NewClient(mock)
			vpcs, nextToken, err := client.ListVPCsPage(context.Background(), tc.inputToken)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(vpcs) != tc.wantLen {
				t.Errorf("len(vpcs) = %d, want %d", len(vpcs), tc.wantLen)
			}
			if tc.wantNextToken == nil && nextToken != nil {
				t.Errorf("nextToken = %s, want nil", *nextToken)
			}
			if tc.wantNextToken != nil {
				if nextToken == nil {
					t.Errorf("nextToken = nil, want %s", *tc.wantNextToken)
				} else if *nextToken != *tc.wantNextToken {
					t.Errorf("nextToken = %s, want %s", *nextToken, *tc.wantNextToken)
				}
			}
			if tc.wantVPCID != "" && len(vpcs) > 0 && vpcs[0].VPCID != tc.wantVPCID {
				t.Errorf("vpcs[0].VPCID = %s, want %s", vpcs[0].VPCID, tc.wantVPCID)
			}
		})
	}
}

func TestListSubnetsPage(t *testing.T) {
	tt := []struct {
		name          string
		vpcID         string
		inputToken    *string
		mockSubnets   []types.Subnet
		mockNextToken *string
		wantLen       int
		wantNextToken *string
		wantSubnetID  string
	}{
		{
			name:       "single page no next token",
			vpcID:      "vpc-abc",
			inputToken: nil,
			mockSubnets: []types.Subnet{
				{
					SubnetId:                awssdk.String("subnet-aaa"),
					CidrBlock:               awssdk.String("10.0.1.0/24"),
					AvailabilityZone:        awssdk.String("us-east-1a"),
					AvailableIpAddressCount: awssdk.Int32(251),
					Tags:                    []types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("pub-a")}},
				},
			},
			mockNextToken: nil,
			wantLen:       1,
			wantNextToken: nil,
			wantSubnetID:  "subnet-aaa",
		},
		{
			name:       "first page with more results",
			vpcID:      "vpc-abc",
			inputToken: nil,
			mockSubnets: []types.Subnet{
				{
					SubnetId:                awssdk.String("subnet-bbb"),
					CidrBlock:               awssdk.String("10.0.2.0/24"),
					AvailabilityZone:        awssdk.String("us-east-1b"),
					AvailableIpAddressCount: awssdk.Int32(100),
				},
			},
			mockNextToken: awssdk.String("subnet-page2"),
			wantLen:       1,
			wantNextToken: awssdk.String("subnet-page2"),
			wantSubnetID:  "subnet-bbb",
		},
		{
			name:          "empty results",
			vpcID:         "vpc-abc",
			inputToken:    nil,
			mockSubnets:   []types.Subnet{},
			mockNextToken: nil,
			wantLen:       0,
			wantNextToken: nil,
			wantSubnetID:  "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockVPCAPI{
				describeSubnetsFunc: func(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error) {
					return &awsec2.DescribeSubnetsOutput{
						Subnets:   tc.mockSubnets,
						NextToken: tc.mockNextToken,
					}, nil
				},
			}
			client := NewClient(mock)
			subnets, nextToken, err := client.ListSubnetsPage(context.Background(), tc.vpcID, tc.inputToken)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(subnets) != tc.wantLen {
				t.Errorf("len(subnets) = %d, want %d", len(subnets), tc.wantLen)
			}
			if tc.wantNextToken == nil && nextToken != nil {
				t.Errorf("nextToken = %s, want nil", *nextToken)
			}
			if tc.wantNextToken != nil {
				if nextToken == nil {
					t.Errorf("nextToken = nil, want %s", *tc.wantNextToken)
				} else if *nextToken != *tc.wantNextToken {
					t.Errorf("nextToken = %s, want %s", *nextToken, *tc.wantNextToken)
				}
			}
			if tc.wantSubnetID != "" && len(subnets) > 0 && subnets[0].SubnetID != tc.wantSubnetID {
				t.Errorf("subnets[0].SubnetID = %s, want %s", subnets[0].SubnetID, tc.wantSubnetID)
			}
		})
	}
}

func TestListSecurityGroupsPage(t *testing.T) {
	tt := []struct {
		name          string
		vpcID         string
		inputToken    *string
		mockSGs       []types.SecurityGroup
		mockNextToken *string
		wantLen       int
		wantNextToken *string
		wantGroupID   string
	}{
		{
			name:       "single page no next token",
			vpcID:      "vpc-abc",
			inputToken: nil,
			mockSGs: []types.SecurityGroup{
				{
					GroupId:     awssdk.String("sg-111"),
					GroupName:   awssdk.String("web-sg"),
					Description: awssdk.String("Web traffic"),
					IpPermissions: []types.IpPermission{
						{IpProtocol: awssdk.String("tcp"), FromPort: awssdk.Int32(80), ToPort: awssdk.Int32(80)},
					},
					IpPermissionsEgress: []types.IpPermission{
						{IpProtocol: awssdk.String("-1")},
					},
				},
			},
			mockNextToken: nil,
			wantLen:       1,
			wantNextToken: nil,
			wantGroupID:   "sg-111",
		},
		{
			name:       "first page with more results",
			vpcID:      "vpc-abc",
			inputToken: nil,
			mockSGs: []types.SecurityGroup{
				{
					GroupId:   awssdk.String("sg-222"),
					GroupName: awssdk.String("db-sg"),
				},
			},
			mockNextToken: awssdk.String("sg-page2"),
			wantLen:       1,
			wantNextToken: awssdk.String("sg-page2"),
			wantGroupID:   "sg-222",
		},
		{
			name:          "empty results",
			vpcID:         "vpc-abc",
			inputToken:    nil,
			mockSGs:       []types.SecurityGroup{},
			mockNextToken: nil,
			wantLen:       0,
			wantNextToken: nil,
			wantGroupID:   "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockVPCAPI{
				describeSecurityGroupsFunc: func(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error) {
					return &awsec2.DescribeSecurityGroupsOutput{
						SecurityGroups: tc.mockSGs,
						NextToken:      tc.mockNextToken,
					}, nil
				},
			}
			client := NewClient(mock)
			sgs, nextToken, err := client.ListSecurityGroupsPage(context.Background(), tc.vpcID, tc.inputToken)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sgs) != tc.wantLen {
				t.Errorf("len(sgs) = %d, want %d", len(sgs), tc.wantLen)
			}
			if tc.wantNextToken == nil && nextToken != nil {
				t.Errorf("nextToken = %s, want nil", *nextToken)
			}
			if tc.wantNextToken != nil {
				if nextToken == nil {
					t.Errorf("nextToken = nil, want %s", *tc.wantNextToken)
				} else if *nextToken != *tc.wantNextToken {
					t.Errorf("nextToken = %s, want %s", *nextToken, *tc.wantNextToken)
				}
			}
			if tc.wantGroupID != "" && len(sgs) > 0 && sgs[0].GroupID != tc.wantGroupID {
				t.Errorf("sgs[0].GroupID = %s, want %s", sgs[0].GroupID, tc.wantGroupID)
			}
		})
	}
}

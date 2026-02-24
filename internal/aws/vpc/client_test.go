package vpc

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockVPCAPI struct {
	describeVpcsFunc             func(ctx context.Context, params *awsec2.DescribeVpcsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeVpcsOutput, error)
	describeSubnetsFunc          func(ctx context.Context, params *awsec2.DescribeSubnetsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSubnetsOutput, error)
	describeSecurityGroupsFunc   func(ctx context.Context, params *awsec2.DescribeSecurityGroupsInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeSecurityGroupsOutput, error)
	describeInternetGatewaysFunc func(ctx context.Context, params *awsec2.DescribeInternetGatewaysInput, optFns ...func(*awsec2.Options)) (*awsec2.DescribeInternetGatewaysOutput, error)
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

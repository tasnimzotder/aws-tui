package eks

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awseks "github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
)

type mockEKSAPI struct {
	listClustersFunc          func(ctx context.Context, params *awseks.ListClustersInput, optFns ...func(*awseks.Options)) (*awseks.ListClustersOutput, error)
	describeClusterFunc       func(ctx context.Context, params *awseks.DescribeClusterInput, optFns ...func(*awseks.Options)) (*awseks.DescribeClusterOutput, error)
	listNodegroupsFunc        func(ctx context.Context, params *awseks.ListNodegroupsInput, optFns ...func(*awseks.Options)) (*awseks.ListNodegroupsOutput, error)
	describeNodegroupFunc     func(ctx context.Context, params *awseks.DescribeNodegroupInput, optFns ...func(*awseks.Options)) (*awseks.DescribeNodegroupOutput, error)
	listAddonsFunc            func(ctx context.Context, params *awseks.ListAddonsInput, optFns ...func(*awseks.Options)) (*awseks.ListAddonsOutput, error)
	describeAddonFunc         func(ctx context.Context, params *awseks.DescribeAddonInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAddonOutput, error)
	listFargateProfilesFunc   func(ctx context.Context, params *awseks.ListFargateProfilesInput, optFns ...func(*awseks.Options)) (*awseks.ListFargateProfilesOutput, error)
	describeFargateProfileFunc func(ctx context.Context, params *awseks.DescribeFargateProfileInput, optFns ...func(*awseks.Options)) (*awseks.DescribeFargateProfileOutput, error)
	listAccessEntriesFunc     func(ctx context.Context, params *awseks.ListAccessEntriesInput, optFns ...func(*awseks.Options)) (*awseks.ListAccessEntriesOutput, error)
	describeAccessEntryFunc   func(ctx context.Context, params *awseks.DescribeAccessEntryInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAccessEntryOutput, error)
}

func (m *mockEKSAPI) ListClusters(ctx context.Context, params *awseks.ListClustersInput, optFns ...func(*awseks.Options)) (*awseks.ListClustersOutput, error) {
	return m.listClustersFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) DescribeCluster(ctx context.Context, params *awseks.DescribeClusterInput, optFns ...func(*awseks.Options)) (*awseks.DescribeClusterOutput, error) {
	return m.describeClusterFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) ListNodegroups(ctx context.Context, params *awseks.ListNodegroupsInput, optFns ...func(*awseks.Options)) (*awseks.ListNodegroupsOutput, error) {
	return m.listNodegroupsFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) DescribeNodegroup(ctx context.Context, params *awseks.DescribeNodegroupInput, optFns ...func(*awseks.Options)) (*awseks.DescribeNodegroupOutput, error) {
	return m.describeNodegroupFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) ListAddons(ctx context.Context, params *awseks.ListAddonsInput, optFns ...func(*awseks.Options)) (*awseks.ListAddonsOutput, error) {
	return m.listAddonsFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) DescribeAddon(ctx context.Context, params *awseks.DescribeAddonInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAddonOutput, error) {
	return m.describeAddonFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) ListFargateProfiles(ctx context.Context, params *awseks.ListFargateProfilesInput, optFns ...func(*awseks.Options)) (*awseks.ListFargateProfilesOutput, error) {
	return m.listFargateProfilesFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) DescribeFargateProfile(ctx context.Context, params *awseks.DescribeFargateProfileInput, optFns ...func(*awseks.Options)) (*awseks.DescribeFargateProfileOutput, error) {
	return m.describeFargateProfileFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) ListAccessEntries(ctx context.Context, params *awseks.ListAccessEntriesInput, optFns ...func(*awseks.Options)) (*awseks.ListAccessEntriesOutput, error) {
	return m.listAccessEntriesFunc(ctx, params, optFns...)
}

func (m *mockEKSAPI) DescribeAccessEntry(ctx context.Context, params *awseks.DescribeAccessEntryInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAccessEntryOutput, error) {
	return m.describeAccessEntryFunc(ctx, params, optFns...)
}

func TestListClusters(t *testing.T) {
	created := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	mock := &mockEKSAPI{
		listClustersFunc: func(ctx context.Context, params *awseks.ListClustersInput, optFns ...func(*awseks.Options)) (*awseks.ListClustersOutput, error) {
			return &awseks.ListClustersOutput{
				Clusters: []string{"my-cluster"},
			}, nil
		},
		describeClusterFunc: func(ctx context.Context, params *awseks.DescribeClusterInput, optFns ...func(*awseks.Options)) (*awseks.DescribeClusterOutput, error) {
			if awssdk.ToString(params.Name) != "my-cluster" {
				t.Errorf("DescribeCluster name = %s, want my-cluster", awssdk.ToString(params.Name))
			}
			return &awseks.DescribeClusterOutput{
				Cluster: &ekstypes.Cluster{
					Name:            awssdk.String("my-cluster"),
					Arn:             awssdk.String("arn:aws:eks:us-east-1:123456789012:cluster/my-cluster"),
					Status:          ekstypes.ClusterStatusActive,
					Version:         awssdk.String("1.28"),
					PlatformVersion: awssdk.String("eks.5"),
					Endpoint:        awssdk.String("https://ABCDEF.gr7.us-east-1.eks.amazonaws.com"),
					RoleArn:         awssdk.String("arn:aws:iam::123456789012:role/eks-role"),
					CreatedAt:       &created,
					CertificateAuthority: &ekstypes.Certificate{
						Data: awssdk.String("LS0tLS1CRUdJTi..."),
					},
					ResourcesVpcConfig: &ekstypes.VpcConfigResponse{
						VpcId:                 awssdk.String("vpc-12345"),
						EndpointPublicAccess:  true,
						EndpointPrivateAccess: true,
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	c := clusters[0]
	if c.Name != "my-cluster" {
		t.Errorf("Name = %s, want my-cluster", c.Name)
	}
	if c.ARN != "arn:aws:eks:us-east-1:123456789012:cluster/my-cluster" {
		t.Errorf("ARN = %s, want arn:aws:eks:us-east-1:123456789012:cluster/my-cluster", c.ARN)
	}
	if c.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", c.Status)
	}
	if c.Version != "1.28" {
		t.Errorf("Version = %s, want 1.28", c.Version)
	}
	if c.PlatformVersion != "eks.5" {
		t.Errorf("PlatformVersion = %s, want eks.5", c.PlatformVersion)
	}
	if c.Endpoint != "https://ABCDEF.gr7.us-east-1.eks.amazonaws.com" {
		t.Errorf("Endpoint = %s, want https://ABCDEF.gr7.us-east-1.eks.amazonaws.com", c.Endpoint)
	}
	if !c.EndpointPublic {
		t.Error("EndpointPublic = false, want true")
	}
	if !c.EndpointPrivate {
		t.Error("EndpointPrivate = false, want true")
	}
	if c.VPCID != "vpc-12345" {
		t.Errorf("VPCID = %s, want vpc-12345", c.VPCID)
	}
	if c.RoleARN != "arn:aws:iam::123456789012:role/eks-role" {
		t.Errorf("RoleARN = %s, want arn:aws:iam::123456789012:role/eks-role", c.RoleARN)
	}
	if c.CertAuthority != "LS0tLS1CRUdJTi..." {
		t.Errorf("CertAuthority = %s, want LS0tLS1CRUdJTi...", c.CertAuthority)
	}
	if !c.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", c.CreatedAt, created)
	}
}

func TestListClusters_Pagination(t *testing.T) {
	callCount := 0

	mock := &mockEKSAPI{
		listClustersFunc: func(ctx context.Context, params *awseks.ListClustersInput, optFns ...func(*awseks.Options)) (*awseks.ListClustersOutput, error) {
			callCount++
			if callCount == 1 {
				if params.NextToken != nil {
					t.Error("first call should have nil NextToken")
				}
				return &awseks.ListClustersOutput{
					Clusters:  []string{"cluster-1"},
					NextToken: awssdk.String("token-page-2"),
				}, nil
			}
			if awssdk.ToString(params.NextToken) != "token-page-2" {
				t.Errorf("NextToken = %s, want token-page-2", awssdk.ToString(params.NextToken))
			}
			return &awseks.ListClustersOutput{
				Clusters: []string{"cluster-2"},
			}, nil
		},
		describeClusterFunc: func(ctx context.Context, params *awseks.DescribeClusterInput, optFns ...func(*awseks.Options)) (*awseks.DescribeClusterOutput, error) {
			return &awseks.DescribeClusterOutput{
				Cluster: &ekstypes.Cluster{
					Name:    params.Name,
					Arn:     awssdk.String("arn:aws:eks:us-east-1:123456789012:cluster/" + awssdk.ToString(params.Name)),
					Status:  ekstypes.ClusterStatusActive,
					Version: awssdk.String("1.28"),
				},
			}, nil
		},
	}

	client := NewClient(mock)
	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	if callCount != 2 {
		t.Errorf("ListClusters called %d times, want 2", callCount)
	}
	if clusters[0].Name != "cluster-1" {
		t.Errorf("clusters[0].Name = %s, want cluster-1", clusters[0].Name)
	}
	if clusters[1].Name != "cluster-2" {
		t.Errorf("clusters[1].Name = %s, want cluster-2", clusters[1].Name)
	}
}

func TestListNodeGroups(t *testing.T) {
	mock := &mockEKSAPI{
		listNodegroupsFunc: func(ctx context.Context, params *awseks.ListNodegroupsInput, optFns ...func(*awseks.Options)) (*awseks.ListNodegroupsOutput, error) {
			if awssdk.ToString(params.ClusterName) != "my-cluster" {
				t.Errorf("ClusterName = %s, want my-cluster", awssdk.ToString(params.ClusterName))
			}
			return &awseks.ListNodegroupsOutput{
				Nodegroups: []string{"ng-1"},
			}, nil
		},
		describeNodegroupFunc: func(ctx context.Context, params *awseks.DescribeNodegroupInput, optFns ...func(*awseks.Options)) (*awseks.DescribeNodegroupOutput, error) {
			if awssdk.ToString(params.NodegroupName) != "ng-1" {
				t.Errorf("NodegroupName = %s, want ng-1", awssdk.ToString(params.NodegroupName))
			}
			return &awseks.DescribeNodegroupOutput{
				Nodegroup: &ekstypes.Nodegroup{
					NodegroupName: awssdk.String("ng-1"),
					NodegroupArn:  awssdk.String("arn:aws:eks:us-east-1:123456789012:nodegroup/my-cluster/ng-1/abc"),
					Status:        ekstypes.NodegroupStatusActive,
					InstanceTypes: []string{"m5.large", "m5.xlarge"},
					AmiType:       ekstypes.AMITypesAl2X8664,
					ScalingConfig: &ekstypes.NodegroupScalingConfig{
						MinSize:     awssdk.Int32(2),
						MaxSize:     awssdk.Int32(10),
						DesiredSize: awssdk.Int32(3),
					},
					Labels: map[string]string{"env": "prod"},
					Taints: []ekstypes.Taint{
						{
							Key:    awssdk.String("dedicated"),
							Value:  awssdk.String("gpu"),
							Effect: ekstypes.TaintEffectNoSchedule,
						},
					},
					Subnets: []string{"subnet-aaa", "subnet-bbb"},
					LaunchTemplate: &ekstypes.LaunchTemplateSpecification{
						Name: awssdk.String("my-lt"),
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	nodeGroups, err := client.ListNodeGroups(context.Background(), "my-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodeGroups) != 1 {
		t.Fatalf("expected 1 node group, got %d", len(nodeGroups))
	}

	ng := nodeGroups[0]
	if ng.Name != "ng-1" {
		t.Errorf("Name = %s, want ng-1", ng.Name)
	}
	if ng.ARN != "arn:aws:eks:us-east-1:123456789012:nodegroup/my-cluster/ng-1/abc" {
		t.Errorf("ARN = %s, want arn:aws:eks:us-east-1:123456789012:nodegroup/my-cluster/ng-1/abc", ng.ARN)
	}
	if ng.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", ng.Status)
	}
	if len(ng.InstanceTypes) != 2 || ng.InstanceTypes[0] != "m5.large" || ng.InstanceTypes[1] != "m5.xlarge" {
		t.Errorf("InstanceTypes = %v, want [m5.large m5.xlarge]", ng.InstanceTypes)
	}
	if ng.AMIType != "AL2_x86_64" {
		t.Errorf("AMIType = %s, want AL2_x86_64", ng.AMIType)
	}
	if ng.MinSize != 2 {
		t.Errorf("MinSize = %d, want 2", ng.MinSize)
	}
	if ng.MaxSize != 10 {
		t.Errorf("MaxSize = %d, want 10", ng.MaxSize)
	}
	if ng.DesiredSize != 3 {
		t.Errorf("DesiredSize = %d, want 3", ng.DesiredSize)
	}
	if ng.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %s, want prod", ng.Labels["env"])
	}
	if len(ng.Taints) != 1 {
		t.Fatalf("expected 1 taint, got %d", len(ng.Taints))
	}
	if ng.Taints[0].Key != "dedicated" || ng.Taints[0].Value != "gpu" || ng.Taints[0].Effect != "NO_SCHEDULE" {
		t.Errorf("Taint = {%s, %s, %s}, want {dedicated, gpu, NO_SCHEDULE}", ng.Taints[0].Key, ng.Taints[0].Value, ng.Taints[0].Effect)
	}
	if len(ng.Subnets) != 2 || ng.Subnets[0] != "subnet-aaa" {
		t.Errorf("Subnets = %v, want [subnet-aaa subnet-bbb]", ng.Subnets)
	}
	if ng.LaunchTemplate != "my-lt" {
		t.Errorf("LaunchTemplate = %s, want my-lt", ng.LaunchTemplate)
	}
}

func TestListAddons(t *testing.T) {
	mock := &mockEKSAPI{
		listAddonsFunc: func(ctx context.Context, params *awseks.ListAddonsInput, optFns ...func(*awseks.Options)) (*awseks.ListAddonsOutput, error) {
			if awssdk.ToString(params.ClusterName) != "my-cluster" {
				t.Errorf("ClusterName = %s, want my-cluster", awssdk.ToString(params.ClusterName))
			}
			return &awseks.ListAddonsOutput{
				Addons: []string{"vpc-cni", "coredns"},
			}, nil
		},
		describeAddonFunc: func(ctx context.Context, params *awseks.DescribeAddonInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAddonOutput, error) {
			name := awssdk.ToString(params.AddonName)
			switch name {
			case "vpc-cni":
				return &awseks.DescribeAddonOutput{
					Addon: &ekstypes.Addon{
						AddonName:           awssdk.String("vpc-cni"),
						AddonArn:            awssdk.String("arn:aws:eks:us-east-1:123456789012:addon/my-cluster/vpc-cni/abc"),
						AddonVersion:        awssdk.String("v1.14.1-eksbuild.1"),
						Status:              ekstypes.AddonStatusActive,
						ServiceAccountRoleArn: awssdk.String("arn:aws:iam::123456789012:role/vpc-cni-role"),
						ConfigurationValues: awssdk.String(`{"env":{"WARM_ENI_TARGET":"1"}}`),
						Health: &ekstypes.AddonHealth{
							Issues: []ekstypes.AddonIssue{},
						},
					},
				}, nil
			case "coredns":
				return &awseks.DescribeAddonOutput{
					Addon: &ekstypes.Addon{
						AddonName:    awssdk.String("coredns"),
						AddonArn:     awssdk.String("arn:aws:eks:us-east-1:123456789012:addon/my-cluster/coredns/def"),
						AddonVersion: awssdk.String("v1.10.1-eksbuild.4"),
						Status:       ekstypes.AddonStatusActive,
					},
				}, nil
			default:
				t.Fatalf("unexpected addon name: %s", name)
				return nil, nil
			}
		},
	}

	client := NewClient(mock)
	addons, err := client.ListAddons(context.Background(), "my-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addons) != 2 {
		t.Fatalf("expected 2 addons, got %d", len(addons))
	}

	if addons[0].Name != "vpc-cni" {
		t.Errorf("addons[0].Name = %s, want vpc-cni", addons[0].Name)
	}
	if addons[0].Version != "v1.14.1-eksbuild.1" {
		t.Errorf("addons[0].Version = %s, want v1.14.1-eksbuild.1", addons[0].Version)
	}
	if addons[0].Status != "ACTIVE" {
		t.Errorf("addons[0].Status = %s, want ACTIVE", addons[0].Status)
	}
	if addons[0].ServiceAccountRole != "arn:aws:iam::123456789012:role/vpc-cni-role" {
		t.Errorf("addons[0].ServiceAccountRole = %s, want arn:aws:iam::123456789012:role/vpc-cni-role", addons[0].ServiceAccountRole)
	}
	if addons[0].ConfigurationValues != `{"env":{"WARM_ENI_TARGET":"1"}}` {
		t.Errorf("addons[0].ConfigurationValues = %s, want {\"env\":{\"WARM_ENI_TARGET\":\"1\"}}", addons[0].ConfigurationValues)
	}
	if addons[0].Health != "" {
		t.Errorf("addons[0].Health = %s, want empty (no issues)", addons[0].Health)
	}

	if addons[1].Name != "coredns" {
		t.Errorf("addons[1].Name = %s, want coredns", addons[1].Name)
	}
	if addons[1].Version != "v1.10.1-eksbuild.4" {
		t.Errorf("addons[1].Version = %s, want v1.10.1-eksbuild.4", addons[1].Version)
	}
}

func TestListFargateProfiles(t *testing.T) {
	mock := &mockEKSAPI{
		listFargateProfilesFunc: func(ctx context.Context, params *awseks.ListFargateProfilesInput, optFns ...func(*awseks.Options)) (*awseks.ListFargateProfilesOutput, error) {
			if awssdk.ToString(params.ClusterName) != "my-cluster" {
				t.Errorf("ClusterName = %s, want my-cluster", awssdk.ToString(params.ClusterName))
			}
			return &awseks.ListFargateProfilesOutput{
				FargateProfileNames: []string{"fp-default"},
			}, nil
		},
		describeFargateProfileFunc: func(ctx context.Context, params *awseks.DescribeFargateProfileInput, optFns ...func(*awseks.Options)) (*awseks.DescribeFargateProfileOutput, error) {
			if awssdk.ToString(params.FargateProfileName) != "fp-default" {
				t.Errorf("FargateProfileName = %s, want fp-default", awssdk.ToString(params.FargateProfileName))
			}
			return &awseks.DescribeFargateProfileOutput{
				FargateProfile: &ekstypes.FargateProfile{
					FargateProfileName: awssdk.String("fp-default"),
					FargateProfileArn:  awssdk.String("arn:aws:eks:us-east-1:123456789012:fargateprofile/my-cluster/fp-default/abc"),
					Status:             ekstypes.FargateProfileStatusActive,
					PodExecutionRoleArn: awssdk.String("arn:aws:iam::123456789012:role/fargate-pod-role"),
					Selectors: []ekstypes.FargateProfileSelector{
						{
							Namespace: awssdk.String("default"),
							Labels:    map[string]string{"app": "web"},
						},
						{
							Namespace: awssdk.String("kube-system"),
						},
					},
					Subnets: []string{"subnet-aaa", "subnet-bbb"},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	profiles, err := client.ListFargateProfiles(context.Background(), "my-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}

	fp := profiles[0]
	if fp.Name != "fp-default" {
		t.Errorf("Name = %s, want fp-default", fp.Name)
	}
	if fp.ARN != "arn:aws:eks:us-east-1:123456789012:fargateprofile/my-cluster/fp-default/abc" {
		t.Errorf("ARN = %s, want arn:aws:eks:us-east-1:123456789012:fargateprofile/my-cluster/fp-default/abc", fp.ARN)
	}
	if fp.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", fp.Status)
	}
	if fp.PodExecutionRole != "arn:aws:iam::123456789012:role/fargate-pod-role" {
		t.Errorf("PodExecutionRole = %s, want arn:aws:iam::123456789012:role/fargate-pod-role", fp.PodExecutionRole)
	}
	if len(fp.Selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d", len(fp.Selectors))
	}
	if fp.Selectors[0].Namespace != "default" {
		t.Errorf("Selectors[0].Namespace = %s, want default", fp.Selectors[0].Namespace)
	}
	if fp.Selectors[0].Labels["app"] != "web" {
		t.Errorf("Selectors[0].Labels[app] = %s, want web", fp.Selectors[0].Labels["app"])
	}
	if fp.Selectors[1].Namespace != "kube-system" {
		t.Errorf("Selectors[1].Namespace = %s, want kube-system", fp.Selectors[1].Namespace)
	}
	if len(fp.Subnets) != 2 || fp.Subnets[0] != "subnet-aaa" {
		t.Errorf("Subnets = %v, want [subnet-aaa subnet-bbb]", fp.Subnets)
	}
}

func TestListAccessEntries(t *testing.T) {
	created := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)

	mock := &mockEKSAPI{
		listAccessEntriesFunc: func(ctx context.Context, params *awseks.ListAccessEntriesInput, optFns ...func(*awseks.Options)) (*awseks.ListAccessEntriesOutput, error) {
			if awssdk.ToString(params.ClusterName) != "my-cluster" {
				t.Errorf("ClusterName = %s, want my-cluster", awssdk.ToString(params.ClusterName))
			}
			return &awseks.ListAccessEntriesOutput{
				AccessEntries: []string{"arn:aws:iam::123456789012:role/admin-role"},
			}, nil
		},
		describeAccessEntryFunc: func(ctx context.Context, params *awseks.DescribeAccessEntryInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAccessEntryOutput, error) {
			if awssdk.ToString(params.PrincipalArn) != "arn:aws:iam::123456789012:role/admin-role" {
				t.Errorf("PrincipalArn = %s, want arn:aws:iam::123456789012:role/admin-role", awssdk.ToString(params.PrincipalArn))
			}
			return &awseks.DescribeAccessEntryOutput{
				AccessEntry: &ekstypes.AccessEntry{
					PrincipalArn:     awssdk.String("arn:aws:iam::123456789012:role/admin-role"),
					Type:             awssdk.String("STANDARD"),
					Username:         awssdk.String("admin"),
					KubernetesGroups: []string{"system:masters"},
					CreatedAt:        &created,
				},
			}, nil
		},
	}

	client := NewClient(mock)
	entries, err := client.ListAccessEntries(context.Background(), "my-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	ae := entries[0]
	if ae.PrincipalARN != "arn:aws:iam::123456789012:role/admin-role" {
		t.Errorf("PrincipalARN = %s, want arn:aws:iam::123456789012:role/admin-role", ae.PrincipalARN)
	}
	if ae.Type != "STANDARD" {
		t.Errorf("Type = %s, want STANDARD", ae.Type)
	}
	if ae.Username != "admin" {
		t.Errorf("Username = %s, want admin", ae.Username)
	}
	if len(ae.Groups) != 1 || ae.Groups[0] != "system:masters" {
		t.Errorf("Groups = %v, want [system:masters]", ae.Groups)
	}
	if !ae.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", ae.CreatedAt, created)
	}
}

func TestDescribeCluster(t *testing.T) {
	created := time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC)

	mock := &mockEKSAPI{
		describeClusterFunc: func(ctx context.Context, params *awseks.DescribeClusterInput, optFns ...func(*awseks.Options)) (*awseks.DescribeClusterOutput, error) {
			if awssdk.ToString(params.Name) != "single-cluster" {
				t.Errorf("DescribeCluster name = %s, want single-cluster", awssdk.ToString(params.Name))
			}
			return &awseks.DescribeClusterOutput{
				Cluster: &ekstypes.Cluster{
					Name:            awssdk.String("single-cluster"),
					Arn:             awssdk.String("arn:aws:eks:us-west-2:123456789012:cluster/single-cluster"),
					Status:          ekstypes.ClusterStatusActive,
					Version:         awssdk.String("1.29"),
					PlatformVersion: awssdk.String("eks.8"),
					Endpoint:        awssdk.String("https://ENDPOINT.gr7.us-west-2.eks.amazonaws.com"),
					RoleArn:         awssdk.String("arn:aws:iam::123456789012:role/cluster-role"),
					CreatedAt:       &created,
					CertificateAuthority: &ekstypes.Certificate{
						Data: awssdk.String("Y2VydGlmaWNhdGVEYXRh"),
					},
					ResourcesVpcConfig: &ekstypes.VpcConfigResponse{
						VpcId:                 awssdk.String("vpc-98765"),
						EndpointPublicAccess:  false,
						EndpointPrivateAccess: true,
					},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	c, err := client.DescribeCluster(context.Background(), "single-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Name != "single-cluster" {
		t.Errorf("Name = %s, want single-cluster", c.Name)
	}
	if c.ARN != "arn:aws:eks:us-west-2:123456789012:cluster/single-cluster" {
		t.Errorf("ARN = %s", c.ARN)
	}
	if c.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", c.Status)
	}
	if c.Version != "1.29" {
		t.Errorf("Version = %s, want 1.29", c.Version)
	}
	if c.PlatformVersion != "eks.8" {
		t.Errorf("PlatformVersion = %s, want eks.8", c.PlatformVersion)
	}
	if c.Endpoint != "https://ENDPOINT.gr7.us-west-2.eks.amazonaws.com" {
		t.Errorf("Endpoint = %s", c.Endpoint)
	}
	if c.EndpointPublic {
		t.Error("EndpointPublic = true, want false")
	}
	if !c.EndpointPrivate {
		t.Error("EndpointPrivate = false, want true")
	}
	if c.VPCID != "vpc-98765" {
		t.Errorf("VPCID = %s, want vpc-98765", c.VPCID)
	}
	if c.RoleARN != "arn:aws:iam::123456789012:role/cluster-role" {
		t.Errorf("RoleARN = %s", c.RoleARN)
	}
	if c.CertAuthority != "Y2VydGlmaWNhdGVEYXRh" {
		t.Errorf("CertAuthority = %s, want Y2VydGlmaWNhdGVEYXRh", c.CertAuthority)
	}
	if !c.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", c.CreatedAt, created)
	}
}

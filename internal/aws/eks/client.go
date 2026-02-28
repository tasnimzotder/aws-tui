package eks

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awseks "github.com/aws/aws-sdk-go-v2/service/eks"
)

type EKSAPI interface {
	ListClusters(ctx context.Context, params *awseks.ListClustersInput, optFns ...func(*awseks.Options)) (*awseks.ListClustersOutput, error)
	DescribeCluster(ctx context.Context, params *awseks.DescribeClusterInput, optFns ...func(*awseks.Options)) (*awseks.DescribeClusterOutput, error)
	ListNodegroups(ctx context.Context, params *awseks.ListNodegroupsInput, optFns ...func(*awseks.Options)) (*awseks.ListNodegroupsOutput, error)
	DescribeNodegroup(ctx context.Context, params *awseks.DescribeNodegroupInput, optFns ...func(*awseks.Options)) (*awseks.DescribeNodegroupOutput, error)
	ListAddons(ctx context.Context, params *awseks.ListAddonsInput, optFns ...func(*awseks.Options)) (*awseks.ListAddonsOutput, error)
	DescribeAddon(ctx context.Context, params *awseks.DescribeAddonInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAddonOutput, error)
	ListFargateProfiles(ctx context.Context, params *awseks.ListFargateProfilesInput, optFns ...func(*awseks.Options)) (*awseks.ListFargateProfilesOutput, error)
	DescribeFargateProfile(ctx context.Context, params *awseks.DescribeFargateProfileInput, optFns ...func(*awseks.Options)) (*awseks.DescribeFargateProfileOutput, error)
	ListAccessEntries(ctx context.Context, params *awseks.ListAccessEntriesInput, optFns ...func(*awseks.Options)) (*awseks.ListAccessEntriesOutput, error)
	DescribeAccessEntry(ctx context.Context, params *awseks.DescribeAccessEntryInput, optFns ...func(*awseks.Options)) (*awseks.DescribeAccessEntryOutput, error)
}

type Client struct {
	api EKSAPI
}

func NewClient(api EKSAPI) *Client {
	return &Client{api: api}
}

func (c *Client) ListClusters(ctx context.Context) ([]EKSCluster, error) {
	var names []string
	var nextToken *string

	for {
		out, err := c.api.ListClusters(ctx, &awseks.ListClustersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListClusters: %w", err)
		}

		names = append(names, out.Clusters...)

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var clusters []EKSCluster
	for _, name := range names {
		cluster, err := c.DescribeCluster(ctx, name)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (c *Client) DescribeCluster(ctx context.Context, name string) (EKSCluster, error) {
	out, err := c.api.DescribeCluster(ctx, &awseks.DescribeClusterInput{
		Name: aws.String(name),
	})
	if err != nil {
		return EKSCluster{}, fmt.Errorf("DescribeCluster(%s): %w", name, err)
	}

	cl := out.Cluster

	var createdAt time.Time
	if cl.CreatedAt != nil {
		createdAt = *cl.CreatedAt
	}

	var certAuthority string
	if cl.CertificateAuthority != nil {
		certAuthority = aws.ToString(cl.CertificateAuthority.Data)
	}

	var vpcID string
	var endpointPublic, endpointPrivate bool
	if cl.ResourcesVpcConfig != nil {
		vpcID = aws.ToString(cl.ResourcesVpcConfig.VpcId)
		endpointPublic = cl.ResourcesVpcConfig.EndpointPublicAccess
		endpointPrivate = cl.ResourcesVpcConfig.EndpointPrivateAccess
	}

	return EKSCluster{
		Name:            aws.ToString(cl.Name),
		ARN:             aws.ToString(cl.Arn),
		Status:          string(cl.Status),
		Version:         aws.ToString(cl.Version),
		PlatformVersion: aws.ToString(cl.PlatformVersion),
		Endpoint:        aws.ToString(cl.Endpoint),
		EndpointPublic:  endpointPublic,
		EndpointPrivate: endpointPrivate,
		VPCID:           vpcID,
		RoleARN:         aws.ToString(cl.RoleArn),
		CertAuthority:   certAuthority,
		CreatedAt:       createdAt,
	}, nil
}

func (c *Client) ListNodeGroups(ctx context.Context, clusterName string) ([]EKSNodeGroup, error) {
	var names []string
	var nextToken *string

	for {
		out, err := c.api.ListNodegroups(ctx, &awseks.ListNodegroupsInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListNodegroups(%s): %w", clusterName, err)
		}

		names = append(names, out.Nodegroups...)

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var nodeGroups []EKSNodeGroup
	for _, name := range names {
		out, err := c.api.DescribeNodegroup(ctx, &awseks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeNodegroup(%s/%s): %w", clusterName, name, err)
		}

		ng := out.Nodegroup

		var minSize, maxSize, desiredSize int
		if ng.ScalingConfig != nil {
			if ng.ScalingConfig.MinSize != nil {
				minSize = int(*ng.ScalingConfig.MinSize)
			}
			if ng.ScalingConfig.MaxSize != nil {
				maxSize = int(*ng.ScalingConfig.MaxSize)
			}
			if ng.ScalingConfig.DesiredSize != nil {
				desiredSize = int(*ng.ScalingConfig.DesiredSize)
			}
		}

		var taints []NodeGroupTaint
		for _, t := range ng.Taints {
			taints = append(taints, NodeGroupTaint{
				Key:    aws.ToString(t.Key),
				Value:  aws.ToString(t.Value),
				Effect: string(t.Effect),
			})
		}

		var launchTemplate string
		if ng.LaunchTemplate != nil {
			launchTemplate = aws.ToString(ng.LaunchTemplate.Name)
		}

		nodeGroups = append(nodeGroups, EKSNodeGroup{
			Name:           aws.ToString(ng.NodegroupName),
			ARN:            aws.ToString(ng.NodegroupArn),
			Status:         string(ng.Status),
			InstanceTypes:  ng.InstanceTypes,
			AMIType:        string(ng.AmiType),
			MinSize:        minSize,
			MaxSize:        maxSize,
			DesiredSize:    desiredSize,
			Labels:         ng.Labels,
			Taints:         taints,
			Subnets:        ng.Subnets,
			LaunchTemplate: launchTemplate,
		})
	}

	return nodeGroups, nil
}

func (c *Client) ListAddons(ctx context.Context, clusterName string) ([]EKSAddon, error) {
	var names []string
	var nextToken *string

	for {
		out, err := c.api.ListAddons(ctx, &awseks.ListAddonsInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListAddons(%s): %w", clusterName, err)
		}

		names = append(names, out.Addons...)

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var addons []EKSAddon
	for _, name := range names {
		out, err := c.api.DescribeAddon(ctx, &awseks.DescribeAddonInput{
			ClusterName: aws.String(clusterName),
			AddonName:   aws.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeAddon(%s/%s): %w", clusterName, name, err)
		}

		a := out.Addon

		var health string
		if a.Health != nil && len(a.Health.Issues) > 0 {
			health = string(a.Health.Issues[0].Code)
		}

		addons = append(addons, EKSAddon{
			Name:                aws.ToString(a.AddonName),
			ARN:                 aws.ToString(a.AddonArn),
			Version:             aws.ToString(a.AddonVersion),
			Status:              string(a.Status),
			Health:              health,
			ServiceAccountRole:  aws.ToString(a.ServiceAccountRoleArn),
			ConfigurationValues: aws.ToString(a.ConfigurationValues),
		})
	}

	return addons, nil
}

func (c *Client) ListFargateProfiles(ctx context.Context, clusterName string) ([]EKSFargateProfile, error) {
	var names []string
	var nextToken *string

	for {
		out, err := c.api.ListFargateProfiles(ctx, &awseks.ListFargateProfilesInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListFargateProfiles(%s): %w", clusterName, err)
		}

		names = append(names, out.FargateProfileNames...)

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var profiles []EKSFargateProfile
	for _, name := range names {
		out, err := c.api.DescribeFargateProfile(ctx, &awseks.DescribeFargateProfileInput{
			ClusterName:        aws.String(clusterName),
			FargateProfileName: aws.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeFargateProfile(%s/%s): %w", clusterName, name, err)
		}

		fp := out.FargateProfile

		var selectors []FargateSelector
		for _, s := range fp.Selectors {
			selectors = append(selectors, FargateSelector{
				Namespace: aws.ToString(s.Namespace),
				Labels:    s.Labels,
			})
		}

		profiles = append(profiles, EKSFargateProfile{
			Name:             aws.ToString(fp.FargateProfileName),
			ARN:              aws.ToString(fp.FargateProfileArn),
			Status:           string(fp.Status),
			PodExecutionRole: aws.ToString(fp.PodExecutionRoleArn),
			Selectors:        selectors,
			Subnets:          fp.Subnets,
		})
	}

	return profiles, nil
}

func (c *Client) ListAccessEntries(ctx context.Context, clusterName string) ([]EKSAccessEntry, error) {
	var principalARNs []string
	var nextToken *string

	for {
		out, err := c.api.ListAccessEntries(ctx, &awseks.ListAccessEntriesInput{
			ClusterName: aws.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListAccessEntries(%s): %w", clusterName, err)
		}

		principalARNs = append(principalARNs, out.AccessEntries...)

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	var entries []EKSAccessEntry
	for _, principalARN := range principalARNs {
		out, err := c.api.DescribeAccessEntry(ctx, &awseks.DescribeAccessEntryInput{
			ClusterName:  aws.String(clusterName),
			PrincipalArn: aws.String(principalARN),
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeAccessEntry(%s/%s): %w", clusterName, principalARN, err)
		}

		ae := out.AccessEntry

		var createdAt time.Time
		if ae.CreatedAt != nil {
			createdAt = *ae.CreatedAt
		}

		entries = append(entries, EKSAccessEntry{
			PrincipalARN: aws.ToString(ae.PrincipalArn),
			Type:         aws.ToString(ae.Type),
			Username:     aws.ToString(ae.Username),
			Groups:       ae.KubernetesGroups,
			CreatedAt:    createdAt,
		})
	}

	return entries, nil
}

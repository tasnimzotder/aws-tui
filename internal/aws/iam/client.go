package iam

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsiam "github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type IAMAPI interface {
	ListUsers(ctx context.Context, params *awsiam.ListUsersInput, optFns ...func(*awsiam.Options)) (*awsiam.ListUsersOutput, error)
	ListRoles(ctx context.Context, params *awsiam.ListRolesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListRolesOutput, error)
	ListPolicies(ctx context.Context, params *awsiam.ListPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListPoliciesOutput, error)
	ListAttachedUserPolicies(ctx context.Context, params *awsiam.ListAttachedUserPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedUserPoliciesOutput, error)
	ListGroupsForUser(ctx context.Context, params *awsiam.ListGroupsForUserInput, optFns ...func(*awsiam.Options)) (*awsiam.ListGroupsForUserOutput, error)
	ListAttachedRolePolicies(ctx context.Context, params *awsiam.ListAttachedRolePoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedRolePoliciesOutput, error)
	ListEntitiesForPolicy(ctx context.Context, params *awsiam.ListEntitiesForPolicyInput, optFns ...func(*awsiam.Options)) (*awsiam.ListEntitiesForPolicyOutput, error)
}

type Client struct {
	api IAMAPI
}

func NewClient(api IAMAPI) *Client {
	return &Client{api: api}
}

func (c *Client) ListUsers(ctx context.Context) ([]IAMUser, error) {
	var users []IAMUser
	var marker *string

	for {
		out, err := c.api.ListUsers(ctx, &awsiam.ListUsersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListUsers: %w", err)
		}

		for _, u := range out.Users {
			var createdAt time.Time
			if u.CreateDate != nil {
				createdAt = *u.CreateDate
			}
			users = append(users, IAMUser{
				Name:      aws.ToString(u.UserName),
				UserID:    aws.ToString(u.UserId),
				ARN:       aws.ToString(u.Arn),
				Path:      aws.ToString(u.Path),
				CreatedAt: createdAt,
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return users, nil
}

func (c *Client) ListRoles(ctx context.Context) ([]IAMRole, error) {
	var roles []IAMRole
	var marker *string

	for {
		out, err := c.api.ListRoles(ctx, &awsiam.ListRolesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListRoles: %w", err)
		}

		for _, r := range out.Roles {
			var createdAt time.Time
			if r.CreateDate != nil {
				createdAt = *r.CreateDate
			}

			policyDoc := aws.ToString(r.AssumeRolePolicyDocument)
			if decoded, err := url.QueryUnescape(policyDoc); err == nil {
				policyDoc = decoded
			}

			roles = append(roles, IAMRole{
				Name:                     aws.ToString(r.RoleName),
				RoleID:                   aws.ToString(r.RoleId),
				ARN:                      aws.ToString(r.Arn),
				Path:                     aws.ToString(r.Path),
				Description:              aws.ToString(r.Description),
				CreatedAt:                createdAt,
				AssumeRolePolicyDocument: policyDoc,
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return roles, nil
}

func (c *Client) ListPolicies(ctx context.Context) ([]IAMPolicy, error) {
	var policies []IAMPolicy
	var marker *string

	for {
		out, err := c.api.ListPolicies(ctx, &awsiam.ListPoliciesInput{
			Scope:  iamtypes.PolicyScopeTypeLocal,
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListPolicies: %w", err)
		}

		for _, p := range out.Policies {
			var createdAt, updatedAt time.Time
			if p.CreateDate != nil {
				createdAt = *p.CreateDate
			}
			if p.UpdateDate != nil {
				updatedAt = *p.UpdateDate
			}

			var attachmentCount int
			if p.AttachmentCount != nil {
				attachmentCount = int(*p.AttachmentCount)
			}

			policies = append(policies, IAMPolicy{
				Name:            aws.ToString(p.PolicyName),
				PolicyID:        aws.ToString(p.PolicyId),
				ARN:             aws.ToString(p.Arn),
				Path:            aws.ToString(p.Path),
				AttachmentCount: attachmentCount,
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return policies, nil
}

func (c *Client) ListAttachedUserPolicies(ctx context.Context, userName string) ([]IAMAttachedPolicy, error) {
	var policies []IAMAttachedPolicy
	var marker *string

	for {
		out, err := c.api.ListAttachedUserPolicies(ctx, &awsiam.ListAttachedUserPoliciesInput{
			UserName: aws.String(userName),
			Marker:   marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListAttachedUserPolicies(%s): %w", userName, err)
		}

		for _, p := range out.AttachedPolicies {
			policies = append(policies, IAMAttachedPolicy{
				Name: aws.ToString(p.PolicyName),
				ARN:  aws.ToString(p.PolicyArn),
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return policies, nil
}

func (c *Client) ListGroupsForUser(ctx context.Context, userName string) ([]IAMGroup, error) {
	var groups []IAMGroup
	var marker *string

	for {
		out, err := c.api.ListGroupsForUser(ctx, &awsiam.ListGroupsForUserInput{
			UserName: aws.String(userName),
			Marker:   marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListGroupsForUser(%s): %w", userName, err)
		}

		for _, g := range out.Groups {
			groups = append(groups, IAMGroup{
				Name: aws.ToString(g.GroupName),
				ARN:  aws.ToString(g.Arn),
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return groups, nil
}

func (c *Client) ListAttachedRolePolicies(ctx context.Context, roleName string) ([]IAMAttachedPolicy, error) {
	var policies []IAMAttachedPolicy
	var marker *string

	for {
		out, err := c.api.ListAttachedRolePolicies(ctx, &awsiam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(roleName),
			Marker:   marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListAttachedRolePolicies(%s): %w", roleName, err)
		}

		for _, p := range out.AttachedPolicies {
			policies = append(policies, IAMAttachedPolicy{
				Name: aws.ToString(p.PolicyName),
				ARN:  aws.ToString(p.PolicyArn),
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return policies, nil
}

func (c *Client) ListEntitiesForPolicy(ctx context.Context, policyARN string) ([]IAMPolicyEntity, error) {
	var entities []IAMPolicyEntity
	var marker *string

	for {
		out, err := c.api.ListEntitiesForPolicy(ctx, &awsiam.ListEntitiesForPolicyInput{
			PolicyArn: aws.String(policyARN),
			Marker:    marker,
		})
		if err != nil {
			return nil, fmt.Errorf("ListEntitiesForPolicy(%s): %w", policyARN, err)
		}

		// Groups first
		for _, g := range out.PolicyGroups {
			entities = append(entities, IAMPolicyEntity{
				Name: aws.ToString(g.GroupName),
				Type: "Group",
			})
		}

		// Then Users
		for _, u := range out.PolicyUsers {
			entities = append(entities, IAMPolicyEntity{
				Name: aws.ToString(u.UserName),
				Type: "User",
			})
		}

		// Then Roles
		for _, r := range out.PolicyRoles {
			entities = append(entities, IAMPolicyEntity{
				Name: aws.ToString(r.RoleName),
				Type: "Role",
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}

	return entities, nil
}

// ListUsersPage fetches a single page of IAM users.
func (c *Client) ListUsersPage(ctx context.Context, marker *string) ([]IAMUser, *string, error) {
	out, err := c.api.ListUsers(ctx, &awsiam.ListUsersInput{
		Marker: marker,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ListUsers: %w", err)
	}

	users := make([]IAMUser, 0, len(out.Users))
	for _, u := range out.Users {
		var createdAt time.Time
		if u.CreateDate != nil {
			createdAt = *u.CreateDate
		}
		users = append(users, IAMUser{
			Name:      aws.ToString(u.UserName),
			UserID:    aws.ToString(u.UserId),
			ARN:       aws.ToString(u.Arn),
			Path:      aws.ToString(u.Path),
			CreatedAt: createdAt,
		})
	}

	var nextMarker *string
	if out.IsTruncated {
		nextMarker = out.Marker
	}
	return users, nextMarker, nil
}

// ListRolesPage fetches a single page of IAM roles.
func (c *Client) ListRolesPage(ctx context.Context, marker *string) ([]IAMRole, *string, error) {
	out, err := c.api.ListRoles(ctx, &awsiam.ListRolesInput{
		Marker: marker,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ListRoles: %w", err)
	}

	roles := make([]IAMRole, 0, len(out.Roles))
	for _, r := range out.Roles {
		var createdAt time.Time
		if r.CreateDate != nil {
			createdAt = *r.CreateDate
		}
		policyDoc := aws.ToString(r.AssumeRolePolicyDocument)
		if decoded, err := url.QueryUnescape(policyDoc); err == nil {
			policyDoc = decoded
		}
		roles = append(roles, IAMRole{
			Name:                     aws.ToString(r.RoleName),
			RoleID:                   aws.ToString(r.RoleId),
			ARN:                      aws.ToString(r.Arn),
			Path:                     aws.ToString(r.Path),
			Description:              aws.ToString(r.Description),
			CreatedAt:                createdAt,
			AssumeRolePolicyDocument: policyDoc,
		})
	}

	var nextMarker *string
	if out.IsTruncated {
		nextMarker = out.Marker
	}
	return roles, nextMarker, nil
}

// ListPoliciesPage fetches a single page of customer-managed IAM policies.
func (c *Client) ListPoliciesPage(ctx context.Context, marker *string) ([]IAMPolicy, *string, error) {
	out, err := c.api.ListPolicies(ctx, &awsiam.ListPoliciesInput{
		Scope:  iamtypes.PolicyScopeTypeLocal,
		Marker: marker,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ListPolicies: %w", err)
	}

	policies := make([]IAMPolicy, 0, len(out.Policies))
	for _, p := range out.Policies {
		var createdAt, updatedAt time.Time
		if p.CreateDate != nil {
			createdAt = *p.CreateDate
		}
		if p.UpdateDate != nil {
			updatedAt = *p.UpdateDate
		}
		var attachmentCount int
		if p.AttachmentCount != nil {
			attachmentCount = int(*p.AttachmentCount)
		}
		policies = append(policies, IAMPolicy{
			Name:            aws.ToString(p.PolicyName),
			PolicyID:        aws.ToString(p.PolicyId),
			ARN:             aws.ToString(p.Arn),
			Path:            aws.ToString(p.Path),
			AttachmentCount: attachmentCount,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		})
	}

	var nextMarker *string
	if out.IsTruncated {
		nextMarker = out.Marker
	}
	return policies, nextMarker, nil
}

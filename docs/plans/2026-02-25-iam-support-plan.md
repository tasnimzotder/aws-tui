# IAM Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add IAM browsing to the services TUI with Users, Roles, and Policies (customer-managed), each with drill-down into attached policies, group memberships, trust policies, and attached entities.

**Architecture:** New IAM client (`internal/aws/iam/`) follows the same interface + mock pattern as other services. IAM sub-menu (like VPC/ECS) routes to three list views. Users and Roles have their own sub-menus for drill-down. Trust policy uses a viewport-based text view (like ECS Config). All IAM APIs paginate with `Marker`/`IsTruncated` — client exhausts all pages internally.

**Tech Stack:** Go 1.26, aws-sdk-go-v2/service/iam, Bubble Tea, existing `TableView[T]` and viewport patterns.

**Design doc:** `docs/plans/2026-02-25-iam-support-design.md`

---

### Task 1: Add IAM SDK dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

Run: `go get github.com/aws/aws-sdk-go-v2/service/iam`

**Step 2: Verify**

Run: `go build ./...`
Expected: Compiles with no errors.

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "add aws-sdk-go-v2/service/iam dependency"
```

---

### Task 2: IAM types

**Files:**
- Create: `internal/aws/iam/types.go`

**Step 1: Create the types file**

```go
package iam

import "time"

type IAMUser struct {
	Name      string
	UserID    string
	ARN       string
	Path      string
	CreatedAt time.Time
}

type IAMRole struct {
	Name                     string
	RoleID                   string
	ARN                      string
	Path                     string
	Description              string
	CreatedAt                time.Time
	AssumeRolePolicyDocument string // URL-encoded JSON
}

type IAMPolicy struct {
	Name            string
	PolicyID        string
	ARN             string
	Path            string
	AttachmentCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type IAMAttachedPolicy struct {
	Name string
	ARN  string
}

type IAMGroup struct {
	Name string
	ARN  string
}

type IAMPolicyEntity struct {
	Name string
	Type string // "User", "Role", "Group"
}
```

**Step 2: Verify**

Run: `go build ./internal/aws/iam/`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/aws/iam/types.go
git commit -m "add IAM types"
```

---

### Task 3: IAM client — ListUsers (TDD)

**Files:**
- Create: `internal/aws/iam/client.go`
- Create: `internal/aws/iam/client_test.go`

**Step 1: Write the failing test**

```go
// internal/aws/iam/client_test.go
package iam

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsiam "github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type mockIAMAPI struct {
	listUsersFunc                func(ctx context.Context, params *awsiam.ListUsersInput, optFns ...func(*awsiam.Options)) (*awsiam.ListUsersOutput, error)
	listRolesFunc                func(ctx context.Context, params *awsiam.ListRolesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListRolesOutput, error)
	listPoliciesFunc             func(ctx context.Context, params *awsiam.ListPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListPoliciesOutput, error)
	listAttachedUserPoliciesFunc func(ctx context.Context, params *awsiam.ListAttachedUserPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedUserPoliciesOutput, error)
	listGroupsForUserFunc        func(ctx context.Context, params *awsiam.ListGroupsForUserInput, optFns ...func(*awsiam.Options)) (*awsiam.ListGroupsForUserOutput, error)
	listAttachedRolePoliciesFunc func(ctx context.Context, params *awsiam.ListAttachedRolePoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedRolePoliciesOutput, error)
	listEntitiesForPolicyFunc    func(ctx context.Context, params *awsiam.ListEntitiesForPolicyInput, optFns ...func(*awsiam.Options)) (*awsiam.ListEntitiesForPolicyOutput, error)
}

func (m *mockIAMAPI) ListUsers(ctx context.Context, params *awsiam.ListUsersInput, optFns ...func(*awsiam.Options)) (*awsiam.ListUsersOutput, error) {
	return m.listUsersFunc(ctx, params, optFns...)
}
func (m *mockIAMAPI) ListRoles(ctx context.Context, params *awsiam.ListRolesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListRolesOutput, error) {
	return m.listRolesFunc(ctx, params, optFns...)
}
func (m *mockIAMAPI) ListPolicies(ctx context.Context, params *awsiam.ListPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListPoliciesOutput, error) {
	return m.listPoliciesFunc(ctx, params, optFns...)
}
func (m *mockIAMAPI) ListAttachedUserPolicies(ctx context.Context, params *awsiam.ListAttachedUserPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedUserPoliciesOutput, error) {
	return m.listAttachedUserPoliciesFunc(ctx, params, optFns...)
}
func (m *mockIAMAPI) ListGroupsForUser(ctx context.Context, params *awsiam.ListGroupsForUserInput, optFns ...func(*awsiam.Options)) (*awsiam.ListGroupsForUserOutput, error) {
	return m.listGroupsForUserFunc(ctx, params, optFns...)
}
func (m *mockIAMAPI) ListAttachedRolePolicies(ctx context.Context, params *awsiam.ListAttachedRolePoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedRolePoliciesOutput, error) {
	return m.listAttachedRolePoliciesFunc(ctx, params, optFns...)
}
func (m *mockIAMAPI) ListEntitiesForPolicy(ctx context.Context, params *awsiam.ListEntitiesForPolicyInput, optFns ...func(*awsiam.Options)) (*awsiam.ListEntitiesForPolicyOutput, error) {
	return m.listEntitiesForPolicyFunc(ctx, params, optFns...)
}

func TestListUsers(t *testing.T) {
	created := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	mock := &mockIAMAPI{
		listUsersFunc: func(ctx context.Context, params *awsiam.ListUsersInput, optFns ...func(*awsiam.Options)) (*awsiam.ListUsersOutput, error) {
			return &awsiam.ListUsersOutput{
				Users: []iamtypes.User{
					{
						UserName:   awssdk.String("alice"),
						UserId:     awssdk.String("AIDA111"),
						Arn:        awssdk.String("arn:aws:iam::123:user/alice"),
						Path:       awssdk.String("/"),
						CreateDate: &created,
					},
					{
						UserName:   awssdk.String("bob"),
						UserId:     awssdk.String("AIDA222"),
						Arn:        awssdk.String("arn:aws:iam::123:user/bob"),
						Path:       awssdk.String("/engineering/"),
						CreateDate: &created,
					},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}

	client := NewClient(mock)
	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "alice" {
		t.Errorf("Name = %s, want alice", users[0].Name)
	}
	if users[0].ARN != "arn:aws:iam::123:user/alice" {
		t.Errorf("ARN = %s, want arn:aws:iam::123:user/alice", users[0].ARN)
	}
	if users[1].Path != "/engineering/" {
		t.Errorf("Path = %s, want /engineering/", users[1].Path)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/aws/iam/ -run TestListUsers -v`
Expected: FAIL — `NewClient` not defined.

**Step 3: Write the client**

```go
// internal/aws/iam/client.go
package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsiam "github.com/aws/aws-sdk-go-v2/service/iam"
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
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		marker = out.Marker
	}

	return users, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/aws/iam/ -run TestListUsers -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/aws/iam/client.go internal/aws/iam/client_test.go
git commit -m "add IAM client with ListUsers"
```

---

### Task 4: IAM client — ListRoles (TDD)

**Files:**
- Modify: `internal/aws/iam/client.go`
- Modify: `internal/aws/iam/client_test.go`

**Step 1: Write the failing test**

```go
func TestListRoles(t *testing.T) {
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockIAMAPI{
		listRolesFunc: func(ctx context.Context, params *awsiam.ListRolesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListRolesOutput, error) {
			return &awsiam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName:                 awssdk.String("lambda-exec"),
						RoleId:                   awssdk.String("AROA111"),
						Arn:                      awssdk.String("arn:aws:iam::123:role/lambda-exec"),
						Path:                     awssdk.String("/service-role/"),
						Description:              awssdk.String("Lambda execution role"),
						CreateDate:               &created,
						AssumeRolePolicyDocument: awssdk.String("%7B%22Version%22%3A%222012-10-17%22%7D"),
					},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}

	client := NewClient(mock)
	roles, err := client.ListRoles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	if roles[0].Name != "lambda-exec" {
		t.Errorf("Name = %s, want lambda-exec", roles[0].Name)
	}
	if roles[0].Description != "Lambda execution role" {
		t.Errorf("Description = %s, want Lambda execution role", roles[0].Description)
	}
	// AssumeRolePolicyDocument should be URL-decoded
	if roles[0].AssumeRolePolicyDocument != `{"Version":"2012-10-17"}` {
		t.Errorf("AssumeRolePolicyDocument = %s, want decoded JSON", roles[0].AssumeRolePolicyDocument)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/aws/iam/ -run TestListRoles -v`
Expected: FAIL — `ListRoles` not defined.

**Step 3: Add ListRoles to client.go**

```go
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
			doc := aws.ToString(r.AssumeRolePolicyDocument)
			if decoded, err := url.QueryUnescape(doc); err == nil {
				doc = decoded
			}
			roles = append(roles, IAMRole{
				Name:                     aws.ToString(r.RoleName),
				RoleID:                   aws.ToString(r.RoleId),
				ARN:                      aws.ToString(r.Arn),
				Path:                     aws.ToString(r.Path),
				Description:              aws.ToString(r.Description),
				CreatedAt:                createdAt,
				AssumeRolePolicyDocument: doc,
			})
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		marker = out.Marker
	}

	return roles, nil
}
```

Add `"net/url"` to the imports in client.go.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/aws/iam/ -run TestListRoles -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/aws/iam/client.go internal/aws/iam/client_test.go
git commit -m "add IAM ListRoles with URL-decoded trust policy"
```

---

### Task 5: IAM client — ListPolicies (TDD)

**Files:**
- Modify: `internal/aws/iam/client.go`
- Modify: `internal/aws/iam/client_test.go`

**Step 1: Write the failing test**

```go
func TestListPolicies(t *testing.T) {
	created := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockIAMAPI{
		listPoliciesFunc: func(ctx context.Context, params *awsiam.ListPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListPoliciesOutput, error) {
			// Verify Scope is "Local" (customer-managed only)
			if params.Scope != "Local" {
				t.Errorf("Scope = %s, want Local", params.Scope)
			}
			return &awsiam.ListPoliciesOutput{
				Policies: []iamtypes.Policy{
					{
						PolicyName:      awssdk.String("my-policy"),
						PolicyId:        awssdk.String("ANPA111"),
						Arn:             awssdk.String("arn:aws:iam::123:policy/my-policy"),
						Path:            awssdk.String("/"),
						AttachmentCount: awssdk.Int32(3),
						CreateDate:      &created,
						UpdateDate:      &updated,
					},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}

	client := NewClient(mock)
	policies, err := client.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].Name != "my-policy" {
		t.Errorf("Name = %s, want my-policy", policies[0].Name)
	}
	if policies[0].AttachmentCount != 3 {
		t.Errorf("AttachmentCount = %d, want 3", policies[0].AttachmentCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/aws/iam/ -run TestListPolicies -v`
Expected: FAIL — `ListPolicies` not defined.

**Step 3: Add ListPolicies to client.go**

```go
func (c *Client) ListPolicies(ctx context.Context) ([]IAMPolicy, error) {
	var policies []IAMPolicy
	var marker *string

	for {
		out, err := c.api.ListPolicies(ctx, &awsiam.ListPoliciesInput{
			Scope:  "Local",
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
			policies = append(policies, IAMPolicy{
				Name:            aws.ToString(p.PolicyName),
				PolicyID:        aws.ToString(p.PolicyId),
				ARN:             aws.ToString(p.Arn),
				Path:            aws.ToString(p.Path),
				AttachmentCount: int(aws.ToInt32(p.AttachmentCount)),
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
			})
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		marker = out.Marker
	}

	return policies, nil
}
```

Note: `ListPoliciesInput.Scope` is a `types.PolicyScopeType` string enum. Check if it's `iamtypes.PolicyScopeTypeLocal` or just use the string `"Local"`. Use whichever compiles.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/aws/iam/ -run TestListPolicies -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/aws/iam/client.go internal/aws/iam/client_test.go
git commit -m "add IAM ListPolicies (customer-managed scope)"
```

---

### Task 6: IAM client — Sub-resource listing methods (TDD)

**Files:**
- Modify: `internal/aws/iam/client.go`
- Modify: `internal/aws/iam/client_test.go`

**Step 1: Write failing tests for all four sub-resource methods**

```go
func TestListAttachedUserPolicies(t *testing.T) {
	mock := &mockIAMAPI{
		listAttachedUserPoliciesFunc: func(ctx context.Context, params *awsiam.ListAttachedUserPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedUserPoliciesOutput, error) {
			if awssdk.ToString(params.UserName) != "alice" {
				t.Errorf("UserName = %s, want alice", awssdk.ToString(params.UserName))
			}
			return &awsiam.ListAttachedUserPoliciesOutput{
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{PolicyName: awssdk.String("ReadOnly"), PolicyArn: awssdk.String("arn:aws:iam::aws:policy/ReadOnly")},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}
	client := NewClient(mock)
	policies, err := client.ListAttachedUserPolicies(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].Name != "ReadOnly" {
		t.Errorf("Name = %s, want ReadOnly", policies[0].Name)
	}
}

func TestListGroupsForUser(t *testing.T) {
	mock := &mockIAMAPI{
		listGroupsForUserFunc: func(ctx context.Context, params *awsiam.ListGroupsForUserInput, optFns ...func(*awsiam.Options)) (*awsiam.ListGroupsForUserOutput, error) {
			return &awsiam.ListGroupsForUserOutput{
				Groups: []iamtypes.Group{
					{GroupName: awssdk.String("Admins"), Arn: awssdk.String("arn:aws:iam::123:group/Admins")},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}
	client := NewClient(mock)
	groups, err := client.ListGroupsForUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Admins" {
		t.Errorf("unexpected groups: %+v", groups)
	}
}

func TestListAttachedRolePolicies(t *testing.T) {
	mock := &mockIAMAPI{
		listAttachedRolePoliciesFunc: func(ctx context.Context, params *awsiam.ListAttachedRolePoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedRolePoliciesOutput, error) {
			return &awsiam.ListAttachedRolePoliciesOutput{
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{PolicyName: awssdk.String("LambdaBasic"), PolicyArn: awssdk.String("arn:aws:iam::aws:policy/LambdaBasic")},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}
	client := NewClient(mock)
	policies, err := client.ListAttachedRolePolicies(context.Background(), "lambda-exec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 1 || policies[0].Name != "LambdaBasic" {
		t.Errorf("unexpected policies: %+v", policies)
	}
}

func TestListEntitiesForPolicy(t *testing.T) {
	mock := &mockIAMAPI{
		listEntitiesForPolicyFunc: func(ctx context.Context, params *awsiam.ListEntitiesForPolicyInput, optFns ...func(*awsiam.Options)) (*awsiam.ListEntitiesForPolicyOutput, error) {
			return &awsiam.ListEntitiesForPolicyOutput{
				PolicyGroups: []iamtypes.PolicyGroup{
					{GroupName: awssdk.String("Devs")},
				},
				PolicyUsers: []iamtypes.PolicyUser{
					{UserName: awssdk.String("alice")},
				},
				PolicyRoles: []iamtypes.PolicyRole{
					{RoleName: awssdk.String("deploy-role")},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}
	client := NewClient(mock)
	entities, err := client.ListEntitiesForPolicy(context.Background(), "arn:aws:iam::123:policy/my-policy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(entities))
	}
	// Should be ordered: Groups, Users, Roles
	if entities[0].Type != "Group" || entities[0].Name != "Devs" {
		t.Errorf("entities[0] = %+v, want Group/Devs", entities[0])
	}
	if entities[1].Type != "User" || entities[1].Name != "alice" {
		t.Errorf("entities[1] = %+v, want User/alice", entities[1])
	}
	if entities[2].Type != "Role" || entities[2].Name != "deploy-role" {
		t.Errorf("entities[2] = %+v, want Role/deploy-role", entities[2])
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/aws/iam/ -run "TestListAttached|TestListGroups|TestListEntities" -v`
Expected: FAIL — methods not defined.

**Step 3: Implement all four methods**

Add to `internal/aws/iam/client.go`:

```go
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
		if !aws.ToBool(out.IsTruncated) {
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
		if !aws.ToBool(out.IsTruncated) {
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
		if !aws.ToBool(out.IsTruncated) {
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
			return nil, fmt.Errorf("ListEntitiesForPolicy: %w", err)
		}
		for _, g := range out.PolicyGroups {
			entities = append(entities, IAMPolicyEntity{Name: aws.ToString(g.GroupName), Type: "Group"})
		}
		for _, u := range out.PolicyUsers {
			entities = append(entities, IAMPolicyEntity{Name: aws.ToString(u.UserName), Type: "User"})
		}
		for _, r := range out.PolicyRoles {
			entities = append(entities, IAMPolicyEntity{Name: aws.ToString(r.RoleName), Type: "Role"})
		}
		if !aws.ToBool(out.IsTruncated) {
			break
		}
		marker = out.Marker
	}

	return entities, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/aws/iam/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/aws/iam/client.go internal/aws/iam/client_test.go
git commit -m "add IAM sub-resource listing methods"
```

---

### Task 7: Wire IAM client into ServiceClient

**Files:**
- Modify: `internal/aws/client.go`

**Step 1: Add IAM to ServiceClient**

Add imports:
```go
"github.com/aws/aws-sdk-go-v2/service/iam"
awsiam "tasnim.dev/aws-tui/internal/aws/iam"
```

Add field to `ServiceClient` struct:
```go
IAM *awsiam.Client
```

Add to `NewServiceClient` return:
```go
IAM: awsiam.NewClient(iam.NewFromConfig(cfg)),
```

**Step 2: Verify**

Run: `go build ./...`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/aws/client.go
git commit -m "wire IAM client into ServiceClient"
```

---

### Task 8: IAM TUI views — sub-menu and list views

**Files:**
- Create: `internal/tui/services/iam.go`

**Step 1: Create the IAM views file**

This file contains:
1. IAM top-level sub-menu (Users / Roles / Policies)
2. Users list view
3. Roles list view
4. Policies list view
5. User sub-menu (Attached Policies / Groups)
6. Role sub-menu (Attached Policies / Trust Policy)
7. User attached policies view
8. User groups view
9. Role attached policies view
10. Policy attached entities view
11. Trust policy text view (viewport-based, like ECS Config)

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// --- IAM Top-level Sub-menu ---

type iamMenuItem struct {
	name string
	desc string
}

func (i iamMenuItem) Title() string       { return i.name }
func (i iamMenuItem) Description() string { return i.desc }
func (i iamMenuItem) FilterValue() string { return i.name }

type IAMSubMenuView struct {
	client *awsclient.ServiceClient
	list   list.Model
}

func NewIAMSubMenuView(client *awsclient.ServiceClient) *IAMSubMenuView {
	items := []list.Item{
		iamMenuItem{name: "Users", desc: "IAM users and their policies/groups"},
		iamMenuItem{name: "Roles", desc: "IAM roles and trust policies"},
		iamMenuItem{name: "Policies", desc: "Customer-managed IAM policies"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &IAMSubMenuView{client: client, list: l}
}

func (v *IAMSubMenuView) Title() string { return "IAM" }
func (v *IAMSubMenuView) Init() tea.Cmd  { return nil }
func (v *IAMSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(iamMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Users":
				return v, pushView(NewIAMUsersView(v.client))
			case "Roles":
				return v, pushView(NewIAMRolesView(v.client))
			case "Policies":
				return v, pushView(NewIAMPoliciesView(v.client))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *IAMSubMenuView) View() string { return v.list.View() }
func (v *IAMSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// --- Users List ---

func NewIAMUsersView(client *awsclient.ServiceClient) *TableView[awsiam.IAMUser] {
	return NewTableView(TableViewConfig[awsiam.IAMUser]{
		Title:       "Users",
		LoadingText: "Loading users...",
		Columns: []table.Column{
			{Title: "Name", Width: 25},
			{Title: "User ID", Width: 22},
			{Title: "Created", Width: 20},
			{Title: "Path", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMUser, error) {
			return client.IAM.ListUsers(ctx)
		},
		RowMapper: func(u awsiam.IAMUser) table.Row {
			return table.Row{u.Name, u.UserID, utils.TimeOrDash(u.CreatedAt, utils.DateOnly), u.Path}
		},
		CopyIDFunc:  func(u awsiam.IAMUser) string { return u.Name },
		CopyARNFunc: func(u awsiam.IAMUser) string { return u.ARN },
		OnEnter: func(u awsiam.IAMUser) tea.Cmd {
			return pushView(NewIAMUserSubMenuView(client, u.Name))
		},
	})
}

// --- Roles List ---

func NewIAMRolesView(client *awsclient.ServiceClient) *TableView[awsiam.IAMRole] {
	return NewTableView(TableViewConfig[awsiam.IAMRole]{
		Title:       "Roles",
		LoadingText: "Loading roles...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Description", Width: 30},
			{Title: "Created", Width: 20},
			{Title: "Path", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMRole, error) {
			return client.IAM.ListRoles(ctx)
		},
		RowMapper: func(r awsiam.IAMRole) table.Row {
			desc := r.Description
			if desc == "" {
				desc = "—"
			}
			return table.Row{r.Name, desc, utils.TimeOrDash(r.CreatedAt, utils.DateOnly), r.Path}
		},
		CopyIDFunc:  func(r awsiam.IAMRole) string { return r.Name },
		CopyARNFunc: func(r awsiam.IAMRole) string { return r.ARN },
		OnEnter: func(r awsiam.IAMRole) tea.Cmd {
			return pushView(NewIAMRoleSubMenuView(client, r.Name, r.AssumeRolePolicyDocument))
		},
	})
}

// --- Policies List ---

func NewIAMPoliciesView(client *awsclient.ServiceClient) *TableView[awsiam.IAMPolicy] {
	return NewTableView(TableViewConfig[awsiam.IAMPolicy]{
		Title:       "Policies",
		LoadingText: "Loading policies...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Attachments", Width: 12},
			{Title: "Created", Width: 20},
			{Title: "Updated", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMPolicy, error) {
			return client.IAM.ListPolicies(ctx)
		},
		RowMapper: func(p awsiam.IAMPolicy) table.Row {
			return table.Row{p.Name, fmt.Sprintf("%d", p.AttachmentCount), utils.TimeOrDash(p.CreatedAt, utils.DateOnly), utils.TimeOrDash(p.UpdatedAt, utils.DateOnly)}
		},
		CopyIDFunc:  func(p awsiam.IAMPolicy) string { return p.Name },
		CopyARNFunc: func(p awsiam.IAMPolicy) string { return p.ARN },
		OnEnter: func(p awsiam.IAMPolicy) tea.Cmd {
			return pushView(NewIAMPolicyEntitiesView(client, p.ARN, p.Name))
		},
	})
}

// --- User Sub-menu ---

type iamUserSubMenuItem struct {
	name string
	desc string
}

func (i iamUserSubMenuItem) Title() string       { return i.name }
func (i iamUserSubMenuItem) Description() string { return i.desc }
func (i iamUserSubMenuItem) FilterValue() string { return i.name }

type IAMUserSubMenuView struct {
	client   *awsclient.ServiceClient
	userName string
	list     list.Model
}

func NewIAMUserSubMenuView(client *awsclient.ServiceClient, userName string) *IAMUserSubMenuView {
	items := []list.Item{
		iamUserSubMenuItem{name: "Attached Policies", desc: "Managed policies attached to this user"},
		iamUserSubMenuItem{name: "Groups", desc: "Groups this user belongs to"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 8)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &IAMUserSubMenuView{client: client, userName: userName, list: l}
}

func (v *IAMUserSubMenuView) Title() string { return v.userName }
func (v *IAMUserSubMenuView) Init() tea.Cmd  { return nil }
func (v *IAMUserSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(iamUserSubMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Attached Policies":
				return v, pushView(NewIAMUserPoliciesView(v.client, v.userName))
			case "Groups":
				return v, pushView(NewIAMUserGroupsView(v.client, v.userName))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *IAMUserSubMenuView) View() string { return v.list.View() }
func (v *IAMUserSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// --- Role Sub-menu ---

type iamRoleSubMenuItem struct {
	name string
	desc string
}

func (i iamRoleSubMenuItem) Title() string       { return i.name }
func (i iamRoleSubMenuItem) Description() string { return i.desc }
func (i iamRoleSubMenuItem) FilterValue() string { return i.name }

type IAMRoleSubMenuView struct {
	client   *awsclient.ServiceClient
	roleName string
	trustDoc string
	list     list.Model
}

func NewIAMRoleSubMenuView(client *awsclient.ServiceClient, roleName, trustDoc string) *IAMRoleSubMenuView {
	items := []list.Item{
		iamRoleSubMenuItem{name: "Attached Policies", desc: "Managed policies attached to this role"},
		iamRoleSubMenuItem{name: "Trust Policy", desc: "Who can assume this role"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 8)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &IAMRoleSubMenuView{client: client, roleName: roleName, trustDoc: trustDoc, list: l}
}

func (v *IAMRoleSubMenuView) Title() string { return v.roleName }
func (v *IAMRoleSubMenuView) Init() tea.Cmd  { return nil }
func (v *IAMRoleSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(iamRoleSubMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Attached Policies":
				return v, pushView(NewIAMRolePoliciesView(v.client, v.roleName))
			case "Trust Policy":
				return v, pushView(NewIAMTrustPolicyView(v.roleName, v.trustDoc))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *IAMRoleSubMenuView) View() string { return v.list.View() }
func (v *IAMRoleSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

// --- User Attached Policies ---

func NewIAMUserPoliciesView(client *awsclient.ServiceClient, userName string) *TableView[awsiam.IAMAttachedPolicy] {
	return NewTableView(TableViewConfig[awsiam.IAMAttachedPolicy]{
		Title:       "Policies",
		LoadingText: "Loading attached policies...",
		Columns: []table.Column{
			{Title: "Policy Name", Width: 35},
			{Title: "ARN", Width: 55},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMAttachedPolicy, error) {
			return client.IAM.ListAttachedUserPolicies(ctx, userName)
		},
		RowMapper: func(p awsiam.IAMAttachedPolicy) table.Row {
			return table.Row{p.Name, p.ARN}
		},
		CopyIDFunc:  func(p awsiam.IAMAttachedPolicy) string { return p.Name },
		CopyARNFunc: func(p awsiam.IAMAttachedPolicy) string { return p.ARN },
	})
}

// --- User Groups ---

func NewIAMUserGroupsView(client *awsclient.ServiceClient, userName string) *TableView[awsiam.IAMGroup] {
	return NewTableView(TableViewConfig[awsiam.IAMGroup]{
		Title:       "Groups",
		LoadingText: "Loading groups...",
		Columns: []table.Column{
			{Title: "Group Name", Width: 30},
			{Title: "ARN", Width: 55},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMGroup, error) {
			return client.IAM.ListGroupsForUser(ctx, userName)
		},
		RowMapper: func(g awsiam.IAMGroup) table.Row {
			return table.Row{g.Name, g.ARN}
		},
		CopyIDFunc:  func(g awsiam.IAMGroup) string { return g.Name },
		CopyARNFunc: func(g awsiam.IAMGroup) string { return g.ARN },
	})
}

// --- Role Attached Policies ---

func NewIAMRolePoliciesView(client *awsclient.ServiceClient, roleName string) *TableView[awsiam.IAMAttachedPolicy] {
	return NewTableView(TableViewConfig[awsiam.IAMAttachedPolicy]{
		Title:       "Policies",
		LoadingText: "Loading attached policies...",
		Columns: []table.Column{
			{Title: "Policy Name", Width: 35},
			{Title: "ARN", Width: 55},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMAttachedPolicy, error) {
			return client.IAM.ListAttachedRolePolicies(ctx, roleName)
		},
		RowMapper: func(p awsiam.IAMAttachedPolicy) table.Row {
			return table.Row{p.Name, p.ARN}
		},
		CopyIDFunc:  func(p awsiam.IAMAttachedPolicy) string { return p.Name },
		CopyARNFunc: func(p awsiam.IAMAttachedPolicy) string { return p.ARN },
	})
}

// --- Policy Attached Entities ---

func NewIAMPolicyEntitiesView(client *awsclient.ServiceClient, policyARN, policyName string) *TableView[awsiam.IAMPolicyEntity] {
	return NewTableView(TableViewConfig[awsiam.IAMPolicyEntity]{
		Title:       policyName,
		LoadingText: "Loading attached entities...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Type", Width: 12},
		},
		FetchFunc: func(ctx context.Context) ([]awsiam.IAMPolicyEntity, error) {
			return client.IAM.ListEntitiesForPolicy(ctx, policyARN)
		},
		RowMapper: func(e awsiam.IAMPolicyEntity) table.Row {
			return table.Row{e.Name, e.Type}
		},
		CopyIDFunc: func(e awsiam.IAMPolicyEntity) string { return e.Name },
	})
}

// --- Trust Policy Text View (viewport-based) ---

type IAMTrustPolicyView struct {
	roleName string
	viewport viewport.Model
	content  string
	ready    bool
	width    int
	height   int
}

func NewIAMTrustPolicyView(roleName, trustDoc string) *IAMTrustPolicyView {
	// Pretty-print the JSON
	content := trustDoc
	var parsed any
	if err := json.Unmarshal([]byte(trustDoc), &parsed); err == nil {
		if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
			content = string(pretty)
		}
	}
	return &IAMTrustPolicyView{
		roleName: roleName,
		content:  content,
		width:    80,
		height:   20,
	}
}

func (v *IAMTrustPolicyView) Title() string { return "Trust Policy" }
func (v *IAMTrustPolicyView) Init() tea.Cmd {
	v.viewport = viewport.New(v.width, v.height)
	v.viewport.SetContent(v.content)
	v.ready = true
	return nil
}
func (v *IAMTrustPolicyView) Update(msg tea.Msg) (View, tea.Cmd) {
	if v.ready {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}
func (v *IAMTrustPolicyView) View() string {
	if v.ready {
		return v.viewport.View()
	}
	return ""
}
func (v *IAMTrustPolicyView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.viewport.Width = width
		v.viewport.Height = height
	}
}
```

**Step 2: Verify**

Run: `go build ./internal/tui/services/`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/tui/services/iam.go
git commit -m "add IAM TUI views with sub-menus and trust policy viewer"
```

---

### Task 9: Wire IAM into root menu + update README

**Files:**
- Modify: `internal/tui/services/root.go`
- Modify: `README.md`

**Step 1: Add IAM to the service list**

In `root.go`, add to `items` slice (after S3):
```go
serviceItem{name: "IAM", desc: "Identity & Access Management — Users, Roles, Policies"},
```

Add to `handleSelection` switch:
```go
case "IAM":
	return func() tea.Msg {
		return PushViewMsg{View: NewIAMSubMenuView(v.client)}
	}
```

**Step 2: Update README**

Add row to supported services table (after S3):
```markdown
| **IAM** | Users → Policies/Groups, Roles → Policies/Trust Policy, Policies → Attached Entities |
```

Update limitations line — remove "IAM" from the "no X" list if present.

**Step 3: Verify**

Run: `go build ./...`
Expected: Compiles.

**Step 4: Commit**

```bash
git add internal/tui/services/root.go README.md
git commit -m "add IAM to services root menu and README"
```

---

### Task 10: Add IAM sub-menus to help context detection

**Files:**
- Modify: `internal/tui/services/help.go`

**Step 1: Add IAM sub-menu views to detectHelpContext**

In the `switch v.(type)` block (after `*ECSServiceSubMenuView`), add:
```go
case *IAMSubMenuView:
	return HelpContextRoot
case *IAMUserSubMenuView:
	return HelpContextRoot
case *IAMRoleSubMenuView:
	return HelpContextRoot
```

**Step 2: Verify**

Run: `go build ./...`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/tui/services/help.go
git commit -m "add IAM sub-menu views to help context detection"
```

---

### Task 11: Full verification

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS.

**Step 2: Build**

Run: `go build -o aws-tui .`
Expected: Compiles.

**Step 3: Manual test (if AWS creds available)**

Run: `./aws-tui services`
- Select IAM → sub-menu appears (Users, Roles, Policies)
- Select Users → list loads with name, user ID, created, path
- Enter on a user → sub-menu (Attached Policies, Groups)
- Select Attached Policies → policy list with name and ARN
- Esc back, select Groups → group list
- Esc back to IAM menu, select Roles → roles list
- Enter on a role → sub-menu (Attached Policies, Trust Policy)
- Select Trust Policy → pretty-printed JSON in scrollable viewport
- Esc back to IAM menu, select Policies → customer-managed policies
- Enter on a policy → attached entities (users/roles/groups)
- c/C copy works on all views
- n/p pagination works on all list views

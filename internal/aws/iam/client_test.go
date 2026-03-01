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
	created1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	created2 := time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC)

	mock := &mockIAMAPI{
		listUsersFunc: func(ctx context.Context, params *awsiam.ListUsersInput, optFns ...func(*awsiam.Options)) (*awsiam.ListUsersOutput, error) {
			return &awsiam.ListUsersOutput{
				Users: []iamtypes.User{
					{
						UserName:   awssdk.String("alice"),
						UserId:     awssdk.String("AIDA1234"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:user/alice"),
						Path:       awssdk.String("/"),
						CreateDate: &created1,
					},
					{
						UserName:   awssdk.String("bob"),
						UserId:     awssdk.String("AIDA5678"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:user/dev/bob"),
						Path:       awssdk.String("/dev/"),
						CreateDate: &created2,
					},
				},
				IsTruncated: false,
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
	if users[0].ARN != "arn:aws:iam::123456789012:user/alice" {
		t.Errorf("ARN = %s, want arn:aws:iam::123456789012:user/alice", users[0].ARN)
	}
	if users[0].Path != "/" {
		t.Errorf("Path = %s, want /", users[0].Path)
	}
	if users[0].UserID != "AIDA1234" {
		t.Errorf("UserID = %s, want AIDA1234", users[0].UserID)
	}
	if !users[0].CreatedAt.Equal(created1) {
		t.Errorf("CreatedAt = %v, want %v", users[0].CreatedAt, created1)
	}

	if users[1].Name != "bob" {
		t.Errorf("Name = %s, want bob", users[1].Name)
	}
	if users[1].Path != "/dev/" {
		t.Errorf("Path = %s, want /dev/", users[1].Path)
	}
}

func TestListRoles(t *testing.T) {
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	mock := &mockIAMAPI{
		listRolesFunc: func(ctx context.Context, params *awsiam.ListRolesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListRolesOutput, error) {
			return &awsiam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName:                 awssdk.String("my-role"),
						RoleId:                   awssdk.String("AROA1234"),
						Arn:                      awssdk.String("arn:aws:iam::123456789012:role/my-role"),
						Path:                     awssdk.String("/"),
						Description:              awssdk.String("A test role"),
						CreateDate:               &created,
						AssumeRolePolicyDocument: awssdk.String("%7B%22Version%22%3A%222012-10-17%22%7D"),
					},
				},
				IsTruncated: false,
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

	if roles[0].Name != "my-role" {
		t.Errorf("Name = %s, want my-role", roles[0].Name)
	}
	if roles[0].RoleID != "AROA1234" {
		t.Errorf("RoleID = %s, want AROA1234", roles[0].RoleID)
	}
	if roles[0].Description != "A test role" {
		t.Errorf("Description = %s, want A test role", roles[0].Description)
	}
	if roles[0].AssumeRolePolicyDocument != `{"Version":"2012-10-17"}` {
		t.Errorf("AssumeRolePolicyDocument = %s, want {\"Version\":\"2012-10-17\"}", roles[0].AssumeRolePolicyDocument)
	}
	if !roles[0].CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", roles[0].CreatedAt, created)
	}
}

func TestListPolicies(t *testing.T) {
	created := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC)

	mock := &mockIAMAPI{
		listPoliciesFunc: func(ctx context.Context, params *awsiam.ListPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListPoliciesOutput, error) {
			if params.Scope != iamtypes.PolicyScopeTypeLocal {
				t.Errorf("Scope = %s, want Local", params.Scope)
			}
			return &awsiam.ListPoliciesOutput{
				Policies: []iamtypes.Policy{
					{
						PolicyName:      awssdk.String("my-policy"),
						PolicyId:        awssdk.String("ANPA1234"),
						Arn:             awssdk.String("arn:aws:iam::123456789012:policy/my-policy"),
						Path:            awssdk.String("/"),
						AttachmentCount: awssdk.Int32(3),
						CreateDate:      &created,
						UpdateDate:      &updated,
					},
				},
				IsTruncated: false,
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

func TestListAttachedUserPolicies(t *testing.T) {
	mock := &mockIAMAPI{
		listAttachedUserPoliciesFunc: func(ctx context.Context, params *awsiam.ListAttachedUserPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedUserPoliciesOutput, error) {
			if awssdk.ToString(params.UserName) != "alice" {
				t.Errorf("UserName = %s, want alice", awssdk.ToString(params.UserName))
			}
			return &awsiam.ListAttachedUserPoliciesOutput{
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{
						PolicyName: awssdk.String("ReadOnly"),
						PolicyArn:  awssdk.String("arn:aws:iam::aws:policy/ReadOnly"),
					},
				},
				IsTruncated: false,
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
	if policies[0].ARN != "arn:aws:iam::aws:policy/ReadOnly" {
		t.Errorf("ARN = %s, want arn:aws:iam::aws:policy/ReadOnly", policies[0].ARN)
	}
}

func TestListGroupsForUser(t *testing.T) {
	mock := &mockIAMAPI{
		listGroupsForUserFunc: func(ctx context.Context, params *awsiam.ListGroupsForUserInput, optFns ...func(*awsiam.Options)) (*awsiam.ListGroupsForUserOutput, error) {
			return &awsiam.ListGroupsForUserOutput{
				Groups: []iamtypes.Group{
					{
						GroupName: awssdk.String("developers"),
						Arn:       awssdk.String("arn:aws:iam::123456789012:group/developers"),
					},
				},
				IsTruncated: false,
			}, nil
		},
	}

	client := NewClient(mock)
	groups, err := client.ListGroupsForUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "developers" {
		t.Errorf("Name = %s, want developers", groups[0].Name)
	}
	if groups[0].ARN != "arn:aws:iam::123456789012:group/developers" {
		t.Errorf("ARN = %s, want arn:aws:iam::123456789012:group/developers", groups[0].ARN)
	}
}

func TestListAttachedRolePolicies(t *testing.T) {
	mock := &mockIAMAPI{
		listAttachedRolePoliciesFunc: func(ctx context.Context, params *awsiam.ListAttachedRolePoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListAttachedRolePoliciesOutput, error) {
			return &awsiam.ListAttachedRolePoliciesOutput{
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{
						PolicyName: awssdk.String("AdminAccess"),
						PolicyArn:  awssdk.String("arn:aws:iam::aws:policy/AdminAccess"),
					},
				},
				IsTruncated: false,
			}, nil
		},
	}

	client := NewClient(mock)
	policies, err := client.ListAttachedRolePolicies(context.Background(), "my-role")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].Name != "AdminAccess" {
		t.Errorf("Name = %s, want AdminAccess", policies[0].Name)
	}
	if policies[0].ARN != "arn:aws:iam::aws:policy/AdminAccess" {
		t.Errorf("ARN = %s, want arn:aws:iam::aws:policy/AdminAccess", policies[0].ARN)
	}
}

func TestListEntitiesForPolicy(t *testing.T) {
	mock := &mockIAMAPI{
		listEntitiesForPolicyFunc: func(ctx context.Context, params *awsiam.ListEntitiesForPolicyInput, optFns ...func(*awsiam.Options)) (*awsiam.ListEntitiesForPolicyOutput, error) {
			return &awsiam.ListEntitiesForPolicyOutput{
				PolicyGroups: []iamtypes.PolicyGroup{
					{GroupName: awssdk.String("admins")},
				},
				PolicyUsers: []iamtypes.PolicyUser{
					{UserName: awssdk.String("alice")},
				},
				PolicyRoles: []iamtypes.PolicyRole{
					{RoleName: awssdk.String("lambda-role")},
				},
				IsTruncated: false,
			}, nil
		},
	}

	client := NewClient(mock)
	entities, err := client.ListEntitiesForPolicy(context.Background(), "arn:aws:iam::123456789012:policy/my-policy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(entities))
	}

	// Verify ordering: Groups, Users, Roles
	if entities[0].Name != "admins" || entities[0].Type != "Group" {
		t.Errorf("entities[0] = {%s, %s}, want {admins, Group}", entities[0].Name, entities[0].Type)
	}
	if entities[1].Name != "alice" || entities[1].Type != "User" {
		t.Errorf("entities[1] = {%s, %s}, want {alice, User}", entities[1].Name, entities[1].Type)
	}
	if entities[2].Name != "lambda-role" || entities[2].Type != "Role" {
		t.Errorf("entities[2] = {%s, %s}, want {lambda-role, Role}", entities[2].Name, entities[2].Type)
	}
}

func TestListUsersPage(t *testing.T) {
	created := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		inMarker       *string
		apiOut         *awsiam.ListUsersOutput
		wantLen        int
		wantUsers      []IAMUser
		wantNextMarker *string
	}{
		{
			name:     "single page no marker",
			inMarker: nil,
			apiOut: &awsiam.ListUsersOutput{
				Users: []iamtypes.User{
					{
						UserName:   awssdk.String("alice"),
						UserId:     awssdk.String("AIDA1234"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:user/alice"),
						Path:       awssdk.String("/"),
						CreateDate: &created,
					},
				},
				IsTruncated: false,
			},
			wantLen: 1,
			wantUsers: []IAMUser{
				{Name: "alice", UserID: "AIDA1234", ARN: "arn:aws:iam::123456789012:user/alice", Path: "/", CreatedAt: created},
			},
			wantNextMarker: nil,
		},
		{
			name:     "first page with more results",
			inMarker: nil,
			apiOut: &awsiam.ListUsersOutput{
				Users: []iamtypes.User{
					{
						UserName:   awssdk.String("bob"),
						UserId:     awssdk.String("AIDA5678"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:user/bob"),
						Path:       awssdk.String("/"),
						CreateDate: &created,
					},
				},
				IsTruncated: true,
				Marker:      awssdk.String("token-page2"),
			},
			wantLen: 1,
			wantUsers: []IAMUser{
				{Name: "bob", UserID: "AIDA5678", ARN: "arn:aws:iam::123456789012:user/bob", Path: "/", CreatedAt: created},
			},
			wantNextMarker: awssdk.String("token-page2"),
		},
		{
			name:     "subsequent page marker in last page",
			inMarker: awssdk.String("token-page2"),
			apiOut: &awsiam.ListUsersOutput{
				Users: []iamtypes.User{
					{
						UserName:   awssdk.String("carol"),
						UserId:     awssdk.String("AIDA9999"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:user/carol"),
						Path:       awssdk.String("/"),
						CreateDate: &created,
					},
				},
				IsTruncated: false,
			},
			wantLen: 1,
			wantUsers: []IAMUser{
				{Name: "carol", UserID: "AIDA9999", ARN: "arn:aws:iam::123456789012:user/carol", Path: "/", CreatedAt: created},
			},
			wantNextMarker: nil,
		},
		{
			name:     "empty results",
			inMarker: nil,
			apiOut: &awsiam.ListUsersOutput{
				Users:       []iamtypes.User{},
				IsTruncated: false,
			},
			wantLen:        0,
			wantUsers:      []IAMUser{},
			wantNextMarker: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				listUsersFunc: func(ctx context.Context, params *awsiam.ListUsersInput, optFns ...func(*awsiam.Options)) (*awsiam.ListUsersOutput, error) {
					if awssdk.ToString(params.Marker) != awssdk.ToString(tt.inMarker) {
						t.Errorf("Marker = %v, want %v", params.Marker, tt.inMarker)
					}
					return tt.apiOut, nil
				},
			}

			client := NewClient(mock)
			users, nextMarker, err := client.ListUsersPage(context.Background(), tt.inMarker)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(users) != tt.wantLen {
				t.Fatalf("len(users) = %d, want %d", len(users), tt.wantLen)
			}
			for i, want := range tt.wantUsers {
				got := users[i]
				if got.Name != want.Name {
					t.Errorf("users[%d].Name = %s, want %s", i, got.Name, want.Name)
				}
				if got.UserID != want.UserID {
					t.Errorf("users[%d].UserID = %s, want %s", i, got.UserID, want.UserID)
				}
				if got.ARN != want.ARN {
					t.Errorf("users[%d].ARN = %s, want %s", i, got.ARN, want.ARN)
				}
				if got.Path != want.Path {
					t.Errorf("users[%d].Path = %s, want %s", i, got.Path, want.Path)
				}
				if !got.CreatedAt.Equal(want.CreatedAt) {
					t.Errorf("users[%d].CreatedAt = %v, want %v", i, got.CreatedAt, want.CreatedAt)
				}
			}
			if awssdk.ToString(nextMarker) != awssdk.ToString(tt.wantNextMarker) {
				t.Errorf("nextMarker = %v, want %v", nextMarker, tt.wantNextMarker)
			}
		})
	}
}

func TestListRolesPage(t *testing.T) {
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		inMarker       *string
		apiOut         *awsiam.ListRolesOutput
		wantLen        int
		wantRoles      []IAMRole
		wantNextMarker *string
	}{
		{
			name:     "single page no marker",
			inMarker: nil,
			apiOut: &awsiam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName:                 awssdk.String("my-role"),
						RoleId:                   awssdk.String("AROA1234"),
						Arn:                      awssdk.String("arn:aws:iam::123456789012:role/my-role"),
						Path:                     awssdk.String("/"),
						Description:              awssdk.String("A test role"),
						CreateDate:               &created,
						AssumeRolePolicyDocument: awssdk.String("%7B%22Version%22%3A%222012-10-17%22%7D"),
					},
				},
				IsTruncated: false,
			},
			wantLen: 1,
			wantRoles: []IAMRole{
				{
					Name:                     "my-role",
					RoleID:                   "AROA1234",
					ARN:                      "arn:aws:iam::123456789012:role/my-role",
					Path:                     "/",
					Description:              "A test role",
					CreatedAt:                created,
					AssumeRolePolicyDocument: `{"Version":"2012-10-17"}`,
				},
			},
			wantNextMarker: nil,
		},
		{
			name:     "first page with more results",
			inMarker: nil,
			apiOut: &awsiam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName:   awssdk.String("role-a"),
						RoleId:     awssdk.String("AROA0001"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:role/role-a"),
						Path:       awssdk.String("/"),
						CreateDate: &created,
					},
				},
				IsTruncated: true,
				Marker:      awssdk.String("token-roles-page2"),
			},
			wantLen: 1,
			wantRoles: []IAMRole{
				{Name: "role-a", RoleID: "AROA0001", ARN: "arn:aws:iam::123456789012:role/role-a", Path: "/", CreatedAt: created},
			},
			wantNextMarker: awssdk.String("token-roles-page2"),
		},
		{
			name:     "subsequent page marker in last page",
			inMarker: awssdk.String("token-roles-page2"),
			apiOut: &awsiam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName:   awssdk.String("role-b"),
						RoleId:     awssdk.String("AROA0002"),
						Arn:        awssdk.String("arn:aws:iam::123456789012:role/role-b"),
						Path:       awssdk.String("/svc/"),
						CreateDate: &created,
					},
				},
				IsTruncated: false,
			},
			wantLen: 1,
			wantRoles: []IAMRole{
				{Name: "role-b", RoleID: "AROA0002", ARN: "arn:aws:iam::123456789012:role/role-b", Path: "/svc/", CreatedAt: created},
			},
			wantNextMarker: nil,
		},
		{
			name:     "empty results",
			inMarker: nil,
			apiOut: &awsiam.ListRolesOutput{
				Roles:       []iamtypes.Role{},
				IsTruncated: false,
			},
			wantLen:        0,
			wantRoles:      []IAMRole{},
			wantNextMarker: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				listRolesFunc: func(ctx context.Context, params *awsiam.ListRolesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListRolesOutput, error) {
					if awssdk.ToString(params.Marker) != awssdk.ToString(tt.inMarker) {
						t.Errorf("Marker = %v, want %v", params.Marker, tt.inMarker)
					}
					return tt.apiOut, nil
				},
			}

			client := NewClient(mock)
			roles, nextMarker, err := client.ListRolesPage(context.Background(), tt.inMarker)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(roles) != tt.wantLen {
				t.Fatalf("len(roles) = %d, want %d", len(roles), tt.wantLen)
			}
			for i, want := range tt.wantRoles {
				got := roles[i]
				if got.Name != want.Name {
					t.Errorf("roles[%d].Name = %s, want %s", i, got.Name, want.Name)
				}
				if got.RoleID != want.RoleID {
					t.Errorf("roles[%d].RoleID = %s, want %s", i, got.RoleID, want.RoleID)
				}
				if got.ARN != want.ARN {
					t.Errorf("roles[%d].ARN = %s, want %s", i, got.ARN, want.ARN)
				}
				if got.Path != want.Path {
					t.Errorf("roles[%d].Path = %s, want %s", i, got.Path, want.Path)
				}
				if got.Description != want.Description {
					t.Errorf("roles[%d].Description = %s, want %s", i, got.Description, want.Description)
				}
				if !got.CreatedAt.Equal(want.CreatedAt) {
					t.Errorf("roles[%d].CreatedAt = %v, want %v", i, got.CreatedAt, want.CreatedAt)
				}
				if want.AssumeRolePolicyDocument != "" && got.AssumeRolePolicyDocument != want.AssumeRolePolicyDocument {
					t.Errorf("roles[%d].AssumeRolePolicyDocument = %s, want %s", i, got.AssumeRolePolicyDocument, want.AssumeRolePolicyDocument)
				}
			}
			if awssdk.ToString(nextMarker) != awssdk.ToString(tt.wantNextMarker) {
				t.Errorf("nextMarker = %v, want %v", nextMarker, tt.wantNextMarker)
			}
		})
	}
}

func TestListPoliciesPage(t *testing.T) {
	created := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		inMarker       *string
		apiOut         *awsiam.ListPoliciesOutput
		wantLen        int
		wantPolicies   []IAMPolicy
		wantNextMarker *string
	}{
		{
			name:     "single page no marker",
			inMarker: nil,
			apiOut: &awsiam.ListPoliciesOutput{
				Policies: []iamtypes.Policy{
					{
						PolicyName:      awssdk.String("my-policy"),
						PolicyId:        awssdk.String("ANPA1234"),
						Arn:             awssdk.String("arn:aws:iam::123456789012:policy/my-policy"),
						Path:            awssdk.String("/"),
						AttachmentCount: awssdk.Int32(3),
						CreateDate:      &created,
						UpdateDate:      &updated,
					},
				},
				IsTruncated: false,
			},
			wantLen: 1,
			wantPolicies: []IAMPolicy{
				{
					Name:            "my-policy",
					PolicyID:        "ANPA1234",
					ARN:             "arn:aws:iam::123456789012:policy/my-policy",
					Path:            "/",
					AttachmentCount: 3,
					CreatedAt:       created,
					UpdatedAt:       updated,
				},
			},
			wantNextMarker: nil,
		},
		{
			name:     "first page with more results",
			inMarker: nil,
			apiOut: &awsiam.ListPoliciesOutput{
				Policies: []iamtypes.Policy{
					{
						PolicyName:      awssdk.String("policy-a"),
						PolicyId:        awssdk.String("ANPA0001"),
						Arn:             awssdk.String("arn:aws:iam::123456789012:policy/policy-a"),
						Path:            awssdk.String("/"),
						AttachmentCount: awssdk.Int32(1),
						CreateDate:      &created,
						UpdateDate:      &updated,
					},
				},
				IsTruncated: true,
				Marker:      awssdk.String("token-policies-page2"),
			},
			wantLen: 1,
			wantPolicies: []IAMPolicy{
				{Name: "policy-a", PolicyID: "ANPA0001", ARN: "arn:aws:iam::123456789012:policy/policy-a", Path: "/", AttachmentCount: 1, CreatedAt: created, UpdatedAt: updated},
			},
			wantNextMarker: awssdk.String("token-policies-page2"),
		},
		{
			name:     "subsequent page marker in last page",
			inMarker: awssdk.String("token-policies-page2"),
			apiOut: &awsiam.ListPoliciesOutput{
				Policies: []iamtypes.Policy{
					{
						PolicyName:      awssdk.String("policy-b"),
						PolicyId:        awssdk.String("ANPA0002"),
						Arn:             awssdk.String("arn:aws:iam::123456789012:policy/policy-b"),
						Path:            awssdk.String("/ops/"),
						AttachmentCount: awssdk.Int32(0),
						CreateDate:      &created,
						UpdateDate:      &updated,
					},
				},
				IsTruncated: false,
			},
			wantLen: 1,
			wantPolicies: []IAMPolicy{
				{Name: "policy-b", PolicyID: "ANPA0002", ARN: "arn:aws:iam::123456789012:policy/policy-b", Path: "/ops/", AttachmentCount: 0, CreatedAt: created, UpdatedAt: updated},
			},
			wantNextMarker: nil,
		},
		{
			name:     "empty results",
			inMarker: nil,
			apiOut: &awsiam.ListPoliciesOutput{
				Policies:    []iamtypes.Policy{},
				IsTruncated: false,
			},
			wantLen:        0,
			wantPolicies:   []IAMPolicy{},
			wantNextMarker: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				listPoliciesFunc: func(ctx context.Context, params *awsiam.ListPoliciesInput, optFns ...func(*awsiam.Options)) (*awsiam.ListPoliciesOutput, error) {
					if params.Scope != iamtypes.PolicyScopeTypeLocal {
						t.Errorf("Scope = %s, want Local", params.Scope)
					}
					if awssdk.ToString(params.Marker) != awssdk.ToString(tt.inMarker) {
						t.Errorf("Marker = %v, want %v", params.Marker, tt.inMarker)
					}
					return tt.apiOut, nil
				},
			}

			client := NewClient(mock)
			policies, nextMarker, err := client.ListPoliciesPage(context.Background(), tt.inMarker)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(policies) != tt.wantLen {
				t.Fatalf("len(policies) = %d, want %d", len(policies), tt.wantLen)
			}
			for i, want := range tt.wantPolicies {
				got := policies[i]
				if got.Name != want.Name {
					t.Errorf("policies[%d].Name = %s, want %s", i, got.Name, want.Name)
				}
				if got.PolicyID != want.PolicyID {
					t.Errorf("policies[%d].PolicyID = %s, want %s", i, got.PolicyID, want.PolicyID)
				}
				if got.ARN != want.ARN {
					t.Errorf("policies[%d].ARN = %s, want %s", i, got.ARN, want.ARN)
				}
				if got.Path != want.Path {
					t.Errorf("policies[%d].Path = %s, want %s", i, got.Path, want.Path)
				}
				if got.AttachmentCount != want.AttachmentCount {
					t.Errorf("policies[%d].AttachmentCount = %d, want %d", i, got.AttachmentCount, want.AttachmentCount)
				}
				if !got.CreatedAt.Equal(want.CreatedAt) {
					t.Errorf("policies[%d].CreatedAt = %v, want %v", i, got.CreatedAt, want.CreatedAt)
				}
				if !got.UpdatedAt.Equal(want.UpdatedAt) {
					t.Errorf("policies[%d].UpdatedAt = %v, want %v", i, got.UpdatedAt, want.UpdatedAt)
				}
			}
			if awssdk.ToString(nextMarker) != awssdk.ToString(tt.wantNextMarker) {
				t.Errorf("nextMarker = %v, want %v", nextMarker, tt.wantNextMarker)
			}
		})
	}
}

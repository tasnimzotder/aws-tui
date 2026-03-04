package iam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsiam "tasnim.dev/aws-tui/internal/aws/iam"
)

// mockClient implements IAMClient for testing.
type mockClient struct {
	users    []awsiam.IAMUser
	roles    []awsiam.IAMRole
	policies []awsiam.IAMPolicy
	err      error
}

func (m *mockClient) ListUsers(ctx context.Context) ([]awsiam.IAMUser, error) {
	return m.users, m.err
}

func (m *mockClient) ListRoles(ctx context.Context) ([]awsiam.IAMRole, error) {
	return m.roles, m.err
}

func (m *mockClient) ListPolicies(ctx context.Context) ([]awsiam.IAMPolicy, error) {
	return m.policies, m.err
}

func (m *mockClient) ListAttachedUserPolicies(ctx context.Context, userName string) ([]awsiam.IAMAttachedPolicy, error) {
	return nil, nil
}

func (m *mockClient) ListGroupsForUser(ctx context.Context, userName string) ([]awsiam.IAMGroup, error) {
	return nil, nil
}

func (m *mockClient) ListAttachedRolePolicies(ctx context.Context, roleName string) ([]awsiam.IAMAttachedPolicy, error) {
	return nil, nil
}

func (m *mockClient) ListEntitiesForPolicy(ctx context.Context, policyARN string) ([]awsiam.IAMPolicyEntity, error) {
	return nil, nil
}

func (m *mockClient) GetPolicyDocument(ctx context.Context, policyARN, versionID string) (string, error) {
	return "", nil
}

func (m *mockClient) ListInlineUserPolicies(ctx context.Context, userName string) ([]awsiam.IAMInlinePolicy, error) {
	return nil, nil
}

func (m *mockClient) ListInlineRolePolicies(ctx context.Context, roleName string) ([]awsiam.IAMInlinePolicy, error) {
	return nil, nil
}

func TestPlugin_Metadata(t *testing.T) {
	p := NewPlugin(&mockClient{})

	assert.Equal(t, "iam", p.ID())
	assert.Equal(t, "IAM", p.Name())
	assert.NotEmpty(t, p.Icon())
}

func TestPlugin_Summary(t *testing.T) {
	tests := []struct {
		name      string
		users     []awsiam.IAMUser
		err       error
		wantTotal int
		wantErr   bool
	}{
		{
			name:      "no users",
			users:     nil,
			wantTotal: 0,
		},
		{
			name: "three users",
			users: []awsiam.IAMUser{
				{Name: "alice"},
				{Name: "bob"},
				{Name: "carol"},
			},
			wantTotal: 3,
		},
		{
			name:    "error",
			err:     assert.AnError,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlugin(&mockClient{users: tt.users, err: tt.err})
			summary, err := p.Summary(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, summary.Total)
			assert.Equal(t, "users", summary.Label)
		})
	}
}

func TestPlugin_Commands(t *testing.T) {
	p := NewPlugin(&mockClient{})
	cmds := p.Commands()

	require.Len(t, cmds, 1)
	assert.Equal(t, "IAM", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "iam")
	assert.Contains(t, cmds[0].Keywords, "users")
	assert.Contains(t, cmds[0].Keywords, "roles")
	assert.Contains(t, cmds[0].Keywords, "policies")
}

func TestPlugin_PollConfig(t *testing.T) {
	p := NewPlugin(&mockClient{})
	cfg := p.PollConfig()

	assert.Equal(t, 10*time.Minute, cfg.IdleInterval)
	assert.Equal(t, time.Duration(0), cfg.ActiveInterval)
	assert.False(t, cfg.IsActive())
}

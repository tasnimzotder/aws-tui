package ecr

import (
	"context"
	"testing"
	"time"

	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	"tasnim.dev/aws-tui/internal/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockECRClient implements ECRClient for testing.
type mockECRClient struct {
	repos  []awsecr.ECRRepo
	images map[string][]awsecr.ECRImage
	err    error
}

func (m *mockECRClient) ListRepositories(ctx context.Context) ([]awsecr.ECRRepo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.repos, nil
}

func (m *mockECRClient) ListImages(ctx context.Context, repoName string) ([]awsecr.ECRImage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.images[repoName], nil
}

func TestPluginMetadata(t *testing.T) {
	p := NewPlugin(nil)
	assert.Equal(t, "ecr", p.ID())
	assert.Equal(t, "ECR", p.Name())
	assert.NotEmpty(t, p.Icon())
}

func TestSummary_WithRepos(t *testing.T) {
	client := &mockECRClient{
		repos: []awsecr.ECRRepo{
			{Name: "app-web", ImageCount: 10},
			{Name: "app-api", ImageCount: 5},
			{Name: "app-worker", ImageCount: 3},
		},
	}
	p := NewPlugin(client)
	summary, err := p.Summary(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 18, summary.Status["images"])
	assert.Equal(t, plugin.HealthHealthy, summary.Health)
	assert.Equal(t, "repositories", summary.Label)
}

func TestSummary_Empty(t *testing.T) {
	client := &mockECRClient{repos: nil}
	p := NewPlugin(client)
	summary, err := p.Summary(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, summary.Total)
	assert.Equal(t, 0, summary.Status["images"])
	assert.Equal(t, plugin.HealthHealthy, summary.Health)
}

func TestSummary_Error(t *testing.T) {
	client := &mockECRClient{err: assert.AnError}
	p := NewPlugin(client)
	_, err := p.Summary(context.Background())

	assert.Error(t, err)
}

func TestCommands(t *testing.T) {
	p := NewPlugin(nil)
	cmds := p.Commands()

	require.Len(t, cmds, 1)
	assert.Equal(t, "ECR Repositories", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "ecr")
	assert.Contains(t, cmds[0].Keywords, "repositories")
	assert.Contains(t, cmds[0].Keywords, "images")
	assert.Contains(t, cmds[0].Keywords, "containers")
}

func TestPollConfig(t *testing.T) {
	p := NewPlugin(nil)
	cfg := p.PollConfig()

	assert.Equal(t, 2*time.Minute, cfg.IdleInterval)
	assert.Equal(t, time.Duration(0), cfg.ActiveInterval)
	assert.Nil(t, cfg.IsActive)
}

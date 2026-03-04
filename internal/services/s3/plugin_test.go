package s3

import (
	"context"
	"testing"
	"time"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements S3Client for testing.
type mockClient struct {
	buckets []awss3.S3Bucket
	objects awss3.ListObjectsResult
	content []byte
	err     error
}

func (m *mockClient) ListBuckets(ctx context.Context) ([]awss3.S3Bucket, error) {
	return m.buckets, m.err
}

func (m *mockClient) ListObjects(ctx context.Context, bucket, prefix, token, region string) (awss3.ListObjectsResult, error) {
	return m.objects, m.err
}

func (m *mockClient) GetObject(ctx context.Context, bucket, key, region string) ([]byte, error) {
	return m.content, m.err
}

func TestPluginMetadata(t *testing.T) {
	p := NewPlugin(&mockClient{})
	assert.Equal(t, "s3", p.ID())
	assert.Equal(t, "S3", p.Name())
	// Icon may be empty; just verify it returns without panic
	_ = p.Icon()
}

func TestSummary(t *testing.T) {
	t.Run("returns bucket count", func(t *testing.T) {
		client := &mockClient{
			buckets: []awss3.S3Bucket{
				{Name: "bucket-1", Region: "us-east-1", CreatedAt: time.Now()},
				{Name: "bucket-2", Region: "us-west-2", CreatedAt: time.Now()},
				{Name: "bucket-3", Region: "us-east-1", CreatedAt: time.Now()},
			},
		}
		p := NewPlugin(client)
		summary, err := p.Summary(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 3, summary.Total)
		assert.Equal(t, 2, summary.Status["us-east-1"])
		assert.Equal(t, 1, summary.Status["us-west-2"])
		assert.Equal(t, "3 buckets", summary.Label)
	})

	t.Run("empty returns unknown health", func(t *testing.T) {
		client := &mockClient{buckets: []awss3.S3Bucket{}}
		p := NewPlugin(client)
		summary, err := p.Summary(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, summary.Total)
		assert.Equal(t, plugin.HealthUnknown, summary.Health)
	})

	t.Run("propagates error", func(t *testing.T) {
		client := &mockClient{err: assert.AnError}
		p := NewPlugin(client)
		_, err := p.Summary(context.Background())
		assert.Error(t, err)
	})
}

func TestCommands(t *testing.T) {
	p := NewPlugin(&mockClient{})
	cmds := p.Commands()
	require.Len(t, cmds, 1)
	assert.Equal(t, "S3 Buckets", cmds[0].Title)
	assert.Contains(t, cmds[0].Keywords, "s3")
	assert.Contains(t, cmds[0].Keywords, "buckets")
	assert.Contains(t, cmds[0].Keywords, "storage")
}

func TestPollConfig(t *testing.T) {
	p := NewPlugin(&mockClient{})
	cfg := p.PollConfig()
	assert.Equal(t, 5*time.Minute, cfg.IdleInterval)
	assert.Equal(t, time.Duration(0), cfg.ActiveInterval)
	assert.False(t, cfg.IsActive())
}

func TestParentPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a/", ""},
		{"a/b/", "a/"},
		{"a/b/c/", "a/b/"},
		{"foo/bar/baz/", "foo/bar/"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parentPrefix(tt.input))
		})
	}
}

func TestIsTextContent(t *testing.T) {
	assert.True(t, isTextContent([]byte("hello world")))
	assert.True(t, isTextContent([]byte("{\"key\": \"value\"}")))
	assert.True(t, isTextContent([]byte("")))
	assert.False(t, isTextContent([]byte{0x00, 0x01, 0x02}))
	assert.False(t, isTextContent([]byte{0xFF, 0xFE}))
}

func TestFormatSize(t *testing.T) {
	assert.Equal(t, "0 B", formatSize(0))
	assert.Equal(t, "512 B", formatSize(512))
	assert.Equal(t, "1.0 KB", formatSize(1024))
	assert.Equal(t, "1.5 MB", formatSize(1572864))
	assert.Equal(t, "2.0 GB", formatSize(2147483648))
}


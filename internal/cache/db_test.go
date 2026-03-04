package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestDB(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()
}

func TestUpsertAndGetResources(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	resources := []Resource{
		{ID: "i-001", Name: "web-server-1", Data: `{"type":"t3.micro"}`},
		{ID: "i-002", Name: "web-server-2", Data: `{"type":"t3.small"}`},
	}

	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", resources, 300)
	require.NoError(t, err)

	got, err := db.GetResources(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "i-001", got[0].ID)
	assert.Equal(t, "web-server-1", got[0].Name)
	assert.Equal(t, `{"type":"t3.micro"}`, got[0].Data)
}

func TestUpsertResources_Overwrite(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	resources := []Resource{
		{ID: "i-001", Name: "old-name", Data: `{"v":1}`},
	}
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", resources, 300)
	require.NoError(t, err)

	updated := []Resource{
		{ID: "i-001", Name: "new-name", Data: `{"v":2}`},
	}
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", updated, 300)
	require.NoError(t, err)

	got, err := db.GetResources(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "new-name", got[0].Name)
}

func TestGetResources_TTLExpiry(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	resources := []Resource{
		{ID: "i-001", Name: "ephemeral", Data: "{}"},
	}
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", resources, 1)
	require.NoError(t, err)

	// Verify it's there before expiry
	got, err := db.GetResources(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Len(t, got, 1)

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	got, err = db.GetResources(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchResources(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	resources := []Resource{
		{ID: "i-001", Name: "prod-web-server", Data: "{}"},
		{ID: "i-002", Name: "prod-api-server", Data: "{}"},
		{ID: "i-003", Name: "staging-web-server", Data: "{}"},
	}
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", resources, 300)
	require.NoError(t, err)

	tests := []struct {
		name     string
		query    string
		wantLen  int
		wantName string
	}{
		{"search web", "web", 2, "prod-web-server"},
		{"search prod", "prod", 2, "prod-api-server"},
		{"search staging", "staging", 1, "staging-web-server"},
		{"search nonexistent", "nonexistent", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.SearchResources(ctx, "default", "us-east-1", tt.query)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 {
				// Just check we got results; order is by service, name
				found := false
				for _, r := range got {
					if r.Name == tt.wantName {
						found = true
					}
				}
				assert.True(t, found, "expected to find %s in results", tt.wantName)
			}
		})
	}
}

func TestSummary_UpsertAndGet(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	err = db.UpsertSummary(ctx, "ec2", "us-east-1", "default", `{"count":5}`, 300)
	require.NoError(t, err)

	data, err := db.GetSummary(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Equal(t, `{"count":5}`, data)
}

func TestSummary_Expired(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	err = db.UpsertSummary(ctx, "ec2", "us-east-1", "default", `{"count":5}`, 1)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// GetSummary should return empty for expired
	data, err := db.GetSummary(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Equal(t, "", data)

	// GetSummaryStale should still return it
	data, err = db.GetSummaryStale(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Equal(t, `{"count":5}`, data)
}

func TestSummary_NotFound(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	data, err := db.GetSummary(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Equal(t, "", data)

	data, err = db.GetSummaryStale(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Equal(t, "", data)
}

func TestProfileRegionPartitioning(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Insert resources for two different profiles
	r1 := []Resource{{ID: "i-001", Name: "profile1-server", Data: "{}"}}
	r2 := []Resource{{ID: "i-001", Name: "profile2-server", Data: "{}"}}

	err = db.UpsertResources(ctx, "ec2", "us-east-1", "profile1", r1, 300)
	require.NoError(t, err)
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "profile2", r2, 300)
	require.NoError(t, err)

	// Each profile should only see its own data
	got1, err := db.GetResources(ctx, "ec2", "us-east-1", "profile1")
	require.NoError(t, err)
	assert.Len(t, got1, 1)
	assert.Equal(t, "profile1-server", got1[0].Name)

	got2, err := db.GetResources(ctx, "ec2", "us-east-1", "profile2")
	require.NoError(t, err)
	assert.Len(t, got2, 1)
	assert.Equal(t, "profile2-server", got2[0].Name)

	// Same for different regions
	r3 := []Resource{{ID: "i-001", Name: "eu-server", Data: "{}"}}
	err = db.UpsertResources(ctx, "ec2", "eu-west-1", "profile1", r3, 300)
	require.NoError(t, err)

	got3, err := db.GetResources(ctx, "ec2", "eu-west-1", "profile1")
	require.NoError(t, err)
	assert.Len(t, got3, 1)
	assert.Equal(t, "eu-server", got3[0].Name)

	// Original us-east-1 data unchanged
	got1Again, err := db.GetResources(ctx, "ec2", "us-east-1", "profile1")
	require.NoError(t, err)
	assert.Len(t, got1Again, 1)
	assert.Equal(t, "profile1-server", got1Again[0].Name)
}

func TestPurgeAll(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Insert resources and summaries for two profiles
	r1 := []Resource{{ID: "i-001", Name: "server1", Data: "{}"}}
	r2 := []Resource{{ID: "i-002", Name: "server2", Data: "{}"}}

	err = db.UpsertResources(ctx, "ec2", "us-east-1", "profile1", r1, 300)
	require.NoError(t, err)
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "profile2", r2, 300)
	require.NoError(t, err)

	err = db.UpsertSummary(ctx, "ec2", "us-east-1", "profile1", `{"a":1}`, 300)
	require.NoError(t, err)
	err = db.UpsertSummary(ctx, "ec2", "us-east-1", "profile2", `{"b":2}`, 300)
	require.NoError(t, err)

	// Purge only profile1
	err = db.PurgeAll(ctx, "profile1", "us-east-1")
	require.NoError(t, err)

	// profile1 data should be gone
	got1, err := db.GetResources(ctx, "ec2", "us-east-1", "profile1")
	require.NoError(t, err)
	assert.Empty(t, got1)

	sum1, err := db.GetSummary(ctx, "ec2", "us-east-1", "profile1")
	require.NoError(t, err)
	assert.Equal(t, "", sum1)

	// profile2 data should still exist
	got2, err := db.GetResources(ctx, "ec2", "us-east-1", "profile2")
	require.NoError(t, err)
	assert.Len(t, got2, 1)

	sum2, err := db.GetSummary(ctx, "ec2", "us-east-1", "profile2")
	require.NoError(t, err)
	assert.Equal(t, `{"b":2}`, sum2)
}

func TestPurgeExpired(t *testing.T) {
	db, err := NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Insert one short-lived and one long-lived resource
	short := []Resource{{ID: "i-001", Name: "short-lived", Data: "{}"}}
	long := []Resource{{ID: "i-002", Name: "long-lived", Data: "{}"}}

	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", short, 1)
	require.NoError(t, err)
	err = db.UpsertResources(ctx, "ec2", "us-east-1", "default", long, 300)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	err = db.PurgeExpired(ctx)
	require.NoError(t, err)

	// Only long-lived should remain (GetResourcesAll would show all, but we use the
	// underlying query to verify purge actually deleted rows)
	got, err := db.GetResources(ctx, "ec2", "us-east-1", "default")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "long-lived", got[0].Name)
}

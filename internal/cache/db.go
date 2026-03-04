package cache

import (
	"context"
	"database/sql"
	_ "embed"
	"time"

	sqlcgen "tasnim.dev/aws-tui/internal/cache/generated"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Resource is a high-level representation of a cached resource.
type Resource struct {
	ID   string
	Name string
	Data string
}

// DB wraps a sql.DB and sqlc-generated Queries with a higher-level API.
type DB struct {
	conn    *sql.DB
	queries *sqlcgen.Queries
}

// New opens a SQLite database at path with WAL mode and runs schema migration.
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, err
	}

	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{
		conn:    conn,
		queries: sqlcgen.New(conn),
	}, nil
}

// NewTestDB creates an in-memory SQLite database for testing.
func NewTestDB() (*DB, error) {
	return New(":memory:")
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// UpsertResources inserts or updates a batch of resources within a transaction.
func (db *DB) UpsertResources(ctx context.Context, service, region, profile string, resources []Resource, ttlSeconds int) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := db.queries.WithTx(tx)
	now := time.Now().Unix()

	for _, r := range resources {
		if err := q.UpsertResource(ctx, sqlcgen.UpsertResourceParams{
			Service:    service,
			ResourceID: r.ID,
			Region:     region,
			Profile:    profile,
			Name:       r.Name,
			Data:       r.Data,
			FetchedAt:  now,
			TtlSeconds: int64(ttlSeconds),
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetResources returns non-expired resources for the given service, region, and profile.
func (db *DB) GetResources(ctx context.Context, service, region, profile string) ([]Resource, error) {
	rows, err := db.queries.GetResources(ctx, sqlcgen.GetResourcesParams{
		Service: service,
		Region:  region,
		Profile: profile,
	})
	if err != nil {
		return nil, err
	}

	result := make([]Resource, 0, len(rows))
	for _, r := range rows {
		result = append(result, Resource{
			ID:   r.ResourceID,
			Name: r.Name,
			Data: r.Data,
		})
	}
	return result, nil
}

// SearchResources performs a LIKE search on resource names.
func (db *DB) SearchResources(ctx context.Context, profile, region, query string) ([]Resource, error) {
	rows, err := db.queries.SearchResources(ctx, sqlcgen.SearchResourcesParams{
		Profile: profile,
		Region:  region,
		Column3: sql.NullString{String: query, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	result := make([]Resource, 0, len(rows))
	for _, r := range rows {
		result = append(result, Resource{
			ID:   r.ResourceID,
			Name: r.Name,
			Data: r.Data,
		})
	}
	return result, nil
}

// UpsertSummary inserts or updates a summary.
func (db *DB) UpsertSummary(ctx context.Context, service, region, profile, data string, ttlSeconds int) error {
	return db.queries.UpsertSummary(ctx, sqlcgen.UpsertSummaryParams{
		Service:    service,
		Region:     region,
		Profile:    profile,
		Data:       data,
		FetchedAt:  time.Now().Unix(),
		TtlSeconds: int64(ttlSeconds),
	})
}

// GetSummary returns the summary data if it exists and is not expired.
// Returns "" if not found or expired.
func (db *DB) GetSummary(ctx context.Context, service, region, profile string) (string, error) {
	s, err := db.queries.GetSummary(ctx, sqlcgen.GetSummaryParams{
		Service: service,
		Region:  region,
		Profile: profile,
	})
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Data, nil
}

// GetSummaryStale returns the summary data even if expired.
// Returns "" if not found.
func (db *DB) GetSummaryStale(ctx context.Context, service, region, profile string) (string, error) {
	s, err := db.queries.GetSummaryStale(ctx, sqlcgen.GetSummaryStaleParams{
		Service: service,
		Region:  region,
		Profile: profile,
	})
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Data, nil
}

// PurgeExpired deletes all expired resources.
func (db *DB) PurgeExpired(ctx context.Context) error {
	return db.queries.PurgeExpired(ctx)
}

// PurgeAll deletes all resources and summaries for a given profile and region.
func (db *DB) PurgeAll(ctx context.Context, profile, region string) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := db.queries.WithTx(tx)

	if err := q.PurgeAll(ctx, sqlcgen.PurgeAllParams{
		Profile: profile,
		Region:  region,
	}); err != nil {
		return err
	}

	if err := q.PurgeSummaries(ctx, sqlcgen.PurgeSummariesParams{
		Profile: profile,
		Region:  region,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

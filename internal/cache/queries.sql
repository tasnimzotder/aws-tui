-- name: GetResources :many
SELECT * FROM resources
WHERE service = ? AND region = ? AND profile = ?
AND (fetched_at + ttl_seconds) > unixepoch();

-- name: GetResourcesAll :many
SELECT * FROM resources
WHERE service = ? AND region = ? AND profile = ?;

-- name: UpsertResource :exec
INSERT INTO resources (service, resource_id, region, profile, name, data, fetched_at, ttl_seconds)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (service, resource_id, region, profile)
DO UPDATE SET name = excluded.name, data = excluded.data,
              fetched_at = excluded.fetched_at, ttl_seconds = excluded.ttl_seconds;

-- name: DeleteResourcesForService :exec
DELETE FROM resources WHERE service = ? AND region = ? AND profile = ?;

-- name: GetSummary :one
SELECT * FROM summaries
WHERE service = ? AND region = ? AND profile = ?
AND (fetched_at + ttl_seconds) > unixepoch();

-- name: GetSummaryStale :one
SELECT * FROM summaries
WHERE service = ? AND region = ? AND profile = ?;

-- name: UpsertSummary :exec
INSERT INTO summaries (service, region, profile, data, fetched_at, ttl_seconds)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (service, region, profile)
DO UPDATE SET data = excluded.data, fetched_at = excluded.fetched_at,
              ttl_seconds = excluded.ttl_seconds;

-- name: SearchResources :many
SELECT * FROM resources
WHERE profile = ? AND region = ? AND name LIKE '%' || ? || '%'
ORDER BY service, name
LIMIT 50;

-- name: PurgeExpired :exec
DELETE FROM resources WHERE (fetched_at + ttl_seconds) < unixepoch();

-- name: PurgeAll :exec
DELETE FROM resources WHERE profile = ? AND region = ?;

-- name: PurgeSummaries :exec
DELETE FROM summaries WHERE profile = ? AND region = ?;

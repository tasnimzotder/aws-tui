# S3 Service Support — Design

## Goal

Add S3 browsing to the services TUI: list all buckets globally, drill into a bucket to navigate prefixes (folders) and objects, with server-side pagination for object listings.

## Navigation Flow

```
Services → S3 → Buckets List → Objects/Prefixes List
                                     ↓ Enter on prefix
                               Deeper prefix level
                                     ↓ L to load more
                               Append next 1000 objects
```

## Buckets View

Columns: **Name**, **Region**, **Created**

- `ListBuckets` returns all buckets globally (not region-scoped)
- Bucket region resolved via `GetBucketLocation` — fetched concurrently (bounded ~10 goroutines) to avoid N+1 slowness
- Client-side pagination (bucket counts are typically manageable)
- Copy: `c` = bucket name, `C` = `arn:aws:s3:::bucket-name`

## Objects View

Columns: **Name**, **Size**, **Last Modified**, **Storage Class**

- Uses `ListObjectsV2` with `Delimiter=/` to get common prefixes and objects at the current level
- Prefixes shown first with a folder indicator (`📁`), then objects
- **Enter** on a prefix drills deeper (pushes new view with that prefix)
- **Esc** goes back up one level
- Copy: `c` = `s3://bucket/key`, `C` = `arn:aws:s3:::bucket/key`

### Server-Side Pagination

- Fetches 1000 items per request (S3 default page size)
- Press `L` to load next page and append to the list
- Status shows "Showing X of X+" when more pages exist
- Standard `n`/`p` client-side pagination applies to loaded items

### Implementation: Extend TableView with LoadMoreFunc

Add an optional `LoadMoreFunc` to `TableViewConfig`:
- When set, `L` key triggers it to fetch and append more items
- Returns new items + whether more pages remain
- Keeps S3 consistent with all other service views
- No impact on existing views (field is optional)

## New Files

| File | Purpose |
|------|---------|
| `internal/aws/s3/types.go` | `S3Bucket`, `S3Object` structs |
| `internal/aws/s3/client.go` | `Client` with `ListBuckets()`, `ListObjects(bucket, prefix, token)` |
| `internal/aws/s3/client_test.go` | Tests with mock S3 API |
| `internal/tui/services/s3.go` | `NewS3BucketsView`, `NewS3ObjectsView` |

## Modified Files

| File | Change |
|------|--------|
| `internal/aws/client.go` | Add `S3 *awss3.Client` to `ServiceClient` |
| `internal/tui/services/root.go` | Add "S3" to service menu + `handleSelection` |
| `internal/tui/services/tableview.go` | Add optional `LoadMoreFunc`, `L` key handler |
| `internal/tui/services/tableview_test.go` | Tests for load-more behavior |
| `go.mod` | Add `github.com/aws/aws-sdk-go-v2/service/s3` |

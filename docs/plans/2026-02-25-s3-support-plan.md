# S3 Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add S3 bucket and object browsing to the services TUI with prefix-based folder navigation and server-side pagination for objects.

**Architecture:** New S3 client (`internal/aws/s3/`) follows the same pattern as ECR — interface + mock-based tests. Buckets list uses standard `TableView`. Objects list uses `TableView` extended with a new `LoadMoreFunc` for server-side pagination (press `L` to load next 1000). Bucket regions fetched concurrently via `GetBucketLocation`.

**Tech Stack:** Go 1.26, aws-sdk-go-v2/service/s3, Bubble Tea, existing `TableView[T]` abstraction.

**Design doc:** `docs/plans/2026-02-25-s3-support-design.md`

---

### Task 1: Add S3 SDK dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

Run: `go get github.com/aws/aws-sdk-go-v2/service/s3`

**Step 2: Verify**

Run: `go build ./...`
Expected: Compiles with no errors.

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "add aws-sdk-go-v2/service/s3 dependency"
```

---

### Task 2: S3 types

**Files:**
- Create: `internal/aws/s3/types.go`

**Step 1: Create the types file**

```go
package s3

import "time"

type S3Bucket struct {
	Name      string
	Region    string
	CreatedAt time.Time
}

type S3Object struct {
	Key          string
	Size         int64
	LastModified time.Time
	StorageClass string
	IsPrefix     bool // true for common prefixes (folders)
}

type ListObjectsResult struct {
	Objects       []S3Object
	NextToken     string // empty = no more pages
	TotalLoaded   int    // running total across all pages
}
```

**Step 2: Verify**

Run: `go build ./internal/aws/s3/`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/aws/s3/types.go
git commit -m "add S3 types for buckets and objects"
```

---

### Task 3: S3 client — ListBuckets (TDD)

**Files:**
- Create: `internal/aws/s3/client.go`
- Create: `internal/aws/s3/client_test.go`

**Step 1: Write the failing test**

```go
// internal/aws/s3/client_test.go
package s3

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type mockS3API struct {
	listBucketsFunc       func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error)
	getBucketLocationFunc func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error)
	listObjectsV2Func    func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error)
}

func (m *mockS3API) ListBuckets(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
	return m.listBucketsFunc(ctx, params, optFns...)
}
func (m *mockS3API) GetBucketLocation(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
	return m.getBucketLocationFunc(ctx, params, optFns...)
}
func (m *mockS3API) ListObjectsV2(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
	return m.listObjectsV2Func(ctx, params, optFns...)
}

func TestListBuckets(t *testing.T) {
	created := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	mock := &mockS3API{
		listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
			return &awss3.ListBucketsOutput{
				Buckets: []s3types.Bucket{
					{Name: awssdk.String("my-bucket"), CreationDate: &created},
					{Name: awssdk.String("other-bucket"), CreationDate: &created},
				},
			}, nil
		},
		getBucketLocationFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
			region := s3types.BucketLocationConstraintUsWest2
			if awssdk.ToString(params.Bucket) == "my-bucket" {
				region = s3types.BucketLocationConstraintUsEast2
			}
			return &awss3.GetBucketLocationOutput{
				LocationConstraint: region,
			}, nil
		},
	}

	client := NewClient(mock)
	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Name != "my-bucket" {
		t.Errorf("Name = %s, want my-bucket", buckets[0].Name)
	}
	if buckets[0].Region != "us-east-2" {
		t.Errorf("Region = %s, want us-east-2", buckets[0].Region)
	}
	if buckets[1].Region != "us-west-2" {
		t.Errorf("Region = %s, want us-west-2", buckets[1].Region)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/aws/s3/ -run TestListBuckets -v`
Expected: FAIL — `NewClient` not defined.

**Step 3: Write the client**

```go
// internal/aws/s3/client.go
package s3

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3API interface {
	ListBuckets(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error)
	GetBucketLocation(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error)
	ListObjectsV2(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error)
}

type Client struct {
	api S3API
}

func NewClient(api S3API) *Client {
	return &Client{api: api}
}

func (c *Client) ListBuckets(ctx context.Context) ([]S3Bucket, error) {
	out, err := c.api.ListBuckets(ctx, &awss3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("ListBuckets: %w", err)
	}

	buckets := make([]S3Bucket, len(out.Buckets))
	for i, b := range out.Buckets {
		var createdAt time.Time
		if b.CreationDate != nil {
			createdAt = *b.CreationDate
		}
		buckets[i] = S3Bucket{
			Name:      aws.ToString(b.Name),
			CreatedAt: createdAt,
		}
	}

	// Fetch regions concurrently (bounded to 10)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	for i := range buckets {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			loc, err := c.api.GetBucketLocation(ctx, &awss3.GetBucketLocationInput{
				Bucket: aws.String(buckets[idx].Name),
			})
			if err != nil {
				return
			}
			region := string(loc.LocationConstraint)
			if region == "" {
				region = "us-east-1" // AWS returns empty for us-east-1
			}
			buckets[idx].Region = region
		}(i)
	}
	wg.Wait()

	return buckets, nil
}
```

Note: add `"time"` to the imports in client.go.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/aws/s3/ -run TestListBuckets -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/aws/s3/client.go internal/aws/s3/client_test.go
git commit -m "add S3 client with ListBuckets and region lookup"
```

---

### Task 4: S3 client — ListObjects (TDD)

**Files:**
- Modify: `internal/aws/s3/client.go`
- Modify: `internal/aws/s3/client_test.go`

**Step 1: Write the failing test**

```go
func TestListObjects(t *testing.T) {
	lastMod := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	mock := &mockS3API{
		listObjectsV2Func: func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
			return &awss3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{
					{Prefix: awssdk.String("logs/")},
				},
				Contents: []s3types.Object{
					{
						Key:          awssdk.String("config.json"),
						Size:         awssdk.Int64(2048),
						LastModified: &lastMod,
						StorageClass: s3types.ObjectStorageClassStandard,
					},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}

	client := NewClient(mock)
	result, err := client.ListObjects(context.Background(), "my-bucket", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Objects) != 2 {
		t.Fatalf("expected 2 objects (1 prefix + 1 object), got %d", len(result.Objects))
	}
	// Prefixes come first
	if !result.Objects[0].IsPrefix || result.Objects[0].Key != "logs/" {
		t.Errorf("first item should be prefix 'logs/', got %+v", result.Objects[0])
	}
	if result.Objects[1].IsPrefix || result.Objects[1].Key != "config.json" {
		t.Errorf("second item should be object 'config.json', got %+v", result.Objects[1])
	}
	if result.Objects[1].Size != 2048 {
		t.Errorf("Size = %d, want 2048", result.Objects[1].Size)
	}
	if result.NextToken != "" {
		t.Errorf("NextToken should be empty, got %q", result.NextToken)
	}
}

func TestListObjects_Pagination(t *testing.T) {
	mock := &mockS3API{
		listObjectsV2Func: func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
			return &awss3.ListObjectsV2Output{
				Contents: []s3types.Object{
					{Key: awssdk.String("file1.txt"), Size: awssdk.Int64(100), StorageClass: s3types.ObjectStorageClassStandard},
				},
				IsTruncated:           awssdk.Bool(true),
				NextContinuationToken: awssdk.String("token-abc"),
			}, nil
		},
	}

	client := NewClient(mock)
	result, err := client.ListObjects(context.Background(), "my-bucket", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextToken != "token-abc" {
		t.Errorf("NextToken = %q, want token-abc", result.NextToken)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/aws/s3/ -run TestListObjects -v`
Expected: FAIL — `ListObjects` not defined.

**Step 3: Write ListObjects**

Add to `internal/aws/s3/client.go`:

```go
func (c *Client) ListObjects(ctx context.Context, bucket, prefix, continuationToken string) (ListObjectsResult, error) {
	input := &awss3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1000),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if continuationToken != "" {
		input.ContinuationToken = aws.String(continuationToken)
	}

	out, err := c.api.ListObjectsV2(ctx, input)
	if err != nil {
		return ListObjectsResult{}, fmt.Errorf("ListObjectsV2: %w", err)
	}

	var objects []S3Object

	// Common prefixes (folders) first
	for _, p := range out.CommonPrefixes {
		objects = append(objects, S3Object{
			Key:      aws.ToString(p.Prefix),
			IsPrefix: true,
		})
	}

	// Then actual objects
	for _, obj := range out.Contents {
		var lastMod time.Time
		if obj.LastModified != nil {
			lastMod = *obj.LastModified
		}
		objects = append(objects, S3Object{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: lastMod,
			StorageClass: string(obj.StorageClass),
		})
	}

	var nextToken string
	if aws.ToBool(out.IsTruncated) && out.NextContinuationToken != nil {
		nextToken = *out.NextContinuationToken
	}

	return ListObjectsResult{
		Objects:   objects,
		NextToken: nextToken,
	}, nil
}
```

Note: `aws.ToInt64` may not exist — use `aws.ToInt64` or dereference manually:
```go
var size int64
if obj.Size != nil {
    size = *obj.Size
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/aws/s3/ -run TestListObjects -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/aws/s3/client.go internal/aws/s3/client_test.go
git commit -m "add S3 ListObjects with prefix navigation and pagination"
```

---

### Task 5: Wire S3 client into ServiceClient

**Files:**
- Modify: `internal/aws/client.go:23-50`

**Step 1: Add S3 to ServiceClient**

Add import:
```go
awss3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
awss3 "tasnim.dev/aws-tui/internal/aws/s3"
```

Add field to `ServiceClient` struct:
```go
S3 *awss3.Client
```

Add to `NewServiceClient` return:
```go
S3: awss3.NewClient(awss3sdk.NewFromConfig(cfg)),
```

**Step 2: Verify**

Run: `go build ./...`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/aws/client.go
git commit -m "wire S3 client into ServiceClient"
```

---

### Task 6: Extend TableView with LoadMoreFunc (TDD)

**Files:**
- Modify: `internal/tui/services/tableview.go:22-34` (config), `145-201` (Update)
- Modify: `internal/tui/services/tableview_test.go`

**Step 1: Write the failing test**

Add to `internal/tui/services/tableview_test.go`:

```go
func TestLoadMore(t *testing.T) {
	loadCalls := 0
	tv := NewTableView(TableViewConfig[testItem]{
		PageSize: 3,
		Columns:  []table.Column{{Title: "ID", Width: 10}},
		FetchFunc: func(ctx context.Context) ([]testItem, error) {
			return []testItem{{id: "a"}, {id: "b"}}, nil
		},
		RowMapper: func(item testItem) table.Row { return table.Row{item.id} },
		LoadMoreFunc: func(ctx context.Context) ([]testItem, bool, error) {
			loadCalls++
			return []testItem{{id: "c"}, {id: "d"}}, false, nil // no more pages
		},
	})

	// Simulate initial data load
	msg := tableDataMsg{viewID: tv.viewID(), items: []testItem{{id: "a"}, {id: "b"}}}
	tv.Update(msg)

	if len(tv.items) != 2 {
		t.Fatalf("initial items = %d, want 2", len(tv.items))
	}

	// Press 'L' to load more
	tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})

	// LoadMore is async — simulate the response
	moreMsg := tableMoreDataMsg{viewID: tv.viewID(), items: []testItem{{id: "c"}, {id: "d"}}, hasMore: false}
	tv.Update(moreMsg)

	if len(tv.items) != 4 {
		t.Fatalf("after load more: items = %d, want 4", len(tv.items))
	}
	if len(tv.allRows) != 4 {
		t.Errorf("after load more: allRows = %d, want 4", len(tv.allRows))
	}
	if loadCalls != 1 {
		t.Errorf("loadCalls = %d, want 1", loadCalls)
	}
}

func TestLoadMoreStatus(t *testing.T) {
	tv := newTestTableViewWithData(5, 3)
	tv.hasMore = true
	status := tv.paginationStatus()
	if !contains(status, "5+") {
		t.Errorf("status should show '5+' when hasMore, got: %s", status)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/services/ -run "TestLoadMore" -v`
Expected: FAIL — `LoadMoreFunc` field not defined.

**Step 3: Implement LoadMoreFunc support**

Add to `TableViewConfig[T]` (after `PageSize`):
```go
LoadMoreFunc func(ctx context.Context) ([]T, bool, error) // optional: returns items, hasMore, error
```

Add to `TableView[T]` struct:
```go
hasMore    bool
loadingMore bool
```

Add new message type (near `tableDataMsg`):
```go
type tableMoreDataMsg struct {
	viewID  uintptr
	items   any
	hasMore bool
}
```

Add to `Update()` in the `tea.KeyMsg` switch (after `"p"` case):
```go
case "L":
    if v.config.LoadMoreFunc != nil && v.hasMore && !v.loadingMore {
        v.loadingMore = true
        return v, v.fetchMore()
    }
```

Add `fetchMore` method:
```go
func (v *TableView[T]) fetchMore() tea.Cmd {
	id := v.viewID()
	loadMore := v.config.LoadMoreFunc
	return func() tea.Msg {
		items, hasMore, err := loadMore(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return tableMoreDataMsg{viewID: id, items: items, hasMore: hasMore}
	}
}
```

Add to `Update()` message switch (after `tableDataMsg` case):
```go
case tableMoreDataMsg:
    if msg.viewID != v.viewID() {
        return v, nil
    }
    newItems, ok := msg.items.([]T)
    if !ok {
        return v, nil
    }
    v.loadingMore = false
    v.hasMore = msg.hasMore
    v.items = append(v.items, newItems...)
    newRows := make([]table.Row, len(newItems))
    for i, item := range newItems {
        newRows[i] = v.config.RowMapper(item)
    }
    v.allRows = append(v.allRows, newRows...)
    v.displayRows = v.allRows
    v.applyPage()
    return v, nil
```

Update `tableDataMsg` handler to set `hasMore`:
```go
// After v.applyPage() in tableDataMsg handler:
if v.config.LoadMoreFunc != nil {
    v.hasMore = true // assume more until first LoadMore returns hasMore=false
}
```

Update `paginationStatus()` to show `+` when `hasMore`:
```go
func (v *TableView[T]) paginationStatus() string {
	total := v.totalPages()
	itemCount := len(v.displayRows)
	if total <= 1 && !v.hasMore {
		return ""
	}
	countStr := fmt.Sprintf("%d", itemCount)
	if v.hasMore {
		countStr += "+"
	}
	if total <= 1 {
		return fmt.Sprintf("(%s items) L to load more", countStr)
	}
	status := fmt.Sprintf("Page %d/%d (%s items)", v.currentPage+1, total, countStr)
	if v.hasMore {
		status += "  L to load more"
	}
	return status
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/services/ -v`
Expected: ALL PASS (new + existing pagination tests).

**Step 5: Commit**

```bash
git add internal/tui/services/tableview.go internal/tui/services/tableview_test.go
git commit -m "add LoadMoreFunc to TableView for server-side pagination"
```

---

### Task 7: S3 buckets TUI view (TDD)

**Files:**
- Create: `internal/tui/services/s3.go`

**Step 1: Create the S3 views file**

```go
package services

import (
	"context"
	"fmt"
	"path"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/utils"
)

func NewS3BucketsView(client *awsclient.ServiceClient) *TableView[awss3.S3Bucket] {
	return NewTableView(TableViewConfig[awss3.S3Bucket]{
		Title:       "S3",
		LoadingText: "Loading buckets...",
		Columns: []table.Column{
			{Title: "Name", Width: 35},
			{Title: "Region", Width: 16},
			{Title: "Created", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awss3.S3Bucket, error) {
			return client.S3.ListBuckets(ctx)
		},
		RowMapper: func(b awss3.S3Bucket) table.Row {
			return table.Row{b.Name, b.Region, utils.TimeOrDash(b.CreatedAt, utils.DateOnly)}
		},
		CopyIDFunc:  func(b awss3.S3Bucket) string { return b.Name },
		CopyARNFunc: func(b awss3.S3Bucket) string { return fmt.Sprintf("arn:aws:s3:::%s", b.Name) },
		OnEnter: func(b awss3.S3Bucket) tea.Cmd {
			return pushView(NewS3ObjectsView(client, b.Name, ""))
		},
	})
}

func NewS3ObjectsView(client *awsclient.ServiceClient, bucket, prefix string) *TableView[awss3.S3Object] {
	title := bucket
	if prefix != "" {
		title = path.Base(prefix[:len(prefix)-1]) + "/" // show last folder name
	}

	var nextToken string

	return NewTableView(TableViewConfig[awss3.S3Object]{
		Title:       title,
		LoadingText: "Loading objects...",
		Columns: []table.Column{
			{Title: "Name", Width: 40},
			{Title: "Size", Width: 12},
			{Title: "Last Modified", Width: 20},
			{Title: "Storage Class", Width: 16},
		},
		FetchFunc: func(ctx context.Context) ([]awss3.S3Object, error) {
			result, err := client.S3.ListObjects(ctx, bucket, prefix, "")
			if err != nil {
				return nil, err
			}
			nextToken = result.NextToken
			return result.Objects, nil
		},
		RowMapper: func(obj awss3.S3Object) table.Row {
			if obj.IsPrefix {
				name := obj.Key
				if prefix != "" {
					name = obj.Key[len(prefix):]
				}
				return table.Row{"📁 " + name, "—", "—", "—"}
			}
			name := obj.Key
			if prefix != "" {
				name = obj.Key[len(prefix):]
			}
			return table.Row{name, formatSize(obj.Size), utils.TimeOrDash(obj.LastModified, utils.DateTime), obj.StorageClass}
		},
		CopyIDFunc: func(obj awss3.S3Object) string {
			return fmt.Sprintf("s3://%s/%s", bucket, obj.Key)
		},
		CopyARNFunc: func(obj awss3.S3Object) string {
			return fmt.Sprintf("arn:aws:s3:::%s/%s", bucket, obj.Key)
		},
		OnEnter: func(obj awss3.S3Object) tea.Cmd {
			if obj.IsPrefix {
				return pushView(NewS3ObjectsView(client, bucket, obj.Key))
			}
			return nil // no drill-down for files
		},
		LoadMoreFunc: func(ctx context.Context) ([]awss3.S3Object, bool, error) {
			if nextToken == "" {
				return nil, false, nil
			}
			result, err := client.S3.ListObjects(ctx, bucket, prefix, nextToken)
			if err != nil {
				return nil, false, err
			}
			nextToken = result.NextToken
			return result.Objects, nextToken != "", nil
		},
	})
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

**Step 2: Verify**

Run: `go build ./internal/tui/services/`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/tui/services/s3.go
git commit -m "add S3 buckets and objects TUI views"
```

---

### Task 8: Wire S3 into root menu

**Files:**
- Modify: `internal/tui/services/root.go:25-31` (items list), `67-91` (handleSelection)

**Step 1: Add S3 to the service list**

Add to `items` slice in `NewRootView` (after ELB):
```go
serviceItem{name: "S3", desc: "Simple Storage Service — Buckets, Objects"},
```

Add to `handleSelection` switch:
```go
case "S3":
    return func() tea.Msg {
        return PushViewMsg{View: NewS3BucketsView(v.client)}
    }
```

**Step 2: Verify**

Run: `go build ./...`
Expected: Compiles.

**Step 3: Commit**

```bash
git add internal/tui/services/root.go
git commit -m "add S3 to services root menu"
```

---

### Task 9: Add L keybinding to help overlay

**Files:**
- Modify: `internal/tui/services/help.go:40-52`

**Step 1: Add L to HelpContextTable bindings**

Add after the `{"p", "Prev page"}` entry:
```go
{"L", "Load more"},
```

**Step 2: Verify**

Run: `go test ./internal/tui/services/ -v`
Expected: ALL PASS.

**Step 3: Commit**

```bash
git add internal/tui/services/help.go
git commit -m "add L (load more) to help overlay"
```

---

### Task 10: Update README

**Files:**
- Modify: `README.md`

**Step 1: Add S3 to the supported services table**

Add row after ELB:
```markdown
| **S3** | Buckets → Objects (prefix/folder navigation) |
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "add S3 to README supported services"
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
- Select S3 → buckets list appears with name, region, created
- Enter on a bucket → objects list with prefixes and objects
- Enter on a prefix → drills into that prefix
- Esc → goes back up
- L → loads more objects (if bucket has > 1000)
- n/p → client-side pagination works
- c → copies s3:// URI
- C → copies ARN

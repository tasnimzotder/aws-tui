# AWS Services Browser Improvements — Design Document

**Date:** 2026-02-25
**Status:** Approved

## Overview

Three enhancements to `aws-utils services`: ECS task detail panel, clipboard copy, and vim-style search/filter. All implemented via shared interfaces in the root Model to avoid duplication across 10+ view files.

## Feature 1: Search/Filter

### Interfaces

```go
type FilterableView interface {
    View
    AllRows() []table.Row
    SetRows(rows []table.Row)
}
```

All table views implement `FilterableView`. List views (root, VPC sub-menu) do not.

### Behavior

- Press `/` to enter filter mode (ignored if current view is not `FilterableView`)
- A `textinput.Model` appears between the header and view content
- On each keystroke, rows are filtered: a row matches if any cell contains the query (case-insensitive)
- Press `Enter` to lock filter and return focus to table navigation
- Press `Esc` while filtering to clear filter and restore all rows
- `/`, `c`, `C` keypresses are suppressed while in filter mode

### State (in root Model)

```go
filtering   bool
filterInput textinput.Model
filterQuery string
```

## Feature 2: Clipboard Copy

### Interfaces

```go
type CopyableView interface {
    View
    CopyID() string
    CopyARN() string
}
```

All table views and the detail view implement `CopyableView`.

### Behavior

- `c` copies `CopyID()` of selected item (short identifier)
- `C` (shift) copies `CopyARN()` (full ARN, falls back to ID)
- Uses `github.com/atotto/clipboard` (already in dependency tree)
- After copying, help bar shows `Copied: <value>` for 2 seconds via `time.After` cmd

### Copy Values Per View

| View | CopyID() | CopyARN() |
|------|----------|-----------|
| EC2 | Instance ID | Instance ID |
| ECS Clusters | Cluster name | Cluster ARN |
| ECS Services | Service name | Service ARN |
| ECS Tasks | Task ID | Task ARN |
| VPCs | VPC ID | VPC ID |
| Subnets | Subnet ID | Subnet ID |
| Security Groups | Group ID | Group ID |
| Internet Gateways | Gateway ID | Gateway ID |
| ECR Repos | Repo name | Repo URI |
| ECR Images | Digest | Digest |
| Task Detail | Task ID | Task ARN |

### Data Changes

Add `ARN string` field to `ECSCluster`, `ECSService`, `ECSTask` structs. Populate from the full ARN in `ListClusters`, `ListServices`, `ListTasks`.

## Feature 3: ECS Task Detail View

### New Data Types

```go
type ECSTaskDetail struct {
    TaskID     string
    TaskARN    string
    Status     string
    TaskDef    string
    StartedAt  time.Time
    StoppedAt  time.Time
    StopCode   string
    StopReason string
    CPU        string
    Memory     string
    Containers []ECSContainerDetail
    NetworkMode string
    PrivateIP  string
    SubnetID   string
}

type ECSContainerDetail struct {
    Name         string
    Image        string
    Status       string
    ExitCode     *int
    LogGroup     string
    LogStream    string
    CPU          int
    Memory       int
    HealthStatus string
}
```

### New API Method

`ECSClient.DescribeTask(ctx, clusterName, taskARN) (*ECSTaskDetail, error)` — calls `DescribeTasks` with a single task ARN, extracts full detail including containers and network configuration. Log group/stream are extracted from container definitions via `DescribeTaskDefinition`.

### ECSAPI Interface Addition

```go
DescribeTaskDefinition(ctx, params, optFns) (*ecs.DescribeTaskDefinitionOutput, error)
```

### View

`TaskDetailView` in `internal/tui/services/detail.go`:

- Uses `viewport.Model` for scrollable key-value content
- Pushed via `PushViewMsg` when pressing Enter on a task in `ECSTasksView`
- Lazy-loads: shows spinner while calling `DescribeTask()`
- Implements `CopyableView` (c = task ID, C = task ARN)
- `Esc`/`Backspace` pops back to task list

### Layout

```
Services › ECS › prod › web-api › abc123def
─────────────────────────────────────────────
  Task ARN       arn:aws:ecs:...
  Status         RUNNING
  Task Def       web-api:42
  Started        2026-02-24 10:30
  CPU            256
  Memory         512
  Network        awsvpc
  Private IP     10.0.1.55
  Subnet         subnet-abc123

  ── Containers ──────────────────────────────
  [app]
    Image        123456.dkr.ecr.../web:latest
    Status       RUNNING
    Health       HEALTHY
    Log Group    /ecs/web-api
    Log Stream   ecs/app/abc123
─────────────────────────────────────────────
```

## Keyboard Shortcuts (Updated)

| Key | Action | Context |
|-----|--------|---------|
| `↑`/`↓` or `j`/`k` | Navigate | Always |
| `Enter` | Drill into / open detail | Table views |
| `Esc` | Clear filter OR go back | Filter → normal |
| `Backspace` | Go back one level | Always |
| `/` | Enter filter mode | Table views |
| `c` | Copy resource ID | Table/detail views |
| `C` | Copy full ARN | Table/detail views |
| `r` | Refresh current view | Always |
| `q` / `Ctrl+C` | Quit | Always |

`Esc` dual behavior: in filter mode clears filter, in normal mode goes back. `/`, `c`, `C` suppressed while filtering.

## Files Changed

| File | Change |
|------|--------|
| `internal/aws/service_types.go` | Add ARN to ECS types, add ECSTaskDetail + ECSContainerDetail |
| `internal/aws/ecs.go` | Store ARNs, add DescribeTask(), add DescribeTaskDefinition to ECSAPI |
| `internal/aws/ecs_test.go` | Test DescribeTask() |
| `internal/tui/services/view.go` | Add FilterableView, CopyableView interfaces + styles |
| `internal/tui/services/model.go` | Filter state, textinput, copy status, handle /, c, C, filter Esc |
| `internal/tui/services/ec2.go` | Implement FilterableView + CopyableView |
| `internal/tui/services/ecs.go` | Implement both interfaces, Enter on task → push detail |
| `internal/tui/services/vpc.go` | Implement both interfaces on 4 views |
| `internal/tui/services/ecr.go` | Implement both interfaces on 2 views |
| `internal/tui/services/detail.go` | New: TaskDetailView with viewport |

# AWS Services Browser — Design Document

**Date:** 2026-02-25
**Status:** Approved

## Overview

A TUI-based AWS resource browser (`aws-utils services --profile <name> --region <region>`) that lets you navigate through EC2, ECS, VPC, and ECR resources using a stack-based drill-down interface with breadcrumb navigation.

## Command

```
aws-utils services --profile <name> --region <region>
```

Both flags are optional. `--profile` falls back to default credential chain. `--region` falls back to the profile's default region.

## Navigation Architecture

### Stack-Based Model

Navigation uses a view stack. Pressing Enter pushes a new view. Esc/Backspace pops the stack. The breadcrumb trail is built from `Title()` of each view in the stack.

```go
type ViewStack struct {
    views []View
}

type View interface {
    Title() string
    View() string
    Update(msg) (View, Cmd)
    Init() Cmd
}
```

### Navigation Hierarchy

```
Services (root list)
├── EC2
│   └── Instances (table with summary header)
├── ECS
│   ├── Clusters
│   │   ├── Services (per cluster)
│   │   │   └── Tasks (per service)
├── VPC
│   ├── VPCs
│   │   ├── Subnets (per VPC)
│   │   ├── Security Groups (per VPC)
│   │   └── Internet Gateways (per VPC)
├── ECR
│   ├── Repositories
│   │   └── Images (per repo)
```

Data is lazy-loaded — only fetched when the user drills into a level.

## Project Structure

```
internal/
├── aws/
│   ├── costexplorer.go      # (existing)
│   ├── types.go             # (existing) + new service types
│   ├── ec2.go               # EC2 client
│   ├── ecs.go               # ECS client
│   ├── vpc.go               # VPC client (uses EC2 API)
│   └── ecr.go               # ECR client
├── tui/
│   ├── model.go             # (existing cost dashboard)
│   ├── styles.go            # (existing) + new styles
│   ├── services/
│   │   ├── model.go         # Root services model with ViewStack
│   │   ├── view.go          # View interface + breadcrumb rendering
│   │   ├── root.go          # Root service list view
│   │   ├── ec2.go           # EC2 instances view
│   │   ├── ecs.go           # ECS cluster/service/task views
│   │   ├── vpc.go           # VPC/subnets/SGs/IGWs views
│   │   └── ecr.go           # ECR repos/images views
cmd/
├── cost.go                  # (existing)
└── services.go              # New services command
```

### Shared Service Client

```go
type ServiceClient struct {
    EC2 EC2API
    ECS ECSAPI
    VPC VPCAPI
    ECR ECRAPI
}
```

## Data Models

### EC2

```go
type EC2Instance struct {
    Name       string
    InstanceID string
    Type       string
    State      string
    PrivateIP  string
    PublicIP   string
}

type EC2Summary struct {
    Total   int
    Running int
    Stopped int
}
```

**API:** `ec2.DescribeInstances`

### ECS

```go
type ECSCluster struct {
    Name             string
    Status           string
    RunningTaskCount int
    ServiceCount     int
}

type ECSService struct {
    Name         string
    Status       string
    DesiredCount int
    RunningCount int
    TaskDef      string
}

type ECSTask struct {
    TaskID       string
    Status       string
    TaskDef      string
    StartedAt    time.Time
    HealthStatus string
}
```

**APIs:** `ecs.ListClusters` → `ecs.DescribeClusters`, `ecs.ListServices` → `ecs.DescribeServices`, `ecs.ListTasks` → `ecs.DescribeTasks`

### VPC

```go
type VPCInfo struct {
    VPCID     string
    Name      string
    CIDR      string
    IsDefault bool
    State     string
}

type SubnetInfo struct {
    SubnetID     string
    Name         string
    CIDR         string
    AZ           string
    AvailableIPs int
}

type SecurityGroupInfo struct {
    GroupID       string
    Name          string
    Description   string
    InboundRules  int
    OutboundRules int
}

type InternetGatewayInfo struct {
    GatewayID string
    Name      string
    State     string
}
```

**APIs:** `ec2.DescribeVpcs`, `ec2.DescribeSubnets`, `ec2.DescribeSecurityGroups`, `ec2.DescribeInternetGateways`

### ECR

```go
type ECRRepo struct {
    Name       string
    URI        string
    ImageCount int
    CreatedAt  time.Time
}

type ECRImage struct {
    Tags     []string
    Digest   string
    SizeMB   float64
    PushedAt time.Time
}
```

**APIs:** `ecr.DescribeRepositories`, `ecr.DescribeImages`

## View Layouts

### Root (Service List)

Vertical list navigated with arrow keys. Enter drills into the selected service.

### EC2 (Instances)

Summary header (Running: N, Stopped: N, Total: N) above a table with columns: Name, Instance ID, Type, State, Private IP, Public IP.

### ECS (3 levels)

- **Clusters:** Table with Cluster, Status, Services, Tasks columns.
- **Services (per cluster):** Table with Name, Status, Desired, Running, Task Def.
- **Tasks (per service):** Table with Task ID, Status, Task Def, Started, Health.

### VPC (VPCs → sub-resources)

- **VPCs:** Table with VPC ID, Name, CIDR, Default, State. Enter shows sub-list.
- **VPC sub-list:** Choice of Subnets, Security Groups, Internet Gateways.
- **Subnets:** Table with Subnet ID, Name, CIDR, AZ, Available IPs.
- **Security Groups:** Table with Group ID, Name, Description, Inbound Rules, Outbound Rules.
- **Internet Gateways:** Table with Gateway ID, Name, State.

### ECR (Repos → Images)

- **Repos:** Table with Repository, Images, Created.
- **Images (per repo):** Table with Tag, Digest, Size, Pushed.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate list/table |
| `Enter` | Drill into selected item |
| `Esc`/`Backspace` | Go back one level |
| `r` | Refresh current view |
| `q` / `Ctrl+C` | Quit |

## Future Considerations (not in v1)

- Search/filter within tables
- More services (Lambda, S3, RDS, etc.)
- Actions (start/stop instances, etc.)
- Copy resource IDs to clipboard

# aws-tui

> **WIP** — This project is under active development and not yet ready for general use.

A terminal UI for browsing and managing AWS resources, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Screenshots

![ECS Task Detail](assets/screenshots/ecs_task.png)

![Cost Dashboard](assets/screenshots/cost.png)

## Features

- **Services Browser** — Browse AWS resources in an interactive TUI with drill-down navigation, filtering, and clipboard copy
- **Server-side Pagination** — First page loads fast; press `L` to load more — no hanging on large accounts
- **Runtime Region Switching** — Press `R` to switch AWS region without restarting
- **EKS Dashboard** — k9s-inspired cluster dashboard with pod exec (native terminal with raw mode, SIGWINCH), log streaming, and port-forwarding — no kubectl required
- **ECS Exec** — Interactive shell into ECS containers via `aws ecs execute-command` with container picker for multi-container tasks
- **Cost Dashboard** — Monthly spend per service with ASCII charts, service drill-down, and historical month navigation

## Supported Services

| Service | What you can browse |
|---------|-------------------|
| **EC2** | Instances — name, type, state, IPs |
| **ECS** | Clusters → Services → Tasks → Logs, with auto-scaling config, deployment history, and container exec |
| **VPC** | VPCs → Dashboard, Subnets, Security Groups, Route Tables, Internet Gateways, NAT Gateways |
| **EKS** | Clusters → Dashboard, Node Groups, Add-ons, Fargate Profiles, Access Entries, Pods, Services, Deployments, Service Accounts (IRSA) |
| **ECR** | Repositories → Images |
| **ELB** | Load Balancers → Listeners → Target Groups |
| **S3** | Buckets → Objects (prefix/folder navigation, file preview, download with progress) |
| **IAM** | Users → Policies/Groups, Roles → Policies/Trust Policy, Policies → Attached Entities |
| **Cost Explorer** | Monthly cost breakdown by service, daily charts, service drill-down, historical month navigation |

### EKS Operations

| Action | Key | Scope | Description |
|--------|-----|-------|-------------|
| Exec | `x` | Pods, Nodes | Interactive shell (prompts for command, defaults to `/bin/sh`) |
| Logs | `l` | Pods | Stream pod container logs with follow, search, and word wrap |
| Port Forward | `f` | Pods | Forward a local port to a pod (e.g. `8080:80`) |
| List Forwards | `F` | Pods, Services | View and manage active port-forward sessions |
| YAML Spec | `e` | All K8s tabs | View resource YAML with syntax highlighting and search |
| Namespace | `N` | K8s tabs | Filter by namespace or clear filter |
| Switch Tab | `Tab` / `1-8` | Cluster detail | Navigate between cluster detail tabs |

Multi-container pods show a container picker before exec/logs.

### ECS Operations

| Action | Key | Scope | Description |
|--------|-----|-------|-------------|
| Exec | `x` | Task detail | Interactive shell into ECS container (requires `session-manager-plugin`) |

Multi-container tasks show a container picker before exec.

### Cost Dashboard

| Key | Action |
|-----|--------|
| `Enter` | Drill down into service daily spend chart |
| `[` / `]` | Navigate to previous / next month (up to 12 months back) |
| `Esc` | Return from service drill-down |
| `r` | Refresh cost data |

## Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Select / drill down |
| `Esc` | Go back |
| `j/k` | Navigate up/down |
| `/` | Filter rows |
| `r` | Refresh data |
| `a` | Toggle auto-refresh |
| `L` | Load more (paginated views) |
| `R` | Switch AWS region |
| `c` / `C` | Copy ID / ARN to clipboard |
| `?` | Toggle context-sensitive help |
| `q` | Quit (with confirmation) |

## Install

### Homebrew

```sh
brew install tasnimzotder/tap/aws-tui
```

## Usage

```sh
# Browse AWS services
awstui services

# View cost data
awstui cost
```

Flags:

```sh
awstui services -p <profile> -r <region>
awstui cost -p <profile>
```

Requires valid AWS credentials (via environment variables, `~/.aws/credentials`, or SSO).

## Build

```sh
go build -o awstui .
```

## Limitations

- **Mostly read-only** — No create, update, or delete operations; browsing only (exceptions: EKS pod exec, ECS exec, and port-forwarding)
- **Single region** — Queries one region at a time (switch with `R`; no cross-region aggregation)
- **Single account** — No multi-account or AWS Organizations support
- **Limited service coverage** — Only the services listed above are supported; no Lambda, RDS, DynamoDB, etc.
- **No real-time updates** — Data is fetched on load; use `r` to refresh or `a` for auto-refresh

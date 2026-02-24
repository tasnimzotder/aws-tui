# aws-tui

> **WIP** — This project is under active development and not yet ready for general use.

A terminal UI for browsing and managing AWS resources, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Services Browser** — Browse AWS resources in an interactive TUI with drill-down navigation, filtering, and clipboard copy
- **Cost Dashboard** — View month-to-date spend per service with ASCII cost graphs

## Supported Services

| Service | What you can browse |
|---------|-------------------|
| **EC2** | Instances — name, type, state, IPs |
| **ECS** | Clusters → Services → Tasks, with auto-scaling config |
| **VPC** | VPCs → Subnets, Security Groups |
| **ECR** | Repositories → Images |
| **ELB** | Load Balancers → Listeners → Target Groups |
| **Cost Explorer** | Month-to-date cost breakdown by service |

## Usage

```sh
# Browse AWS services
aws-tui services

# View cost data
aws-tui cost
```

Flags:

```sh
aws-tui services -p <profile> -r <region>
aws-tui cost -p <profile>
```

Requires valid AWS credentials (via environment variables, `~/.aws/credentials`, or SSO).

## Build

```sh
go build -o aws-tui .
```

## Limitations

- **Read-only** — No create, update, or delete operations; browsing only
- **Single region** — Queries one region at a time (no cross-region aggregation)
- **Single account** — No multi-account or AWS Organizations support
- **Client-side pagination** — All resources are fetched at once then paginated locally; very large accounts may see slow initial loads
- **Limited service coverage** — Only the services listed above are supported; no S3, Lambda, RDS, IAM, etc.
- **No real-time updates** — Data is fetched on load; use `r` to manually refresh
- **Cost data scope** — Cost dashboard shows month-to-date only; no custom date ranges or historical trends

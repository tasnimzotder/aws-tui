# AWS Cost Dashboard — Design Document

**Date:** 2026-02-25
**Status:** Approved

## Overview

A CLI tool (`aws-utils cost`) that displays a Bubble Tea TUI dashboard showing AWS cost data: today's spend, month-to-date total, end-of-month forecast, and a per-service cost breakdown.

## Architecture

### Project Structure

```
aws-utils/
├── main.go                  # Entry point, CLI parsing (cobra)
├── cmd/
│   └── cost.go              # `cost` command setup
├── internal/
│   ├── aws/
│   │   └── costexplorer.go  # AWS Cost Explorer API client wrapper
│   └── tui/
│       ├── model.go         # Bubble Tea model (state, Init, Update, View)
│       ├── styles.go        # Lip Gloss styles
│       └── components.go    # Reusable view components
├── go.mod
└── go.sum
```

### Dependencies

- **CLI:** `cobra`
- **TUI:** `bubbletea`, `lipgloss`, `bubbles` (table component)
- **AWS:** `aws-sdk-go-v2` with `costexplorer` service client

## Data Flow

### API Calls (2 per refresh)

1. **`GetCostAndUsage`** — actual spend
   - Time range: start of current month → today
   - Granularity: `DAILY`
   - Group by: `SERVICE`
   - Metric: `UnblendedCost`

2. **`GetCostForecast`** — projected end-of-month
   - Time range: tomorrow → end of current month
   - Granularity: `MONTHLY`
   - Metric: `UNBLENDED_COST`

### Data Model

```go
type CostData struct {
    TodaySpend    float64
    MTDSpend      float64
    ForecastSpend float64
    Currency      string
    TopServices   []ServiceCost
    LastUpdated   time.Time
}

type ServiceCost struct {
    Name string
    Cost float64
}
```

### Authentication

The `--profile` flag sets the AWS profile via `config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))`. Falls back to default credential chain when no flag is provided.

## TUI Dashboard

### Layout

```
┌─────────────────────────────────────────────────┐
│  AWS Cost Dashboard          profile: my-profile │
├─────────────────────────────────────────────────┤
│  Today: $12.34        MTD: $187.52               │
│  Forecast: $245.80    (Feb 2026)                 │
├─────────────────────────────────────────────────┤
│  Top Services                                    │
│  ┌─────────────────────────────────┬───────────┐ │
│  │ Service                         │ MTD Cost  │ │
│  ├─────────────────────────────────┼───────────┤ │
│  │ Amazon EC2                      │ $89.12    │ │
│  │ Amazon S3                       │ $42.30    │ │
│  │ ...                             │ ...       │ │
│  └─────────────────────────────────┴───────────┘ │
│  Press q to quit • r to refresh                  │
└─────────────────────────────────────────────────┘
```

### Interactions

- **Launch:** Loading spinner while fetching data
- **`q` / `ctrl+c`:** Quit
- **`r`:** Re-fetch data from AWS
- Table shows top 10 services sorted by cost descending

### Styling

Lip Gloss with subtle colors and clean borders. Minimal, terminal-friendly aesthetic.

## Cost Note

AWS Cost Explorer API charges $0.01 per request. Each dashboard view/refresh costs ~$0.02 (2 API calls).

## Future Considerations (not in v1)

- Tab navigation for daily trends and deeper drill-down
- Multi-account support
- Resource exploration commands

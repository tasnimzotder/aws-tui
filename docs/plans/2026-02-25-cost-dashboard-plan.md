# Cost Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `aws-utils cost --profile <name>` — a Bubble Tea TUI dashboard showing today's spend, MTD total, forecast, and top services from AWS Cost Explorer.

**Architecture:** Cobra CLI parses flags and launches a Bubble Tea program. The TUI model fetches data from AWS Cost Explorer via two API calls (`GetCostAndUsage` + `GetCostForecast`), then renders a styled dashboard using Lip Gloss and the Bubbles table component.

**Tech Stack:** Go 1.25, cobra, bubbletea, bubbles (table, spinner), lipgloss, aws-sdk-go-v2 (config, costexplorer)

---

### Task 1: Project Scaffolding & Dependencies

**Files:**
- Modify: `go.mod`
- Create: `main.go`
- Create: `cmd/cost.go`

**Step 1: Install dependencies**

Run:
```bash
cd /Users/tasnim/developments/v0/aws-utils
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/aws/aws-sdk-go-v2/config@latest
go get github.com/aws/aws-sdk-go-v2/service/costexplorer@latest
```

**Step 2: Create `main.go` with root cobra command**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"tasnim.dev/aws-utils/cmd"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "aws-utils",
		Short: "AWS utility tools",
	}

	rootCmd.AddCommand(cmd.NewCostCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Create `cmd/cost.go` with stub cost command**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCostCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show AWS cost dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Cost dashboard (profile: %s) — coming soon\n", profile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")

	return cmd
}
```

**Step 4: Verify it compiles and runs**

Run:
```bash
go build -o aws-utils . && ./aws-utils cost --profile test
```
Expected: `Cost dashboard (profile: test) — coming soon`

**Step 5: Commit**

```bash
git add main.go cmd/cost.go go.mod go.sum
git commit -m "feat: scaffold CLI with cobra and cost subcommand"
```

---

### Task 2: AWS Cost Explorer Client — Data Types & Interface

**Files:**
- Create: `internal/aws/types.go`
- Create: `internal/aws/costexplorer.go`

**Step 1: Create data types**

Create `internal/aws/types.go`:

```go
package aws

import "time"

// CostData holds all cost information for the dashboard.
type CostData struct {
	TodaySpend    float64
	MTDSpend      float64
	ForecastSpend float64
	Currency      string
	TopServices   []ServiceCost
	LastUpdated   time.Time
}

// ServiceCost represents cost for a single AWS service.
type ServiceCost struct {
	Name string
	Cost float64
}
```

**Step 2: Create Cost Explorer client with `FetchCostData` method**

Create `internal/aws/costexplorer.go`:

```go
package aws

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// CostExplorerAPI is the subset of the AWS Cost Explorer client we use.
type CostExplorerAPI interface {
	GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
	GetCostForecast(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error)
}

// Client wraps the AWS Cost Explorer API.
type Client struct {
	ce CostExplorerAPI
}

// NewClient creates a new Cost Explorer client with the given profile.
func NewClient(ctx context.Context, profile string) (*Client, error) {
	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &Client{ce: costexplorer.NewFromConfig(cfg)}, nil
}

// NewClientWithAPI creates a client with a custom API implementation (for testing).
func NewClientWithAPI(api CostExplorerAPI) *Client {
	return &Client{ce: api}
}

// FetchCostData retrieves cost and forecast data from AWS Cost Explorer.
func (c *Client) FetchCostData(ctx context.Context) (*CostData, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// GetCostAndUsage: month start to today, grouped by SERVICE
	usageOut, err := c.ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(monthStart.Format("2006-01-02")),
			End:   aws.String(tomorrow.Format("2006-01-02")),
		},
		Granularity: types.GranularityDaily,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []types.GroupDefinition{
			{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GetCostAndUsage: %w", err)
	}

	// Aggregate per-service costs and compute today + MTD
	serviceMap := make(map[string]float64)
	var todaySpend, mtdSpend float64
	var currency string

	for _, result := range usageOut.ResultsByTime {
		isToday := aws.ToString(result.TimePeriod.Start) == today.Format("2006-01-02")
		for _, group := range result.Groups {
			svcName := group.Keys[0]
			amount, _ := strconv.ParseFloat(aws.ToString(group.Metrics["UnblendedCost"].Amount), 64)
			unit := aws.ToString(group.Metrics["UnblendedCost"].Unit)
			if currency == "" {
				currency = unit
			}
			serviceMap[svcName] += amount
			mtdSpend += amount
			if isToday {
				todaySpend += amount
			}
		}
	}

	// Sort services by cost descending
	services := make([]ServiceCost, 0, len(serviceMap))
	for name, cost := range serviceMap {
		services = append(services, ServiceCost{Name: name, Cost: cost})
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Cost > services[j].Cost
	})

	// Cap at top 10
	if len(services) > 10 {
		services = services[:10]
	}

	// GetCostForecast: tomorrow to month end
	var forecastSpend float64
	if tomorrow.Before(monthEnd) {
		forecastOut, err := c.ce.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(tomorrow.Format("2006-01-02")),
				End:   aws.String(monthEnd.Format("2006-01-02")),
			},
			Granularity: types.GranularityMonthly,
			Metric:      types.MetricUnblendedCost,
		})
		if err != nil {
			return nil, fmt.Errorf("GetCostForecast: %w", err)
		}
		forecastSpend, _ = strconv.ParseFloat(aws.ToString(forecastOut.Total.Amount), 64)
	}

	return &CostData{
		TodaySpend:    todaySpend,
		MTDSpend:      mtdSpend,
		ForecastSpend: forecastSpend,
		Currency:      currency,
		TopServices:   services,
		LastUpdated:   now,
	}, nil
}
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/aws/types.go internal/aws/costexplorer.go
git commit -m "feat: add AWS Cost Explorer client with FetchCostData"
```

---

### Task 3: AWS Cost Explorer Client — Unit Tests

**Files:**
- Create: `internal/aws/costexplorer_test.go`

**Step 1: Write tests with mock API**

Create `internal/aws/costexplorer_test.go`:

```go
package aws

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// mockCostExplorerAPI implements CostExplorerAPI for testing.
type mockCostExplorerAPI struct {
	getCostAndUsageFunc  func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
	getCostForecastFunc  func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error)
}

func (m *mockCostExplorerAPI) GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	return m.getCostAndUsageFunc(ctx, params, optFns...)
}

func (m *mockCostExplorerAPI) GetCostForecast(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
	return m.getCostForecastFunc(ctx, params, optFns...)
}

func TestFetchCostData_AggregatesCorrectly(t *testing.T) {
	mock := &mockCostExplorerAPI{
		getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
			return &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-01"),
							End:   awssdk.String("2026-02-02"),
						},
						Groups: []types.Group{
							{
								Keys: []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("50.00"), Unit: awssdk.String("USD")},
								},
							},
							{
								Keys: []string{"Amazon S3"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("20.00"), Unit: awssdk.String("USD")},
								},
							},
						},
					},
				},
			}, nil
		},
		getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
			return &costexplorer.GetCostForecastOutput{
				Total: &types.MetricValue{
					Amount: awssdk.String("300.00"),
					Unit:   awssdk.String("USD"),
				},
			}, nil
		},
	}

	client := NewClientWithAPI(mock)
	data, err := client.FetchCostData(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.MTDSpend != 70.00 {
		t.Errorf("MTDSpend = %f, want 70.00", data.MTDSpend)
	}
	if data.ForecastSpend != 300.00 {
		t.Errorf("ForecastSpend = %f, want 300.00", data.ForecastSpend)
	}
	if data.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", data.Currency)
	}
	if len(data.TopServices) != 2 {
		t.Fatalf("TopServices length = %d, want 2", len(data.TopServices))
	}
	if data.TopServices[0].Name != "Amazon EC2" {
		t.Errorf("TopServices[0].Name = %s, want Amazon EC2", data.TopServices[0].Name)
	}
	if data.TopServices[0].Cost != 50.00 {
		t.Errorf("TopServices[0].Cost = %f, want 50.00", data.TopServices[0].Cost)
	}
}

func TestFetchCostData_SortsServicesByDescendingCost(t *testing.T) {
	mock := &mockCostExplorerAPI{
		getCostAndUsageFunc: func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
			return &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						TimePeriod: &types.DateInterval{
							Start: awssdk.String("2026-02-01"),
							End:   awssdk.String("2026-02-02"),
						},
						Groups: []types.Group{
							{
								Keys: []string{"AWS Lambda"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("5.00"), Unit: awssdk.String("USD")},
								},
							},
							{
								Keys: []string{"Amazon EC2"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("100.00"), Unit: awssdk.String("USD")},
								},
							},
							{
								Keys: []string{"Amazon S3"},
								Metrics: map[string]types.MetricValue{
									"UnblendedCost": {Amount: awssdk.String("30.00"), Unit: awssdk.String("USD")},
								},
							},
						},
					},
				},
			}, nil
		},
		getCostForecastFunc: func(ctx context.Context, params *costexplorer.GetCostForecastInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostForecastOutput, error) {
			return &costexplorer.GetCostForecastOutput{
				Total: &types.MetricValue{Amount: awssdk.String("200.00"), Unit: awssdk.String("USD")},
			}, nil
		},
	}

	client := NewClientWithAPI(mock)
	data, err := client.FetchCostData(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"Amazon EC2", "Amazon S3", "AWS Lambda"}
	for i, svc := range data.TopServices {
		if svc.Name != expected[i] {
			t.Errorf("TopServices[%d].Name = %s, want %s", i, svc.Name, expected[i])
		}
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/aws/ -v`
Expected: Both tests PASS

**Step 3: Commit**

```bash
git add internal/aws/costexplorer_test.go
git commit -m "test: add unit tests for Cost Explorer client"
```

---

### Task 4: TUI Styles

**Files:**
- Create: `internal/tui/styles.go`

**Step 1: Create Lip Gloss styles**

Create `internal/tui/styles.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#A78BFA")
	mutedColor     = lipgloss.Color("#6B7280")
	successColor   = lipgloss.Color("#10B981")
	warningColor   = lipgloss.Color("#F59E0B")

	// Layout
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(mutedColor).
			Padding(0, 1)

	metricLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	metricValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(successColor)

	forecastValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warningColor)

	profileStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(1, 0, 0, 0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	dashboardStyle = lipgloss.NewStyle().
			Padding(1, 2)
)
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat: add Lip Gloss styles for cost dashboard"
```

---

### Task 5: TUI Model — State, Init, Update, View

**Files:**
- Create: `internal/tui/model.go`

**Step 1: Create the Bubble Tea model**

Create `internal/tui/model.go`:

```go
package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

// Messages
type costDataMsg struct{ data *awsclient.CostData }
type errMsg struct{ err error }

// Model holds the TUI state.
type Model struct {
	client  *awsclient.Client
	profile string
	data    *awsclient.CostData
	err     error
	loading bool
	spinner spinner.Model
	table   table.Model
}

// NewModel creates a new TUI model.
func NewModel(client *awsclient.Client, profile string) Model {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(primaryColor)),
	)

	columns := []table.Column{
		{Title: "Service", Width: 35},
		{Title: "MTD Cost", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(mutedColor).
		BorderBottom(true).
		Bold(true)
	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(styles)

	return Model{
		client:  client,
		profile: profile,
		loading: true,
		spinner: s,
		table:   t,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchCost())
}

func (m Model) fetchCost() tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.FetchCostData(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return costDataMsg{data: data}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchCost())
		}

	case costDataMsg:
		m.data = msg.data
		m.loading = false
		m.table.SetRows(m.buildRows())
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	// Header
	profileText := "default"
	if m.profile != "" {
		profileText = m.profile
	}
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		titleStyle.Render("AWS Cost Dashboard"),
		"   ",
		profileStyle.Render(fmt.Sprintf("profile: %s", profileText)),
	)

	if m.loading {
		return dashboardStyle.Render(
			header + "\n\n" + m.spinner.View() + " Fetching cost data...\n",
		)
	}

	if m.err != nil {
		return dashboardStyle.Render(
			header + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) +
				"\n\n" + helpStyle.Render("Press r to retry • q to quit"),
		)
	}

	if m.data == nil {
		return dashboardStyle.Render(header + "\n\nNo data available.\n")
	}

	// Metrics
	monthLabel := m.data.LastUpdated.Format("Jan 2006")
	metrics := lipgloss.JoinHorizontal(
		lipgloss.Top,
		metricLabelStyle.Render("Today: ")+metricValueStyle.Render(formatCurrency(m.data.TodaySpend, m.data.Currency)),
		"        ",
		metricLabelStyle.Render("MTD: ")+metricValueStyle.Render(formatCurrency(m.data.MTDSpend, m.data.Currency)),
	)
	forecast := metricLabelStyle.Render("Forecast: ") +
		forecastValueStyle.Render(formatCurrency(m.data.ForecastSpend, m.data.Currency)) +
		metricLabelStyle.Render(fmt.Sprintf("    (%s)", monthLabel))

	// Table
	tableView := "\n" + metricLabelStyle.Render("Top Services") + "\n" + m.table.View()

	// Help
	help := helpStyle.Render("Press q to quit • r to refresh")

	return dashboardStyle.Render(
		headerStyle.Render(header) + "\n\n" +
			metrics + "\n" +
			forecast + "\n" +
			tableView + "\n" +
			help,
	)
}

func (m Model) buildRows() []table.Row {
	if m.data == nil {
		return nil
	}
	rows := make([]table.Row, len(m.data.TopServices))
	for i, svc := range m.data.TopServices {
		rows[i] = table.Row{svc.Name, formatCurrency(svc.Cost, m.data.Currency)}
	}
	return rows
}

func formatCurrency(amount float64, currency string) string {
	symbol := "$"
	if currency != "" && currency != "USD" {
		symbol = currency + " "
	}
	return fmt.Sprintf("%s%.2f", symbol, amount)
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat: add Bubble Tea model for cost dashboard TUI"
```

---

### Task 6: Wire TUI Into Cobra Command

**Files:**
- Modify: `cmd/cost.go`

**Step 1: Update `cmd/cost.go` to launch the TUI**

Replace the full contents of `cmd/cost.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	awsclient "tasnim.dev/aws-utils/internal/aws"
	"tasnim.dev/aws-utils/internal/tui"
)

func NewCostCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Show AWS cost dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := awsclient.NewClient(context.Background(), profile)
			if err != nil {
				return fmt.Errorf("initializing AWS client: %w", err)
			}

			model := tui.NewModel(client, profile)
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")

	return cmd
}
```

**Step 2: Verify it compiles**

Run: `go build -o aws-utils .`
Expected: Binary compiles successfully

**Step 3: Commit**

```bash
git add cmd/cost.go
git commit -m "feat: wire TUI into cobra cost command"
```

---

### Task 7: Manual Integration Test

**Step 1: Build and run with a real AWS profile**

Run:
```bash
go build -o aws-utils . && ./aws-utils cost --profile <your-profile>
```

Expected: Dashboard appears in alt-screen with loading spinner, then shows cost data. Press `q` to quit, `r` to refresh.

**Step 2: Test error handling**

Run with a bogus profile:
```bash
./aws-utils cost --profile nonexistent-profile-xyz
```

Expected: Dashboard shows error message with retry prompt.

**Step 3: Test no-profile default**

Run:
```bash
./aws-utils cost
```

Expected: Uses default AWS credential chain.

**Step 4: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: address issues found during manual testing"
```

---

### Task 8: TUI View Rendering Tests

**Files:**
- Create: `internal/tui/model_test.go`

**Step 1: Write view rendering tests**

Create `internal/tui/model_test.go`:

```go
package tui

import (
	"strings"
	"testing"
	"time"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

func TestView_Loading(t *testing.T) {
	m := NewModel(nil, "test-profile")
	m.loading = true

	view := m.View()
	if !strings.Contains(view, "Fetching cost data") {
		t.Error("loading view should contain 'Fetching cost data'")
	}
	if !strings.Contains(view, "test-profile") {
		t.Error("loading view should show profile name")
	}
}

func TestView_WithData(t *testing.T) {
	m := NewModel(nil, "prod")
	m.loading = false
	m.data = &awsclient.CostData{
		TodaySpend:    12.34,
		MTDSpend:      187.52,
		ForecastSpend: 245.80,
		Currency:      "USD",
		TopServices: []awsclient.ServiceCost{
			{Name: "Amazon EC2", Cost: 89.12},
			{Name: "Amazon S3", Cost: 42.30},
		},
		LastUpdated: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
	}
	m.table.SetRows(m.buildRows())

	view := m.View()
	if !strings.Contains(view, "$12.34") {
		t.Error("view should show today's spend")
	}
	if !strings.Contains(view, "$187.52") {
		t.Error("view should show MTD spend")
	}
	if !strings.Contains(view, "$245.80") {
		t.Error("view should show forecast")
	}
	if !strings.Contains(view, "Amazon EC2") {
		t.Error("view should show top service")
	}
	if !strings.Contains(view, "prod") {
		t.Error("view should show profile name")
	}
}

func TestView_Error(t *testing.T) {
	m := NewModel(nil, "broken")
	m.loading = false
	m.err = fmt.Errorf("access denied")

	view := m.View()
	if !strings.Contains(view, "access denied") {
		t.Error("error view should show error message")
	}
	if !strings.Contains(view, "retry") {
		t.Error("error view should mention retry")
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		amount   float64
		currency string
		want     string
	}{
		{12.34, "USD", "$12.34"},
		{0.00, "USD", "$0.00"},
		{1234.56, "", "$1234.56"},
		{50.00, "EUR", "EUR 50.00"},
	}

	for _, tt := range tests {
		got := formatCurrency(tt.amount, tt.currency)
		if got != tt.want {
			t.Errorf("formatCurrency(%f, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
		}
	}
}
```

Note: You'll need to add `"fmt"` to the imports for the error test.

**Step 2: Run tests**

Run: `go test ./internal/tui/ -v`
Expected: All tests PASS

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All tests across both packages PASS

**Step 4: Commit**

```bash
git add internal/tui/model_test.go
git commit -m "test: add TUI view rendering tests"
```

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Scaffolding & deps | `main.go`, `cmd/cost.go`, `go.mod` |
| 2 | AWS Cost Explorer client | `internal/aws/types.go`, `internal/aws/costexplorer.go` |
| 3 | AWS client unit tests | `internal/aws/costexplorer_test.go` |
| 4 | TUI styles | `internal/tui/styles.go` |
| 5 | TUI model (Init/Update/View) | `internal/tui/model.go` |
| 6 | Wire TUI into cobra | `cmd/cost.go` |
| 7 | Manual integration test | — |
| 8 | TUI view rendering tests | `internal/tui/model_test.go` |

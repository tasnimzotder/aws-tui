# Services Browser Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add vim-style search/filter, clipboard copy (`c`/`C`), and ECS task detail panel to the services browser.

**Architecture:** Two new optional interfaces (`FilterableView`, `CopyableView`) let the root Model handle filter/copy logic centrally. Each table view implements both. ECS task detail is a new `TaskDetailView` pushed via existing `PushViewMsg` stack. ARN fields added to ECS types for copy support.

**Tech Stack:** Go 1.25, bubbletea, bubbles (table, textinput, viewport, spinner), lipgloss, atotto/clipboard, aws-sdk-go-v2/service/ecs

---

### Task 1: Add ARN Fields to ECS Data Types

**Files:**
- Modify: `internal/aws/service_types.go`
- Modify: `internal/aws/ecs.go`
- Modify: `internal/aws/ecs_test.go`

**Step 1: Add ARN field to ECS structs**

In `internal/aws/service_types.go`, add `ARN string` to `ECSCluster`, `ECSService`, and `ECSTask`:

```go
type ECSCluster struct {
	Name             string
	ARN              string
	Status           string
	RunningTaskCount int
	ServiceCount     int
}

type ECSService struct {
	Name         string
	ARN          string
	Status       string
	DesiredCount int
	RunningCount int
	TaskDef      string
}

type ECSTask struct {
	TaskID       string
	ARN          string
	Status       string
	TaskDef      string
	StartedAt    time.Time
	HealthStatus string
}
```

**Step 2: Populate ARNs in ecs.go**

In `internal/aws/ecs.go`, update `ListClusters` to store the ARN:

In the `ListClusters` method, change the cluster construction to:
```go
clusters[i] = ECSCluster{
    Name:             awssdk.ToString(cl.ClusterName),
    ARN:              awssdk.ToString(cl.ClusterArn),
    Status:           awssdk.ToString(cl.Status),
    RunningTaskCount: int(cl.RunningTasksCount),
    ServiceCount:     int(cl.ActiveServicesCount),
}
```

In the `ListServices` method, change the service construction to:
```go
services[i] = ECSService{
    Name:         awssdk.ToString(svc.ServiceName),
    ARN:          awssdk.ToString(svc.ServiceArn),
    Status:       awssdk.ToString(svc.Status),
    DesiredCount: int(svc.DesiredCount),
    RunningCount: int(svc.RunningCount),
    TaskDef:      taskDef,
}
```

In the `ListTasks` method, store the full ARN before shortening the task ID:
```go
tasks[i] = ECSTask{
    TaskID:       taskID,
    ARN:          awssdk.ToString(t.TaskArn),
    Status:       awssdk.ToString(t.LastStatus),
    TaskDef:      taskDef,
    StartedAt:    startedAt,
    HealthStatus: string(t.HealthStatus),
}
```

**Step 3: Update test to verify ARN**

In `internal/aws/ecs_test.go`, add an ARN assertion to `TestListClusters`:

After the existing `ServiceCount` check, add:
```go
if clusters[0].ARN != "arn:aws:ecs:us-east-1:123456:cluster/prod" {
    t.Errorf("ARN = %s, want arn:aws:ecs:us-east-1:123456:cluster/prod", clusters[0].ARN)
}
```

Also update the mock `describeClustersFunc` to include `ClusterArn`:
```go
ClusterArn:          awssdk.String("arn:aws:ecs:us-east-1:123456:cluster/prod"),
```

**Step 4: Run tests**

Run: `go test ./internal/aws/ -v -run TestListClusters`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/aws/service_types.go internal/aws/ecs.go internal/aws/ecs_test.go
git commit -m "feat: add ARN fields to ECS data types"
```

---

### Task 2: Add ECSTaskDetail Types and DescribeTask API

**Files:**
- Modify: `internal/aws/service_types.go`
- Modify: `internal/aws/ecs.go`
- Create: `internal/aws/ecs_detail_test.go`

**Step 1: Add ECSTaskDetail and ECSContainerDetail types**

Append to `internal/aws/service_types.go`:

```go
// ECS Detail (for task detail view)

type ECSTaskDetail struct {
	TaskID      string
	TaskARN     string
	Status      string
	TaskDef     string
	StartedAt   time.Time
	StoppedAt   time.Time
	StopCode    string
	StopReason  string
	CPU         string
	Memory      string
	Containers  []ECSContainerDetail
	NetworkMode string
	PrivateIP   string
	SubnetID    string
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

**Step 2: Add DescribeTaskDefinition to ECSAPI interface**

In `internal/aws/ecs.go`, add to the `ECSAPI` interface:

```go
DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
```

**Step 3: Add DescribeTask method**

Add to `internal/aws/ecs.go`:

```go
func (c *ECSClient) DescribeTask(ctx context.Context, clusterName, taskARN string) (*ECSTaskDetail, error) {
	descOut, err := c.api.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: awssdk.String(clusterName),
		Tasks:   []string{taskARN},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeTasks: %w", err)
	}
	if len(descOut.Tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskARN)
	}

	t := descOut.Tasks[0]

	taskID := awssdk.ToString(t.TaskArn)
	if parts := strings.Split(taskID, "/"); len(parts) > 2 {
		taskID = parts[len(parts)-1]
	}

	taskDef := awssdk.ToString(t.TaskDefinitionArn)
	taskDefShort := taskDef
	if parts := strings.Split(taskDef, "/"); len(parts) > 1 {
		taskDefShort = parts[len(parts)-1]
	}

	var startedAt, stoppedAt time.Time
	if t.StartedAt != nil {
		startedAt = *t.StartedAt
	}
	if t.StoppedAt != nil {
		stoppedAt = *t.StoppedAt
	}

	detail := &ECSTaskDetail{
		TaskID:     taskID,
		TaskARN:    awssdk.ToString(t.TaskArn),
		Status:     awssdk.ToString(t.LastStatus),
		TaskDef:    taskDefShort,
		StartedAt:  startedAt,
		StoppedAt:  stoppedAt,
		StopCode:   string(t.StopCode),
		StopReason: awssdk.ToString(t.StoppedReason),
		CPU:        awssdk.ToString(t.Cpu),
		Memory:     awssdk.ToString(t.Memory),
	}

	// Extract network info
	for _, att := range t.Attachments {
		if awssdk.ToString(att.Type) == "ElasticNetworkInterface" {
			for _, kv := range att.Details {
				switch awssdk.ToString(kv.Name) {
				case "privateIPv4Address":
					detail.PrivateIP = awssdk.ToString(kv.Value)
				case "subnetId":
					detail.SubnetID = awssdk.ToString(kv.Value)
				case "networkInterfaceId":
					detail.NetworkMode = "awsvpc"
				}
			}
		}
	}

	// Get log configuration from task definition
	logConfigs := map[string][2]string{} // containerName -> [logGroup, logStreamPrefix]
	tdOut, err := c.api.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: awssdk.String(taskDef),
	})
	if err == nil && tdOut.TaskDefinition != nil {
		for _, cd := range tdOut.TaskDefinition.ContainerDefinitions {
			name := awssdk.ToString(cd.Name)
			if cd.LogConfiguration != nil && cd.LogConfiguration.Options != nil {
				logConfigs[name] = [2]string{
					cd.LogConfiguration.Options["awslogs-group"],
					cd.LogConfiguration.Options["awslogs-stream-prefix"],
				}
			}
		}
	}

	// Build container details
	detail.Containers = make([]ECSContainerDetail, len(t.Containers))
	for i, c := range t.Containers {
		var exitCode *int
		if c.ExitCode != nil {
			ec := int(*c.ExitCode)
			exitCode = &ec
		}

		cd := ECSContainerDetail{
			Name:         awssdk.ToString(c.Name),
			Image:        awssdk.ToString(c.Image),
			Status:       awssdk.ToString(c.LastStatus),
			ExitCode:     exitCode,
			CPU:          int(awssdk.ToInt32(c.Cpu)),
			Memory:       int(awssdk.ToInt32(c.Memory)),
			HealthStatus: string(c.HealthStatus),
		}

		// Attach log info
		if lc, ok := logConfigs[cd.Name]; ok {
			cd.LogGroup = lc[0]
			if lc[1] != "" {
				cd.LogStream = lc[1] + "/" + cd.Name + "/" + taskID
			}
		}

		detail.Containers[i] = cd
	}

	return detail, nil
}
```

**Step 4: Write test**

Create `internal/aws/ecs_detail_test.go`:

```go
package aws

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestDescribeTask(t *testing.T) {
	started := time.Date(2026, 2, 24, 10, 30, 0, 0, time.UTC)
	mock := &mockECSAPI{
		listClustersFunc:     nil,
		describeClustersFunc: nil,
		listServicesFunc:     nil,
		describeServicesFunc: nil,
		listTasksFunc:        nil,
		describeTasksFunc: func(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
			return &ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{
					{
						TaskArn:           awssdk.String("arn:aws:ecs:us-east-1:123456:task/prod/abc123def"),
						LastStatus:        awssdk.String("RUNNING"),
						TaskDefinitionArn: awssdk.String("arn:aws:ecs:us-east-1:123456:task-definition/web-api:42"),
						StartedAt:         &started,
						Cpu:               awssdk.String("256"),
						Memory:            awssdk.String("512"),
						Attachments: []ecstypes.Attachment{
							{
								Type: awssdk.String("ElasticNetworkInterface"),
								Details: []ecstypes.KeyValuePair{
									{Name: awssdk.String("privateIPv4Address"), Value: awssdk.String("10.0.1.55")},
									{Name: awssdk.String("subnetId"), Value: awssdk.String("subnet-abc123")},
									{Name: awssdk.String("networkInterfaceId"), Value: awssdk.String("eni-xyz")},
								},
							},
						},
						Containers: []ecstypes.Container{
							{
								Name:         awssdk.String("app"),
								Image:        awssdk.String("123456.dkr.ecr.us-east-1.amazonaws.com/web:latest"),
								LastStatus:   awssdk.String("RUNNING"),
								HealthStatus: ecstypes.HealthStatusHealthy,
							},
						},
					},
				},
			}, nil
		},
		describeTaskDefinitionFunc: func(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
			return &ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name: awssdk.String("app"),
							LogConfiguration: &ecstypes.LogConfiguration{
								LogDriver: ecstypes.LogDriverAwslogs,
								Options: map[string]string{
									"awslogs-group":         "/ecs/web-api",
									"awslogs-stream-prefix": "ecs",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewECSClient(mock)
	detail, err := client.DescribeTask(context.Background(), "prod", "arn:aws:ecs:us-east-1:123456:task/prod/abc123def")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.TaskID != "abc123def" {
		t.Errorf("TaskID = %s, want abc123def", detail.TaskID)
	}
	if detail.CPU != "256" {
		t.Errorf("CPU = %s, want 256", detail.CPU)
	}
	if detail.PrivateIP != "10.0.1.55" {
		t.Errorf("PrivateIP = %s, want 10.0.1.55", detail.PrivateIP)
	}
	if detail.NetworkMode != "awsvpc" {
		t.Errorf("NetworkMode = %s, want awsvpc", detail.NetworkMode)
	}
	if len(detail.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(detail.Containers))
	}
	if detail.Containers[0].Name != "app" {
		t.Errorf("Container.Name = %s, want app", detail.Containers[0].Name)
	}
	if detail.Containers[0].LogGroup != "/ecs/web-api" {
		t.Errorf("Container.LogGroup = %s, want /ecs/web-api", detail.Containers[0].LogGroup)
	}
	if detail.Containers[0].LogStream != "ecs/app/abc123def" {
		t.Errorf("Container.LogStream = %s, want ecs/app/abc123def", detail.Containers[0].LogStream)
	}
}
```

**Step 5: Add describeTaskDefinitionFunc to mockECSAPI**

In `internal/aws/ecs_test.go`, add the new field to the mock struct:

Add field:
```go
describeTaskDefinitionFunc func(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
```

Add method:
```go
func (m *mockECSAPI) DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
	return m.describeTaskDefinitionFunc(ctx, params, optFns...)
}
```

**Step 6: Run tests**

Run: `go test ./internal/aws/ -v -run TestDescribeTask`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/aws/service_types.go internal/aws/ecs.go internal/aws/ecs_test.go internal/aws/ecs_detail_test.go
git commit -m "feat: add ECS task detail API with container and log info"
```

---

### Task 3: Add FilterableView and CopyableView Interfaces

**Files:**
- Modify: `internal/tui/services/view.go`

**Step 1: Add interfaces and new styles**

In `internal/tui/services/view.go`, add after the `PopViewMsg` struct:

```go
// FilterableView is implemented by views that support text filtering.
type FilterableView interface {
	View
	AllRows() []table.Row
	SetRows(rows []table.Row)
}

// CopyableView is implemented by views that support clipboard copy.
type CopyableView interface {
	View
	CopyID() string
	CopyARN() string
}
```

Add import for `"github.com/charmbracelet/bubbles/table"`.

Add these styles to the var block:

```go
svcFilterStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#7C3AED"))

svcCopiedStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#10B981")).
    Bold(true)
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/tui/services/view.go
git commit -m "feat: add FilterableView and CopyableView interfaces"
```

---

### Task 4: Add Filter and Copy Logic to Root Model

**Files:**
- Modify: `internal/tui/services/model.go`

**Step 1: Update model with filter and copy state**

Replace the entire `internal/tui/services/model.go` with:

```go
package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

type clearCopiedMsg struct{}

// Model is the root Bubble Tea model for the services browser.
type Model struct {
	client  *awsclient.ServiceClient
	profile string
	region  string
	stack   []View

	// Filter state
	filtering   bool
	filterInput textinput.Model
	filterQuery string

	// Copy status
	copiedText string
}

// NewModel creates a new services browser model.
func NewModel(client *awsclient.ServiceClient, profile, region string) Model {
	root := NewRootView(client)

	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 64

	return Model{
		client:      client,
		profile:     profile,
		region:      region,
		stack:       []View{root},
		filterInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.stack) > 0 {
		return m.stack[len(m.stack)-1].Init()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearCopiedMsg:
		m.copiedText = ""
		return m, nil

	case tea.KeyMsg:
		// Filter mode input handling
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filterQuery = ""
				m.filterInput.SetValue("")
				// Restore all rows
				if fv, ok := m.currentFilterable(); ok {
					fv.SetRows(fv.AllRows())
				}
				return m, nil
			case "enter":
				m.filtering = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.filterQuery = m.filterInput.Value()
				// Apply filter
				if fv, ok := m.currentFilterable(); ok {
					m.applyFilter(fv)
				}
				return m, cmd
			}
		}

		// Normal mode
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				return m, nil
			}
			return m, tea.Quit
		case "backspace":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				return m, nil
			}
		case "/":
			if _, ok := m.currentFilterable(); ok {
				m.filtering = true
				m.filterInput.SetValue("")
				m.filterInput.Focus()
				return m, textinput.Blink
			}
		case "c":
			if cv, ok := m.currentCopyable(); ok {
				id := cv.CopyID()
				if id != "" {
					clipboard.WriteAll(id)
					m.copiedText = id
					return m, m.clearCopiedAfter()
				}
			}
		case "C":
			if cv, ok := m.currentCopyable(); ok {
				arn := cv.CopyARN()
				if arn != "" {
					clipboard.WriteAll(arn)
					m.copiedText = arn
					return m, m.clearCopiedAfter()
				}
			}
		}

	case PushViewMsg:
		m.stack = append(m.stack, msg.View)
		return m, msg.View.Init()

	case PopViewMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, nil
	}

	// Delegate to current view
	if len(m.stack) > 0 {
		current := m.stack[len(m.stack)-1]
		updated, cmd := current.Update(msg)
		m.stack[len(m.stack)-1] = updated
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	// Build breadcrumb
	titles := make([]string, len(m.stack))
	for i, v := range m.stack {
		titles[i] = v.Title()
	}
	breadcrumb := renderBreadcrumb(titles)

	// Profile and region info
	profileText := "default"
	if m.profile != "" {
		profileText = m.profile
	}
	regionText := "default"
	if m.region != "" {
		regionText = m.region
	}
	info := svcProfileStyle.Render(fmt.Sprintf("profile: %s  region: %s", profileText, regionText))

	header := lipgloss.JoinHorizontal(lipgloss.Top, breadcrumb, "   ", info)

	// Filter bar
	filterBar := ""
	if m.filtering {
		filterBar = svcFilterStyle.Render("/ ") + m.filterInput.View() + "\n"
	} else if m.filterQuery != "" {
		filterBar = svcFilterStyle.Render(fmt.Sprintf("filter: %s", m.filterQuery)) + "\n"
	}

	// Current view content
	content := ""
	if len(m.stack) > 0 {
		content = m.stack[len(m.stack)-1].View()
	}

	// Help / copy status
	var help string
	if m.copiedText != "" {
		help = svcCopiedStyle.Render(fmt.Sprintf("Copied: %s", m.copiedText))
	} else if m.filtering {
		help = svcHelpStyle.Render("Enter to lock filter • Esc to clear")
	} else if len(m.stack) <= 1 {
		help = svcHelpStyle.Render("Enter to select • / to filter • q to quit")
	} else {
		help = svcHelpStyle.Render("Esc to go back • r to refresh • / to filter • c to copy • q to quit")
	}

	return svcDashboardStyle.Render(
		svcHeaderStyle.Render(header) + "\n\n" +
			filterBar +
			content + "\n" +
			help,
	)
}

func (m Model) currentFilterable() (FilterableView, bool) {
	if len(m.stack) == 0 {
		return nil, false
	}
	fv, ok := m.stack[len(m.stack)-1].(FilterableView)
	return fv, ok
}

func (m Model) currentCopyable() (CopyableView, bool) {
	if len(m.stack) == 0 {
		return nil, false
	}
	cv, ok := m.stack[len(m.stack)-1].(CopyableView)
	return cv, ok
}

func (m Model) applyFilter(fv FilterableView) {
	if m.filterQuery == "" {
		fv.SetRows(fv.AllRows())
		return
	}
	query := strings.ToLower(m.filterQuery)
	var filtered []table.Row
	for _, row := range fv.AllRows() {
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), query) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	fv.SetRows(filtered)
}

func (m Model) clearCopiedAfter() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearCopiedMsg{}
	})
}
```

**Step 2: Add missing import for table**

The `applyFilter` method uses `table.Row`, so ensure the import includes:
```go
"github.com/charmbracelet/bubbles/table"
```

(Already included in the full replacement above.)

**Step 3: Verify build**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/tui/services/model.go
git commit -m "feat: add filter and clipboard copy logic to root Model"
```

---

### Task 5: Implement FilterableView + CopyableView on EC2View

**Files:**
- Modify: `internal/tui/services/ec2.go`

**Step 1: Add allRows field and interface methods**

Add `allRows []table.Row` field to `EC2View`:

```go
type EC2View struct {
	client    *awsclient.ServiceClient
	instances []awsclient.EC2Instance
	summary   awsclient.EC2Summary
	table     table.Model
	spinner   spinner.Model
	loading   bool
	err       error
	allRows   []table.Row
}
```

In the `Update` method, after building `rows` in the `ec2DataMsg` case, store them:
```go
v.allRows = rows
```

Add these methods after the `View()` method:

```go
func (v *EC2View) AllRows() []table.Row { return v.allRows }
func (v *EC2View) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *EC2View) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.instances) {
		return v.instances[idx].InstanceID
	}
	return ""
}

func (v *EC2View) CopyARN() string {
	return v.CopyID()
}
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/tui/services/ec2.go
git commit -m "feat: implement FilterableView + CopyableView on EC2View"
```

---

### Task 6: Implement FilterableView + CopyableView on ECS Views

**Files:**
- Modify: `internal/tui/services/ecs.go`

**Step 1: Add allRows + interface methods to ECSClustersView**

Add `allRows []table.Row` field to `ECSClustersView`. In `Update`, after `v.table.SetRows(rows)`, add `v.allRows = rows`.

Add methods:

```go
func (v *ECSClustersView) AllRows() []table.Row { return v.allRows }
func (v *ECSClustersView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *ECSClustersView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.clusters) {
		return v.clusters[idx].Name
	}
	return ""
}

func (v *ECSClustersView) CopyARN() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.clusters) {
		return v.clusters[idx].ARN
	}
	return ""
}
```

**Step 2: Add allRows + interface methods to ECSServicesView**

Same pattern. Add `allRows []table.Row` field. Store in Update. Add methods:

```go
func (v *ECSServicesView) AllRows() []table.Row { return v.allRows }
func (v *ECSServicesView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *ECSServicesView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.services) {
		return v.services[idx].Name
	}
	return ""
}

func (v *ECSServicesView) CopyARN() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.services) {
		return v.services[idx].ARN
	}
	return ""
}
```

**Step 3: Add allRows + interface methods to ECSTasksView**

Same pattern. Add `allRows []table.Row` field. Store in Update. Add methods:

```go
func (v *ECSTasksView) AllRows() []table.Row { return v.allRows }
func (v *ECSTasksView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *ECSTasksView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.tasks) {
		return v.tasks[idx].TaskID
	}
	return ""
}

func (v *ECSTasksView) CopyARN() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.tasks) {
		return v.tasks[idx].ARN
	}
	return ""
}
```

**Step 4: Add Enter handler on ECSTasksView to push detail**

In `ECSTasksView.Update`, add an `"enter"` case in the `tea.KeyMsg` switch:

```go
case tea.KeyMsg:
    switch msg.String() {
    case "r":
        v.loading = true
        v.err = nil
        return v, tea.Batch(v.spinner.Tick, v.fetchData())
    case "enter":
        idx := v.table.Cursor()
        if idx >= 0 && idx < len(v.tasks) {
            task := v.tasks[idx]
            return v, func() tea.Msg {
                return PushViewMsg{View: NewTaskDetailView(v.client, v.clusterName, task.ARN)}
            }
        }
    }
```

Note: `NewTaskDetailView` doesn't exist yet — it will be created in Task 8. This is fine because Tasks 6–8 will be committed after Task 8 if needed to compile.

**Step 5: Verify build (may fail until Task 8)**

Run: `go build ./...`
Expected: May fail with "undefined: NewTaskDetailView" — this will be resolved in Task 8.

**Step 6: Commit (defer if build fails)**

```bash
git add internal/tui/services/ecs.go
git commit -m "feat: implement FilterableView + CopyableView on ECS views"
```

---

### Task 7: Implement FilterableView + CopyableView on VPC and ECR Views

**Files:**
- Modify: `internal/tui/services/vpc.go`
- Modify: `internal/tui/services/ecr.go`

**Step 1: VPCListView**

Add `allRows []table.Row` field. Store in Update after `v.table.SetRows(rows)`. Add:

```go
func (v *VPCListView) AllRows() []table.Row { return v.allRows }
func (v *VPCListView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *VPCListView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.vpcs) {
		return v.vpcs[idx].VPCID
	}
	return ""
}

func (v *VPCListView) CopyARN() string { return v.CopyID() }
```

**Step 2: SubnetsView**

Add `allRows []table.Row` field. Store in Update. Add:

```go
func (v *SubnetsView) AllRows() []table.Row { return v.allRows }
func (v *SubnetsView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *SubnetsView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.subnets) {
		return v.subnets[idx].SubnetID
	}
	return ""
}

func (v *SubnetsView) CopyARN() string { return v.CopyID() }
```

**Step 3: SecurityGroupsView**

Add `allRows []table.Row` field. Store in Update. Add:

```go
func (v *SecurityGroupsView) AllRows() []table.Row { return v.allRows }
func (v *SecurityGroupsView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *SecurityGroupsView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.sgs) {
		return v.sgs[idx].GroupID
	}
	return ""
}

func (v *SecurityGroupsView) CopyARN() string { return v.CopyID() }
```

**Step 4: IGWView**

Add `allRows []table.Row` field. Store in Update. Add:

```go
func (v *IGWView) AllRows() []table.Row { return v.allRows }
func (v *IGWView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *IGWView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.igws) {
		return v.igws[idx].GatewayID
	}
	return ""
}

func (v *IGWView) CopyARN() string { return v.CopyID() }
```

**Step 5: ECRReposView**

Add `allRows []table.Row` field. Store in Update. Add:

```go
func (v *ECRReposView) AllRows() []table.Row { return v.allRows }
func (v *ECRReposView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *ECRReposView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.repos) {
		return v.repos[idx].Name
	}
	return ""
}

func (v *ECRReposView) CopyARN() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.repos) {
		return v.repos[idx].URI
	}
	return ""
}
```

**Step 6: ECRImagesView**

Add `allRows []table.Row` field. Store in Update. Add:

```go
func (v *ECRImagesView) AllRows() []table.Row { return v.allRows }
func (v *ECRImagesView) SetRows(rows []table.Row) { v.table.SetRows(rows) }

func (v *ECRImagesView) CopyID() string {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.images) {
		return v.images[idx].Digest
	}
	return ""
}

func (v *ECRImagesView) CopyARN() string { return v.CopyID() }
```

**Step 7: Commit**

```bash
git add internal/tui/services/vpc.go internal/tui/services/ecr.go
git commit -m "feat: implement FilterableView + CopyableView on VPC and ECR views"
```

---

### Task 8: ECS Task Detail View

**Files:**
- Create: `internal/tui/services/detail.go`

**Step 1: Create TaskDetailView**

Create `internal/tui/services/detail.go`:

```go
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

type taskDetailMsg struct{ detail *awsclient.ECSTaskDetail }

type TaskDetailView struct {
	client      *awsclient.ServiceClient
	clusterName string
	taskARN     string
	detail      *awsclient.ECSTaskDetail
	viewport    viewport.Model
	spinner     spinner.Model
	loading     bool
	err         error
	ready       bool
}

func NewTaskDetailView(client *awsclient.ServiceClient, clusterName, taskARN string) *TaskDetailView {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &TaskDetailView{
		client:      client,
		clusterName: clusterName,
		taskARN:     taskARN,
		spinner:     sp,
		loading:     true,
	}
}

func (v *TaskDetailView) Title() string {
	if v.detail != nil {
		return v.detail.TaskID
	}
	return "Task"
}

func (v *TaskDetailView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}

func (v *TaskDetailView) fetchData() tea.Cmd {
	return func() tea.Msg {
		detail, err := v.client.ECS.DescribeTask(context.Background(), v.clusterName, v.taskARN)
		if err != nil {
			return errViewMsg{err: err}
		}
		return taskDetailMsg{detail: detail}
	}
}

func (v *TaskDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case taskDetailMsg:
		v.detail = msg.detail
		v.loading = false
		content := v.renderDetail()
		v.viewport = viewport.New(80, 20)
		v.viewport.SetContent(content)
		v.ready = true
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	if v.ready {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *TaskDetailView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading task details..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if v.ready {
		return v.viewport.View()
	}
	return ""
}

func (v *TaskDetailView) renderDetail() string {
	d := v.detail
	var b strings.Builder

	labelStyle := svcMutedStyle.Copy().Width(16)
	valueStyle := lipgloss.NewStyle()

	row := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render(label), valueStyle.Render(value)))
	}

	row("Task ARN", d.TaskARN)
	row("Status", d.Status)
	row("Task Def", d.TaskDef)

	if !d.StartedAt.IsZero() {
		row("Started", d.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if !d.StoppedAt.IsZero() {
		row("Stopped", d.StoppedAt.Format("2006-01-02 15:04:05"))
	}
	if d.StopCode != "" {
		row("Stop Code", d.StopCode)
	}
	if d.StopReason != "" {
		row("Stop Reason", d.StopReason)
	}

	row("CPU", d.CPU)
	row("Memory", d.Memory)

	if d.NetworkMode != "" {
		row("Network", d.NetworkMode)
	}
	if d.PrivateIP != "" {
		row("Private IP", d.PrivateIP)
	}
	if d.SubnetID != "" {
		row("Subnet", d.SubnetID)
	}

	// Containers
	if len(d.Containers) > 0 {
		b.WriteString("\n")
		b.WriteString(svcMutedStyle.Render("  ── Containers ──────────────────────────────") + "\n")
		for _, c := range d.Containers {
			b.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("[%s]", c.Name))))
			row("  Image", c.Image)
			row("  Status", c.Status)
			if c.ExitCode != nil {
				row("  Exit Code", fmt.Sprintf("%d", *c.ExitCode))
			}
			if c.HealthStatus != "" {
				row("  Health", c.HealthStatus)
			}
			if c.CPU > 0 {
				row("  CPU", fmt.Sprintf("%d", c.CPU))
			}
			if c.Memory > 0 {
				row("  Memory", fmt.Sprintf("%d", c.Memory))
			}
			if c.LogGroup != "" {
				row("  Log Group", c.LogGroup)
			}
			if c.LogStream != "" {
				row("  Log Stream", c.LogStream)
			}
		}
	}

	return b.String()
}

// CopyableView implementation
func (v *TaskDetailView) CopyID() string {
	if v.detail != nil {
		return v.detail.TaskID
	}
	return ""
}

func (v *TaskDetailView) CopyARN() string {
	if v.detail != nil {
		return v.detail.TaskARN
	}
	return ""
}
```

**Step 2: Verify full build**

Run: `go build ./...`
Expected: No errors (all views now exist)

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/tui/services/detail.go
git commit -m "feat: add ECS task detail view with container and log info"
```

---

### Task 9: Final Verification and Install

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Build binary**

Run: `go build -o aws-utils .`
Expected: Binary compiles

**Step 3: Verify help**

Run: `./aws-utils services --help`
Expected: Shows services command with --profile and --region flags

**Step 4: Clean up and commit any remaining changes**

```bash
rm ./aws-utils
git status
```

If clean, done. If any unstaged changes, commit them.

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Add ARN fields to ECS types | `service_types.go`, `ecs.go`, `ecs_test.go` |
| 2 | ECS task detail API + test | `service_types.go`, `ecs.go`, `ecs_detail_test.go` |
| 3 | FilterableView + CopyableView interfaces | `view.go` |
| 4 | Filter + copy logic in root Model | `model.go` |
| 5 | EC2 implements interfaces | `ec2.go` |
| 6 | ECS views implement interfaces + Enter→detail | `ecs.go` |
| 7 | VPC + ECR views implement interfaces | `vpc.go`, `ecr.go` |
| 8 | TaskDetailView with viewport | `detail.go` |
| 9 | Final verification | — |

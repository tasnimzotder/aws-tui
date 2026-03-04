package ecs

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// --- messages ---

type clustersLoadedMsg struct {
	clusters []ecs.ECSCluster
	err      error
}

type servicesLoadedMsg struct {
	services []ecs.ECSService
	err      error
}

type tasksLoadedMsg struct {
	tasks []ecs.ECSTask
	err   error
}

// --- status dot helper ---

func statusDot(status string) string {
	switch strings.ToUpper(status) {
	case "ACTIVE", "RUNNING":
		return "●"
	case "PROVISIONING", "PENDING", "ACTIVATING":
		return "◐"
	case "DRAINING", "DEPROVISIONING", "STOPPING":
		return "○"
	default:
		return "○"
	}
}

// =============================================================================
// ClusterListView
// =============================================================================

// ClusterListView shows a table of ECS clusters.
type ClusterListView struct {
	client   ECSClient
	router   plugin.Router
	table    ui.TableView[ecs.ECSCluster]
	loading  bool
	err      error
	skeleton ui.Skeleton
	region   string
	profile  string
}

// NewClusterListView creates a new cluster list view.
func NewClusterListView(client ECSClient, router plugin.Router, region, profile string) *ClusterListView {
	cols := []ui.Column[ecs.ECSCluster]{
		{Title: "Status", Width: 6, Field: func(c ecs.ECSCluster) string { return statusDot(c.Status) }},
		{Title: "Name", Width: 30, Field: func(c ecs.ECSCluster) string { return c.Name }},
		{Title: "Services", Width: 10, Field: func(c ecs.ECSCluster) string { return fmt.Sprintf("%d", c.ServiceCount) }},
		{Title: "Running", Width: 10, Field: func(c ecs.ECSCluster) string { return fmt.Sprintf("%d", c.RunningTaskCount) }},
	}
	return &ClusterListView{
		client:   client,
		router:   router,
		table:    ui.NewTableView(cols, nil, func(c ecs.ECSCluster) string { return c.ARN }),
		loading:  true,
		skeleton: ui.NewSkeleton(60, 5),
		region:   region,
		profile:  profile,
	}
}

func (v *ClusterListView) Title() string { return "ECS Clusters" }

func (v *ClusterListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view services"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

func (v *ClusterListView) Init() tea.Cmd {
	return v.fetchClusters()
}

func (v *ClusterListView) fetchClusters() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		clusters, err := v.client.ListClusters(ctx)
		return clustersLoadedMsg{clusters: clusters, err: err}
	}
}

func (v *ClusterListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clustersLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.table.SetItems(msg.clusters)
		return v, nil

	case tea.KeyPressMsg:
		if v.loading {
			return v, nil
		}
		switch msg.String() {
		case "enter":
			selected := v.table.SelectedItem()
			if selected.Name != "" {
				view := NewServiceListView(v.client, v.router, selected.Name, v.region, v.profile)
				v.router.Push(view)
				return v, view.Init()
			}
			return v, nil
		case "esc", "backspace":
			v.router.Pop()
			return v, nil
		case "r":
			v.loading = true
			return v, v.fetchClusters()
		}

		var cmd tea.Cmd
		v.table, cmd = v.table.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *ClusterListView) View() tea.View {
	if v.loading {
		return tea.NewView(v.skeleton.View())
	}
	if v.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v", v.err))
	}
	if v.table.FilteredCount() == 0 {
		return tea.NewView("No ECS clusters found.")
	}
	return tea.NewView(v.table.View())
}

// =============================================================================
// ServiceListView
// =============================================================================

// ServiceListView shows a table of ECS services within a cluster.
type ServiceListView struct {
	client      ECSClient
	router      plugin.Router
	clusterName string
	table       ui.TableView[ecs.ECSService]
	loading     bool
	err         error
	skeleton    ui.Skeleton
	region      string
	profile     string
}

// NewServiceListView creates a service list view for the given cluster.
func NewServiceListView(client ECSClient, router plugin.Router, clusterName, region, profile string) *ServiceListView {
	cols := []ui.Column[ecs.ECSService]{
		{Title: "Status", Width: 6, Field: func(s ecs.ECSService) string { return statusDot(s.Status) }},
		{Title: "Name", Width: 30, Field: func(s ecs.ECSService) string { return s.Name }},
		{Title: "Task Def", Width: 25, Field: func(s ecs.ECSService) string { return s.TaskDef }},
		{Title: "Desired", Width: 8, Field: func(s ecs.ECSService) string { return fmt.Sprintf("%d", s.DesiredCount) }},
		{Title: "Running", Width: 8, Field: func(s ecs.ECSService) string { return fmt.Sprintf("%d", s.RunningCount) }},
		{Title: "Pending", Width: 8, Field: func(s ecs.ECSService) string { return fmt.Sprintf("%d", s.PendingCount) }},
	}
	return &ServiceListView{
		client:      client,
		router:      router,
		clusterName: clusterName,
		table:       ui.NewTableView(cols, nil, func(s ecs.ECSService) string { return s.ARN }),
		loading:     true,
		skeleton:    ui.NewSkeleton(60, 5),
		region:      region,
		profile:     profile,
	}
}

func (v *ServiceListView) Title() string {
	return fmt.Sprintf("Services — %s", v.clusterName)
}

func (v *ServiceListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view tasks"},
		{Key: "esc", Desc: "back"},
		{Key: "/", Desc: "filter"},
	}
}

func (v *ServiceListView) Init() tea.Cmd {
	return v.fetchServices()
}

func (v *ServiceListView) fetchServices() tea.Cmd {
	cluster := v.clusterName
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		services, err := v.client.ListServices(ctx, cluster)
		return servicesLoadedMsg{services: services, err: err}
	}
}

func (v *ServiceListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case servicesLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.table.SetItems(msg.services)
		return v, nil

	case tea.KeyPressMsg:
		if v.loading {
			return v, nil
		}
		switch msg.String() {
		case "enter":
			selected := v.table.SelectedItem()
			if selected.Name != "" {
				view := NewTaskListView(v.client, v.router, v.clusterName, selected.Name, v.region, v.profile)
				v.router.Push(view)
				return v, view.Init()
			}
			return v, nil
		case "esc", "backspace":
			v.router.Pop()
			return v, nil
		case "r":
			v.loading = true
			return v, v.fetchServices()
		}

		var cmd tea.Cmd
		v.table, cmd = v.table.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *ServiceListView) View() tea.View {
	if v.loading {
		return tea.NewView(v.skeleton.View())
	}
	if v.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v", v.err))
	}
	if v.table.FilteredCount() == 0 {
		return tea.NewView("No services found in this cluster.")
	}
	return tea.NewView(v.table.View())
}

// =============================================================================
// TaskListView
// =============================================================================

// TaskListView shows a table of ECS tasks within a service.
type TaskListView struct {
	client      ECSClient
	router      plugin.Router
	clusterName string
	serviceName string
	table       ui.TableView[ecs.ECSTask]
	loading     bool
	err         error
	skeleton    ui.Skeleton
	region      string
	profile     string
}

// NewTaskListView creates a task list view for the given cluster and service.
func NewTaskListView(client ECSClient, router plugin.Router, clusterName, serviceName, region, profile string) *TaskListView {
	cols := []ui.Column[ecs.ECSTask]{
		{Title: "Status", Width: 6, Field: func(t ecs.ECSTask) string { return statusDot(t.Status) }},
		{Title: "Task ID", Width: 38, Field: func(t ecs.ECSTask) string { return t.TaskID }},
		{Title: "Task Def", Width: 25, Field: func(t ecs.ECSTask) string { return t.TaskDef }},
		{Title: "Health", Width: 10, Field: func(t ecs.ECSTask) string { return t.HealthStatus }},
		{Title: "Started", Width: 20, Field: func(t ecs.ECSTask) string {
			if t.StartedAt.IsZero() {
				return "-"
			}
			return t.StartedAt.Format("2006-01-02 15:04")
		}},
	}
	return &TaskListView{
		client:      client,
		router:      router,
		clusterName: clusterName,
		serviceName: serviceName,
		table:       ui.NewTableView(cols, nil, func(t ecs.ECSTask) string { return t.ARN }),
		loading:     true,
		skeleton:    ui.NewSkeleton(60, 5),
		region:      region,
		profile:     profile,
	}
}

func (v *TaskListView) Title() string {
	return fmt.Sprintf("Tasks — %s/%s", v.clusterName, v.serviceName)
}

func (v *TaskListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view detail"},
		{Key: "esc", Desc: "back"},
		{Key: "/", Desc: "filter"},
	}
}

func (v *TaskListView) Init() tea.Cmd {
	return v.fetchTasks()
}

func (v *TaskListView) fetchTasks() tea.Cmd {
	cluster, service := v.clusterName, v.serviceName
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tasks, err := v.client.ListTasks(ctx, cluster, service)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (v *TaskListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tasksLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.table.SetItems(msg.tasks)
		return v, nil

	case tea.KeyPressMsg:
		if v.loading {
			return v, nil
		}
		switch msg.String() {
		case "enter":
			selected := v.table.SelectedItem()
			if selected.ARN != "" {
				v.router.NavigateDetail("ecs", v.clusterName+"/"+selected.ARN)
			}
			return v, nil
		case "esc", "backspace":
			v.router.Pop()
			return v, nil
		case "r":
			v.loading = true
			return v, v.fetchTasks()
		}

		var cmd tea.Cmd
		v.table, cmd = v.table.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *TaskListView) View() tea.View {
	if v.loading {
		return tea.NewView(v.skeleton.View())
	}
	if v.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v", v.err))
	}
	if v.table.FilteredCount() == 0 {
		return tea.NewView("No tasks found for this service.")
	}
	return tea.NewView(v.table.View())
}

package ecs

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

type ecsExecFinishedMsg struct{ err error }

// --- messages ---

type serviceDetailLoadedMsg struct {
	detail *ecs.ECSServiceDetail
	err    error
}

type taskDetailLoadedMsg struct {
	detail *ecs.ECSTaskDetail
	err    error
}

// =============================================================================
// DetailView
// =============================================================================

// DetailView shows detailed information about a cluster/service or task.
// The id format is "cluster/taskARN" for tasks, or "cluster/serviceName" for services.
type DetailView struct {
	client  ECSClient
	router  plugin.Router
	id      string
	tabs    ui.TabController
	loading bool
	err     error
	region  string
	profile string

	serviceDetail *ecs.ECSServiceDetail
	taskDetail    *ecs.ECSTaskDetail
	isTask        bool
}

// NewDetailView creates a detail view. The id is expected in format "cluster/resourceID".
func NewDetailView(client ECSClient, router plugin.Router, id, region, profile string) *DetailView {
	// Determine if this is a task (ARN contains ":task/") or a service
	isTask := strings.Contains(id, ":task/")

	var tabTitles []string
	if isTask {
		tabTitles = []string{"Overview", "Containers"}
	} else {
		tabTitles = []string{"Overview", "Deployments", "Events"}
	}

	return &DetailView{
		client:  client,
		router:  router,
		id:      id,
		tabs:    ui.NewTabController(tabTitles),
		loading: true,
		isTask:  isTask,
		region:  region,
		profile: profile,
	}
}

func (v *DetailView) Title() string {
	if v.taskDetail != nil {
		return fmt.Sprintf("Task — %s", v.taskDetail.TaskID)
	}
	if v.serviceDetail != nil {
		return fmt.Sprintf("Service — %s", v.serviceDetail.Name)
	}
	return "ECS Detail"
}

func (v *DetailView) KeyHints() []plugin.KeyHint {
	hints := []plugin.KeyHint{
		{Key: "[/]", Desc: "switch tab"},
		{Key: "esc", Desc: "back"},
	}
	if v.isTask && v.taskDetail != nil && v.taskDetail.Status == "RUNNING" {
		hints = append(hints, plugin.KeyHint{Key: "x", Desc: "exec into task"})
	}
	return hints
}

// parseID splits "cluster/resource" into cluster and resource parts.
func parseID(id string) (cluster, resource string) {
	idx := strings.Index(id, "/")
	if idx < 0 {
		return id, ""
	}
	return id[:idx], id[idx+1:]
}

func (v *DetailView) Init() tea.Cmd {
	cluster, resource := parseID(v.id)
	if v.isTask {
		return v.fetchTaskDetail(cluster, resource)
	}
	return v.fetchServiceDetail(cluster, resource)
}

func (v *DetailView) fetchServiceDetail(cluster, service string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		detail, err := v.client.DescribeService(ctx, cluster, service)
		return serviceDetailLoadedMsg{detail: detail, err: err}
	}
}

func (v *DetailView) fetchTaskDetail(cluster, taskARN string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		detail, err := v.client.DescribeTask(ctx, cluster, taskARN)
		return taskDetailLoadedMsg{detail: detail, err: err}
	}
}

func (v *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case serviceDetailLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.serviceDetail = msg.detail
		return v, nil

	case taskDetailLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.taskDetail = msg.detail
		return v, nil

	case ecsExecFinishedMsg:
		if msg.err != nil {
			v.router.Toast(plugin.ToastError, "ECS exec failed: "+msg.err.Error())
		}
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			v.router.Pop()
			return v, nil
		case "x":
			if v.isTask && v.taskDetail != nil && v.taskDetail.Status == "RUNNING" && len(v.taskDetail.Containers) > 0 {
				return v, v.execTask()
			}
			return v, nil
		}

		var cmd tea.Cmd
		v.tabs, cmd = v.tabs.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *DetailView) execTask() tea.Cmd {
	cluster, _ := parseID(v.id)
	container := v.taskDetail.Containers[0].Name
	args := []string{
		"ecs", "execute-command",
		"--cluster", cluster,
		"--task", v.taskDetail.TaskARN,
		"--container", container,
		"--interactive",
		"--command", "/bin/sh",
	}
	if v.region != "" {
		args = append(args, "--region", v.region)
	}
	if v.profile != "" {
		args = append(args, "--profile", v.profile)
	}
	c := exec.Command("aws", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ecsExecFinishedMsg{err: err}
	})
}

func (v *DetailView) View() tea.View {
	if v.loading {
		skel := ui.NewSkeleton(60, 8)
		return tea.NewView(skel.View())
	}
	if v.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v", v.err))
	}

	var b strings.Builder
	b.WriteString(v.tabs.View())
	b.WriteString("\n\n")

	if v.isTask && v.taskDetail != nil {
		b.WriteString(v.renderTaskTab())
	} else if v.serviceDetail != nil {
		b.WriteString(v.renderServiceTab())
	}

	return tea.NewView(b.String())
}

// --- service tab rendering ---

func (v *DetailView) renderServiceTab() string {
	d := v.serviceDetail
	switch v.tabs.Active() {
	case 0: // Overview
		return v.renderServiceOverview(d)
	case 1: // Deployments
		return v.renderDeployments(d)
	case 2: // Events
		return v.renderEvents(d)
	}
	return ""
}

func (v *DetailView) renderServiceOverview(d *ecs.ECSServiceDetail) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Name:         %s\n", d.Name))
	b.WriteString(fmt.Sprintf("Status:       %s\n", d.Status))
	b.WriteString(fmt.Sprintf("Launch Type:  %s\n", d.LaunchType))
	b.WriteString(fmt.Sprintf("Task Def:     %s\n", d.TaskDef))
	b.WriteString(fmt.Sprintf("Desired:      %d\n", d.DesiredCount))
	b.WriteString(fmt.Sprintf("Running:      %d\n", d.RunningCount))
	b.WriteString(fmt.Sprintf("Pending:      %d\n", d.PendingCount))
	b.WriteString(fmt.Sprintf("Exec Command: %v\n", d.EnableExecuteCommand))

	if len(d.LoadBalancers) > 0 {
		b.WriteString("\nLoad Balancers:\n")
		for _, lb := range d.LoadBalancers {
			b.WriteString(fmt.Sprintf("  %s → %s:%d\n", lb.TargetGroupARN, lb.ContainerName, lb.ContainerPort))
		}
	}

	return b.String()
}

func (v *DetailView) renderDeployments(d *ecs.ECSServiceDetail) string {
	if len(d.Deployments) == 0 {
		return "No deployments."
	}
	var b strings.Builder
	for _, dep := range d.Deployments {
		b.WriteString(fmt.Sprintf("ID:       %s\n", dep.ID))
		b.WriteString(fmt.Sprintf("Status:   %s\n", dep.Status))
		b.WriteString(fmt.Sprintf("Rollout:  %s\n", dep.RolloutState))
		b.WriteString(fmt.Sprintf("Task Def: %s\n", dep.TaskDef))
		b.WriteString(fmt.Sprintf("Desired:  %d  Running: %d  Pending: %d\n", dep.DesiredCount, dep.RunningCount, dep.PendingCount))
		if !dep.CreatedAt.IsZero() {
			b.WriteString(fmt.Sprintf("Created:  %s\n", dep.CreatedAt.Format(time.RFC3339)))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (v *DetailView) renderEvents(d *ecs.ECSServiceDetail) string {
	if len(d.Events) == 0 {
		return "No recent events."
	}
	var b strings.Builder
	limit := len(d.Events)
	if limit > 20 {
		limit = 20
	}
	for _, ev := range d.Events[:limit] {
		ts := ev.CreatedAt.Format("15:04:05")
		b.WriteString(fmt.Sprintf("[%s] %s\n", ts, ev.Message))
	}
	return b.String()
}

// --- task tab rendering ---

func (v *DetailView) renderTaskTab() string {
	d := v.taskDetail
	switch v.tabs.Active() {
	case 0: // Overview
		return v.renderTaskOverview(d)
	case 1: // Containers
		return v.renderContainers(d)
	}
	return ""
}

func (v *DetailView) renderTaskOverview(d *ecs.ECSTaskDetail) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Task ID:      %s\n", d.TaskID))
	b.WriteString(fmt.Sprintf("Status:       %s\n", d.Status))
	b.WriteString(fmt.Sprintf("Task Def:     %s\n", d.TaskDef))
	b.WriteString(fmt.Sprintf("CPU:          %s\n", d.CPU))
	b.WriteString(fmt.Sprintf("Memory:       %s\n", d.Memory))
	b.WriteString(fmt.Sprintf("Network:      %s\n", d.NetworkMode))
	b.WriteString(fmt.Sprintf("Private IP:   %s\n", d.PrivateIP))
	b.WriteString(fmt.Sprintf("Subnet:       %s\n", d.SubnetID))

	if !d.StartedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Started:      %s\n", d.StartedAt.Format(time.RFC3339)))
	}
	if !d.StoppedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Stopped:      %s\n", d.StoppedAt.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("Stop Code:    %s\n", d.StopCode))
		b.WriteString(fmt.Sprintf("Stop Reason:  %s\n", d.StopReason))
	}

	return b.String()
}

func (v *DetailView) renderContainers(d *ecs.ECSTaskDetail) string {
	if len(d.Containers) == 0 {
		return "No containers."
	}
	var b strings.Builder
	for _, c := range d.Containers {
		b.WriteString(fmt.Sprintf("Name:    %s\n", c.Name))
		b.WriteString(fmt.Sprintf("Image:   %s\n", c.Image))
		b.WriteString(fmt.Sprintf("Status:  %s\n", c.Status))
		b.WriteString(fmt.Sprintf("Health:  %s\n", c.HealthStatus))
		if c.ExitCode != nil {
			b.WriteString(fmt.Sprintf("Exit:    %d\n", *c.ExitCode))
		}
		if c.LogGroup != "" {
			b.WriteString(fmt.Sprintf("Logs:    %s / %s\n", c.LogGroup, c.LogStream))
		}
		b.WriteString("\n")
	}
	return b.String()
}

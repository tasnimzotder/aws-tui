package services

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsecs "tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// --- Service Config View (viewport-based) ---

type ecsServiceConfigMsg struct {
	detail *awsecs.ECSServiceDetail
}

type ECSServiceConfigView struct {
	client      *awsclient.ServiceClient
	clusterName string
	serviceName string
	serviceARN  string
	viewport    viewport.Model
	spinner     spinner.Model
	loading     bool
	err         error
	ready       bool
	width       int
	height      int
}

func NewECSServiceConfigView(client *awsclient.ServiceClient, clusterName, serviceName, serviceARN string) *ECSServiceConfigView {
	return &ECSServiceConfigView{
		client:      client,
		clusterName: clusterName,
		serviceName: serviceName,
		serviceARN:  serviceARN,
		spinner:     theme.NewSpinner(),
		loading:     true,
		width:       80,
		height:      20,
	}
}

func (v *ECSServiceConfigView) Title() string { return "Config" }
func (v *ECSServiceConfigView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}

func (v *ECSServiceConfigView) fetchData() tea.Cmd {
	return func() tea.Msg {
		detail, err := v.client.ECS.DescribeService(context.Background(), v.clusterName, v.serviceName)
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecsServiceConfigMsg{detail: detail}
	}
}

func (v *ECSServiceConfigView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecsServiceConfigMsg:
		v.loading = false
		v.viewport = viewport.New(viewport.WithWidth(v.width), viewport.WithHeight(v.height))
		v.viewport.SetContent(v.renderContent(msg.detail))
		v.ready = true
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tea.KeyPressMsg:
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

func (v *ECSServiceConfigView) renderContent(d *awsecs.ECSServiceDetail) string {
	db := utils.NewDetailBuilder(24, theme.MutedStyle)

	db.Section("Service Configuration")
	db.Row("Service Name", d.Name)
	db.Row("ARN", d.ARN)
	db.Row("Status", d.Status)
	db.Row("Task Definition", d.TaskDef)
	db.Row("Launch Type", d.LaunchType)

	execCmd := "Disabled"
	if d.EnableExecuteCommand {
		execCmd = "Enabled"
	}
	db.Row("Execute Command", execCmd)

	db.Row("Desired Count", fmt.Sprintf("%d", d.DesiredCount))
	db.Row("Running Count", fmt.Sprintf("%d", d.RunningCount))
	db.Row("Pending Count", fmt.Sprintf("%d", d.PendingCount))
	db.Blank()

	if len(d.PlacementConstraints) > 0 {
		db.Section("Placement Constraints")
		for _, pc := range d.PlacementConstraints {
			db.Row("Type", pc.Type)
			if pc.Expression != "" {
				db.Row("Expression", pc.Expression)
			}
		}
		db.Blank()
	}

	if len(d.PlacementStrategy) > 0 {
		db.Section("Placement Strategy")
		for _, ps := range d.PlacementStrategy {
			db.Row("Type", ps.Type)
			if ps.Field != "" {
				db.Row("Field", ps.Field)
			}
		}
		db.Blank()
	}

	return db.String()
}

func (v *ECSServiceConfigView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading service configuration..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if v.ready {
		return v.viewport.View()
	}
	return ""
}

func (v *ECSServiceConfigView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.viewport.SetWidth(width)
		v.viewport.SetHeight(height)
	}
}

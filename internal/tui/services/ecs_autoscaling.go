package services

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsautoscaling "tasnim.dev/aws-tui/internal/aws/autoscaling"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// --- Auto Scaling View (viewport-based) ---

type ecsAutoScalingMsg struct {
	targets  []awsautoscaling.AutoScalingTarget
	policies []awsautoscaling.AutoScalingPolicy
}

type ECSAutoScalingView struct {
	client      *awsclient.ServiceClient
	clusterName string
	serviceName string
	viewport    viewport.Model
	spinner     spinner.Model
	loading     bool
	err         error
	ready       bool
	width       int
	height      int
}

func NewECSAutoScalingView(client *awsclient.ServiceClient, clusterName, serviceName string) *ECSAutoScalingView {
	return &ECSAutoScalingView{
		client:      client,
		clusterName: clusterName,
		serviceName: serviceName,
		spinner:     theme.NewSpinner(),
		loading:     true,
		width:       80,
		height:      20,
	}
}

func (v *ECSAutoScalingView) Title() string { return "Auto Scaling" }
func (v *ECSAutoScalingView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}

func (v *ECSAutoScalingView) fetchData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		targets, err := v.client.AutoScaling.GetECSScalingTargets(ctx, v.clusterName, v.serviceName)
		if err != nil {
			return errViewMsg{err: err}
		}
		policies, err := v.client.AutoScaling.GetECSScalingPolicies(ctx, v.clusterName, v.serviceName)
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecsAutoScalingMsg{targets: targets, policies: policies}
	}
}

func (v *ECSAutoScalingView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecsAutoScalingMsg:
		v.loading = false
		v.viewport = viewport.New(viewport.WithWidth(v.width), viewport.WithHeight(v.height))
		v.viewport.SetContent(v.renderContent(msg.targets, msg.policies))
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

func (v *ECSAutoScalingView) renderContent(targets []awsautoscaling.AutoScalingTarget, policies []awsautoscaling.AutoScalingPolicy) string {
	if len(targets) == 0 && len(policies) == 0 {
		return theme.MutedStyle.Render("  No auto scaling configured")
	}

	db := utils.NewDetailBuilder(16, theme.MutedStyle)

	for _, t := range targets {
		db.Section("Scaling Target")
		db.Row("Resource", t.ResourceID)
		db.Row("Min Capacity", fmt.Sprintf("%d", t.MinCapacity))
		db.Row("Max Capacity", fmt.Sprintf("%d", t.MaxCapacity))
		db.Blank()
	}

	for _, p := range policies {
		db.Section("Scaling Policy")
		db.Row("Policy Name", p.PolicyName)
		db.Row("Policy Type", p.PolicyType)
		if p.TargetValue > 0 {
			db.Row("Target Value", fmt.Sprintf("%.1f", p.TargetValue))
		}
		if p.MetricName != "" {
			db.Row("Metric", p.MetricName)
		}
		db.Blank()
	}

	return db.String()
}

func (v *ECSAutoScalingView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading auto scaling configuration..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if v.ready {
		return v.viewport.View()
	}
	return ""
}

func (v *ECSAutoScalingView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.viewport.SetWidth(width)
		v.viewport.SetHeight(height)
	}
}

func (v *ECSAutoScalingView) CopyID() string  { return v.serviceName }
func (v *ECSAutoScalingView) CopyARN() string { return "" }

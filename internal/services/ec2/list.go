package ec2

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// instancesMsg carries the result of fetching instances.
type instancesMsg struct {
	instances []awsec2.EC2Instance
	err       error
}

// ListView displays EC2 instances in a table.
type ListView struct {
	client  EC2Client
	router  plugin.Router
	table   ui.TableView[awsec2.EC2Instance]
	loading bool
	err     error
	region  string
	profile string
}

// NewListView creates a new EC2 ListView.
func NewListView(client EC2Client, router plugin.Router, region, profile string) *ListView {
	cols := ec2Columns()
	tv := ui.NewTableView(cols, nil, func(i awsec2.EC2Instance) string {
		return i.InstanceID
	})
	return &ListView{
		client:  client,
		router:  router,
		table:   tv,
		loading: true,
		region:  region,
		profile: profile,
	}
}

func ec2Columns() []ui.Column[awsec2.EC2Instance] {
	return []ui.Column[awsec2.EC2Instance]{
		{Title: "State", Width: 3, Field: func(i awsec2.EC2Instance) string {
			return stateDot(i.State)
		}},
		{Title: "Name", Width: 24, Field: func(i awsec2.EC2Instance) string { return i.Name }},
		{Title: "Instance ID", Width: 20, Field: func(i awsec2.EC2Instance) string { return i.InstanceID }},
		{Title: "Type", Width: 14, Field: func(i awsec2.EC2Instance) string { return i.Type }},
		{Title: "AZ", Width: 14, Field: func(i awsec2.EC2Instance) string { return i.AZ }},
		{Title: "Private IP", Width: 16, Field: func(i awsec2.EC2Instance) string { return i.PrivateIP }},
		{Title: "Public IP", Width: 16, Field: func(i awsec2.EC2Instance) string { return i.PublicIP }},
	}
}

var (
	greenDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
	yellowDot = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●")
	redDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("●")
	grayDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("●")
)

func stateDot(state string) string {
	switch state {
	case "running":
		return greenDot
	case "pending", "stopping":
		return yellowDot
	case "stopped", "terminated", "shutting-down":
		return redDot
	default:
		return grayDot
	}
}

func (lv *ListView) fetchInstances() tea.Cmd {
	client := lv.client
	return func() tea.Msg {
		instances, _, err := client.ListInstances(context.TODO())
		return instancesMsg{instances: instances, err: err}
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchInstances()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case instancesMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.table.SetItems(msg.instances)
		return lv, nil

	case tea.KeyPressMsg:
		if lv.loading {
			return lv, nil
		}

		switch msg.String() {
		case "enter":
			if id := lv.table.SelectedID(); id != "" {
				view := NewDetailView(lv.client, lv.router, id, lv.region, lv.profile)
				lv.router.Push(view)
				return lv, view.Init()
			}
			return lv, nil
		case "esc", "backspace":
			lv.router.Pop()
			return lv, nil
		case "r":
			lv.loading = true
			return lv, lv.fetchInstances()
		}
	}

	var cmd tea.Cmd
	lv.table, cmd = lv.table.Update(msg)
	return lv, cmd
}

func (lv *ListView) View() tea.View {
	if lv.loading {
		skel := ui.NewSkeleton(80, 6)
		return tea.NewView(skel.View())
	}
	if lv.err != nil {
		return tea.NewView("Error: " + lv.err.Error())
	}
	return tea.NewView(lv.table.View())
}

func (lv *ListView) Title() string { return "EC2 Instances" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

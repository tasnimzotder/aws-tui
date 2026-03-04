package elb

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// loadBalancersMsg carries the result of fetching load balancers.
type loadBalancersMsg struct {
	lbs []awselb.ELBLoadBalancer
	err error
}

// ListView displays ELB load balancers in a table.
type ListView struct {
	client  *awselb.Client
	router  plugin.Router
	table   ui.TableView[awselb.ELBLoadBalancer]
	loading bool
	err     error
}

// NewListView creates a new ELB ListView.
func NewListView(client *awselb.Client, router plugin.Router) *ListView {
	cols := elbColumns()
	tv := ui.NewTableView(cols, nil, func(lb awselb.ELBLoadBalancer) string {
		return lb.ARN
	})
	return &ListView{
		client:  client,
		router:  router,
		table:   tv,
		loading: true,
	}
}

var (
	greenDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
	yellowDot = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●")
	redDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("●")
	grayDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("●")
)

func lbStateDot(state string) string {
	switch state {
	case "active":
		return greenDot
	case "provisioning":
		return yellowDot
	case "failed":
		return redDot
	default:
		return grayDot
	}
}

func elbColumns() []ui.Column[awselb.ELBLoadBalancer] {
	return []ui.Column[awselb.ELBLoadBalancer]{
		{Title: "State", Width: 3, Field: func(lb awselb.ELBLoadBalancer) string {
			return lbStateDot(lb.State)
		}},
		{Title: "Name", Width: 28, Field: func(lb awselb.ELBLoadBalancer) string { return lb.Name }},
		{Title: "Type", Width: 12, Field: func(lb awselb.ELBLoadBalancer) string { return lb.Type }},
		{Title: "Scheme", Width: 16, Field: func(lb awselb.ELBLoadBalancer) string { return lb.Scheme }},
		{Title: "DNS Name", Width: 48, Field: func(lb awselb.ELBLoadBalancer) string { return lb.DNSName }},
		{Title: "VPC", Width: 22, Field: func(lb awselb.ELBLoadBalancer) string { return lb.VPCID }},
		{Title: "Created", Width: 20, Field: func(lb awselb.ELBLoadBalancer) string {
			if lb.CreatedAt.IsZero() {
				return "—"
			}
			return lb.CreatedAt.Format("2006-01-02 15:04")
		}},
	}
}

func (lv *ListView) fetchLoadBalancers() tea.Cmd {
	client := lv.client
	return func() tea.Msg {
		lbs, err := client.ListLoadBalancers(context.Background())
		return loadBalancersMsg{lbs: lbs, err: err}
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchLoadBalancers()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadBalancersMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.table.SetItems(msg.lbs)
		return lv, nil

	case tea.KeyPressMsg:
		if lv.loading {
			return lv, nil
		}

		switch msg.String() {
		case "enter":
			if id := lv.table.SelectedID(); id != "" {
				view := NewDetailView(lv.client, lv.router, id)
				lv.router.Push(view)
				return lv, view.Init()
			}
			return lv, nil
		case "esc", "backspace":
			lv.router.Pop()
			return lv, nil
		case "r":
			lv.loading = true
			return lv, lv.fetchLoadBalancers()
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
		return tea.NewView(fmt.Sprintf("Error: %s", lv.err.Error()))
	}
	return tea.NewView(lv.table.View())
}

func (lv *ListView) Title() string { return "Load Balancers" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

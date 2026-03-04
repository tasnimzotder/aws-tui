package elb

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// detailLoadedMsg carries the result of loading LB detail data.
type detailLoadedMsg struct {
	lb         awselb.ELBLoadBalancer
	listeners  []awselb.ELBListener
	tgs        []awselb.ELBTargetGroup
	attributes []awselb.ELBAttribute
	err        error
}

// DetailView shows detailed information for a single load balancer.
type DetailView struct {
	client     *awselb.Client
	router     plugin.Router
	lbARN      string
	lb         *awselb.ELBLoadBalancer
	listeners  []awselb.ELBListener
	tgs        []awselb.ELBTargetGroup
	attributes []awselb.ELBAttribute
	tabs       ui.TabController
	loading    bool
	err        error
}

// NewDetailView creates a DetailView for the given load balancer ARN.
func NewDetailView(client *awselb.Client, router plugin.Router, lbARN string) *DetailView {
	return &DetailView{
		client:  client,
		router:  router,
		lbARN:   lbARN,
		tabs:    ui.NewTabController([]string{"Overview", "Listeners", "Target Groups", "Attributes"}),
		loading: true,
	}
}

func (dv *DetailView) loadDetail() tea.Cmd {
	client := dv.client
	lbARN := dv.lbARN
	return func() tea.Msg {
		ctx := context.Background()

		lbs, err := client.ListLoadBalancers(ctx)
		if err != nil {
			return detailLoadedMsg{err: err}
		}

		var found *awselb.ELBLoadBalancer
		for _, lb := range lbs {
			if lb.ARN == lbARN {
				found = &lb
				break
			}
		}
		if found == nil {
			return detailLoadedMsg{err: fmt.Errorf("load balancer not found: %s", lbARN)}
		}

		listeners, _ := client.ListListeners(ctx, lbARN)
		tgs, _ := client.ListTargetGroups(ctx, lbARN)
		attrs, _ := client.GetLoadBalancerAttributes(ctx, lbARN)

		return detailLoadedMsg{lb: *found, listeners: listeners, tgs: tgs, attributes: attrs}
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.loadDetail()
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detailLoadedMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.lb = &msg.lb
		dv.listeners = msg.listeners
		dv.tgs = msg.tgs
		dv.attributes = msg.attributes
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		case "enter":
			// Drill into target group to see targets.
			if dv.tabs.Active() == 2 && len(dv.tgs) > 0 {
				view := NewTGDetailView(dv.client, dv.router, dv.tgs)
				dv.router.Push(view)
				return dv, view.Init()
			}
			return dv, nil
		}
	}

	var cmd tea.Cmd
	dv.tabs, cmd = dv.tabs.Update(msg)
	return dv, cmd
}

func (dv *DetailView) View() tea.View {
	if dv.loading {
		skel := ui.NewSkeleton(60, 8)
		return tea.NewView(skel.View())
	}
	if dv.err != nil {
		return tea.NewView("Error: " + dv.err.Error())
	}

	var b strings.Builder
	b.WriteString(dv.tabs.View())
	b.WriteString("\n\n")

	switch dv.tabs.Active() {
	case 0:
		b.WriteString(dv.renderOverview())
	case 1:
		b.WriteString(dv.renderListeners())
	case 2:
		b.WriteString(dv.renderTargetGroups())
	case 3:
		b.WriteString(dv.renderAttributes())
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderOverview() string {
	lb := dv.lb
	rows := []ui.KV{
		{"Name", lb.Name},
		{"ARN", lb.ARN},
		{"Type", lb.Type},
		{"State", lb.State},
		{"Scheme", lb.Scheme},
		{"DNS Name", lb.DNSName},
		{"VPC", lb.VPCID},
		{"Listeners", fmt.Sprintf("%d", len(dv.listeners))},
		{"Target Groups", fmt.Sprintf("%d", len(dv.tgs))},
		{"Created", lb.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
	}
	return ui.RenderKV(rows, 20, 0)
}

func (dv *DetailView) renderListeners() string {
	if len(dv.listeners) == 0 {
		return "No listeners configured."
	}

	cols := []ui.Column[awselb.ELBListener]{
		{Title: "Port", Width: 8, Field: func(l awselb.ELBListener) string {
			return fmt.Sprintf("%d", l.Port)
		}},
		{Title: "Protocol", Width: 10, Field: func(l awselb.ELBListener) string { return l.Protocol }},
		{Title: "Default Action", Width: 40, Field: func(l awselb.ELBListener) string { return l.DefaultAction }},
		{Title: "SSL Policy", Width: 28, Field: func(l awselb.ELBListener) string {
			if l.SSLPolicy == "" {
				return "—"
			}
			return l.SSLPolicy
		}},
	}
	tv := ui.NewTableView(cols, dv.listeners, func(l awselb.ELBListener) string { return l.ARN })
	return tv.View()
}

func (dv *DetailView) renderTargetGroups() string {
	if len(dv.tgs) == 0 {
		return "No target groups attached."
	}

	cols := []ui.Column[awselb.ELBTargetGroup]{
		{Title: "Name", Width: 28, Field: func(tg awselb.ELBTargetGroup) string { return tg.Name }},
		{Title: "Protocol", Width: 10, Field: func(tg awselb.ELBTargetGroup) string { return tg.Protocol }},
		{Title: "Port", Width: 8, Field: func(tg awselb.ELBTargetGroup) string {
			return fmt.Sprintf("%d", tg.Port)
		}},
		{Title: "Type", Width: 10, Field: func(tg awselb.ELBTargetGroup) string { return tg.TargetType }},
		{Title: "Healthy", Width: 8, Field: func(tg awselb.ELBTargetGroup) string {
			return fmt.Sprintf("%d", tg.HealthyCount)
		}},
		{Title: "Unhealthy", Width: 10, Field: func(tg awselb.ELBTargetGroup) string {
			return fmt.Sprintf("%d", tg.UnhealthyCount)
		}},
	}
	tv := ui.NewTableView(cols, dv.tgs, func(tg awselb.ELBTargetGroup) string { return tg.ARN })
	return tv.View()
}

func (dv *DetailView) renderAttributes() string {
	if len(dv.attributes) == 0 {
		return "No attributes."
	}

	cols := []ui.Column[awselb.ELBAttribute]{
		{Title: "Key", Width: 48, Field: func(a awselb.ELBAttribute) string { return a.Key }},
		{Title: "Value", Width: 30, Field: func(a awselb.ELBAttribute) string { return a.Value }},
	}
	tv := ui.NewTableView(cols, dv.attributes, func(a awselb.ELBAttribute) string { return a.Key })
	return tv.View()
}

func (dv *DetailView) Title() string {
	if dv.lb != nil {
		return dv.lb.Name
	}
	return "Load Balancer"
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	hints := []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "[/]", Desc: "switch tab"},
	}
	if dv.tabs.Active() == 2 {
		hints = append(hints, plugin.KeyHint{Key: "enter", Desc: "view targets"})
	}
	return hints
}

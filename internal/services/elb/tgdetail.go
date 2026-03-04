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

type targetsMsg struct {
	tgName  string
	targets []awselb.ELBTarget
	err     error
}

// TGDetailView shows targets for each target group of a load balancer.
type TGDetailView struct {
	client  *awselb.Client
	router  plugin.Router
	tgs     []awselb.ELBTargetGroup
	targets map[string][]awselb.ELBTarget // keyed by TG name
	tabs    ui.TabController
	loading bool
	err     error
}

func NewTGDetailView(client *awselb.Client, router plugin.Router, tgs []awselb.ELBTargetGroup) *TGDetailView {
	names := make([]string, len(tgs))
	for i, tg := range tgs {
		names[i] = tg.Name
	}
	return &TGDetailView{
		client:  client,
		router:  router,
		tgs:     tgs,
		targets: make(map[string][]awselb.ELBTarget),
		tabs:    ui.NewTabController(names),
		loading: true,
	}
}

func (v *TGDetailView) Init() tea.Cmd {
	client := v.client
	tgs := v.tgs
	var cmds []tea.Cmd
	for _, tg := range tgs {
		tg := tg
		cmds = append(cmds, func() tea.Msg {
			targets, err := client.ListTargets(context.TODO(), tg.ARN)
			return targetsMsg{tgName: tg.Name, targets: targets, err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (v *TGDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case targetsMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.targets[msg.tgName] = msg.targets
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			v.router.Pop()
			return v, nil
		}
	}

	var cmd tea.Cmd
	v.tabs, cmd = v.tabs.Update(msg)
	return v, cmd
}

func (v *TGDetailView) View() tea.View {
	if v.loading && len(v.targets) == 0 {
		skel := ui.NewSkeleton(60, 6)
		return tea.NewView(skel.View())
	}
	if v.err != nil {
		return tea.NewView("Error: " + v.err.Error())
	}

	var b strings.Builder
	b.WriteString(v.tabs.View())
	b.WriteString("\n\n")

	active := v.tabs.Active()
	if active < len(v.tgs) {
		tg := v.tgs[active]

		// TG overview.
		rows := []ui.KV{
			{"Name", tg.Name},
			{"ARN", tg.ARN},
			{"Protocol", tg.Protocol},
			{"Port", fmt.Sprintf("%d", tg.Port)},
			{"Target Type", tg.TargetType},
			{"Healthy", fmt.Sprintf("%d", tg.HealthyCount)},
			{"Unhealthy", fmt.Sprintf("%d", tg.UnhealthyCount)},
		}
		b.WriteString(ui.RenderKV(rows, 16, 0))
		b.WriteString("\n")

		// Targets table.
		targets := v.targets[tg.Name]
		if len(targets) == 0 {
			b.WriteString("No registered targets.")
		} else {
			cols := []ui.Column[awselb.ELBTarget]{
				{Title: "ID", Width: 24, Field: func(t awselb.ELBTarget) string { return t.ID }},
				{Title: "Port", Width: 8, Field: func(t awselb.ELBTarget) string {
					return fmt.Sprintf("%d", t.Port)
				}},
				{Title: "AZ", Width: 14, Field: func(t awselb.ELBTarget) string { return t.AZ }},
				{Title: "Health", Width: 12, Field: func(t awselb.ELBTarget) string { return t.HealthState }},
				{Title: "Reason", Width: 20, Field: func(t awselb.ELBTarget) string { return t.HealthReason }},
				{Title: "Description", Width: 30, Field: func(t awselb.ELBTarget) string { return t.HealthDesc }},
			}
			tv := ui.NewTableView(cols, targets, func(t awselb.ELBTarget) string {
				return fmt.Sprintf("%s:%d", t.ID, t.Port)
			})
			b.WriteString(tv.View())
		}
	}

	return tea.NewView(b.String())
}

func (v *TGDetailView) Title() string {
	return "Target Groups"
}

func (v *TGDetailView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "[/]", Desc: "switch TG"},
	}
}

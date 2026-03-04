package vpc

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// vpcsMsg carries the result of fetching VPCs.
type vpcsMsg struct {
	vpcs []awsvpc.VPCInfo
	err  error
}

// ListView displays VPCs in a table.
type ListView struct {
	client  VPCClient
	router  plugin.Router
	table   ui.TableView[awsvpc.VPCInfo]
	loading bool
	err     error
}

// NewListView creates a new VPC ListView.
func NewListView(client VPCClient, router plugin.Router) *ListView {
	cols := vpcColumns()
	tv := ui.NewTableView(cols, nil, func(v awsvpc.VPCInfo) string {
		return v.VPCID
	})
	return &ListView{
		client:  client,
		router:  router,
		table:   tv,
		loading: true,
	}
}

func vpcColumns() []ui.Column[awsvpc.VPCInfo] {
	return []ui.Column[awsvpc.VPCInfo]{
		{Title: "Name", Width: 24, Field: func(v awsvpc.VPCInfo) string { return v.Name }},
		{Title: "VPC ID", Width: 22, Field: func(v awsvpc.VPCInfo) string { return v.VPCID }},
		{Title: "CIDR", Width: 20, Field: func(v awsvpc.VPCInfo) string { return v.CIDR }},
		{Title: "State", Width: 12, Field: func(v awsvpc.VPCInfo) string { return v.State }},
		{Title: "Default", Width: 8, Field: func(v awsvpc.VPCInfo) string {
			if v.IsDefault {
				return "Yes"
			}
			return "No"
		}},
	}
}

func (lv *ListView) fetchVPCs() tea.Cmd {
	client := lv.client
	return func() tea.Msg {
		vpcs, err := client.ListVPCs(context.TODO())
		return vpcsMsg{vpcs: vpcs, err: err}
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchVPCs()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vpcsMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.table.SetItems(msg.vpcs)
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
			return lv, lv.fetchVPCs()
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

func (lv *ListView) Title() string { return "VPCs" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

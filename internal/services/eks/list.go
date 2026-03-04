package eks

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// clustersMsg carries the result of fetching clusters.
type clustersMsg struct {
	clusters []awseks.EKSCluster
	err      error
}

// ListView displays EKS clusters in a table.
type ListView struct {
	client  *awseks.Client
	router  plugin.Router
	table   ui.TableView[awseks.EKSCluster]
	loading bool
	err     error
	region  string
	profile string
}

// NewListView creates a new EKS ListView.
func NewListView(client *awseks.Client, router plugin.Router, region, profile string) *ListView {
	cols := clusterColumns()
	tv := ui.NewTableView(cols, nil, func(c awseks.EKSCluster) string {
		return c.Name
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

func clusterColumns() []ui.Column[awseks.EKSCluster] {
	return []ui.Column[awseks.EKSCluster]{
		{Title: "Status", Width: 3, Field: func(c awseks.EKSCluster) string {
			return clusterStatusDot(c.Status)
		}},
		{Title: "Name", Width: 28, Field: func(c awseks.EKSCluster) string { return c.Name }},
		{Title: "Version", Width: 10, Field: func(c awseks.EKSCluster) string { return c.Version }},
		{Title: "Platform", Width: 16, Field: func(c awseks.EKSCluster) string { return c.PlatformVersion }},
		{Title: "Endpoint", Width: 50, Field: func(c awseks.EKSCluster) string { return c.Endpoint }},
	}
}

var (
	greenDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("●")
	yellowDot = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("●")
	redDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("●")
	grayDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("●")
)

func clusterStatusDot(status string) string {
	switch status {
	case "ACTIVE":
		return greenDot
	case "CREATING", "UPDATING":
		return yellowDot
	case "DELETING", "FAILED":
		return redDot
	default:
		return grayDot
	}
}

func (lv *ListView) fetchClusters() tea.Cmd {
	client := lv.client
	return func() tea.Msg {
		clusters, err := client.ListClusters(context.TODO())
		return clustersMsg{clusters: clusters, err: err}
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchClusters()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clustersMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.table.SetItems(msg.clusters)
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
			return lv, lv.fetchClusters()
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

func (lv *ListView) Title() string { return "EKS Clusters" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

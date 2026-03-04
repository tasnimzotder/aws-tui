package ecr

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// reposMsg carries the result of fetching repositories.
type reposMsg struct {
	repos []awsecr.ECRRepo
	err   error
}

// ListView displays ECR repositories in a table.
type ListView struct {
	client  ECRClient
	router  plugin.Router
	table   ui.TableView[awsecr.ECRRepo]
	loading bool
	err     error
}

// NewListView creates a new ECR ListView.
func NewListView(client ECRClient, router plugin.Router) *ListView {
	cols := repoColumns()
	tv := ui.NewTableView(cols, nil, func(r awsecr.ECRRepo) string {
		return r.Name
	})
	return &ListView{
		client:  client,
		router:  router,
		table:   tv,
		loading: true,
	}
}

func repoColumns() []ui.Column[awsecr.ECRRepo] {
	return []ui.Column[awsecr.ECRRepo]{
		{Title: "Name", Width: 36, Field: func(r awsecr.ECRRepo) string { return r.Name }},
		{Title: "URI", Width: 48, Field: func(r awsecr.ECRRepo) string { return r.URI }},
		{Title: "Images", Width: 8, Field: func(r awsecr.ECRRepo) string {
			return fmt.Sprintf("%d", r.ImageCount)
		}},
		{Title: "Created", Width: 20, Field: func(r awsecr.ECRRepo) string {
			if r.CreatedAt.IsZero() {
				return "-"
			}
			return r.CreatedAt.Format("2006-01-02 15:04")
		}},
	}
}

func (lv *ListView) fetchRepos() tea.Cmd {
	client := lv.client
	return func() tea.Msg {
		repos, err := client.ListRepositories(context.Background())
		return reposMsg{repos: repos, err: err}
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchRepos()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reposMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.table.SetItems(msg.repos)
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
			return lv, lv.fetchRepos()
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

func (lv *ListView) Title() string { return "ECR Repositories" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view images"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

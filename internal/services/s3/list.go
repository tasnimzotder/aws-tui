package s3

import (
	"context"

	tea "charm.land/bubbletea/v2"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// bucketsMsg carries the result of fetching buckets.
type bucketsMsg struct {
	buckets []awss3.S3Bucket
	err     error
}

// ListView displays S3 buckets in a table.
type ListView struct {
	client  S3Client
	router  plugin.Router
	table   ui.TableView[awss3.S3Bucket]
	buckets []awss3.S3Bucket
	loading bool
	err     error
}

// NewListView creates a new S3 bucket ListView.
func NewListView(client S3Client, router plugin.Router) *ListView {
	cols := bucketColumns()
	tv := ui.NewTableView(cols, nil, func(b awss3.S3Bucket) string {
		return b.Name
	})
	return &ListView{
		client:  client,
		router:  router,
		table:   tv,
		loading: true,
	}
}

func bucketColumns() []ui.Column[awss3.S3Bucket] {
	return []ui.Column[awss3.S3Bucket]{
		{Title: "Name", Width: 40, Field: func(b awss3.S3Bucket) string { return b.Name }},
		{Title: "Region", Width: 16, Field: func(b awss3.S3Bucket) string { return b.Region }},
		{Title: "Created", Width: 20, Field: func(b awss3.S3Bucket) string {
			if b.CreatedAt.IsZero() {
				return ""
			}
			return b.CreatedAt.Format("2006-01-02 15:04")
		}},
	}
}

func (lv *ListView) fetchBuckets() tea.Cmd {
	client := lv.client
	return func() tea.Msg {
		buckets, err := client.ListBuckets(context.TODO())
		return bucketsMsg{buckets: buckets, err: err}
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.fetchBuckets()
}

func (lv *ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case bucketsMsg:
		lv.loading = false
		if msg.err != nil {
			lv.err = msg.err
			return lv, nil
		}
		lv.buckets = msg.buckets
		lv.table.SetItems(msg.buckets)
		return lv, nil

	case tea.KeyPressMsg:
		if lv.loading {
			return lv, nil
		}

		switch msg.String() {
		case "enter":
			selected := lv.table.SelectedItem()
			if selected.Name != "" {
				view := NewDetailView(lv.client, lv.router, selected.Name, selected.Region)
				lv.router.Push(view)
				return lv, view.Init()
			}
			return lv, nil
		case "esc", "backspace":
			lv.router.Pop()
			return lv, nil
		case "r":
			lv.loading = true
			return lv, lv.fetchBuckets()
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

func (lv *ListView) Title() string { return "S3 Buckets" }

func (lv *ListView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "browse bucket"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

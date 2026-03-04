package ecr

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// imagesMsg carries the result of fetching images for a repository.
type imagesMsg struct {
	images []awsecr.ECRImage
	err    error
}

// DetailView shows images for a single ECR repository.
type DetailView struct {
	client   ECRClient
	router   plugin.Router
	repoName string
	images   []awsecr.ECRImage
	table    ui.TableView[awsecr.ECRImage]
	tabs     ui.TabController
	loading  bool
	err      error
}

// NewDetailView creates a DetailView for the given repository name.
func NewDetailView(client ECRClient, router plugin.Router, repoName string) *DetailView {
	cols := imageColumns()
	tv := ui.NewTableView(cols, nil, func(img awsecr.ECRImage) string {
		return img.Digest
	})
	return &DetailView{
		client:   client,
		router:   router,
		repoName: repoName,
		table:    tv,
		tabs:     ui.NewTabController([]string{"Images", "Overview"}),
		loading:  true,
	}
}

func imageColumns() []ui.Column[awsecr.ECRImage] {
	return []ui.Column[awsecr.ECRImage]{
		{Title: "Tags", Width: 30, Field: func(img awsecr.ECRImage) string {
			if len(img.Tags) == 0 {
				return "<untagged>"
			}
			return strings.Join(img.Tags, ", ")
		}},
		{Title: "Digest", Width: 24, Field: func(img awsecr.ECRImage) string { return img.Digest }},
		{Title: "Size (MB)", Width: 12, Field: func(img awsecr.ECRImage) string {
			return fmt.Sprintf("%.1f", img.SizeMB)
		}},
		{Title: "Pushed At", Width: 20, Field: func(img awsecr.ECRImage) string {
			if img.PushedAt.IsZero() {
				return "-"
			}
			return img.PushedAt.Format("2006-01-02 15:04")
		}},
	}
}

func (dv *DetailView) fetchImages() tea.Cmd {
	client := dv.client
	repoName := dv.repoName
	return func() tea.Msg {
		images, err := client.ListImages(context.Background(), repoName)
		return imagesMsg{images: images, err: err}
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.fetchImages()
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imagesMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.images = msg.images
		dv.table.SetItems(msg.images)
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		case "r":
			if !dv.loading {
				dv.loading = true
				return dv, dv.fetchImages()
			}
			return dv, nil
		}
	}

	var cmd tea.Cmd
	dv.tabs, cmd = dv.tabs.Update(msg)

	if dv.tabs.Active() == 0 {
		var tableCmd tea.Cmd
		dv.table, tableCmd = dv.table.Update(msg)
		if tableCmd != nil {
			cmd = tableCmd
		}
	}

	return dv, cmd
}

func (dv *DetailView) View() tea.View {
	if dv.loading {
		skel := ui.NewSkeleton(80, 6)
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
		b.WriteString(dv.table.View())
	case 1:
		b.WriteString(dv.renderOverview())
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderOverview() string {
	totalImages := len(dv.images)
	totalSize := 0.0
	for _, img := range dv.images {
		totalSize += img.SizeMB
	}

	rows := []ui.KV{
		{K: "Repository", V: dv.repoName},
		{K: "Total Images", V: fmt.Sprintf("%d", totalImages)},
		{K: "Total Size", V: fmt.Sprintf("%.1f MB", totalSize)},
	}
	return ui.RenderKV(rows, 20, 0)
}

func (dv *DetailView) Title() string {
	return dv.repoName
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "r", Desc: "refresh"},
		{Key: "[/]", Desc: "switch tab"},
		{Key: "1-2", Desc: "jump to tab"},
	}
}

package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/constants"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type s3ObjectContentMsg struct {
	data []byte
}

type S3ObjectContentView struct {
	client   *awsclient.ServiceClient
	bucket   string
	key      string
	region   string
	size     int64
	viewport viewport.Model
	spinner  spinner.Model
	loading  bool
	err      error
	ready    bool
	width    int
	height   int
}

func NewS3ObjectContentView(client *awsclient.ServiceClient, obj awss3.S3Object, bucket, region string) *S3ObjectContentView {
	return &S3ObjectContentView{
		client:  client,
		bucket:  bucket,
		key:     obj.Key,
		region:  region,
		size:    obj.Size,
		spinner: theme.NewSpinner(),
		loading: true,
		width:   80,
		height:  20,
	}
}

func (v *S3ObjectContentView) Title() string { return path.Base(v.key) }

func (v *S3ObjectContentView) Init() tea.Cmd {
	if v.size > constants.MaxViewFileSize {
		v.loading = false
		v.err = fmt.Errorf("file too large to view (%s, max %s)",
			formatSize(v.size), formatSize(constants.MaxViewFileSize))
		return nil
	}
	return tea.Batch(v.spinner.Tick, v.fetchContent())
}

func (v *S3ObjectContentView) fetchContent() tea.Cmd {
	return func() tea.Msg {
		data, err := v.client.S3.GetObject(context.Background(), v.bucket, v.key, v.region)
		if err != nil {
			return errViewMsg{err: err}
		}
		return s3ObjectContentMsg{data: data}
	}
}

func (v *S3ObjectContentView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case s3ObjectContentMsg:
		v.loading = false
		v.viewport = v.newTextViewport()
		v.viewport.SetContent(v.formatText(msg.data))
		v.ready = true
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			if v.size <= constants.MaxViewFileSize {
				v.loading = true
				v.err = nil
				return v, tea.Batch(v.spinner.Tick, v.fetchContent())
			}
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

func (v *S3ObjectContentView) newTextViewport() viewport.Model {
	vp := viewport.New(
		viewport.WithWidth(v.width),
		viewport.WithHeight(v.height-2),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	ext := strings.ToLower(path.Ext(v.key))
	if isCodeFile(ext) {
		vp.LeftGutterFunc = func(info viewport.GutterContext) string {
			if info.Soft {
				return "     " + theme.MutedStyle.Render("│ ")
			}
			if info.Index >= info.TotalLines {
				return "   " + theme.MutedStyle.Render("~ │ ")
			}
			return theme.MutedStyle.Render(fmt.Sprintf("%4d │ ", info.Index+1))
		}
	}

	return vp
}

func isCodeFile(ext string) bool {
	switch ext {
	case ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".css", ".js", ".ts",
		".go", ".py", ".rb", ".rs", ".java", ".kt", ".c", ".cpp", ".h", ".sh",
		".tf", ".hcl", ".sql", ".graphql", ".proto", ".md", ".txt", ".csv", ".log",
		".env", ".ini", ".conf", ".cfg":
		return true
	}
	return false
}

func (v *S3ObjectContentView) formatText(data []byte) string {
	if !utf8.Valid(data) {
		return theme.ErrorStyle.Render("Binary file — cannot display content") +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back, then d to download")
	}

	ext := strings.ToLower(path.Ext(v.key))
	if ext == ".json" || ext == ".jsonl" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err == nil {
			return buf.String()
		}
	}

	return string(data)
}

func (v *S3ObjectContentView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading content..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.ready {
		return ""
	}

	status := theme.MutedStyle.Render(
		fmt.Sprintf(" %s  %s  %.0f%%  ↑/↓ scroll  Esc back",
			path.Base(v.key), formatSize(v.size), v.viewport.ScrollPercent()*100),
	)
	return v.viewport.View() + "\n" + status
}

func (v *S3ObjectContentView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.ready {
		v.viewport.SetWidth(width)
		v.viewport.SetHeight(height - 2)
	}
}

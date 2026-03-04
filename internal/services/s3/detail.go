package s3

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// objectsMsg carries the result of listing objects.
type objectsMsg struct {
	result awss3.ListObjectsResult
	err    error
}

// fileContentMsg carries the result of fetching an object's content.
type fileContentMsg struct {
	key     string
	content []byte
	err     error
}

// DetailView provides an object browser for an S3 bucket.
type DetailView struct {
	client S3Client
	router plugin.Router
	bucket string
	region string
	prefix string

	table   ui.TableView[awss3.S3Object]
	loading bool
	err     error

	// File preview state
	previewing     bool
	previewKey     string
	previewBody    string
	previewLines   []string
	previewScroll  int
	previewError   error
	viewportHeight int
}

// NewDetailView creates a new S3 bucket detail/object browser view.
func NewDetailView(client S3Client, router plugin.Router, bucket, region string) *DetailView {
	cols := objectColumns()
	tv := ui.NewTableView(cols, nil, func(o awss3.S3Object) string {
		return o.Key
	})
	return &DetailView{
		client:  client,
		router:  router,
		bucket:  bucket,
		region:  region,
		table:   tv,
		loading: true,
	}
}

func objectColumns() []ui.Column[awss3.S3Object] {
	return []ui.Column[awss3.S3Object]{
		{Title: "Name", Width: 44, Field: func(o awss3.S3Object) string {
			name := o.Key
			if o.IsPrefix {
				// Show only the folder name, strip trailing slash for display
				name = strings.TrimSuffix(name, "/")
				parts := strings.Split(name, "/")
				return parts[len(parts)-1] + "/"
			}
			// Show only the file name
			return path.Base(name)
		}},
		{Title: "Size", Width: 12, Field: func(o awss3.S3Object) string {
			if o.IsPrefix {
				return "-"
			}
			return formatSize(o.Size)
		}},
		{Title: "Last Modified", Width: 20, Field: func(o awss3.S3Object) string {
			if o.IsPrefix || o.LastModified.IsZero() {
				return "-"
			}
			return o.LastModified.Format("2006-01-02 15:04")
		}},
		{Title: "Storage Class", Width: 16, Field: func(o awss3.S3Object) string {
			if o.IsPrefix {
				return "-"
			}
			if o.StorageClass == "" {
				return "STANDARD"
			}
			return o.StorageClass
		}},
	}
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func (dv *DetailView) fetchObjects() tea.Cmd {
	client := dv.client
	bucket := dv.bucket
	prefix := dv.prefix
	region := dv.region
	return func() tea.Msg {
		result, err := client.ListObjects(context.TODO(), bucket, prefix, "", region)
		return objectsMsg{result: result, err: err}
	}
}

func (dv *DetailView) fetchFileContent(key string) tea.Cmd {
	client := dv.client
	bucket := dv.bucket
	region := dv.region
	return func() tea.Msg {
		content, err := client.GetObject(context.TODO(), bucket, key, region)
		return fileContentMsg{key: key, content: content, err: err}
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.fetchObjects()
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case objectsMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.table.SetItems(msg.result.Objects)
		return dv, nil

	case fileContentMsg:
		dv.loading = false
		dv.previewing = true
		dv.previewKey = msg.key
		dv.previewScroll = 0
		if msg.err != nil {
			dv.previewError = msg.err
			return dv, nil
		}
		dv.previewError = nil
		if isTextContent(msg.content) {
			body := string(msg.content)
			const maxPreview = 8192
			if len(body) > maxPreview {
				body = body[:maxPreview] + "\n... (truncated)"
			}
			// Apply syntax highlighting based on file extension
			highlighted := highlightCode(path.Base(msg.key), body)
			dv.previewBody = highlighted
			dv.previewLines = strings.Split(highlighted, "\n")
		} else {
			dv.previewBody = fmt.Sprintf("[Binary file — %s]", formatSize(int64(len(msg.content))))
			dv.previewLines = []string{dv.previewBody}
		}
		return dv, nil

	case tea.WindowSizeMsg:
		dv.viewportHeight = msg.Height
		return dv, nil

	case tea.KeyPressMsg:
		if dv.loading {
			return dv, nil
		}

		// If previewing a file, handle scroll and esc
		if dv.previewing {
			switch msg.String() {
			case "esc", "backspace":
				dv.previewing = false
				dv.previewKey = ""
				dv.previewBody = ""
				dv.previewLines = nil
				dv.previewScroll = 0
				dv.previewError = nil
			case "j", "down":
				maxScroll := len(dv.previewLines) - dv.previewVisibleLines()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if dv.previewScroll < maxScroll {
					dv.previewScroll++
				}
			case "k", "up":
				if dv.previewScroll > 0 {
					dv.previewScroll--
				}
			case "d":
				// Half-page down
				jump := dv.previewVisibleLines() / 2
				maxScroll := len(dv.previewLines) - dv.previewVisibleLines()
				if maxScroll < 0 {
					maxScroll = 0
				}
				dv.previewScroll += jump
				if dv.previewScroll > maxScroll {
					dv.previewScroll = maxScroll
				}
			case "u":
				// Half-page up
				jump := dv.previewVisibleLines() / 2
				dv.previewScroll -= jump
				if dv.previewScroll < 0 {
					dv.previewScroll = 0
				}
			case "g":
				dv.previewScroll = 0
			case "G":
				maxScroll := len(dv.previewLines) - dv.previewVisibleLines()
				if maxScroll < 0 {
					maxScroll = 0
				}
				dv.previewScroll = maxScroll
			}
			return dv, nil
		}

		switch msg.String() {
		case "enter":
			selected := dv.table.SelectedItem()
			if selected.Key == "" {
				return dv, nil
			}
			if selected.IsPrefix {
				// Navigate into folder
				dv.prefix = selected.Key
				dv.loading = true
				dv.err = nil
				return dv, dv.fetchObjects()
			}
			// Fetch file content for preview
			dv.loading = true
			return dv, dv.fetchFileContent(selected.Key)

		case "esc", "backspace":
			if dv.prefix == "" {
				// At bucket root, go back to bucket list
				dv.router.Pop()
				return dv, nil
			}
			// Go up one level
			dv.prefix = parentPrefix(dv.prefix)
			dv.loading = true
			dv.err = nil
			return dv, dv.fetchObjects()

		case "r":
			dv.loading = true
			return dv, dv.fetchObjects()
		}
	}

	var cmd tea.Cmd
	dv.table, cmd = dv.table.Update(msg)
	return dv, cmd
}

// parentPrefix returns the parent prefix for the given prefix.
// For example, "a/b/c/" returns "a/b/", and "a/" returns "".
func parentPrefix(prefix string) string {
	trimmed := strings.TrimSuffix(prefix, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx < 0 {
		return ""
	}
	return trimmed[:idx+1]
}

var (
	breadcrumbStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))
)

func (dv *DetailView) View() tea.View {
	if dv.loading {
		skel := ui.NewSkeleton(80, 6)
		return tea.NewView(dv.breadcrumb() + "\n" + skel.View())
	}
	if dv.err != nil {
		return tea.NewView(dv.breadcrumb() + "\nError: " + dv.err.Error())
	}

	if dv.previewing {
		return tea.NewView(dv.renderPreview())
	}

	return tea.NewView(dv.breadcrumb() + "\n\n" + dv.table.View())
}

func (dv *DetailView) breadcrumb() string {
	parts := []string{dv.bucket}
	if dv.prefix != "" {
		segments := strings.Split(strings.TrimSuffix(dv.prefix, "/"), "/")
		parts = append(parts, segments...)
	}
	return breadcrumbStyle.Render(strings.Join(parts, " / "))
}

// previewVisibleLines returns how many lines fit in the viewport for the preview.
func (dv *DetailView) previewVisibleLines() int {
	// Reserve lines for: app breadcrumb (1), S3 breadcrumb (1), "Preview:" header (1),
	// blank line (1), scroll indicator (1), status bar (1), safety (1) = 7
	h := dv.viewportHeight - 7
	if h < 5 {
		h = 30 // fallback if window size unknown
	}
	return h
}

func (dv *DetailView) renderPreview() string {
	var b strings.Builder
	b.WriteString(dv.breadcrumb())
	b.WriteString("\n")
	b.WriteString(previewHeaderStyle.Render("Preview: " + path.Base(dv.previewKey)))
	b.WriteString("\n\n")
	if dv.previewError != nil {
		b.WriteString("Error: " + dv.previewError.Error())
	} else {
		visible := dv.previewVisibleLines()
		start := dv.previewScroll
		end := start + visible
		if end > len(dv.previewLines) {
			end = len(dv.previewLines)
		}
		for i := start; i < end; i++ {
			b.WriteString(dv.previewLines[i])
			if i < end-1 {
				b.WriteByte('\n')
			}
		}
		// Scroll indicator
		total := len(dv.previewLines)
		if total > visible {
			pct := 0
			if total-visible > 0 {
				pct = (dv.previewScroll * 100) / (total - visible)
			}
			b.WriteString(fmt.Sprintf("\n── %d%% ── j/k scroll · d/u half-page · g/G top/bottom · esc close", pct))
		}
	}
	return b.String()
}

func (dv *DetailView) Title() string {
	return "s3://" + dv.bucket
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	if dv.previewing {
		return []plugin.KeyHint{
			{Key: "j/k", Desc: "scroll"},
			{Key: "d/u", Desc: "half-page"},
			{Key: "g/G", Desc: "top/bottom"},
			{Key: "esc", Desc: "close"},
		}
	}
	hints := []plugin.KeyHint{
		{Key: "enter", Desc: "open"},
		{Key: "esc", Desc: "back"},
		{Key: "r", Desc: "refresh"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
	return hints
}

// highlightCode applies syntax highlighting based on the file extension.
// Returns the original text if highlighting is not available.
func highlightCode(filename, code string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	formatter := formatters.Get("terminal256")

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return code
	}
	return buf.String()
}

// isTextContent checks whether the content appears to be valid UTF-8 text.
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	// Check a sample (first 512 bytes) for valid UTF-8 and absence of null bytes
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	if !utf8.Valid(sample) {
		return false
	}
	for _, b := range sample {
		if b == 0 {
			return false
		}
	}
	return true
}

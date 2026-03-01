package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

// TextView is a reusable text viewer component with syntax highlighting,
// search, and soft-wrap toggle. It implements the View and ResizableView interfaces.
type TextView struct {
	title    string
	filename string
	rawText  string // processed (highlighted) text content
	isBinary bool
	viewport viewport.Model
	ready    bool

	// Search
	searching   bool
	searchInput textinput.Model
	searchQuery string
	matchCount  int
	matchIndex  int

	// Wrap toggle
	softWrap bool

	width  int
	height int
}

var searchHighlightStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#F59E0B")).
	Foreground(lipgloss.Color("#000000"))

// NewTextView creates a new TextView from content bytes and a filename.
// The filename is used for syntax highlighting detection and display.
func NewTextView(title string, content []byte, filename string) *TextView {
	tv := &TextView{
		title:    title,
		filename: filename,
		softWrap: true,
		width:    80,
		height:   24,
	}

	if !utf8.Valid(content) {
		tv.isBinary = true
		return tv
	}

	text := string(content)

	// Pretty-print JSON
	ext := strings.ToLower(path.Ext(filename))
	if ext == ".json" || ext == ".jsonl" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, content, "", "  "); err == nil {
			text = buf.String()
		}
	}

	tv.rawText = tv.highlight(text)
	return tv
}

// highlight applies syntax highlighting to text using chroma.
func (tv *TextView) highlight(text string) string {
	lexer := lexers.Match(tv.filename)
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		return text
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return text
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return text
	}
	return buf.String()
}

func (tv *TextView) newViewport() viewport.Model {
	h := tv.height - 2 // reserve space for status bar
	if h < 1 {
		h = 1
	}

	vp := viewport.New(
		viewport.WithWidth(tv.width),
		viewport.WithHeight(h),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = tv.softWrap
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	ext := strings.ToLower(path.Ext(tv.filename))
	if isCodeFile(ext) {
		vp.LeftGutterFunc = func(info viewport.GutterContext) string {
			if info.Soft {
				return "     " + theme.MutedStyle.Render("\u2502 ")
			}
			if info.Index >= info.TotalLines {
				return "   " + theme.MutedStyle.Render("~ \u2502 ")
			}
			return theme.MutedStyle.Render(fmt.Sprintf("%4d \u2502 ", info.Index+1))
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

func (tv *TextView) Title() string { return tv.title }

func (tv *TextView) HelpContext() *HelpContext {
	ctx := HelpContextTextView
	return &ctx
}

func (tv *TextView) Init() tea.Cmd {
	if tv.isBinary {
		return nil
	}
	tv.viewport = tv.newViewport()
	tv.viewport.SetContent(tv.rawText)
	tv.ready = true

	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 256
	tv.searchInput = ti

	return nil
}

func (tv *TextView) Update(msg tea.Msg) (View, tea.Cmd) {
	if tv.isBinary {
		return tv, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()

		if tv.searching {
			switch key {
			case "enter":
				tv.applySearch()
				tv.searching = false
				tv.searchInput.Blur()
				return tv, nil
			case "esc":
				tv.cancelSearch()
				tv.searching = false
				tv.searchInput.Blur()
				return tv, nil
			default:
				var cmd tea.Cmd
				tv.searchInput, cmd = tv.searchInput.Update(msg)
				return tv, cmd
			}
		}

		// Normal mode keybindings
		switch key {
		case "/":
			tv.searching = true
			tv.searchInput.SetValue("")
			tv.searchInput.Focus()
			return tv, textinput.Blink
		case "n":
			if tv.matchCount > 0 {
				tv.matchIndex = (tv.matchIndex + 1) % tv.matchCount
				tv.scrollToMatch()
			}
			return tv, nil
		case "shift+n", "N":
			if tv.matchCount > 0 {
				tv.matchIndex = (tv.matchIndex - 1 + tv.matchCount) % tv.matchCount
				tv.scrollToMatch()
			}
			return tv, nil
		case "w":
			tv.softWrap = !tv.softWrap
			tv.viewport.SoftWrap = tv.softWrap
			return tv, nil
		}
	}

	if tv.ready {
		var cmd tea.Cmd
		tv.viewport, cmd = tv.viewport.Update(msg)
		return tv, cmd
	}
	return tv, nil
}

func (tv *TextView) applySearch() {
	query := tv.searchInput.Value()
	if query == "" {
		tv.cancelSearch()
		return
	}
	tv.searchQuery = query

	// Count matches case-insensitively in the raw text
	lower := strings.ToLower(tv.rawText)
	lowerQuery := strings.ToLower(query)
	tv.matchCount = strings.Count(lower, lowerQuery)
	tv.matchIndex = 0

	if tv.matchCount == 0 {
		// Restore original content without highlights
		tv.viewport.SetContent(tv.rawText)
		return
	}

	// Highlight all matches and set content
	highlighted := tv.highlightMatches(tv.rawText, query)
	tv.viewport.SetContent(highlighted)
	tv.scrollToMatch()
}

func (tv *TextView) cancelSearch() {
	tv.searchQuery = ""
	tv.matchCount = 0
	tv.matchIndex = 0
	tv.viewport.SetContent(tv.rawText)
}

// highlightMatches highlights all case-insensitive occurrences of query in text.
func (tv *TextView) highlightMatches(text, query string) string {
	if query == "" {
		return text
	}
	lower := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	var result strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], lowerQuery)
		if idx < 0 {
			result.WriteString(text[i:])
			break
		}
		result.WriteString(text[i : i+idx])
		matchText := text[i+idx : i+idx+len(query)]
		result.WriteString(searchHighlightStyle.Render(matchText))
		i += idx + len(query)
	}
	return result.String()
}

// scrollToMatch scrolls the viewport to the current match.
func (tv *TextView) scrollToMatch() {
	if tv.matchCount == 0 || !tv.ready {
		return
	}

	lower := strings.ToLower(tv.rawText)
	lowerQuery := strings.ToLower(tv.searchQuery)

	// Find the Nth match
	pos := 0
	for n := 0; n <= tv.matchIndex; n++ {
		idx := strings.Index(lower[pos:], lowerQuery)
		if idx < 0 {
			return
		}
		if n == tv.matchIndex {
			pos += idx
			break
		}
		pos += idx + len(lowerQuery)
	}

	// Count the line number of the match
	line := strings.Count(tv.rawText[:pos], "\n")
	tv.viewport.SetYOffset(line)
}

func (tv *TextView) View() string {
	if tv.isBinary {
		return theme.ErrorStyle.Render("Binary file \u2014 cannot display content") +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if !tv.ready {
		return ""
	}

	// Build status bar
	var status strings.Builder

	// Filename
	status.WriteString(" ")
	status.WriteString(tv.filename)

	// Scroll percent
	status.WriteString(fmt.Sprintf("  %.0f%%", tv.viewport.ScrollPercent()*100))

	// Wrap mode
	if tv.softWrap {
		status.WriteString("  wrap")
	}

	// Keybinding hints
	status.WriteString("  / search  w wrap  n/N next/prev")

	// Search info
	if tv.searchQuery != "" {
		if tv.matchCount > 0 {
			status.WriteString(fmt.Sprintf("  [%d/%d]", tv.matchIndex+1, tv.matchCount))
		} else {
			status.WriteString("  no match")
		}
	}

	status.WriteString("  Esc back")

	statusLine := theme.MutedStyle.Render(status.String())

	if tv.searching {
		return tv.viewport.View() + "\n/" + tv.searchInput.View()
	}

	return tv.viewport.View() + "\n" + statusLine
}

func (tv *TextView) SetSize(width, height int) {
	tv.width = width
	tv.height = height
	if tv.ready {
		tv.viewport.SetWidth(width)
		h := height - 2
		if h < 1 {
			h = 1
		}
		tv.viewport.SetHeight(h)
	}
}

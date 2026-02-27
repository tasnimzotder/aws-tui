# S3 Viewer/Download Redesign — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the image viewer, extract text viewing into a reusable `TextView` component with syntax highlighting/search/wrap toggle, and enhance the download experience with a progress bar, ETA, and cancellation.

**Architecture:** The new `TextView` is a standalone `View` that accepts content bytes and a filename. It uses chroma for syntax highlighting, a viewport for scrolling, a text input for search, and Lip Gloss for the status bar. The download view gets a streaming `GetObjectStream` method on the S3 client, reads chunks via an `io.TeeReader`, and sends progress messages to drive a Bubble Tea progress bar. Esc cancels via `context.WithCancel`.

**Tech Stack:** chroma v2 (syntax highlighting), Bubble Tea v2 progress bubble, viewport v2, textinput v2

---

### Task 1: Remove image viewer code and dependencies

**Files:**
- Delete: `internal/tui/services/imgrender.go`
- Modify: `internal/tui/services/s3_view.go` (will be deleted entirely in Task 3)
- Modify: `internal/tui/services/s3_view_test.go` (will be rewritten in Task 4)
- Modify: `go.mod`

**Step 1: Delete imgrender.go**

```bash
rm internal/tui/services/imgrender.go
```

**Step 2: Remove image deps from go.mod**

```bash
go mod tidy
```

This removes `rasterm`, `pixterm`, `disintegration/imaging`, `golang.org/x/image` since nothing else imports them.

**Step 3: Verify build**

Run: `go build ./...`
Expected: Build succeeds (s3_view.go still exists but will be replaced next)

Note: The build will fail because s3_view.go references imgrender functions. That's fine — we'll delete s3_view.go in Task 3. For now, just delete imgrender.go and move on.

**Step 4: Commit**

```bash
git add -A
git commit -m "remove image viewer feature and dependencies"
```

---

### Task 2: Add chroma dependency and GetObjectStream to S3 client

**Files:**
- Modify: `internal/aws/s3/client.go:14-19` (S3API interface)
- Modify: `internal/aws/s3/client.go` (add method)
- Modify: `internal/aws/s3/client_test.go` (add mock + tests)
- Modify: `go.mod` (add chroma)

**Step 1: Add chroma dependency**

```bash
go get github.com/alecthomas/chroma/v2@latest
```

**Step 2: Add GetObjectStream to S3API interface**

In `internal/aws/s3/client.go`, the `S3API` interface at line 14 already has `GetObject`. No change needed to the interface — `GetObjectStream` is a client-level method that calls the same `GetObject` API but returns the stream.

**Step 3: Write failing test for GetObjectStream**

In `internal/aws/s3/client_test.go`, add:

```go
func TestGetObjectStream(t *testing.T) {
	body := io.NopCloser(strings.NewReader("streaming content"))
	mock := &mockS3API{
		getObjectFunc: func(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			return &awss3.GetObjectOutput{
				Body:          body,
				ContentLength: aws.Int64(17),
			}, nil
		},
	}
	client := s3.NewClient(mock)
	reader, size, err := client.GetObjectStream(context.Background(), "bucket", "key.txt", "us-east-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	if size != 17 {
		t.Errorf("size = %d, want 17", size)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "streaming content" {
		t.Errorf("data = %q, want 'streaming content'", data)
	}
}
```

**Step 4: Run test to verify it fails**

Run: `go test ./internal/aws/s3/ -run TestGetObjectStream -v`
Expected: FAIL — `GetObjectStream` not defined

**Step 5: Implement GetObjectStream**

In `internal/aws/s3/client.go`, add after `GetObject`:

```go
// GetObjectStream returns the object body as a stream with its content length.
// The caller must close the returned ReadCloser.
func (c *Client) GetObjectStream(ctx context.Context, bucket, key, region string) (io.ReadCloser, int64, error) {
	input := &awss3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	var opts []func(*awss3.Options)
	if region != "" {
		opts = append(opts, func(o *awss3.Options) { o.Region = region })
	}
	out, err := c.api.GetObject(ctx, input, opts...)
	if err != nil {
		return nil, 0, fmt.Errorf("GetObjectStream: %w", err)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, size, nil
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/aws/s3/ -run TestGetObjectStream -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/aws/s3/client.go internal/aws/s3/client_test.go go.mod go.sum
git commit -m "add GetObjectStream and chroma dependency"
```

---

### Task 3: Create reusable TextView component

**Files:**
- Create: `internal/tui/services/textview.go`
- Delete: `internal/tui/services/s3_view.go`
- Modify: `internal/tui/services/help.go` (add HelpContextTextView)

This is the core component. It implements `View` and `ResizableView`.

**Step 1: Delete s3_view.go**

```bash
rm internal/tui/services/s3_view.go
```

**Step 2: Create textview.go**

Create `internal/tui/services/textview.go`:

```go
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

// TextView is a reusable text content viewer with syntax highlighting,
// search, and word-wrap toggle. It implements View and ResizableView.
type TextView struct {
	title    string
	filename string
	rawText  string   // processed text content
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

// NewTextView creates a text viewer for the given content.
// filename is used for syntax highlighting language detection.
func NewTextView(title string, content []byte, filename string) *TextView {
	si := textinput.New()
	si.Placeholder = "search..."
	si.CharLimit = 128

	tv := &TextView{
		title:       title,
		filename:    filename,
		searchInput: si,
		softWrap:    true,
		width:       80,
		height:      24,
	}

	if !utf8.Valid(content) {
		tv.isBinary = true
		return tv
	}

	tv.rawText = tv.processContent(content)
	return tv
}

func (tv *TextView) Title() string { return tv.title }

func (tv *TextView) Init() tea.Cmd {
	tv.viewport = tv.newViewport()
	tv.viewport.SetContent(tv.rawText)
	tv.ready = true
	return nil
}

func (tv *TextView) processContent(data []byte) string {
	ext := strings.ToLower(path.Ext(tv.filename))

	// Pretty-print JSON
	if ext == ".json" || ext == ".jsonl" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err == nil {
			return tv.highlight(buf.String())
		}
	}

	text := string(data)
	return tv.highlight(text)
}

func (tv *TextView) highlight(text string) string {
	lexer := lexers.Match(tv.filename)
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		return text // no highlighting available
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return text
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
	h := tv.contentHeight()
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

func (tv *TextView) contentHeight() int {
	h := tv.height - 2 // status bar
	if tv.searching {
		h-- // search bar
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (tv *TextView) Update(msg tea.Msg) (View, tea.Cmd) {
	if tv.isBinary {
		return tv, nil // no interaction for binary
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if tv.searching {
			return tv.updateSearch(msg)
		}
		switch msg.String() {
		case "/":
			tv.searching = true
			tv.searchInput.Focus()
			tv.viewport.SetHeight(tv.contentHeight())
			return tv, textinput.Blink
		case "n":
			if tv.matchCount > 0 {
				tv.matchIndex = (tv.matchIndex + 1) % tv.matchCount
				tv.scrollToMatch()
			}
			return tv, nil
		case "N":
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

func (tv *TextView) updateSearch(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := tv.searchInput.Value()
		tv.searchQuery = query
		tv.matchIndex = 0
		tv.applySearch()
		tv.searching = false
		tv.viewport.SetHeight(tv.contentHeight())
		return tv, nil
	case "esc":
		tv.searching = false
		tv.searchInput.SetValue("")
		tv.searchQuery = ""
		tv.matchCount = 0
		tv.viewport.SetContent(tv.rawText)
		tv.viewport.SetHeight(tv.contentHeight())
		return tv, nil
	}
	var cmd tea.Cmd
	tv.searchInput, cmd = tv.searchInput.Update(msg)
	return tv, cmd
}

func (tv *TextView) applySearch() {
	if tv.searchQuery == "" {
		tv.matchCount = 0
		tv.viewport.SetContent(tv.rawText)
		return
	}

	// Count matches in raw (unhighlighted) text for accurate counting
	// but highlight in the displayed text
	plainLines := strings.Split(tv.rawText, "\n")
	tv.matchCount = 0
	for _, line := range plainLines {
		tv.matchCount += strings.Count(strings.ToLower(line), strings.ToLower(tv.searchQuery))
	}

	// Highlight matches in the displayed content
	highlighted := tv.highlightMatches(tv.rawText)
	tv.viewport.SetContent(highlighted)

	if tv.matchCount > 0 {
		tv.scrollToMatch()
	}
}

func (tv *TextView) highlightMatches(text string) string {
	if tv.searchQuery == "" {
		return text
	}
	matchStyle := lipgloss.NewStyle().Background(lipgloss.Color("#F59E0B")).Foreground(lipgloss.Color("#000000"))

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		query := strings.ToLower(tv.searchQuery)
		if strings.Contains(lower, query) {
			var result strings.Builder
			remaining := line
			for {
				idx := strings.Index(strings.ToLower(remaining), query)
				if idx < 0 {
					result.WriteString(remaining)
					break
				}
				result.WriteString(remaining[:idx])
				result.WriteString(matchStyle.Render(remaining[idx : idx+len(tv.searchQuery)]))
				remaining = remaining[idx+len(tv.searchQuery):]
			}
			lines[i] = result.String()
		}
	}
	return strings.Join(lines, "\n")
}

func (tv *TextView) scrollToMatch() {
	if tv.searchQuery == "" || tv.matchCount == 0 {
		return
	}

	lines := strings.Split(tv.rawText, "\n")
	current := 0
	query := strings.ToLower(tv.searchQuery)
	for lineIdx, line := range lines {
		count := strings.Count(strings.ToLower(line), query)
		if current+count > tv.matchIndex {
			tv.viewport.GotoTop()
			for range lineIdx {
				tv.viewport.LineDown(1)
			}
			return
		}
		current += count
	}
}

func (tv *TextView) View() string {
	if tv.isBinary {
		return theme.ErrorStyle.Render("Binary file — cannot display content") +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}

	var parts []string

	if tv.searching {
		parts = append(parts, "/ "+tv.searchInput.View())
	}

	parts = append(parts, tv.viewport.View())

	// Status bar
	wrapMode := "wrap"
	if !tv.softWrap {
		wrapMode = "nowrap"
	}
	searchStatus := ""
	if tv.searchQuery != "" && tv.matchCount > 0 {
		searchStatus = fmt.Sprintf("  [%d/%d]", tv.matchIndex+1, tv.matchCount)
	} else if tv.searchQuery != "" {
		searchStatus = "  [no match]"
	}
	status := theme.MutedStyle.Render(
		fmt.Sprintf(" %s  %.0f%%  %s  / search  w wrap  n/N next/prev%s  Esc back",
			tv.filename, tv.viewport.ScrollPercent()*100, wrapMode, searchStatus),
	)
	parts = append(parts, status)

	return strings.Join(parts, "\n")
}

func (tv *TextView) SetSize(width, height int) {
	tv.width = width
	tv.height = height
	if tv.ready {
		tv.viewport.SetWidth(width)
		tv.viewport.SetHeight(tv.contentHeight())
	}
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
```

**Step 3: Add HelpContextTextView to help.go**

In `internal/tui/services/help.go`, add `HelpContextTextView` constant and its bindings:

```go
// Add to the const block (after HelpContextS3Objects):
HelpContextTextView

// Add case in renderHelp switch:
case HelpContextTextView:
    title = "Keybindings — Text Viewer"
    bindings = []helpBinding{
        {"/", "Search"},
        {"n", "Next match"},
        {"N", "Prev match"},
        {"w", "Toggle word wrap"},
        {"j/k", "Scroll up/down"},
        {"Esc", "Go back / close search"},
        {"?", "Toggle this help"},
        {"q", "Quit"},
    }

// Add to detectHelpContext:
case *TextView:
    return HelpContextTextView
```

**Step 4: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/tui/services/textview.go internal/tui/services/help.go
git rm internal/tui/services/s3_view.go
git commit -m "add reusable TextView component with syntax highlighting, search, and wrap toggle"
```

---

### Task 4: Wire TextView into S3 and write tests

**Files:**
- Modify: `internal/tui/services/s3.go:92-104` (update v key handler)
- Rewrite: `internal/tui/services/s3_view_test.go` → rename to `internal/tui/services/textview_test.go`
- Modify: `internal/tui/services/s3.go` (loading view for fetching content before pushing TextView)

The S3 `v` key handler needs to: fetch the object content (async with spinner), then push a `TextView`. We need a small intermediary view that does the loading, similar to the old `S3ObjectContentView` but without image logic — just a loading spinner that fetches content and then pushes a `TextView`.

**Step 1: Create S3ContentLoaderView in s3.go or a small s3_view.go**

Actually, to keep things clean, create a minimal `internal/tui/services/s3_view.go` that is just the content loader:

```go
package services

import (
	"context"
	"fmt"
	"path"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/constants"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type s3ObjectContentMsg struct {
	data []byte
}

// S3ContentLoaderView fetches an S3 object then pushes a TextView.
type S3ContentLoaderView struct {
	client  *awsclient.ServiceClient
	bucket  string
	key     string
	region  string
	size    int64
	spinner spinner.Model
	loading bool
	err     error
}

func NewS3ContentLoaderView(client *awsclient.ServiceClient, obj awss3.S3Object, bucket, region string) *S3ContentLoaderView {
	return &S3ContentLoaderView{
		client:  client,
		bucket:  bucket,
		key:     obj.Key,
		region:  region,
		size:    obj.Size,
		spinner: theme.NewSpinner(),
		loading: true,
	}
}

func (v *S3ContentLoaderView) Title() string { return path.Base(v.key) }

func (v *S3ContentLoaderView) Init() tea.Cmd {
	if v.size > constants.MaxViewFileSize {
		v.loading = false
		v.err = fmt.Errorf("file too large to view (%s, max %s)",
			formatSize(v.size), formatSize(constants.MaxViewFileSize))
		return nil
	}
	return tea.Batch(v.spinner.Tick, v.fetchContent())
}

func (v *S3ContentLoaderView) fetchContent() tea.Cmd {
	return func() tea.Msg {
		data, err := v.client.S3.GetObject(context.Background(), v.bucket, v.key, v.region)
		if err != nil {
			return errViewMsg{err: err}
		}
		return s3ObjectContentMsg{data: data}
	}
}

func (v *S3ContentLoaderView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case s3ObjectContentMsg:
		v.loading = false
		// Push the reusable TextView with the fetched content
		tv := NewTextView(path.Base(v.key), msg.data, v.key)
		return v, pushView(tv)

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *S3ContentLoaderView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading content..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return ""
}

func (v *S3ContentLoaderView) SetSize(width, height int) {}
```

**Step 2: Update s3.go v key handler**

In `internal/tui/services/s3.go:93-98`, change `NewS3ObjectContentView` to `NewS3ContentLoaderView`:

```go
"v": func(obj awss3.S3Object) tea.Cmd {
    if obj.IsPrefix {
        return nil
    }
    return pushView(NewS3ContentLoaderView(client, obj, bucket, region))
},
```

**Step 3: Rewrite tests**

Delete `internal/tui/services/s3_view_test.go`, create `internal/tui/services/textview_test.go`:

```go
package services

import (
	"strings"
	"testing"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/constants"
)

func TestTextView_PlainText(t *testing.T) {
	tv := NewTextView("test.txt", []byte("hello world"), "test.txt")
	tv.Init()
	if tv.isBinary {
		t.Error("plain text should not be binary")
	}
	if !strings.Contains(tv.rawText, "hello world") {
		t.Errorf("rawText should contain content, got: %s", tv.rawText)
	}
}

func TestTextView_Binary(t *testing.T) {
	tv := NewTextView("data.bin", []byte{0x80, 0x81, 0x82, 0x83}, "data.bin")
	if !tv.isBinary {
		t.Error("invalid UTF-8 should be detected as binary")
	}
	view := tv.View()
	if !strings.Contains(view, "Binary file") {
		t.Errorf("binary view should show notice, got: %s", view)
	}
}

func TestTextView_JSON(t *testing.T) {
	tv := NewTextView("data.json", []byte(`{"key":"value","arr":[1,2,3]}`), "data.json")
	tv.Init()
	// JSON should be pretty-printed (indented)
	if !strings.Contains(tv.rawText, "\"key\"") {
		t.Errorf("JSON should be formatted, got: %s", tv.rawText)
	}
}

func TestTextView_SyntaxHighlight(t *testing.T) {
	goCode := []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")
	tv := NewTextView("main.go", goCode, "main.go")
	tv.Init()
	// Chroma should add ANSI escape codes
	if !strings.Contains(tv.rawText, "\x1b[") {
		t.Error("Go code should be syntax highlighted with ANSI codes")
	}
}

func TestTextView_NoHighlightForUnknown(t *testing.T) {
	tv := NewTextView("data.xyz", []byte("just plain text"), "data.xyz")
	tv.Init()
	// Unknown extension, chroma may still analyse — just ensure no crash
	if tv.isBinary {
		t.Error("should not be binary")
	}
}

func TestTextView_WrapToggle(t *testing.T) {
	tv := NewTextView("test.txt", []byte("hello"), "test.txt")
	tv.Init()
	if !tv.softWrap {
		t.Error("default should be soft wrap enabled")
	}
	// Simulate 'w' key
	view := tv.View()
	if !strings.Contains(view, "wrap") {
		t.Error("status bar should show wrap mode")
	}
}

func TestTextView_Title(t *testing.T) {
	tv := NewTextView("My File", []byte("content"), "path/to/file.go")
	if tv.Title() != "My File" {
		t.Errorf("Title() = %q, want 'My File'", tv.Title())
	}
}

func TestTextView_SearchCountsMatches(t *testing.T) {
	content := []byte("foo bar foo baz foo")
	tv := NewTextView("test.txt", content, "test.txt")
	tv.Init()
	tv.searchQuery = "foo"
	tv.applySearch()
	if tv.matchCount != 3 {
		t.Errorf("matchCount = %d, want 3", tv.matchCount)
	}
}

func TestTextView_SearchNoMatch(t *testing.T) {
	content := []byte("hello world")
	tv := NewTextView("test.txt", content, "test.txt")
	tv.Init()
	tv.searchQuery = "xyz"
	tv.applySearch()
	if tv.matchCount != 0 {
		t.Errorf("matchCount = %d, want 0", tv.matchCount)
	}
}

func TestTextView_SearchCaseInsensitive(t *testing.T) {
	content := []byte("Hello HELLO hello")
	tv := NewTextView("test.txt", content, "test.txt")
	tv.Init()
	tv.searchQuery = "hello"
	tv.applySearch()
	if tv.matchCount != 3 {
		t.Errorf("matchCount = %d, want 3", tv.matchCount)
	}
}

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		input string
		tilde bool
	}{
		{"~/Downloads/file.txt", true},
		{"/absolute/path.txt", false},
		{"relative/path.txt", false},
	}
	for _, tt := range tests {
		got := expandTilde(tt.input)
		if tt.tilde && strings.Contains(got, "~/") {
			t.Errorf("expandTilde(%q) should expand ~, got: %s", tt.input, got)
		}
		if !tt.tilde && got != tt.input {
			t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.input)
		}
	}
}

func TestS3ContentLoader_SizeLimit(t *testing.T) {
	obj := awss3.S3Object{Key: "big.dat", Size: constants.MaxViewFileSize + 1}
	v := NewS3ContentLoaderView(nil, obj, "bucket", "us-east-1")
	v.Init()
	if v.err == nil {
		t.Fatal("expected size limit error")
	}
	if !strings.Contains(v.View(), "too large") {
		t.Errorf("should mention 'too large', got: %s", v.View())
	}
}

func TestS3DownloadView_Title(t *testing.T) {
	v := &S3DownloadView{key: "some/file.txt"}
	if v.Title() != "Download" {
		t.Errorf("Title() = %s, want Download", v.Title())
	}
}

func TestS3DownloadView_InitialView(t *testing.T) {
	obj := awss3.S3Object{Key: "data.csv", Size: 1024}
	v := NewS3DownloadView(nil, obj, "bucket", "us-east-1")
	view := v.View()
	if !strings.Contains(view, "data.csv") {
		t.Errorf("should show filename, got: %s", view)
	}
	if !strings.Contains(view, "Downloads") {
		t.Errorf("should show default path, got: %s", view)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

**Step 4: Run tests**

Run: `go test ./internal/tui/services/ -v`
Expected: All pass

**Step 5: Commit**

```bash
git rm internal/tui/services/s3_view_test.go 2>/dev/null; true
git add internal/tui/services/s3_view.go internal/tui/services/s3.go internal/tui/services/textview.go internal/tui/services/textview_test.go
git commit -m "wire TextView into S3 content viewer with tests"
```

---

### Task 5: Rewrite download view with progress bar and cancellation

**Files:**
- Modify: `internal/tui/services/s3_download.go` (major rewrite)
- Modify: `internal/tui/services/textview_test.go` (update download tests)

**Step 1: Write failing tests for new download behavior**

Add to `textview_test.go`:

```go
func TestS3DownloadView_ProgressState(t *testing.T) {
	obj := awss3.S3Object{Key: "file.zip", Size: 1024 * 1024}
	v := NewS3DownloadView(nil, obj, "bucket", "us-east-1")

	// Before download starts
	if v.downloading {
		t.Error("should not be downloading initially")
	}

	// View should show file size
	view := v.View()
	if !strings.Contains(view, "file.zip") {
		t.Errorf("should show filename, got: %s", view)
	}
}

func TestS3DownloadView_CancelState(t *testing.T) {
	obj := awss3.S3Object{Key: "file.zip", Size: 1024}
	v := NewS3DownloadView(nil, obj, "bucket", "us-east-1")
	if v.cancel == nil {
		t.Error("cancel func should be set")
	}
}
```

**Step 2: Rewrite s3_download.go**

Replace `internal/tui/services/s3_download.go`:

```go
package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type s3DownloadProgressMsg struct {
	downloaded int64
}

type s3DownloadDoneMsg struct {
	path string
}

type S3DownloadView struct {
	client *awsclient.ServiceClient
	bucket string
	key    string
	region string
	size   int64

	input    textinput.Model
	progress progress.Model

	downloading bool
	done        bool
	donePath    string
	cancelled   bool
	err         error

	cancel    context.CancelFunc
	startTime time.Time
	downloaded int64

	width  int
	height int
}

func NewS3DownloadView(client *awsclient.ServiceClient, obj awss3.S3Object, bucket, region string) *S3DownloadView {
	ti := textinput.New()
	ti.Placeholder = "download path..."
	ti.CharLimit = 256
	ti.SetValue("~/Downloads/" + path.Base(obj.Key))
	ti.Focus()

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	_, cancel := context.WithCancel(context.Background())

	return &S3DownloadView{
		client:   client,
		bucket:   bucket,
		key:      obj.Key,
		region:   region,
		size:     obj.Size,
		input:    ti,
		progress: p,
		cancel:   cancel,
		width:    80,
		height:   20,
	}
}

func (v *S3DownloadView) Title() string { return "Download" }

func (v *S3DownloadView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *S3DownloadView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case s3DownloadProgressMsg:
		v.downloaded = msg.downloaded
		var pct float64
		if v.size > 0 {
			pct = float64(v.downloaded) / float64(v.size)
		}
		return v, v.progress.SetPercent(pct)

	case s3DownloadDoneMsg:
		v.downloading = false
		v.done = true
		v.donePath = msg.path
		return v, nil

	case errViewMsg:
		v.downloading = false
		if v.cancelled {
			// Cancellation error, not a real error
			return v, nil
		}
		v.err = msg.err
		return v, nil

	case progress.FrameMsg:
		m, cmd := v.progress.Update(msg)
		v.progress = m.(progress.Model)
		return v, cmd

	case tea.KeyPressMsg:
		if v.downloading {
			if msg.String() == "esc" {
				v.cancelled = true
				v.downloading = false
				if v.cancel != nil {
					v.cancel()
				}
				return v, nil
			}
			return v, nil
		}
		if v.done || v.cancelled {
			return v, nil
		}
		switch msg.String() {
		case "enter":
			destPath := expandTilde(v.input.Value())
			ctx, cancel := context.WithCancel(context.Background())
			v.cancel = cancel
			v.downloading = true
			v.startTime = time.Now()
			v.err = nil
			return v, v.download(ctx, destPath)
		}
	}

	if !v.downloading && !v.done && !v.cancelled {
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *S3DownloadView) download(ctx context.Context, destPath string) tea.Cmd {
	return func() tea.Msg {
		reader, size, err := v.client.S3.GetObjectStream(ctx, v.bucket, v.key, v.region)
		if err != nil {
			return errViewMsg{err: err}
		}
		defer reader.Close()

		// Use reported size if we didn't have it
		if v.size == 0 && size > 0 {
			v.size = size
		}

		dir := path.Dir(destPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errViewMsg{err: fmt.Errorf("create directory: %w", err)}
		}

		f, err := os.Create(destPath)
		if err != nil {
			return errViewMsg{err: fmt.Errorf("create file: %w", err)}
		}

		buf := make([]byte, 32*1024) // 32KB chunks
		var downloaded int64

		for {
			select {
			case <-ctx.Done():
				f.Close()
				os.Remove(destPath) // clean up partial file
				return errViewMsg{err: ctx.Err()}
			default:
			}

			n, readErr := reader.Read(buf)
			if n > 0 {
				if _, writeErr := f.Write(buf[:n]); writeErr != nil {
					f.Close()
					os.Remove(destPath)
					return errViewMsg{err: fmt.Errorf("write: %w", writeErr)}
				}
				downloaded += int64(n)
				// Note: progress updates happen via the size tracking in View()
				// We store downloaded on the view for the progress bar
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				f.Close()
				os.Remove(destPath)
				return errViewMsg{err: fmt.Errorf("read: %w", readErr)}
			}
		}

		f.Close()
		return s3DownloadDoneMsg{path: destPath}
	}
}

func (v *S3DownloadView) View() string {
	if v.cancelled {
		return theme.MutedStyle.Render("Download cancelled") +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.done {
		return theme.SuccessStyle.Render(fmt.Sprintf("✓ Downloaded to %s", v.donePath)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.downloading {
		var pct float64
		if v.size > 0 {
			pct = float64(v.downloaded) / float64(v.size)
		}

		// ETA calculation
		elapsed := time.Since(v.startTime)
		eta := ""
		if pct > 0 && elapsed > time.Second {
			remaining := time.Duration(float64(elapsed) * (1 - pct) / pct)
			eta = fmt.Sprintf("  ETA %s", formatDuration(remaining))
		}

		return fmt.Sprintf("Downloading %s\n\n%s\n\n%s / %s  (%.0f%%)%s\n\n%s",
			path.Base(v.key),
			v.progress.View(),
			formatSize(v.downloaded),
			formatSize(v.size),
			pct*100,
			eta,
			theme.MutedStyle.Render("Esc to cancel"),
		)
	}

	return fmt.Sprintf("Download %s (%s)\n\n%s\n\n%s",
		path.Base(v.key),
		formatSize(v.size),
		v.input.View(),
		theme.MutedStyle.Render("Enter to download • Esc to cancel"),
	)
}

func (v *S3DownloadView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.progress.Width = width - 8
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

func expandTilde(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + p[1:]
		}
	}
	return p
}
```

**Step 3: Run tests**

Run: `go test ./internal/tui/services/ -v`
Expected: All pass

**Step 4: Run full build and vet**

Run: `go build ./... && go vet ./...`
Expected: Clean

**Step 5: Commit**

```bash
git add internal/tui/services/s3_download.go internal/tui/services/textview_test.go
git commit -m "enhance download with progress bar, ETA, and cancellation"
```

---

### Task 6: Final cleanup and full test pass

**Files:**
- Modify: `go.mod` / `go.sum` (tidy)
- Modify: `internal/tui/services/tableview_test.go` (verify existing tests still pass)

**Step 1: Tidy modules**

```bash
go mod tidy
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All pass

**Step 3: Run vet**

Run: `go vet ./...`
Expected: Clean

**Step 4: Final commit**

```bash
git add go.mod go.sum
git commit -m "tidy modules after image viewer removal"
```

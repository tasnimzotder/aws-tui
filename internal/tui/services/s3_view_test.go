package services

import (
	"strings"
	"testing"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/constants"
)

// --- S3ContentLoaderView tests ---

func TestS3ContentLoaderView_SizeLimit(t *testing.T) {
	obj := awss3.S3Object{
		Key:  "big-file.dat",
		Size: constants.MaxViewFileSize + 1,
	}
	v := NewS3ContentLoaderView(nil, obj, "bucket", "us-east-1")
	v.Init()

	if v.err == nil {
		t.Fatal("expected size limit error, got nil")
	}
	if !contains(v.View(), "too large") {
		t.Errorf("error should mention 'too large', got: %s", v.View())
	}
}

func TestS3ContentLoaderView_UnderSizeLimit(t *testing.T) {
	obj := awss3.S3Object{
		Key:  "small.txt",
		Size: 100,
	}
	v := NewS3ContentLoaderView(nil, obj, "bucket", "us-east-1")
	if v.err != nil {
		t.Errorf("unexpected error for small file: %v", v.err)
	}
}

func TestS3ContentLoaderView_Title(t *testing.T) {
	v := &S3ContentLoaderView{key: "path/to/file.txt"}
	if v.Title() != "file.txt" {
		t.Errorf("Title() = %s, want file.txt", v.Title())
	}
}

// --- TextView tests ---

func TestTextView_Binary(t *testing.T) {
	tv := NewTextView("data.bin", []byte{0x80, 0x81, 0x82, 0x83}, "data.bin")
	tv.Init()
	if !tv.isBinary {
		t.Error("expected isBinary to be true for invalid UTF-8")
	}
	if !contains(tv.View(), "Binary file") {
		t.Errorf("expected binary notice, got: %s", tv.View())
	}
}

func TestTextView_PlainText(t *testing.T) {
	tv := NewTextView("readme.txt", []byte("hello world"), "readme.txt")
	if tv.isBinary {
		t.Error("expected isBinary to be false for valid UTF-8")
	}
}

func TestTextView_JSONPrettyPrint(t *testing.T) {
	tv := NewTextView("data.json", []byte(`{"key":"value","arr":[1,2,3]}`), "data.json")
	if !contains(tv.rawText, "\"key\"") {
		t.Errorf("expected pretty-printed JSON, got: %s", tv.rawText)
	}
}

func TestTextView_InvalidJSON(t *testing.T) {
	raw := `{"key": "value", broken}`
	tv := NewTextView("broken.json", []byte(raw), "broken.json")
	if !contains(tv.rawText, "key") {
		t.Errorf("expected raw text fallback to contain 'key', got: %s", tv.rawText)
	}
}

func TestTextView_SyntaxHighlight(t *testing.T) {
	goCode := []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")
	tv := NewTextView("main.go", goCode, "main.go")
	// Chroma should add ANSI escape codes for Go syntax
	if !contains(tv.rawText, "\x1b[") {
		t.Error("Go code should be syntax highlighted with ANSI codes")
	}
}

func TestTextView_NoHighlightForUnknown(t *testing.T) {
	tv := NewTextView("data.xyz", []byte("just plain text"), "data.xyz")
	if tv.isBinary {
		t.Error("should not be binary")
	}
}

func TestTextView_Title(t *testing.T) {
	tv := NewTextView("My File", []byte("content"), "path/to/file.txt")
	if tv.Title() != "My File" {
		t.Errorf("Title() = %q, want 'My File'", tv.Title())
	}
}

func TestTextView_DefaultWrapEnabled(t *testing.T) {
	tv := NewTextView("test.txt", []byte("hello"), "test.txt")
	tv.Init()
	if !tv.softWrap {
		t.Error("default should be soft wrap enabled")
	}
}

func TestTextView_ViewShowsStatusBar(t *testing.T) {
	tv := NewTextView("test.txt", []byte("hello"), "test.txt")
	tv.Init()
	view := tv.View()
	if !contains(view, "wrap") {
		t.Error("status bar should show wrap mode")
	}
	if !contains(view, "search") {
		t.Error("status bar should show search hint")
	}
	if !contains(view, "Esc back") {
		t.Error("status bar should show Esc hint")
	}
}

func TestTextView_SearchCountsMatches(t *testing.T) {
	content := []byte("foo bar foo baz foo")
	tv := NewTextView("test.txt", content, "test.txt")
	tv.Init()
	tv.searchInput.SetValue("foo")
	tv.applySearch()
	if tv.matchCount != 3 {
		t.Errorf("matchCount = %d, want 3", tv.matchCount)
	}
}

func TestTextView_SearchNoMatch(t *testing.T) {
	content := []byte("hello world")
	tv := NewTextView("test.txt", content, "test.txt")
	tv.Init()
	tv.searchInput.SetValue("xyz")
	tv.applySearch()
	if tv.matchCount != 0 {
		t.Errorf("matchCount = %d, want 0", tv.matchCount)
	}
}

func TestTextView_SearchCaseInsensitive(t *testing.T) {
	content := []byte("Hello HELLO hello")
	tv := NewTextView("test.txt", content, "test.txt")
	tv.Init()
	tv.searchInput.SetValue("hello")
	tv.applySearch()
	if tv.matchCount != 3 {
		t.Errorf("matchCount = %d, want 3", tv.matchCount)
	}
}

func TestTextView_SearchEmptyQuery(t *testing.T) {
	tv := NewTextView("test.txt", []byte("content"), "test.txt")
	tv.Init()
	tv.searchInput.SetValue("")
	tv.applySearch()
	if tv.matchCount != 0 {
		t.Errorf("matchCount should be 0 for empty query, got %d", tv.matchCount)
	}
}

func TestTextView_CancelSearchClearsState(t *testing.T) {
	tv := NewTextView("test.txt", []byte("foo bar foo"), "test.txt")
	tv.Init()
	tv.searchInput.SetValue("foo")
	tv.applySearch()
	if tv.matchCount != 2 {
		t.Fatalf("expected 2 matches, got %d", tv.matchCount)
	}
	tv.cancelSearch()
	if tv.matchCount != 0 {
		t.Errorf("matchCount should be 0 after cancel, got %d", tv.matchCount)
	}
	if tv.searchQuery != "" {
		t.Errorf("searchQuery should be empty after cancel, got %q", tv.searchQuery)
	}
}

func TestTextView_HighlightMatches(t *testing.T) {
	tv := NewTextView("test.txt", []byte("hello world hello"), "test.txt")
	tv.Init()
	highlighted := tv.highlightMatches("hello world hello", "hello")
	// Should contain ANSI codes from the highlight style
	if !contains(highlighted, "\x1b[") {
		t.Error("highlighted text should contain ANSI escape codes")
	}
	// Should still contain the non-matching part
	if !contains(highlighted, "world") {
		t.Error("highlighted text should preserve non-matching parts")
	}
}

func TestTextView_WrapToggle(t *testing.T) {
	tv := NewTextView("test.txt", []byte("hello"), "test.txt")
	tv.Init()
	if !tv.softWrap {
		t.Fatal("default should be wrap on")
	}
	tv.softWrap = !tv.softWrap
	tv.viewport.SoftWrap = tv.softWrap
	if tv.softWrap {
		t.Error("after toggle, wrap should be off")
	}
}

// --- Download view tests ---

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
	if !contains(view, "data.csv") {
		t.Errorf("initial view should show filename, got: %s", view)
	}
	if !contains(view, "Downloads") {
		t.Errorf("initial view should show default path, got: %s", view)
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

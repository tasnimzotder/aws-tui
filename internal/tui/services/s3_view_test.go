package services

import (
	"strings"
	"testing"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/constants"
)

func TestS3ContentLoaderView_SizeLimit(t *testing.T) {
	obj := awss3.S3Object{
		Key:  "big-file.dat",
		Size: constants.MaxViewFileSize + 1,
	}
	v := NewS3ObjectContentView(nil, obj, "bucket", "us-east-1")
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
	v := NewS3ObjectContentView(nil, obj, "bucket", "us-east-1")
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
	// rawText should contain pretty-printed JSON (possibly with syntax highlighting)
	if !contains(tv.rawText, "\"key\"") {
		t.Errorf("expected pretty-printed JSON, got: %s", tv.rawText)
	}
}

func TestTextView_InvalidJSON(t *testing.T) {
	raw := `{"key": "value", broken}`
	tv := NewTextView("broken.json", []byte(raw), "broken.json")
	// Should fall back to raw text (possibly highlighted), but must contain original content
	if !contains(tv.rawText, "key") {
		t.Errorf("expected raw text fallback to contain 'key', got: %s", tv.rawText)
	}
}

func TestTextView_Title(t *testing.T) {
	tv := NewTextView("file.txt", []byte("content"), "path/to/file.txt")
	if tv.Title() != "file.txt" {
		t.Errorf("Title() = %s, want file.txt", tv.Title())
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

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

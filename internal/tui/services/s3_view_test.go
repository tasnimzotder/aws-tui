package services

import (
	"strings"
	"testing"

	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/constants"
)

func TestS3ObjectContentView_SizeLimit(t *testing.T) {
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

func TestS3ObjectContentView_UnderSizeLimit(t *testing.T) {
	obj := awss3.S3Object{
		Key:  "small.txt",
		Size: 100,
	}
	v := NewS3ObjectContentView(nil, obj, "bucket", "us-east-1")
	if v.err != nil {
		t.Errorf("unexpected error for small file: %v", v.err)
	}
}

func TestS3ObjectContentView_FormatText_Plain(t *testing.T) {
	v := &S3ObjectContentView{key: "readme.txt"}
	content := v.formatText([]byte("hello world"))
	if content != "hello world" {
		t.Errorf("expected plain text, got: %s", content)
	}
}

func TestS3ObjectContentView_FormatText_Binary(t *testing.T) {
	v := &S3ObjectContentView{key: "data.bin"}
	content := v.formatText([]byte{0x80, 0x81, 0x82, 0x83})
	if !contains(content, "Binary file") {
		t.Errorf("expected binary notice, got: %s", content)
	}
}

func TestS3ObjectContentView_FormatText_JSON(t *testing.T) {
	v := &S3ObjectContentView{key: "data.json"}
	content := v.formatText([]byte(`{"key":"value","arr":[1,2,3]}`))
	if !contains(content, "\"key\": \"value\"") {
		t.Errorf("expected pretty-printed JSON, got: %s", content)
	}
}

func TestS3ObjectContentView_FormatText_InvalidJSON(t *testing.T) {
	v := &S3ObjectContentView{key: "broken.json"}
	raw := `{"key": "value", broken}`
	content := v.formatText([]byte(raw))
	if content != raw {
		t.Errorf("expected raw text fallback, got: %s", content)
	}
}

func TestS3ObjectContentView_Title(t *testing.T) {
	v := &S3ObjectContentView{key: "path/to/file.txt"}
	if v.Title() != "file.txt" {
		t.Errorf("Title() = %s, want file.txt", v.Title())
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

package services

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type s3DownloadDoneMsg struct {
	path string
}

type S3DownloadView struct {
	client      *awsclient.ServiceClient
	bucket      string
	key         string
	region      string
	input       textinput.Model
	spinner     spinner.Model
	downloading bool
	done        bool
	donePath    string
	err         error
	width       int
	height      int
}

func NewS3DownloadView(client *awsclient.ServiceClient, obj awss3.S3Object, bucket, region string) *S3DownloadView {
	ti := textinput.New()
	ti.Placeholder = "download path..."
	ti.CharLimit = 256

	defaultPath := "~/Downloads/" + path.Base(obj.Key)
	ti.SetValue(defaultPath)
	ti.Focus()

	return &S3DownloadView{
		client:  client,
		bucket:  bucket,
		key:     obj.Key,
		region:  region,
		input:   ti,
		spinner: theme.NewSpinner(),
		width:   80,
		height:  20,
	}
}

func (v *S3DownloadView) Title() string { return "Download" }

func (v *S3DownloadView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *S3DownloadView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case s3DownloadDoneMsg:
		v.downloading = false
		v.done = true
		v.donePath = msg.path
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.downloading = false
		return v, nil

	case tea.KeyPressMsg:
		if v.downloading {
			return v, nil
		}
		switch msg.String() {
		case "enter":
			if !v.done {
				v.downloading = true
				v.err = nil
				destPath := expandTilde(v.input.Value())
				return v, tea.Batch(v.spinner.Tick, v.download(destPath))
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	if !v.downloading && !v.done {
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *S3DownloadView) download(destPath string) tea.Cmd {
	return func() tea.Msg {
		data, err := v.client.S3.GetObject(context.Background(), v.bucket, v.key, v.region)
		if err != nil {
			return errViewMsg{err: err}
		}

		// Ensure parent directory exists
		dir := path.Dir(destPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errViewMsg{err: fmt.Errorf("create directory: %w", err)}
		}

		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return errViewMsg{err: fmt.Errorf("write file: %w", err)}
		}

		return s3DownloadDoneMsg{path: destPath}
	}
}

func (v *S3DownloadView) View() string {
	if v.downloading {
		return v.spinner.View() + " Downloading..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.done {
		return theme.SuccessStyle.Render(fmt.Sprintf("Downloaded to %s", v.donePath)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}

	return fmt.Sprintf("Download %s to:\n\n%s\n\n%s",
		path.Base(v.key),
		v.input.View(),
		theme.MutedStyle.Render("Enter to download • Esc to cancel"),
	)
}

func (v *S3DownloadView) SetSize(width, height int) {
	v.width = width
	v.height = height
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

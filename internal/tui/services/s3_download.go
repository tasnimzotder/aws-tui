package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type s3DownloadDoneMsg struct {
	path string
}

type s3DownloadTickMsg struct{}

type S3DownloadView struct {
	client      *awsclient.ServiceClient
	bucket      string
	key         string
	region      string
	size        int64
	input       textinput.Model
	progress    progress.Model
	downloading bool
	done        bool
	donePath    string
	cancelled   bool
	err         error
	cancel      context.CancelFunc
	startTime   time.Time
	downloaded  int64 // accessed atomically from download goroutine
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

	// Create a no-op cancel func so it's never nil
	_, cancel := context.WithCancel(context.Background())

	return &S3DownloadView{
		client:   client,
		bucket:   bucket,
		key:      obj.Key,
		region:   region,
		size:     obj.Size,
		input:    ti,
		progress: progress.New(progress.WithDefaultBlend(), progress.WithoutPercentage()),
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
	case s3DownloadDoneMsg:
		v.downloading = false
		v.done = true
		v.donePath = msg.path
		return v, nil

	case errViewMsg:
		if v.cancelled {
			// Don't overwrite cancelled state with the "download cancelled" error
			return v, nil
		}
		v.err = msg.err
		v.downloading = false
		return v, nil

	case s3DownloadTickMsg:
		if !v.downloading {
			return v, nil
		}
		downloaded := atomic.LoadInt64(&v.downloaded)
		var percent float64
		if v.size > 0 {
			percent = float64(downloaded) / float64(v.size)
			if percent > 1.0 {
				percent = 1.0
			}
		}
		cmd := v.progress.SetPercent(percent)
		return v, tea.Batch(cmd, downloadTick())

	case progress.FrameMsg:
		var cmd tea.Cmd
		v.progress, cmd = v.progress.Update(msg)
		return v, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if v.downloading {
				v.downloading = false
				v.cancelled = true
				v.cancel()
				return v, nil
			}
		case "enter":
			if !v.downloading && !v.done && !v.cancelled && v.err == nil {
				ctx, cancel := context.WithCancel(context.Background())
				v.cancel = cancel
				v.downloading = true
				v.err = nil
				v.startTime = time.Now()
				atomic.StoreInt64(&v.downloaded, 0)
				destPath := expandTilde(v.input.Value())
				return v, tea.Batch(v.download(ctx, destPath), downloadTick())
			}
		}

	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		return v, nil
	}

	if !v.downloading && !v.done && !v.cancelled && v.err == nil {
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}
	return v, nil
}

func downloadTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return s3DownloadTickMsg{}
	})
}

func (v *S3DownloadView) download(ctx context.Context, destPath string) tea.Cmd {
	return func() tea.Msg {
		reader, size, err := v.client.S3.GetObjectStream(ctx, v.bucket, v.key, v.region)
		if err != nil {
			return errViewMsg{err: err}
		}
		defer reader.Close()

		// Use actual content length if available
		if size > 0 {
			v.size = size
		}

		// Ensure parent directory exists
		dir := path.Dir(destPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errViewMsg{err: fmt.Errorf("create directory: %w", err)}
		}

		f, err := os.Create(destPath)
		if err != nil {
			return errViewMsg{err: fmt.Errorf("create file: %w", err)}
		}

		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				f.Close()
				os.Remove(destPath)
				return errViewMsg{err: fmt.Errorf("download cancelled")}
			default:
			}

			n, readErr := reader.Read(buf)
			if n > 0 {
				if _, writeErr := f.Write(buf[:n]); writeErr != nil {
					f.Close()
					os.Remove(destPath)
					return errViewMsg{err: fmt.Errorf("write file: %w", writeErr)}
				}
				atomic.AddInt64(&v.downloaded, int64(n))
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				f.Close()
				os.Remove(destPath)
				return errViewMsg{err: readErr}
			}
		}

		f.Close()
		return s3DownloadDoneMsg{path: destPath}
	}
}

func (v *S3DownloadView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.cancelled {
		return theme.ErrorStyle.Render("Download cancelled") +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.done {
		return theme.SuccessStyle.Render(fmt.Sprintf("Downloaded to %s", v.donePath)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.downloading {
		downloaded := atomic.LoadInt64(&v.downloaded)
		var percent float64
		if v.size > 0 {
			percent = float64(downloaded) / float64(v.size) * 100
			if percent > 100 {
				percent = 100
			}
		}

		progressBar := v.progress.View()

		stats := fmt.Sprintf("%s / %s  %.0f%%",
			formatSize(downloaded),
			formatSize(v.size),
			percent,
		)

		var eta string
		elapsed := time.Since(v.startTime)
		if downloaded > 0 && v.size > 0 && percent < 100 {
			remaining := time.Duration(float64(elapsed) * (float64(v.size-downloaded) / float64(downloaded)))
			eta = fmt.Sprintf("  ETA %s", formatDuration(remaining))
		}

		return fmt.Sprintf("Downloading %s\n\n%s\n\n%s%s\n\n%s",
			path.Base(v.key),
			progressBar,
			stats,
			eta,
			theme.MutedStyle.Render("Esc to cancel"),
		)
	}

	return fmt.Sprintf("Download %s (%s) to:\n\n%s\n\n%s",
		path.Base(v.key),
		formatSize(v.size),
		v.input.View(),
		theme.MutedStyle.Render("Enter to download - Esc to cancel"),
	)
}

func (v *S3DownloadView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.progress.SetWidth(width - 8)
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

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

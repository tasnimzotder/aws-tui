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

// NewS3ObjectContentView is a backward-compat alias for s3.go which still references this.
func NewS3ObjectContentView(client *awsclient.ServiceClient, obj awss3.S3Object, bucket, region string) *S3ContentLoaderView {
	return NewS3ContentLoaderView(client, obj, bucket, region)
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

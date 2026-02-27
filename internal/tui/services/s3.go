package services

import (
	"context"
	"fmt"
	"path"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awss3 "tasnim.dev/aws-tui/internal/aws/s3"
	"tasnim.dev/aws-tui/internal/utils"
)

func NewS3BucketsView(client *awsclient.ServiceClient) *TableView[awss3.S3Bucket] {
	return NewTableView(TableViewConfig[awss3.S3Bucket]{
		Title:       "S3",
		LoadingText: "Loading buckets...",
		Columns: []table.Column{
			{Title: "Name", Width: 35},
			{Title: "Region", Width: 16},
			{Title: "Created", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awss3.S3Bucket, error) {
			return client.S3.ListBuckets(ctx)
		},
		RowMapper: func(b awss3.S3Bucket) table.Row {
			return table.Row{b.Name, b.Region, utils.TimeOrDash(b.CreatedAt, utils.DateOnly)}
		},
		CopyIDFunc:  func(b awss3.S3Bucket) string { return b.Name },
		CopyARNFunc: func(b awss3.S3Bucket) string { return fmt.Sprintf("arn:aws:s3:::%s", b.Name) },
		OnEnter: func(b awss3.S3Bucket) tea.Cmd {
			return pushView(NewS3ObjectsView(client, b.Name, "", b.Region))
		},
	})
}

func NewS3ObjectsView(client *awsclient.ServiceClient, bucket, prefix, region string) *TableView[awss3.S3Object] {
	title := bucket
	if prefix != "" {
		title = path.Base(prefix[:len(prefix)-1]) + "/"
	}

	var nextToken string

	s3HelpCtx := HelpContextS3Objects
	return NewTableView(TableViewConfig[awss3.S3Object]{
		Title:       title,
		LoadingText: "Loading objects...",
		HelpCtx:     &s3HelpCtx,
		Columns: []table.Column{
			{Title: "Name", Width: 40},
			{Title: "Size", Width: 12},
			{Title: "Last Modified", Width: 20},
			{Title: "Storage Class", Width: 16},
		},
		FetchFunc: func(ctx context.Context) ([]awss3.S3Object, error) {
			result, err := client.S3.ListObjects(ctx, bucket, prefix, "", region)
			if err != nil {
				return nil, err
			}
			nextToken = result.NextToken
			return result.Objects, nil
		},
		RowMapper: func(obj awss3.S3Object) table.Row {
			if obj.IsPrefix {
				name := obj.Key
				if prefix != "" {
					name = obj.Key[len(prefix):]
				}
				return table.Row{"\U0001F4C1 " + name, "—", "—", "—"}
			}
			name := obj.Key
			if prefix != "" {
				name = obj.Key[len(prefix):]
			}
			return table.Row{name, formatSize(obj.Size), utils.TimeOrDash(obj.LastModified, utils.DateTime), obj.StorageClass}
		},
		CopyIDFunc: func(obj awss3.S3Object) string {
			return fmt.Sprintf("s3://%s/%s", bucket, obj.Key)
		},
		CopyARNFunc: func(obj awss3.S3Object) string {
			return fmt.Sprintf("arn:aws:s3:::%s/%s", bucket, obj.Key)
		},
		OnEnter: func(obj awss3.S3Object) tea.Cmd {
			if obj.IsPrefix {
				return pushView(NewS3ObjectsView(client, bucket, obj.Key, region))
			}
			return nil
		},
		KeyHandlers: map[string]func(awss3.S3Object) tea.Cmd{
			"v": func(obj awss3.S3Object) tea.Cmd {
				if obj.IsPrefix {
					return nil
				}
				return pushView(NewS3ContentLoaderView(client, obj, bucket, region))
			},
			"d": func(obj awss3.S3Object) tea.Cmd {
				if obj.IsPrefix {
					return nil
				}
				return pushView(NewS3DownloadView(client, obj, bucket, region))
			},
		},
		LoadMoreFunc: func(ctx context.Context) ([]awss3.S3Object, bool, error) {
			if nextToken == "" {
				return nil, false, nil
			}
			result, err := client.S3.ListObjects(ctx, bucket, prefix, nextToken, region)
			if err != nil {
				return nil, false, err
			}
			nextToken = result.NextToken
			return result.Objects, nextToken != "", nil
		},
	})
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

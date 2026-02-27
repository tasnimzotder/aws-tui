package services

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsecr "tasnim.dev/aws-tui/internal/aws/ecr"
	"tasnim.dev/aws-tui/internal/utils"
)

func NewECRReposView(client *awsclient.ServiceClient) *TableView[awsecr.ECRRepo] {
	return NewTableView(TableViewConfig[awsecr.ECRRepo]{
		Title:       "ECR",
		LoadingText: "Loading repositories...",
		Columns: []table.Column{
			{Title: "Repository", Width: 35},
			{Title: "Images", Width: 8},
			{Title: "Created", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awsecr.ECRRepo, error) {
			return client.ECR.ListRepositories(ctx)
		},
		RowMapper: func(r awsecr.ECRRepo) table.Row {
			return table.Row{r.Name, fmt.Sprintf("%d", r.ImageCount), utils.TimeOrDash(r.CreatedAt, utils.DateOnly)}
		},
		CopyIDFunc:  func(r awsecr.ECRRepo) string { return r.Name },
		CopyARNFunc: func(r awsecr.ECRRepo) string { return r.URI },
		OnEnter: func(r awsecr.ECRRepo) tea.Cmd {
			return pushView(NewECRImagesView(client, r.Name))
		},
	})
}

func NewECRImagesView(client *awsclient.ServiceClient, repoName string) *TableView[awsecr.ECRImage] {
	return NewTableView(TableViewConfig[awsecr.ECRImage]{
		Title:       repoName,
		LoadingText: "Loading images...",
		Columns: []table.Column{
			{Title: "Tag", Width: 25},
			{Title: "Digest", Width: 22},
			{Title: "Size (MB)", Width: 10},
			{Title: "Pushed", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awsecr.ECRImage, error) {
			return client.ECR.ListImages(ctx, repoName)
		},
		RowMapper: func(img awsecr.ECRImage) table.Row {
			tag := "—"
			if len(img.Tags) > 0 {
				tag = strings.Join(img.Tags, ", ")
			}
			return table.Row{tag, img.Digest, fmt.Sprintf("%.1f", img.SizeMB), utils.TimeOrDash(img.PushedAt, utils.DateTime)}
		},
		CopyIDFunc: func(img awsecr.ECRImage) string { return img.Digest },
	})
}

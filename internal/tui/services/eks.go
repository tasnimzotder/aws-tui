package services

import (
	"context"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

func NewEKSClustersView(client *awsclient.ServiceClient) *TableView[awseks.EKSCluster] {
	var nextToken *string
	return NewTableView(TableViewConfig[awseks.EKSCluster]{
		Title:       "EKS",
		LoadingText: "Loading EKS clusters...",
		Columns: []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Status", Width: 12},
			{Title: "K8s Version", Width: 12},
			{Title: "Platform", Width: 10},
			{Title: "Created", Width: 20},
		},
		FetchFuncPaged: func(ctx context.Context) ([]awseks.EKSCluster, bool, error) {
			nextToken = nil
			clusters, nt, err := client.EKS.ListClustersPage(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return clusters, nt != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awseks.EKSCluster, bool, error) {
			clusters, nt, err := client.EKS.ListClustersPage(ctx, nextToken)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return clusters, nt != nil, nil
		},
		RowMapper: func(cl awseks.EKSCluster) table.Row {
			return table.Row{cl.Name, theme.RenderStatus(cl.Status), cl.Version, cl.PlatformVersion, utils.TimeOrDash(cl.CreatedAt, utils.DateOnly)}
		},
		CopyIDFunc:  func(cl awseks.EKSCluster) string { return cl.Name },
		CopyARNFunc: func(cl awseks.EKSCluster) string { return cl.ARN },
		OnEnter: func(cl awseks.EKSCluster) tea.Cmd {
			region := client.Cfg.Region
			return pushView(NewEKSClusterDetailView(client, cl, region))
		},
	})
}

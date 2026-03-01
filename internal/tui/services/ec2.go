package services

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

func NewEC2View(client *awsclient.ServiceClient, profile, region string) *TableView[ec2ViewItem] {
	helpCtx := HelpContextEC2
	var nextToken *string
	var accSummary awsec2.EC2Summary
	return NewTableView(TableViewConfig[ec2ViewItem]{
		Title:       "EC2",
		LoadingText: "Loading EC2 instances...",
		Columns: []table.Column{
			{Title: "Name", Width: 20},
			{Title: "Instance ID", Width: 22},
			{Title: "Type", Width: 12},
			{Title: "State", Width: 10},
			{Title: "AZ", Width: 14},
			{Title: "Arch", Width: 8},
			{Title: "Private IP", Width: 16},
			{Title: "Public IP", Width: 16},
		},
		FetchFuncPaged: func(ctx context.Context) ([]ec2ViewItem, bool, error) {
			nextToken = nil
			accSummary = awsec2.EC2Summary{}
			instances, summary, nt, err := client.EC2.ListInstancesPage(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			accSummary = summary
			items := make([]ec2ViewItem, len(instances))
			for i, inst := range instances {
				items[i] = ec2ViewItem{instance: inst, summary: accSummary}
			}
			return items, nt != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]ec2ViewItem, bool, error) {
			instances, summary, nt, err := client.EC2.ListInstancesPage(ctx, nextToken)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			accSummary.Running += summary.Running
			accSummary.Stopped += summary.Stopped
			accSummary.Total += summary.Total
			items := make([]ec2ViewItem, len(instances))
			for i, inst := range instances {
				items[i] = ec2ViewItem{instance: inst, summary: accSummary}
			}
			return items, nt != nil, nil
		},
		RowMapper: func(item ec2ViewItem) table.Row {
			i := item.instance
			return table.Row{i.Name, i.InstanceID, i.Type, theme.RenderStatus(i.State), i.AZ, i.Architecture, i.PrivateIP, i.PublicIP}
		},
		CopyIDFunc: func(item ec2ViewItem) string {
			return item.instance.InstanceID
		},
		SummaryFunc: func(items []ec2ViewItem) string {
			if len(items) == 0 {
				return ""
			}
			s := accSummary
			return fmt.Sprintf(
				"%s %s   %s %s   %s %s",
				theme.MutedStyle.Render("Running:"), theme.SuccessStyle.Render(fmt.Sprintf("%d", s.Running)),
				theme.MutedStyle.Render("Stopped:"), theme.MutedStyle.Render(fmt.Sprintf("%d", s.Stopped)),
				theme.MutedStyle.Render("Total:"), theme.MutedStyle.Render(fmt.Sprintf("%d", s.Total)),
			)
		},
		OnEnter: func(item ec2ViewItem) tea.Cmd {
			return pushView(NewEC2DetailView(client, item.instance, profile, region))
		},
		KeyHandlers: map[string]func(ec2ViewItem) tea.Cmd{
			"x": func(item ec2ViewItem) tea.Cmd {
				return pushView(newSSMInputView(item.instance, profile, region))
			},
		},
		HelpCtx:      &helpCtx,
		HeightOffset: 3,
	})
}

// ec2ViewItem bundles an instance with the summary (computed once during fetch).
type ec2ViewItem struct {
	instance awsec2.EC2Instance
	summary  awsec2.EC2Summary
}

// pushView is a helper to create a PushViewMsg command.
func pushView(v View) tea.Cmd {
	return func() tea.Msg { return PushViewMsg{View: v} }
}

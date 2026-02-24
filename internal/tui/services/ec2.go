package services

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

func NewEC2View(client *awsclient.ServiceClient) *TableView[ec2ViewItem] {
	return NewTableView(TableViewConfig[ec2ViewItem]{
		Title:       "EC2",
		LoadingText: "Loading EC2 instances...",
		Columns: []table.Column{
			{Title: "Name", Width: 20},
			{Title: "Instance ID", Width: 22},
			{Title: "Type", Width: 12},
			{Title: "State", Width: 10},
			{Title: "Private IP", Width: 16},
			{Title: "Public IP", Width: 16},
		},
		FetchFunc: func(ctx context.Context) ([]ec2ViewItem, error) {
			instances, summary, err := client.EC2.ListInstances(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]ec2ViewItem, len(instances))
			for i, inst := range instances {
				items[i] = ec2ViewItem{instance: inst, summary: summary}
			}
			return items, nil
		},
		RowMapper: func(item ec2ViewItem) table.Row {
			i := item.instance
			return table.Row{i.Name, i.InstanceID, i.Type, i.State, i.PrivateIP, i.PublicIP}
		},
		CopyIDFunc: func(item ec2ViewItem) string {
			return item.instance.InstanceID
		},
		SummaryFunc: func(items []ec2ViewItem) string {
			if len(items) == 0 {
				return ""
			}
			s := items[0].summary
			return fmt.Sprintf(
				"%s %s   %s %s   %s %s",
				theme.MutedStyle.Render("Running:"), theme.SuccessStyle.Render(fmt.Sprintf("%d", s.Running)),
				theme.MutedStyle.Render("Stopped:"), theme.MutedStyle.Render(fmt.Sprintf("%d", s.Stopped)),
				theme.MutedStyle.Render("Total:"), theme.MutedStyle.Render(fmt.Sprintf("%d", s.Total)),
			)
		},
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

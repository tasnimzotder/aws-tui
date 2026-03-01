package services

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

func NewELBLoadBalancersView(client *awsclient.ServiceClient) *TableView[awselb.ELBLoadBalancer] {
	helpCtx := HelpContextELB
	return NewTableView(TableViewConfig[awselb.ELBLoadBalancer]{
		Title:       "ELB",
		LoadingText: "Loading load balancers...",
		Columns: []table.Column{
			{Title: "Name", Width: 25},
			{Title: "Type", Width: 12},
			{Title: "State", Width: 10},
			{Title: "Scheme", Width: 16},
			{Title: "DNS Name", Width: 35},
			{Title: "VPC", Width: 15},
			{Title: "Created", Width: 12},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBLoadBalancer, error) {
			return client.ELB.ListLoadBalancers(ctx)
		},
		RowMapper: func(lb awselb.ELBLoadBalancer) table.Row {
			created := "—"
			if !lb.CreatedAt.IsZero() {
				created = lb.CreatedAt.Format("2006-01-02")
			}
			return table.Row{lb.Name, lb.Type, theme.RenderStatus(lb.State), lb.Scheme, lb.DNSName, lb.VPCID, created}
		},
		CopyIDFunc:  func(lb awselb.ELBLoadBalancer) string { return lb.Name },
		CopyARNFunc: func(lb awselb.ELBLoadBalancer) string { return lb.ARN },
		OnEnter: func(lb awselb.ELBLoadBalancer) tea.Cmd {
			return pushView(NewELBDetailView(client, lb))
		},
		HelpCtx: &helpCtx,
	})
}

func NewELBListenersView(client *awsclient.ServiceClient, lbARN, lbName string) *TableView[awselb.ELBListener] {
	return NewTableView(TableViewConfig[awselb.ELBListener]{
		Title:       lbName,
		LoadingText: "Loading listeners...",
		Columns: []table.Column{
			{Title: "Port", Width: 8},
			{Title: "Protocol", Width: 10},
			{Title: "Default Action", Width: 50},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBListener, error) {
			return client.ELB.ListListeners(ctx, lbARN)
		},
		RowMapper: func(l awselb.ELBListener) table.Row {
			return table.Row{fmt.Sprintf("%d", l.Port), l.Protocol, l.DefaultAction}
		},
		CopyIDFunc:  func(l awselb.ELBListener) string { return fmt.Sprintf("%d", l.Port) },
		CopyARNFunc: func(l awselb.ELBListener) string { return l.ARN },
	})
}

func NewELBTargetGroupsView(client *awsclient.ServiceClient, listenerARN, title string) *TableView[awselb.ELBTargetGroup] {
	return NewTableView(TableViewConfig[awselb.ELBTargetGroup]{
		Title:       title,
		LoadingText: "Loading target groups...",
		Columns: []table.Column{
			{Title: "Name", Width: 25},
			{Title: "Protocol", Width: 10},
			{Title: "Port", Width: 8},
			{Title: "Target Type", Width: 12},
			{Title: "Healthy", Width: 8},
			{Title: "Unhealthy", Width: 10},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBTargetGroup, error) {
			return client.ELB.ListListenerTargetGroups(ctx, listenerARN)
		},
		RowMapper: func(tg awselb.ELBTargetGroup) table.Row {
			return table.Row{
				tg.Name,
				tg.Protocol,
				fmt.Sprintf("%d", tg.Port),
				tg.TargetType,
				fmt.Sprintf("%d", tg.HealthyCount),
				fmt.Sprintf("%d", tg.UnhealthyCount),
			}
		},
		CopyIDFunc:  func(tg awselb.ELBTargetGroup) string { return tg.Name },
		CopyARNFunc: func(tg awselb.ELBTargetGroup) string { return tg.ARN },
	})
}

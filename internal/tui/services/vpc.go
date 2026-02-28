package services

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
)

func NewVPCListView(client *awsclient.ServiceClient) *TableView[awsvpc.VPCInfo] {
	return NewTableView(TableViewConfig[awsvpc.VPCInfo]{
		Title:       "VPC",
		LoadingText: "Loading VPCs...",
		Columns: []table.Column{
			{Title: "VPC ID", Width: 24},
			{Title: "Name", Width: 20},
			{Title: "CIDR", Width: 18},
			{Title: "Default", Width: 8},
			{Title: "State", Width: 12},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.VPCInfo, error) {
			return client.VPC.ListVPCs(ctx)
		},
		RowMapper: func(vpc awsvpc.VPCInfo) table.Row {
			def := "No"
			if vpc.IsDefault {
				def = "Yes"
			}
			return table.Row{vpc.VPCID, vpc.Name, vpc.CIDR, def, vpc.State}
		},
		CopyIDFunc: func(vpc awsvpc.VPCInfo) string { return vpc.VPCID },
		OnEnter: func(vpc awsvpc.VPCInfo) tea.Cmd {
			return pushView(NewVPCDetailView(client, vpc))
		},
	})
}

// --- Subnets View ---

func NewSubnetsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.SubnetInfo] {
	return NewTableView(TableViewConfig[awsvpc.SubnetInfo]{
		Title:       "Subnets",
		LoadingText: "Loading subnets...",
		Columns: []table.Column{
			{Title: "Subnet ID", Width: 26},
			{Title: "Name", Width: 20},
			{Title: "CIDR", Width: 18},
			{Title: "AZ", Width: 14},
			{Title: "Available IPs", Width: 14},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.SubnetInfo, error) {
			return client.VPC.ListSubnets(ctx, vpcID)
		},
		RowMapper: func(s awsvpc.SubnetInfo) table.Row {
			return table.Row{s.SubnetID, s.Name, s.CIDR, s.AZ, fmt.Sprintf("%d", s.AvailableIPs)}
		},
		CopyIDFunc: func(s awsvpc.SubnetInfo) string { return s.SubnetID },
	})
}

// --- Security Groups View ---

func NewSecurityGroupsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.SecurityGroupInfo] {
	return NewTableView(TableViewConfig[awsvpc.SecurityGroupInfo]{
		Title:       "Security Groups",
		LoadingText: "Loading security groups...",
		Columns: []table.Column{
			{Title: "Group ID", Width: 24},
			{Title: "Name", Width: 22},
			{Title: "Description", Width: 30},
			{Title: "Inbound", Width: 8},
			{Title: "Outbound", Width: 9},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.SecurityGroupInfo, error) {
			return client.VPC.ListSecurityGroups(ctx, vpcID)
		},
		RowMapper: func(sg awsvpc.SecurityGroupInfo) table.Row {
			return table.Row{sg.GroupID, sg.Name, sg.Description, fmt.Sprintf("%d", sg.InboundRules), fmt.Sprintf("%d", sg.OutboundRules)}
		},
		CopyIDFunc: func(sg awsvpc.SecurityGroupInfo) string { return sg.GroupID },
		OnEnter: func(sg awsvpc.SecurityGroupInfo) tea.Cmd {
			return pushView(NewSecurityGroupDetailView(client, sg))
		},
	})
}

// --- Internet Gateways View ---

func NewIGWView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.InternetGatewayInfo] {
	return NewTableView(TableViewConfig[awsvpc.InternetGatewayInfo]{
		Title:       "Internet Gateways",
		LoadingText: "Loading internet gateways...",
		Columns: []table.Column{
			{Title: "Gateway ID", Width: 26},
			{Title: "Name", Width: 25},
			{Title: "State", Width: 12},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.InternetGatewayInfo, error) {
			return client.VPC.ListInternetGateways(ctx, vpcID)
		},
		RowMapper: func(igw awsvpc.InternetGatewayInfo) table.Row {
			return table.Row{igw.GatewayID, igw.Name, igw.State}
		},
		CopyIDFunc: func(igw awsvpc.InternetGatewayInfo) string { return igw.GatewayID },
	})
}

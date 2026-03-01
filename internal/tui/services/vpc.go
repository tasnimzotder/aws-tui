package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

func NewVPCListView(client *awsclient.ServiceClient) *TableView[awsvpc.VPCInfo] {
	vpcHelp := HelpContextVPC
	var nextToken *string
	return NewTableView(TableViewConfig[awsvpc.VPCInfo]{
		Title:       "VPC",
		LoadingText: "Loading VPCs...",
		HelpCtx:     &vpcHelp,
		Columns: []table.Column{
			{Title: "VPC ID", Width: 24},
			{Title: "Name", Width: 20},
			{Title: "CIDR", Width: 18},
			{Title: "Default", Width: 8},
			{Title: "State", Width: 12},
		},
		FetchFuncPaged: func(ctx context.Context) ([]awsvpc.VPCInfo, bool, error) {
			nextToken = nil
			vpcs, nt, err := client.VPC.ListVPCsPage(ctx, nil)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return vpcs, nt != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awsvpc.VPCInfo, bool, error) {
			vpcs, nt, err := client.VPC.ListVPCsPage(ctx, nextToken)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return vpcs, nt != nil, nil
		},
		RowMapper: func(vpc awsvpc.VPCInfo) table.Row {
			def := "No"
			if vpc.IsDefault {
				def = "Yes"
			}
			return table.Row{vpc.VPCID, vpc.Name, vpc.CIDR, def, theme.RenderStatus(vpc.State)}
		},
		CopyIDFunc: func(vpc awsvpc.VPCInfo) string { return vpc.VPCID },
		OnEnter: func(vpc awsvpc.VPCInfo) tea.Cmd {
			return pushView(NewVPCDetailView(client, vpc))
		},
	})
}

// --- Subnets View ---

func NewSubnetsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.SubnetInfo] {
	var nextToken *string
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
		FetchFuncPaged: func(ctx context.Context) ([]awsvpc.SubnetInfo, bool, error) {
			nextToken = nil
			subnets, nt, err := client.VPC.ListSubnetsPage(ctx, vpcID, nil)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return subnets, nt != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awsvpc.SubnetInfo, bool, error) {
			subnets, nt, err := client.VPC.ListSubnetsPage(ctx, vpcID, nextToken)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return subnets, nt != nil, nil
		},
		RowMapper: func(s awsvpc.SubnetInfo) table.Row {
			return table.Row{s.SubnetID, s.Name, s.CIDR, s.AZ, fmt.Sprintf("%d", s.AvailableIPs)}
		},
		CopyIDFunc: func(s awsvpc.SubnetInfo) string { return s.SubnetID },
	})
}

// --- Security Groups View ---

func NewSecurityGroupsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.SecurityGroupInfo] {
	var nextToken *string
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
		FetchFuncPaged: func(ctx context.Context) ([]awsvpc.SecurityGroupInfo, bool, error) {
			nextToken = nil
			sgs, nt, err := client.VPC.ListSecurityGroupsPage(ctx, vpcID, nil)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return sgs, nt != nil, nil
		},
		LoadMoreFunc: func(ctx context.Context) ([]awsvpc.SecurityGroupInfo, bool, error) {
			sgs, nt, err := client.VPC.ListSecurityGroupsPage(ctx, vpcID, nextToken)
			if err != nil {
				return nil, false, err
			}
			nextToken = nt
			return sgs, nt != nil, nil
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
			return table.Row{igw.GatewayID, igw.Name, theme.RenderStatus(igw.State)}
		},
		CopyIDFunc: func(igw awsvpc.InternetGatewayInfo) string { return igw.GatewayID },
	})
}

// --- VPC Endpoints View ---

func NewVPCEndpointsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.VPCEndpointInfo] {
	return NewTableView(TableViewConfig[awsvpc.VPCEndpointInfo]{
		Title:       "VPC Endpoints",
		LoadingText: "Loading VPC endpoints...",
		Columns: []table.Column{
			{Title: "Endpoint ID", Width: 26},
			{Title: "Service", Width: 36},
			{Title: "Type", Width: 12},
			{Title: "State", Width: 12},
			{Title: "Subnets", Width: 8},
			{Title: "Route Tables", Width: 13},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.VPCEndpointInfo, error) {
			return client.VPC.ListVPCEndpoints(ctx, vpcID)
		},
		RowMapper: func(ep awsvpc.VPCEndpointInfo) table.Row {
			return table.Row{ep.EndpointID, ep.ServiceName, ep.Type, theme.RenderStatus(ep.State),
				fmt.Sprintf("%d", len(ep.SubnetIDs)),
				fmt.Sprintf("%d", len(ep.RouteTableIDs))}
		},
		CopyIDFunc: func(ep awsvpc.VPCEndpointInfo) string { return ep.EndpointID },
	})
}

// --- VPC Peering View ---

func NewVPCPeeringView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.VPCPeeringInfo] {
	return NewTableView(TableViewConfig[awsvpc.VPCPeeringInfo]{
		Title:       "VPC Peering",
		LoadingText: "Loading VPC peering connections...",
		Columns: []table.Column{
			{Title: "Peering ID", Width: 24},
			{Title: "Name", Width: 16},
			{Title: "Status", Width: 12},
			{Title: "Requester VPC", Width: 24},
			{Title: "Requester CIDR", Width: 16},
			{Title: "Accepter VPC", Width: 24},
			{Title: "Accepter CIDR", Width: 16},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.VPCPeeringInfo, error) {
			return client.VPC.ListVPCPeering(ctx, vpcID)
		},
		RowMapper: func(p awsvpc.VPCPeeringInfo) table.Row {
			return table.Row{p.PeeringID, p.Name, theme.RenderStatus(p.Status),
				p.RequesterVPC, p.RequesterCIDR,
				p.AccepterVPC, p.AccepterCIDR}
		},
		CopyIDFunc: func(p awsvpc.VPCPeeringInfo) string { return p.PeeringID },
	})
}

// --- Network ACLs View ---

func NewNACLsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.NetworkACLInfo] {
	return NewTableView(TableViewConfig[awsvpc.NetworkACLInfo]{
		Title:       "Network ACLs",
		LoadingText: "Loading network ACLs...",
		Columns: []table.Column{
			{Title: "NACL ID", Width: 24},
			{Title: "Name", Width: 20},
			{Title: "Default", Width: 8},
			{Title: "Inbound", Width: 8},
			{Title: "Outbound", Width: 9},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.NetworkACLInfo, error) {
			return client.VPC.ListNetworkACLs(ctx, vpcID)
		},
		RowMapper: func(n awsvpc.NetworkACLInfo) table.Row {
			def := "No"
			if n.IsDefault {
				def = "Yes"
			}
			return table.Row{n.NACLID, n.Name, def,
				fmt.Sprintf("%d", n.Inbound), fmt.Sprintf("%d", n.Outbound)}
		},
		CopyIDFunc: func(n awsvpc.NetworkACLInfo) string { return n.NACLID },
		OnEnter: func(n awsvpc.NetworkACLInfo) tea.Cmd {
			title := n.NACLID
			if n.Name != "" {
				title = n.NACLID + " - " + n.Name
			}
			return pushView(NewNACLEntriesView(client, n.NACLID, title))
		},
	})
}

// --- Flow Logs View ---

func NewFlowLogsView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.FlowLogInfo] {
	return NewTableView(TableViewConfig[awsvpc.FlowLogInfo]{
		Title:       "Flow Logs",
		LoadingText: "Loading flow logs...",
		Columns: []table.Column{
			{Title: "Flow Log ID", Width: 24},
			{Title: "Status", Width: 10},
			{Title: "Traffic Type", Width: 12},
			{Title: "Destination", Width: 50},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.FlowLogInfo, error) {
			return client.VPC.ListFlowLogs(ctx, vpcID)
		},
		RowMapper: func(fl awsvpc.FlowLogInfo) table.Row {
			return table.Row{fl.FlowLogID, theme.RenderStatus(fl.Status), fl.TrafficType, fl.LogDestination}
		},
		CopyIDFunc: func(fl awsvpc.FlowLogInfo) string { return fl.FlowLogID },
	})
}

// --- VPC Tags View ---

// vpcTagsMsg carries fetched tags.
type vpcTagsMsg struct {
	tags map[string]string
}

// vpcTagsErrMsg signals an error fetching tags.
type vpcTagsErrMsg struct {
	err error
}

// VPCTagsView shows VPC tags in a scrollable viewport.
type VPCTagsView struct {
	client   *awsclient.ServiceClient
	vpcID    string
	tags     map[string]string
	loaded   bool
	err      error
	viewport viewport.Model
	vpReady  bool
	spinner  spinner.Model
	width    int
	height   int
}

func NewVPCTagsView(client *awsclient.ServiceClient, vpcID string) *VPCTagsView {
	return &VPCTagsView{
		client:  client,
		vpcID:   vpcID,
		spinner: theme.NewSpinner(),
	}
}

func (v *VPCTagsView) Title() string { return "VPC Tags" }

func (v *VPCTagsView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchTags())
}

func (v *VPCTagsView) fetchTags() tea.Cmd {
	client := v.client
	vpcID := v.vpcID
	return func() tea.Msg {
		tags, err := client.VPC.GetVPCTags(context.Background(), vpcID)
		if err != nil {
			return vpcTagsErrMsg{err: err}
		}
		return vpcTagsMsg{tags: tags}
	}
}

func (v *VPCTagsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case vpcTagsMsg:
		v.tags = msg.tags
		v.loaded = true
		v.initViewport()
		return v, nil

	case vpcTagsErrMsg:
		v.err = msg.err
		v.loaded = true
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if v.vpReady {
			v.viewport.SetWidth(v.width)
			h := v.height - 2
			if h < 1 {
				h = 1
			}
			v.viewport.SetHeight(h)
		}
		return v, nil

	case spinner.TickMsg:
		if !v.loaded {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
	}

	if v.vpReady {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *VPCTagsView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *VPCTagsView) renderContent() string {
	if v.err != nil {
		return fmt.Sprintf("Error loading tags: %s", v.err.Error())
	}
	if len(v.tags) == 0 {
		return "No tags"
	}

	keys := make([]string, 0, len(v.tags))
	for k := range v.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s = %s\n", k, v.tags[k])
	}
	return b.String()
}

func (v *VPCTagsView) View() string {
	if !v.loaded {
		return v.spinner.View() + " Loading tags..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.vpReady {
		return ""
	}
	return v.viewport.View()
}

func (v *VPCTagsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.vpReady {
		v.viewport.SetWidth(width)
		h := height - 2
		if h < 1 {
			h = 1
		}
		v.viewport.SetHeight(h)
	}
}

package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// VPC Detail View (dashboard + tabbed navigation)
// ---------------------------------------------------------------------------

// vpcCountsMsg carries resource counts fetched in the background.
type vpcCountsMsg struct {
	subnets   int
	sgs       int
	rts       int
	igws      int
	nats      int
	endpoints int
	peerings  int
	nacls     int
}

// VPCDetailView shows a dashboard with VPC info and resource counts, plus a
// horizontal tab bar with switchable sub-resource tables.
type VPCDetailView struct {
	client *awsclient.ServiceClient
	vpc    awsvpc.VPCInfo

	// Dashboard counts
	subnetCount   int
	sgCount       int
	rtCount       int
	igwCount      int
	natCount      int
	endpointCount int
	peeringCount  int
	naclCount     int
	countsLoaded  bool

	tabs *TabController

	loading bool
	spinner spinner.Model
	err     error
	width   int
	height  int
}

// NewVPCDetailView creates a new VPC detail view with dashboard and tabs.
func NewVPCDetailView(client *awsclient.ServiceClient, vpc awsvpc.VPCInfo) *VPCDetailView {
	v := &VPCDetailView{
		client:  client,
		vpc:     vpc,
		loading: true,
		spinner: theme.NewSpinner(),
	}
	v.tabs = NewTabController(
		[]string{
			"Subnets", "Security Groups", "Route Tables", "Internet Gateways", "NAT Gateways",
			"Endpoints", "Peering", "NACLs", "Flow Logs", "Tags",
		},
		v.createTab,
	)
	return v
}

func (v *VPCDetailView) createTab(idx int) View {
	switch idx {
	case 0:
		return NewSubnetsView(v.client, v.vpc.VPCID)
	case 1:
		return NewSecurityGroupsView(v.client, v.vpc.VPCID)
	case 2:
		return NewRouteTablesView(v.client, v.vpc.VPCID)
	case 3:
		return NewIGWView(v.client, v.vpc.VPCID)
	case 4:
		return NewNATGatewaysView(v.client, v.vpc.VPCID)
	case 5:
		return NewVPCEndpointsView(v.client, v.vpc.VPCID)
	case 6:
		return NewVPCPeeringView(v.client, v.vpc.VPCID)
	case 7:
		return NewNACLsView(v.client, v.vpc.VPCID)
	case 8:
		return NewFlowLogsView(v.client, v.vpc.VPCID)
	case 9:
		return NewVPCTagsView(v.client, v.vpc.VPCID)
	}
	return nil
}

func (v *VPCDetailView) Title() string {
	if v.vpc.Name != "" {
		return v.vpc.Name
	}
	return v.vpc.VPCID
}

func (v *VPCDetailView) HelpContext() *HelpContext {
	ctx := HelpContextVPCDetail
	return &ctx
}

func (v *VPCDetailView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchCounts())
}

func (v *VPCDetailView) fetchCounts() tea.Cmd {
	client := v.client
	vpcID := v.vpc.VPCID
	return func() tea.Msg {
		ctx := context.Background()
		var wg sync.WaitGroup
		var subnets, sgs, rts, igws, nats, endpoints, peerings, nacls int
		var mu sync.Mutex

		fetch := func(fn func()) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				fn()
			}()
		}

		fetch(func() {
			items, err := client.VPC.ListSubnets(ctx, vpcID)
			if err == nil {
				mu.Lock()
				subnets = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListSecurityGroups(ctx, vpcID)
			if err == nil {
				mu.Lock()
				sgs = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListRouteTables(ctx, vpcID)
			if err == nil {
				mu.Lock()
				rts = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListInternetGateways(ctx, vpcID)
			if err == nil {
				mu.Lock()
				igws = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListNATGateways(ctx, vpcID)
			if err == nil {
				mu.Lock()
				nats = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListVPCEndpoints(ctx, vpcID)
			if err == nil {
				mu.Lock()
				endpoints = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListVPCPeering(ctx, vpcID)
			if err == nil {
				mu.Lock()
				peerings = len(items)
				mu.Unlock()
			}
		})
		fetch(func() {
			items, err := client.VPC.ListNetworkACLs(ctx, vpcID)
			if err == nil {
				mu.Lock()
				nacls = len(items)
				mu.Unlock()
			}
		})

		wg.Wait()
		return vpcCountsMsg{subnets: subnets, sgs: sgs, rts: rts, igws: igws, nats: nats,
			endpoints: endpoints, peerings: peerings, nacls: nacls}
	}
}

func (v *VPCDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case vpcCountsMsg:
		v.subnetCount = msg.subnets
		v.sgCount = msg.sgs
		v.rtCount = msg.rts
		v.igwCount = msg.igws
		v.natCount = msg.nats
		v.endpointCount = msg.endpoints
		v.peeringCount = msg.peerings
		v.naclCount = msg.nacls
		v.countsLoaded = true
		v.loading = false
		// Initialize first tab
		cmd := v.tabs.SwitchTab(0)
		v.tabs.ResizeActive(v.width, v.contentHeight())
		return v, cmd

	case tea.KeyPressMsg:
		key := msg.String()
		if handled, cmd := v.tabs.HandleKey(key); handled {
			v.tabs.ResizeActive(v.width, v.contentHeight())
			return v, cmd
		}
		return v, v.tabs.DelegateUpdate(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.tabs.ResizeActive(v.width, v.contentHeight())
		return v, nil

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
		return v, v.tabs.DelegateUpdate(msg)

	default:
		return v, v.tabs.DelegateUpdate(msg)
	}
}

// contentHeight calculates the height available for the tab content area.
// Dashboard box takes ~5 lines, tab bar takes ~2 lines.
func (v *VPCDetailView) contentHeight() int {
	h := v.height - 7 // dashboard (~5) + tab bar (2)
	if h < 3 {
		h = 3
	}
	return h
}

func (v *VPCDetailView) View() string {
	var sections []string

	// 1. Dashboard box
	sections = append(sections, v.renderDashboard())

	// 2. Tab bar
	sections = append(sections, v.tabs.RenderTabBar())

	// 3. Tab content
	if av := v.tabs.ActiveView(); av != nil {
		sections = append(sections, av.View())
	} else if v.loading {
		sections = append(sections, v.spinner.View()+" Loading VPC resources...")
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *VPCDetailView) renderDashboard() string {
	vpc := v.vpc
	label := theme.MutedStyle

	// Title: VPC ID + Name + status badge
	title := theme.DashboardTitleStyle.Render(vpc.VPCID)
	if vpc.Name != "" {
		title += "  " + vpc.Name
	}
	title += "  " + theme.RenderStatus(vpc.State)

	def := "No"
	if vpc.IsDefault {
		def = "Yes"
	}
	line1 := label.Render("CIDR: ") + vpc.CIDR + label.Render("  Default: ") + def

	var line2 string
	if v.countsLoaded {
		line2 = label.Render("Subnets: ") + fmt.Sprintf("%d", v.subnetCount) +
			label.Render("   SGs: ") + fmt.Sprintf("%d", v.sgCount) +
			label.Render("   Route Tables: ") + fmt.Sprintf("%d", v.rtCount) +
			label.Render("   IGWs: ") + fmt.Sprintf("%d", v.igwCount)
		line2 += "\n" + label.Render("NAT Gateways: ") + fmt.Sprintf("%d", v.natCount) +
			label.Render("   Endpoints: ") + fmt.Sprintf("%d", v.endpointCount) +
			label.Render("   Peering: ") + fmt.Sprintf("%d", v.peeringCount) +
			label.Render("   NACLs: ") + fmt.Sprintf("%d", v.naclCount)
	} else {
		line2 = theme.MutedStyle.Render("Loading resource counts...")
	}

	boxStyle := theme.DashboardBoxStyle
	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4)
	}

	return boxStyle.Render(title + "\n" + line1 + "\n" + line2)
}

// SetSize implements ResizableView.
func (v *VPCDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.tabs.ResizeActive(v.width, v.contentHeight())
}

// ---------------------------------------------------------------------------
// Route Tables View
// ---------------------------------------------------------------------------

func NewRouteTablesView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.RouteTableInfo] {
	return NewTableView(TableViewConfig[awsvpc.RouteTableInfo]{
		Title:       "Route Tables",
		LoadingText: "Loading route tables...",
		Columns: []table.Column{
			{Title: "Route Table ID", Width: 24},
			{Title: "Name", Width: 20},
			{Title: "Main", Width: 6},
			{Title: "Routes", Width: 8},
			{Title: "Associations", Width: 14},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.RouteTableInfo, error) {
			return client.VPC.ListRouteTables(ctx, vpcID)
		},
		RowMapper: func(rt awsvpc.RouteTableInfo) table.Row {
			main := "No"
			if rt.IsMain {
				main = "Yes"
			}
			return table.Row{rt.RouteTableID, rt.Name, main,
				fmt.Sprintf("%d", len(rt.Routes)),
				fmt.Sprintf("%d", len(rt.Associations))}
		},
		CopyIDFunc: func(rt awsvpc.RouteTableInfo) string { return rt.RouteTableID },
		OnEnter: func(rt awsvpc.RouteTableInfo) tea.Cmd {
			return pushView(NewRouteTableDetailView(rt))
		},
	})
}

// ---------------------------------------------------------------------------
// Route Table Detail View
// ---------------------------------------------------------------------------

// RouteTableDetailView shows routes and associated subnets for a route table.
type RouteTableDetailView struct {
	rt            awsvpc.RouteTableInfo
	width, height int
	viewport      viewport.Model
	vpReady       bool
}

// NewRouteTableDetailView creates a detail view for the given route table.
func NewRouteTableDetailView(rt awsvpc.RouteTableInfo) *RouteTableDetailView {
	return &RouteTableDetailView{rt: rt}
}

func (v *RouteTableDetailView) Title() string { return "Route Table: " + v.rt.RouteTableID }

func (v *RouteTableDetailView) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *RouteTableDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *RouteTableDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	rt := v.rt

	// Header
	name := rt.RouteTableID
	if rt.Name != "" {
		name = fmt.Sprintf("%s (%s)", rt.RouteTableID, rt.Name)
	}
	main := "No"
	if rt.IsMain {
		main = "Yes"
	}
	b.WriteString(bold.Render(fmt.Sprintf("Route Table: %s", name)))
	b.WriteString(fmt.Sprintf("   Main: %s\n", main))
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")

	// Associated Subnets
	if len(rt.Associations) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Associated Subnets:"))
		b.WriteString("\n")
		for _, assoc := range rt.Associations {
			if assoc.SubnetID != "" {
				b.WriteString(fmt.Sprintf("  %s\n", assoc.SubnetID))
			}
		}
	}

	// Routes
	if len(rt.Routes) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Routes:"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %-20s %-20s %-10s %s\n", "DESTINATION", "TARGET", "STATUS", "ORIGIN"))
		for _, route := range rt.Routes {
			b.WriteString(fmt.Sprintf("  %-20s %-20s %-10s %s\n",
				route.Destination, route.Target, route.Status, route.Origin))
		}
	}

	return b.String()
}

func (v *RouteTableDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
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
	}

	if v.vpReady {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *RouteTableDetailView) View() string {
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *RouteTableDetailView) SetSize(width, height int) {
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

// ---------------------------------------------------------------------------
// NAT Gateways View
// ---------------------------------------------------------------------------

func NewNATGatewaysView(client *awsclient.ServiceClient, vpcID string) *TableView[awsvpc.NATGatewayInfo] {
	return NewTableView(TableViewConfig[awsvpc.NATGatewayInfo]{
		Title:       "NAT Gateways",
		LoadingText: "Loading NAT gateways...",
		Columns: []table.Column{
			{Title: "Gateway ID", Width: 24},
			{Title: "Name", Width: 20},
			{Title: "State", Width: 12},
			{Title: "Type", Width: 10},
			{Title: "Subnet", Width: 26},
			{Title: "Elastic IP", Width: 16},
		},
		FetchFunc: func(ctx context.Context) ([]awsvpc.NATGatewayInfo, error) {
			return client.VPC.ListNATGateways(ctx, vpcID)
		},
		RowMapper: func(ng awsvpc.NATGatewayInfo) table.Row {
			return table.Row{ng.GatewayID, ng.Name, theme.RenderStatus(ng.State), ng.Type, ng.SubnetID, ng.ElasticIP}
		},
		CopyIDFunc: func(ng awsvpc.NATGatewayInfo) string { return ng.GatewayID },
	})
}

// ---------------------------------------------------------------------------
// Security Group Detail View
// ---------------------------------------------------------------------------

// sgRulesMsg carries fetched security group rules.
type sgRulesMsg struct {
	rules []awsvpc.SecurityGroupRule
}

// sgRulesErrMsg signals an error fetching SG rules.
type sgRulesErrMsg struct {
	err error
}

// SecurityGroupDetailView shows inbound and outbound rules for a security group.
type SecurityGroupDetailView struct {
	client        *awsclient.ServiceClient
	sg            awsvpc.SecurityGroupInfo
	rules         []awsvpc.SecurityGroupRule
	loaded        bool
	err           error
	viewport      viewport.Model
	vpReady       bool
	spinner       spinner.Model
	width, height int
}

// NewSecurityGroupDetailView creates a detail view for the given security group.
func NewSecurityGroupDetailView(client *awsclient.ServiceClient, sg awsvpc.SecurityGroupInfo) *SecurityGroupDetailView {
	return &SecurityGroupDetailView{
		client:  client,
		sg:      sg,
		spinner: theme.NewSpinner(),
	}
}

func (v *SecurityGroupDetailView) Title() string { return "Security Group: " + v.sg.GroupID }

func (v *SecurityGroupDetailView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchRules())
}

func (v *SecurityGroupDetailView) fetchRules() tea.Cmd {
	client := v.client
	groupID := v.sg.GroupID
	return func() tea.Msg {
		ctx := context.Background()
		rules, err := client.VPC.ListSecurityGroupRules(ctx, groupID)
		if err != nil {
			return sgRulesErrMsg{err: err}
		}
		return sgRulesMsg{rules: rules}
	}
}

func (v *SecurityGroupDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case sgRulesMsg:
		v.rules = msg.rules
		v.loaded = true
		v.initViewport()
		return v, nil

	case sgRulesErrMsg:
		v.err = msg.err
		v.loaded = true
		v.initViewport()
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

func (v *SecurityGroupDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *SecurityGroupDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	sg := v.sg

	// Header
	name := sg.GroupID
	if sg.Name != "" {
		name = fmt.Sprintf("%s (%s)", sg.GroupID, sg.Name)
	}
	b.WriteString(bold.Render(fmt.Sprintf("Security Group: %s", name)))
	b.WriteString("\n")
	if sg.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", sg.Description))
	}
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")

	if v.err != nil {
		b.WriteString(fmt.Sprintf("\nError loading rules: %s\n", v.err.Error()))
		return b.String()
	}

	// Separate inbound and outbound
	var inbound, outbound []awsvpc.SecurityGroupRule
	for _, r := range v.rules {
		if r.Direction == "inbound" {
			inbound = append(inbound, r)
		} else {
			outbound = append(outbound, r)
		}
	}

	// Inbound Rules
	b.WriteString("\n")
	b.WriteString(bold.Render("Inbound Rules:"))
	b.WriteString("\n")
	if len(inbound) == 0 {
		b.WriteString("  (none)\n")
	} else {
		b.WriteString(fmt.Sprintf("  %-10s %-10s %-22s %s\n", "PROTOCOL", "PORT", "SOURCE", "DESCRIPTION"))
		for _, r := range inbound {
			b.WriteString(fmt.Sprintf("  %-10s %-10s %-22s %s\n", r.Protocol, r.PortRange, r.Source, r.Description))
		}
	}

	// Outbound Rules
	b.WriteString("\n")
	b.WriteString(bold.Render("Outbound Rules:"))
	b.WriteString("\n")
	if len(outbound) == 0 {
		b.WriteString("  (none)\n")
	} else {
		b.WriteString(fmt.Sprintf("  %-10s %-10s %-22s %s\n", "PROTOCOL", "PORT", "DESTINATION", "DESCRIPTION"))
		for _, r := range outbound {
			b.WriteString(fmt.Sprintf("  %-10s %-10s %-22s %s\n", r.Protocol, r.PortRange, r.Source, r.Description))
		}
	}

	return b.String()
}

func (v *SecurityGroupDetailView) View() string {
	if !v.loaded {
		return v.spinner.View() + " Loading security group rules..."
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *SecurityGroupDetailView) SetSize(width, height int) {
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

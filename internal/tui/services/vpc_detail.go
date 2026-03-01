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

	// Tabs
	activeTab int
	tabNames  []string
	tabViews  []View // lazily initialized

	loading bool
	spinner spinner.Model
	err     error
	width   int
	height  int
}

// NewVPCDetailView creates a new VPC detail view with dashboard and tabs.
func NewVPCDetailView(client *awsclient.ServiceClient, vpc awsvpc.VPCInfo) *VPCDetailView {
	return &VPCDetailView{
		client: client,
		vpc:    vpc,
		tabNames: []string{
			"Subnets", "Security Groups", "Route Tables", "Internet Gateways", "NAT Gateways",
			"Endpoints", "Peering", "NACLs", "Flow Logs", "Tags",
		},
		tabViews: make([]View, 10),
		loading:  true,
		spinner:  theme.NewSpinner(),
	}
}

func (v *VPCDetailView) Title() string {
	if v.vpc.Name != "" {
		return v.vpc.Name
	}
	return v.vpc.VPCID
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
		v.initTab(0)
		if v.tabViews[0] != nil {
			cmd := v.tabViews[0].Init()
			return v, cmd
		}
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			return v, v.switchTab((v.activeTab + 1) % len(v.tabNames))
		case "shift+tab":
			next := v.activeTab - 1
			if next < 0 {
				next = len(v.tabNames) - 1
			}
			return v, v.switchTab(next)
		case "1":
			return v, v.switchTab(0)
		case "2":
			return v, v.switchTab(1)
		case "3":
			return v, v.switchTab(2)
		case "4":
			return v, v.switchTab(3)
		case "5":
			return v, v.switchTab(4)
		case "6":
			return v, v.switchTab(5)
		case "7":
			return v, v.switchTab(6)
		case "8":
			return v, v.switchTab(7)
		case "9":
			return v, v.switchTab(8)
		case "0":
			return v, v.switchTab(9)
		default:
			// Delegate to active tab view
			if v.tabViews[v.activeTab] != nil {
				updated, cmd := v.tabViews[v.activeTab].Update(msg)
				v.tabViews[v.activeTab] = updated
				return v, cmd
			}
		}

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.resizeActiveTab()
		return v, nil

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
		// Delegate spinner ticks to active tab (it may have its own spinner)
		if v.tabViews[v.activeTab] != nil {
			updated, cmd := v.tabViews[v.activeTab].Update(msg)
			v.tabViews[v.activeTab] = updated
			return v, cmd
		}
		return v, nil

	default:
		// Delegate all other messages to active tab view
		if v.tabViews[v.activeTab] != nil {
			updated, cmd := v.tabViews[v.activeTab].Update(msg)
			v.tabViews[v.activeTab] = updated
			return v, cmd
		}
	}

	return v, nil
}

func (v *VPCDetailView) switchTab(idx int) tea.Cmd {
	v.activeTab = idx
	v.initTab(idx)
	v.resizeActiveTab()
	if v.tabViews[idx] != nil {
		return v.tabViews[idx].Init()
	}
	return nil
}

func (v *VPCDetailView) initTab(idx int) {
	if v.tabViews[idx] != nil {
		return
	}
	switch idx {
	case 0:
		v.tabViews[idx] = NewSubnetsView(v.client, v.vpc.VPCID)
	case 1:
		v.tabViews[idx] = NewSecurityGroupsView(v.client, v.vpc.VPCID)
	case 2:
		v.tabViews[idx] = NewRouteTablesView(v.client, v.vpc.VPCID)
	case 3:
		v.tabViews[idx] = NewIGWView(v.client, v.vpc.VPCID)
	case 4:
		v.tabViews[idx] = NewNATGatewaysView(v.client, v.vpc.VPCID)
	case 5:
		v.tabViews[idx] = NewVPCEndpointsView(v.client, v.vpc.VPCID)
	case 6:
		v.tabViews[idx] = NewVPCPeeringView(v.client, v.vpc.VPCID)
	case 7:
		v.tabViews[idx] = NewNACLsView(v.client, v.vpc.VPCID)
	case 8:
		v.tabViews[idx] = NewFlowLogsView(v.client, v.vpc.VPCID)
	case 9:
		v.tabViews[idx] = NewVPCTagsView(v.client, v.vpc.VPCID)
	}
}

func (v *VPCDetailView) resizeActiveTab() {
	if v.tabViews[v.activeTab] == nil {
		return
	}
	if rv, ok := v.tabViews[v.activeTab].(ResizableView); ok {
		contentHeight := v.contentHeight()
		rv.SetSize(v.width, contentHeight)
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
	sections = append(sections, v.renderTabBar())

	// 3. Tab content
	if v.tabViews[v.activeTab] != nil {
		sections = append(sections, v.tabViews[v.activeTab].View())
	} else if v.loading {
		sections = append(sections, v.spinner.View()+" Loading VPC resources...")
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *VPCDetailView) renderDashboard() string {
	vpc := v.vpc

	// Info lines
	def := "No"
	if vpc.IsDefault {
		def = "Yes"
	}
	line1 := fmt.Sprintf("CIDR: %s  State: %s  Default: %s", vpc.CIDR, vpc.State, def)

	var line2 string
	if v.countsLoaded {
		line2 = fmt.Sprintf("Subnets: %d   SGs: %d   Route Tables: %d   IGWs: %d",
			v.subnetCount, v.sgCount, v.rtCount, v.igwCount)
		line2 += fmt.Sprintf("\nNAT Gateways: %d   Endpoints: %d   Peering: %d   NACLs: %d",
			v.natCount, v.endpointCount, v.peeringCount, v.naclCount)
	} else {
		line2 = "Loading resource counts..."
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 1)

	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4)
	}

	titleStyle := lipgloss.NewStyle().Bold(true)
	title := vpc.VPCID
	if vpc.Name != "" {
		title = vpc.VPCID + " - " + vpc.Name
	}

	header := titleStyle.Render(title)
	content := line1 + "\n" + line2

	return boxStyle.Render(header + "\n" + content)
}

func (v *VPCDetailView) renderTabBar() string {
	var tabs []string
	for i, name := range v.tabNames {
		key := fmt.Sprintf("%d", i+1)
		if i == 9 {
			key = "0"
		}
		label := key + ":" + name
		if i == v.activeTab {
			tabs = append(tabs, theme.TabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, theme.TabInactiveStyle.Render(label))
		}
	}
	bar := strings.Join(tabs, "")
	return theme.TabBarStyle.Render(bar)
}

// SetSize implements ResizableView.
func (v *VPCDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.resizeActiveTab()
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
	w := v.width
	if w < 20 {
		w = 80
	}
	vp := viewport.New(
		viewport.WithWidth(w),
		viewport.WithHeight(h),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	vp.SetContent(v.renderContent())
	v.viewport = vp
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
			return table.Row{ng.GatewayID, ng.Name, ng.State, ng.Type, ng.SubnetID, ng.ElasticIP}
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
	w := v.width
	if w < 20 {
		w = 80
	}
	vp := viewport.New(
		viewport.WithWidth(w),
		viewport.WithHeight(h),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	vp.SetContent(v.renderContent())
	v.viewport = vp
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

// ---------------------------------------------------------------------------
// NACL Entries View (sub-view when drilling into a NACL)
// ---------------------------------------------------------------------------

// naclEntriesMsg carries fetched NACL entries.
type naclEntriesMsg struct {
	entries []awsvpc.NetworkACLEntry
}

// naclEntriesErrMsg signals an error fetching NACL entries.
type naclEntriesErrMsg struct {
	err error
}

// NACLEntriesView shows inbound and outbound entries for a Network ACL.
type NACLEntriesView struct {
	client        *awsclient.ServiceClient
	naclID        string
	title         string
	entries       []awsvpc.NetworkACLEntry
	loaded        bool
	err           error
	viewport      viewport.Model
	vpReady       bool
	spinner       spinner.Model
	width, height int
}

func NewNACLEntriesView(client *awsclient.ServiceClient, naclID, title string) *NACLEntriesView {
	return &NACLEntriesView{
		client:  client,
		naclID:  naclID,
		title:   title,
		spinner: theme.NewSpinner(),
	}
}

func (v *NACLEntriesView) Title() string { return "NACL: " + v.title }

func (v *NACLEntriesView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchEntries())
}

func (v *NACLEntriesView) fetchEntries() tea.Cmd {
	client := v.client
	naclID := v.naclID
	return func() tea.Msg {
		entries, err := client.VPC.ListNetworkACLEntries(context.Background(), naclID)
		if err != nil {
			return naclEntriesErrMsg{err: err}
		}
		return naclEntriesMsg{entries: entries}
	}
}

func (v *NACLEntriesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case naclEntriesMsg:
		v.entries = msg.entries
		v.loaded = true
		v.initViewport()
		return v, nil

	case naclEntriesErrMsg:
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

func (v *NACLEntriesView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	w := v.width
	if w < 20 {
		w = 80
	}
	vp := viewport.New(
		viewport.WithWidth(w),
		viewport.WithHeight(h),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	vp.SetContent(v.renderContent())
	v.viewport = vp
	v.vpReady = true
}

func (v *NACLEntriesView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	b.WriteString(bold.Render("Network ACL: " + v.title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 80))
	b.WriteString("\n")

	if v.err != nil {
		fmt.Fprintf(&b, "\nError loading entries: %s\n", v.err.Error())
		return b.String()
	}

	// Separate inbound and outbound
	var inbound, outbound []awsvpc.NetworkACLEntry
	for _, e := range v.entries {
		if e.Direction == "inbound" {
			inbound = append(inbound, e)
		} else {
			outbound = append(outbound, e)
		}
	}

	b.WriteString("\n")
	b.WriteString(bold.Render("Inbound Rules:"))
	b.WriteString("\n")
	if len(inbound) == 0 {
		b.WriteString("  (none)\n")
	} else {
		fmt.Fprintf(&b, "  %-8s %-10s %-12s %-20s %s\n", "RULE#", "PROTOCOL", "PORT", "CIDR", "ACTION")
		for _, e := range inbound {
			fmt.Fprintf(&b, "  %-8d %-10s %-12s %-20s %s\n", e.RuleNumber, e.Protocol, e.PortRange, e.CIDRBlock, e.Action)
		}
	}

	b.WriteString("\n")
	b.WriteString(bold.Render("Outbound Rules:"))
	b.WriteString("\n")
	if len(outbound) == 0 {
		b.WriteString("  (none)\n")
	} else {
		fmt.Fprintf(&b, "  %-8s %-10s %-12s %-20s %s\n", "RULE#", "PROTOCOL", "PORT", "CIDR", "ACTION")
		for _, e := range outbound {
			fmt.Fprintf(&b, "  %-8d %-10s %-12s %-20s %s\n", e.RuleNumber, e.Protocol, e.PortRange, e.CIDRBlock, e.Action)
		}
	}

	return b.String()
}

func (v *NACLEntriesView) View() string {
	if !v.loaded {
		return v.spinner.View() + " Loading NACL entries..."
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *NACLEntriesView) SetSize(width, height int) {
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

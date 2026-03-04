package vpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

// Tab indices for the detail view.
const (
	tabOverview = iota
	tabSubnets
	tabSecurityGroups
	tabRouteTables
	tabNATGateways
	tabEndpoints
	tabPeering
	tabNACLs
	tabFlowLogs
)

var tabTitles = []string{
	"Overview",
	"Subnets",
	"Security Groups",
	"Route Tables",
	"NAT Gateways",
	"Endpoints",
	"Peering",
	"NACLs",
	"Flow Logs",
}

// Detail messages for async data loading.
type (
	overviewMsg struct {
		tags map[string]string
		igws []awsvpc.InternetGatewayInfo
		err  error
	}
	subnetsMsg struct {
		items []awsvpc.SubnetInfo
		err   error
	}
	securityGroupsMsg struct {
		items []awsvpc.SecurityGroupInfo
		err   error
	}
	routeTablesMsg struct {
		items []awsvpc.RouteTableInfo
		err   error
	}
	natGatewaysMsg struct {
		items []awsvpc.NATGatewayInfo
		err   error
	}
	endpointsMsg struct {
		items []awsvpc.VPCEndpointInfo
		err   error
	}
	peeringMsg struct {
		items []awsvpc.VPCPeeringInfo
		err   error
	}
	naclsMsg struct {
		items []awsvpc.NetworkACLInfo
		err   error
	}
	flowLogsMsg struct {
		items []awsvpc.FlowLogInfo
		err   error
	}
)

// DetailView shows VPC details with tabbed sub-resource views.
type DetailView struct {
	client VPCClient
	router plugin.Router
	vpcID  string
	tabs   ui.TabController

	// Tab data
	tags map[string]string
	igws []awsvpc.InternetGatewayInfo

	subnets        ui.TableView[awsvpc.SubnetInfo]
	securityGroups ui.TableView[awsvpc.SecurityGroupInfo]
	routeTables    ui.TableView[awsvpc.RouteTableInfo]
	natGateways    ui.TableView[awsvpc.NATGatewayInfo]
	endpoints      ui.TableView[awsvpc.VPCEndpointInfo]
	peering        ui.TableView[awsvpc.VPCPeeringInfo]
	nacls          ui.TableView[awsvpc.NetworkACLInfo]
	flowLogs       ui.TableView[awsvpc.FlowLogInfo]

	// Tracks which tabs have been loaded.
	loaded  [9]bool
	loading [9]bool
	errors  [9]error
}

// NewDetailView creates a VPC DetailView for the given VPC ID.
func NewDetailView(client VPCClient, router plugin.Router, vpcID string) *DetailView {
	return &DetailView{
		client:         client,
		router:         router,
		vpcID:          vpcID,
		tabs:           ui.NewTabController(tabTitles),
		subnets:        newSubnetTable(nil),
		securityGroups: newSecurityGroupTable(nil),
		routeTables:    newRouteTableTable(nil),
		natGateways:    newNATGatewayTable(nil),
		endpoints:      newEndpointTable(nil),
		peering:        newPeeringTable(nil),
		nacls:          newNACLTable(nil),
		flowLogs:       newFlowLogTable(nil),
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.loadTab(tabOverview)
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case overviewMsg:
		dv.loading[tabOverview] = false
		dv.loaded[tabOverview] = true
		if msg.err != nil {
			dv.errors[tabOverview] = msg.err
			return dv, nil
		}
		dv.tags = msg.tags
		dv.igws = msg.igws
		return dv, nil

	case subnetsMsg:
		dv.loading[tabSubnets] = false
		dv.loaded[tabSubnets] = true
		if msg.err != nil {
			dv.errors[tabSubnets] = msg.err
			return dv, nil
		}
		dv.subnets.SetItems(msg.items)
		return dv, nil

	case securityGroupsMsg:
		dv.loading[tabSecurityGroups] = false
		dv.loaded[tabSecurityGroups] = true
		if msg.err != nil {
			dv.errors[tabSecurityGroups] = msg.err
			return dv, nil
		}
		dv.securityGroups.SetItems(msg.items)
		return dv, nil

	case routeTablesMsg:
		dv.loading[tabRouteTables] = false
		dv.loaded[tabRouteTables] = true
		if msg.err != nil {
			dv.errors[tabRouteTables] = msg.err
			return dv, nil
		}
		dv.routeTables.SetItems(msg.items)
		return dv, nil

	case natGatewaysMsg:
		dv.loading[tabNATGateways] = false
		dv.loaded[tabNATGateways] = true
		if msg.err != nil {
			dv.errors[tabNATGateways] = msg.err
			return dv, nil
		}
		dv.natGateways.SetItems(msg.items)
		return dv, nil

	case endpointsMsg:
		dv.loading[tabEndpoints] = false
		dv.loaded[tabEndpoints] = true
		if msg.err != nil {
			dv.errors[tabEndpoints] = msg.err
			return dv, nil
		}
		dv.endpoints.SetItems(msg.items)
		return dv, nil

	case peeringMsg:
		dv.loading[tabPeering] = false
		dv.loaded[tabPeering] = true
		if msg.err != nil {
			dv.errors[tabPeering] = msg.err
			return dv, nil
		}
		dv.peering.SetItems(msg.items)
		return dv, nil

	case naclsMsg:
		dv.loading[tabNACLs] = false
		dv.loaded[tabNACLs] = true
		if msg.err != nil {
			dv.errors[tabNACLs] = msg.err
			return dv, nil
		}
		dv.nacls.SetItems(msg.items)
		return dv, nil

	case flowLogsMsg:
		dv.loading[tabFlowLogs] = false
		dv.loaded[tabFlowLogs] = true
		if msg.err != nil {
			dv.errors[tabFlowLogs] = msg.err
			return dv, nil
		}
		dv.flowLogs.SetItems(msg.items)
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		case "r":
			active := dv.tabs.Active()
			dv.loaded[active] = false
			return dv, dv.loadTab(active)
		case "enter":
			view := dv.drillDown()
			if view != nil {
				dv.router.Push(view)
				return dv, view.Init()
			}
			return dv, nil
		}

		// Tab navigation
		prevTab := dv.tabs.Active()
		var tabCmd tea.Cmd
		dv.tabs, tabCmd = dv.tabs.Update(msg)
		if dv.tabs.Active() != prevTab {
			cmd := dv.loadTab(dv.tabs.Active())
			if cmd != nil {
				return dv, tea.Batch(tabCmd, cmd)
			}
			return dv, tabCmd
		}

		// Delegate to the active tab's table
		var cmd tea.Cmd
		switch dv.tabs.Active() {
		case tabSubnets:
			dv.subnets, cmd = dv.subnets.Update(msg)
		case tabSecurityGroups:
			dv.securityGroups, cmd = dv.securityGroups.Update(msg)
		case tabRouteTables:
			dv.routeTables, cmd = dv.routeTables.Update(msg)
		case tabNATGateways:
			dv.natGateways, cmd = dv.natGateways.Update(msg)
		case tabEndpoints:
			dv.endpoints, cmd = dv.endpoints.Update(msg)
		case tabPeering:
			dv.peering, cmd = dv.peering.Update(msg)
		case tabNACLs:
			dv.nacls, cmd = dv.nacls.Update(msg)
		case tabFlowLogs:
			dv.flowLogs, cmd = dv.flowLogs.Update(msg)
		}
		return dv, cmd
	}

	return dv, nil
}

func (dv *DetailView) loadTab(tab int) tea.Cmd {
	if dv.loaded[tab] {
		return nil
	}
	dv.loading[tab] = true
	client := dv.client
	vpcID := dv.vpcID

	switch tab {
	case tabOverview:
		return func() tea.Msg {
			ctx := context.TODO()
			tags, err := client.GetVPCTags(ctx, vpcID)
			if err != nil {
				return overviewMsg{err: err}
			}
			igws, err := client.ListInternetGateways(ctx, vpcID)
			if err != nil {
				return overviewMsg{err: err}
			}
			return overviewMsg{tags: tags, igws: igws}
		}
	case tabSubnets:
		return func() tea.Msg {
			items, err := client.ListSubnets(context.TODO(), vpcID)
			return subnetsMsg{items: items, err: err}
		}
	case tabSecurityGroups:
		return func() tea.Msg {
			items, err := client.ListSecurityGroups(context.TODO(), vpcID)
			return securityGroupsMsg{items: items, err: err}
		}
	case tabRouteTables:
		return func() tea.Msg {
			items, err := client.ListRouteTables(context.TODO(), vpcID)
			return routeTablesMsg{items: items, err: err}
		}
	case tabNATGateways:
		return func() tea.Msg {
			items, err := client.ListNATGateways(context.TODO(), vpcID)
			return natGatewaysMsg{items: items, err: err}
		}
	case tabEndpoints:
		return func() tea.Msg {
			items, err := client.ListVPCEndpoints(context.TODO(), vpcID)
			return endpointsMsg{items: items, err: err}
		}
	case tabPeering:
		return func() tea.Msg {
			items, err := client.ListVPCPeering(context.TODO(), vpcID)
			return peeringMsg{items: items, err: err}
		}
	case tabNACLs:
		return func() tea.Msg {
			items, err := client.ListNetworkACLs(context.TODO(), vpcID)
			return naclsMsg{items: items, err: err}
		}
	case tabFlowLogs:
		return func() tea.Msg {
			items, err := client.ListFlowLogs(context.TODO(), vpcID)
			return flowLogsMsg{items: items, err: err}
		}
	}
	return nil
}

func (dv *DetailView) View() tea.View {
	var b strings.Builder

	b.WriteString(dv.tabs.View())
	b.WriteString("\n\n")

	active := dv.tabs.Active()

	if dv.loading[active] {
		skel := ui.NewSkeleton(80, 6)
		b.WriteString(skel.View())
		return tea.NewView(b.String())
	}

	if dv.errors[active] != nil {
		b.WriteString(fmt.Sprintf("Error: %s", dv.errors[active].Error()))
		return tea.NewView(b.String())
	}

	switch active {
	case tabOverview:
		b.WriteString(dv.renderOverview())
	case tabSubnets:
		b.WriteString(dv.subnets.View())
	case tabSecurityGroups:
		b.WriteString(dv.securityGroups.View())
	case tabRouteTables:
		b.WriteString(dv.routeTables.View())
	case tabNATGateways:
		b.WriteString(dv.natGateways.View())
	case tabEndpoints:
		b.WriteString(dv.endpoints.View())
	case tabPeering:
		b.WriteString(dv.peering.View())
	case tabNACLs:
		b.WriteString(dv.nacls.View())
	case tabFlowLogs:
		b.WriteString(dv.flowLogs.View())
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderOverview() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("VPC ID:  %s\n", dv.vpcID))

	if dv.tags != nil {
		if name, ok := dv.tags["Name"]; ok {
			b.WriteString(fmt.Sprintf("Name:    %s\n", name))
		}
		b.WriteString("\nTags:\n")
		for k, v := range dv.tags {
			b.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
	}

	if len(dv.igws) > 0 {
		b.WriteString("\nInternet Gateways:\n")
		for _, igw := range dv.igws {
			name := igw.GatewayID
			if igw.Name != "" {
				name = fmt.Sprintf("%s (%s)", igw.Name, igw.GatewayID)
			}
			b.WriteString(fmt.Sprintf("  %s  [%s]\n", name, igw.State))
		}
	}

	return b.String()
}

func (dv *DetailView) drillDown() plugin.View {
	switch dv.tabs.Active() {
	case tabSubnets:
		item := dv.subnets.SelectedItem()
		if item.SubnetID != "" {
			return NewSubnetDetailView(dv.client, dv.router, item)
		}
	case tabSecurityGroups:
		item := dv.securityGroups.SelectedItem()
		if item.GroupID != "" {
			return NewSGDetailView(dv.client, dv.router, item)
		}
	case tabRouteTables:
		item := dv.routeTables.SelectedItem()
		if item.RouteTableID != "" {
			return NewRouteTableDetailView(dv.client, dv.router, item)
		}
	case tabNATGateways:
		item := dv.natGateways.SelectedItem()
		if item.GatewayID != "" {
			return NewNATGatewayDetailView(dv.client, dv.router, item)
		}
	case tabEndpoints:
		item := dv.endpoints.SelectedItem()
		if item.EndpointID != "" {
			return NewEndpointDetailView(dv.client, dv.router, item)
		}
	case tabPeering:
		item := dv.peering.SelectedItem()
		if item.PeeringID != "" {
			return NewPeeringDetailView(dv.client, dv.router, item)
		}
	case tabNACLs:
		item := dv.nacls.SelectedItem()
		if item.NACLID != "" {
			return NewNACLDetailView(dv.client, dv.router, item)
		}
	case tabFlowLogs:
		item := dv.flowLogs.SelectedItem()
		if item.FlowLogID != "" {
			return NewFlowLogDetailView(dv.client, dv.router, item)
		}
	}
	return nil
}

func (dv *DetailView) Title() string {
	return fmt.Sprintf("VPC: %s", dv.vpcID)
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	return []plugin.KeyHint{
		{Key: "enter", Desc: "view details"},
		{Key: "]/[", Desc: "switch tab"},
		{Key: "1-9", Desc: "jump to tab"},
		{Key: "r", Desc: "refresh tab"},
		{Key: "esc", Desc: "back"},
		{Key: "/", Desc: "filter"},
		{Key: "s", Desc: "sort"},
	}
}

// Table constructors for each sub-resource.

func newSubnetTable(items []awsvpc.SubnetInfo) ui.TableView[awsvpc.SubnetInfo] {
	cols := []ui.Column[awsvpc.SubnetInfo]{
		{Title: "Name", Width: 24, Field: func(s awsvpc.SubnetInfo) string { return s.Name }},
		{Title: "Subnet ID", Width: 26, Field: func(s awsvpc.SubnetInfo) string { return s.SubnetID }},
		{Title: "CIDR", Width: 20, Field: func(s awsvpc.SubnetInfo) string { return s.CIDR }},
		{Title: "AZ", Width: 16, Field: func(s awsvpc.SubnetInfo) string { return s.AZ }},
		{Title: "Available IPs", Width: 14, Field: func(s awsvpc.SubnetInfo) string { return strconv.Itoa(s.AvailableIPs) }},
	}
	return ui.NewTableView(cols, items, func(s awsvpc.SubnetInfo) string { return s.SubnetID })
}

func newSecurityGroupTable(items []awsvpc.SecurityGroupInfo) ui.TableView[awsvpc.SecurityGroupInfo] {
	cols := []ui.Column[awsvpc.SecurityGroupInfo]{
		{Title: "Name", Width: 24, Field: func(sg awsvpc.SecurityGroupInfo) string { return sg.Name }},
		{Title: "Group ID", Width: 22, Field: func(sg awsvpc.SecurityGroupInfo) string { return sg.GroupID }},
		{Title: "Description", Width: 30, Field: func(sg awsvpc.SecurityGroupInfo) string { return sg.Description }},
		{Title: "Inbound", Width: 8, Field: func(sg awsvpc.SecurityGroupInfo) string { return strconv.Itoa(sg.InboundRules) }},
		{Title: "Outbound", Width: 9, Field: func(sg awsvpc.SecurityGroupInfo) string { return strconv.Itoa(sg.OutboundRules) }},
	}
	return ui.NewTableView(cols, items, func(sg awsvpc.SecurityGroupInfo) string { return sg.GroupID })
}

func newRouteTableTable(items []awsvpc.RouteTableInfo) ui.TableView[awsvpc.RouteTableInfo] {
	cols := []ui.Column[awsvpc.RouteTableInfo]{
		{Title: "Name", Width: 24, Field: func(rt awsvpc.RouteTableInfo) string { return rt.Name }},
		{Title: "Route Table ID", Width: 26, Field: func(rt awsvpc.RouteTableInfo) string { return rt.RouteTableID }},
		{Title: "Main", Width: 6, Field: func(rt awsvpc.RouteTableInfo) string {
			if rt.IsMain {
				return "Yes"
			}
			return "No"
		}},
		{Title: "Routes", Width: 7, Field: func(rt awsvpc.RouteTableInfo) string { return strconv.Itoa(len(rt.Routes)) }},
		{Title: "Associations", Width: 13, Field: func(rt awsvpc.RouteTableInfo) string { return strconv.Itoa(len(rt.Associations)) }},
	}
	return ui.NewTableView(cols, items, func(rt awsvpc.RouteTableInfo) string { return rt.RouteTableID })
}

func newNATGatewayTable(items []awsvpc.NATGatewayInfo) ui.TableView[awsvpc.NATGatewayInfo] {
	cols := []ui.Column[awsvpc.NATGatewayInfo]{
		{Title: "Name", Width: 24, Field: func(ng awsvpc.NATGatewayInfo) string { return ng.Name }},
		{Title: "Gateway ID", Width: 24, Field: func(ng awsvpc.NATGatewayInfo) string { return ng.GatewayID }},
		{Title: "State", Width: 12, Field: func(ng awsvpc.NATGatewayInfo) string { return ng.State }},
		{Title: "Type", Width: 8, Field: func(ng awsvpc.NATGatewayInfo) string { return ng.Type }},
		{Title: "Subnet", Width: 26, Field: func(ng awsvpc.NATGatewayInfo) string { return ng.SubnetID }},
		{Title: "Elastic IP", Width: 16, Field: func(ng awsvpc.NATGatewayInfo) string { return ng.ElasticIP }},
	}
	return ui.NewTableView(cols, items, func(ng awsvpc.NATGatewayInfo) string { return ng.GatewayID })
}

func newEndpointTable(items []awsvpc.VPCEndpointInfo) ui.TableView[awsvpc.VPCEndpointInfo] {
	cols := []ui.Column[awsvpc.VPCEndpointInfo]{
		{Title: "Endpoint ID", Width: 26, Field: func(ep awsvpc.VPCEndpointInfo) string { return ep.EndpointID }},
		{Title: "Service", Width: 40, Field: func(ep awsvpc.VPCEndpointInfo) string { return ep.ServiceName }},
		{Title: "Type", Width: 12, Field: func(ep awsvpc.VPCEndpointInfo) string { return ep.Type }},
		{Title: "State", Width: 12, Field: func(ep awsvpc.VPCEndpointInfo) string { return ep.State }},
	}
	return ui.NewTableView(cols, items, func(ep awsvpc.VPCEndpointInfo) string { return ep.EndpointID })
}

func newPeeringTable(items []awsvpc.VPCPeeringInfo) ui.TableView[awsvpc.VPCPeeringInfo] {
	cols := []ui.Column[awsvpc.VPCPeeringInfo]{
		{Title: "Name", Width: 20, Field: func(p awsvpc.VPCPeeringInfo) string { return p.Name }},
		{Title: "Peering ID", Width: 26, Field: func(p awsvpc.VPCPeeringInfo) string { return p.PeeringID }},
		{Title: "Status", Width: 14, Field: func(p awsvpc.VPCPeeringInfo) string { return p.Status }},
		{Title: "Requester VPC", Width: 22, Field: func(p awsvpc.VPCPeeringInfo) string { return p.RequesterVPC }},
		{Title: "Accepter VPC", Width: 22, Field: func(p awsvpc.VPCPeeringInfo) string { return p.AccepterVPC }},
	}
	return ui.NewTableView(cols, items, func(p awsvpc.VPCPeeringInfo) string { return p.PeeringID })
}

func newNACLTable(items []awsvpc.NetworkACLInfo) ui.TableView[awsvpc.NetworkACLInfo] {
	cols := []ui.Column[awsvpc.NetworkACLInfo]{
		{Title: "Name", Width: 24, Field: func(n awsvpc.NetworkACLInfo) string { return n.Name }},
		{Title: "NACL ID", Width: 26, Field: func(n awsvpc.NetworkACLInfo) string { return n.NACLID }},
		{Title: "Default", Width: 8, Field: func(n awsvpc.NetworkACLInfo) string {
			if n.IsDefault {
				return "Yes"
			}
			return "No"
		}},
		{Title: "Inbound", Width: 8, Field: func(n awsvpc.NetworkACLInfo) string { return strconv.Itoa(n.Inbound) }},
		{Title: "Outbound", Width: 9, Field: func(n awsvpc.NetworkACLInfo) string { return strconv.Itoa(n.Outbound) }},
	}
	return ui.NewTableView(cols, items, func(n awsvpc.NetworkACLInfo) string { return n.NACLID })
}

func newFlowLogTable(items []awsvpc.FlowLogInfo) ui.TableView[awsvpc.FlowLogInfo] {
	cols := []ui.Column[awsvpc.FlowLogInfo]{
		{Title: "Flow Log ID", Width: 24, Field: func(fl awsvpc.FlowLogInfo) string { return fl.FlowLogID }},
		{Title: "Status", Width: 10, Field: func(fl awsvpc.FlowLogInfo) string { return fl.Status }},
		{Title: "Traffic", Width: 10, Field: func(fl awsvpc.FlowLogInfo) string { return fl.TrafficType }},
		{Title: "Destination", Width: 40, Field: func(fl awsvpc.FlowLogInfo) string { return fl.LogDestination }},
	}
	return ui.NewTableView(cols, items, func(fl awsvpc.FlowLogInfo) string { return fl.FlowLogID })
}

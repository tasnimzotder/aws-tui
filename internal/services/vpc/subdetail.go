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

// Sub-detail async messages.
type sgRulesMsg struct {
	rules []awsvpc.SecurityGroupRule
	err   error
}

type naclEntriesMsg struct {
	entries []awsvpc.NetworkACLEntry
	err     error
}

// SubDetailView shows details for a single VPC sub-resource.
type SubDetailView struct {
	client VPCClient
	router plugin.Router
	kind   string // "subnet", "sg", "rt", "nat", "endpoint", "peering", "nacl", "flowlog"
	title  string

	kvRows []ui.KV

	// Security group rules
	sgRules   []awsvpc.SecurityGroupRule
	sgGroupID string

	// Route table
	routes       []awsvpc.RouteEntry
	associations []awsvpc.RouteTableAssociation

	// NACL entries
	naclEntries []awsvpc.NetworkACLEntry
	naclID      string

	// Endpoint
	endpointSubnets     []string
	endpointRouteTables []string

	tabs    ui.TabController
	loading bool
	err     error
}

// --- Constructors for each sub-resource type ---

func NewSubnetDetailView(client VPCClient, router plugin.Router, s awsvpc.SubnetInfo) *SubDetailView {
	return &SubDetailView{
		client: client,
		router: router,
		kind:   "subnet",
		title:  fmt.Sprintf("Subnet: %s", nameOrID(s.Name, s.SubnetID)),
		kvRows: []ui.KV{
			{"Subnet ID", s.SubnetID},
			{"Name", s.Name},
			{"CIDR", s.CIDR},
			{"Availability Zone", s.AZ},
			{"Available IPs", strconv.Itoa(s.AvailableIPs)},
		},
	}
}

func NewSGDetailView(client VPCClient, router plugin.Router, sg awsvpc.SecurityGroupInfo) *SubDetailView {
	return &SubDetailView{
		client:    client,
		router:    router,
		kind:      "sg",
		title:     fmt.Sprintf("SG: %s", nameOrID(sg.Name, sg.GroupID)),
		sgGroupID: sg.GroupID,
		tabs:      ui.NewTabController([]string{"Overview", "Inbound Rules", "Outbound Rules"}),
		loading:   true,
		kvRows: []ui.KV{
			{"Group ID", sg.GroupID},
			{"Name", sg.Name},
			{"Description", sg.Description},
			{"Inbound Rules", strconv.Itoa(sg.InboundRules)},
			{"Outbound Rules", strconv.Itoa(sg.OutboundRules)},
		},
	}
}

func NewRouteTableDetailView(client VPCClient, router plugin.Router, rt awsvpc.RouteTableInfo) *SubDetailView {
	isMain := "No"
	if rt.IsMain {
		isMain = "Yes"
	}
	return &SubDetailView{
		client:       client,
		router:       router,
		kind:         "rt",
		title:        fmt.Sprintf("Route Table: %s", nameOrID(rt.Name, rt.RouteTableID)),
		tabs:         ui.NewTabController([]string{"Overview", "Routes", "Associations"}),
		routes:       rt.Routes,
		associations: rt.Associations,
		kvRows: []ui.KV{
			{"Route Table ID", rt.RouteTableID},
			{"Name", rt.Name},
			{"Main", isMain},
			{"Routes", strconv.Itoa(len(rt.Routes))},
			{"Associations", strconv.Itoa(len(rt.Associations))},
		},
	}
}

func NewNATGatewayDetailView(client VPCClient, router plugin.Router, ng awsvpc.NATGatewayInfo) *SubDetailView {
	return &SubDetailView{
		client: client,
		router: router,
		kind:   "nat",
		title:  fmt.Sprintf("NAT GW: %s", nameOrID(ng.Name, ng.GatewayID)),
		kvRows: []ui.KV{
			{"Gateway ID", ng.GatewayID},
			{"Name", ng.Name},
			{"State", ng.State},
			{"Type", ng.Type},
			{"Subnet", ng.SubnetID},
			{"Elastic IP", ng.ElasticIP},
			{"Private IP", ng.PrivateIP},
		},
	}
}

func NewEndpointDetailView(client VPCClient, router plugin.Router, ep awsvpc.VPCEndpointInfo) *SubDetailView {
	return &SubDetailView{
		client:              client,
		router:              router,
		kind:                "endpoint",
		title:               fmt.Sprintf("Endpoint: %s", ep.EndpointID),
		tabs:                ui.NewTabController([]string{"Overview", "Subnets", "Route Tables"}),
		endpointSubnets:     ep.SubnetIDs,
		endpointRouteTables: ep.RouteTableIDs,
		kvRows: []ui.KV{
			{"Endpoint ID", ep.EndpointID},
			{"Service", ep.ServiceName},
			{"Type", ep.Type},
			{"State", ep.State},
			{"Subnets", strconv.Itoa(len(ep.SubnetIDs))},
			{"Route Tables", strconv.Itoa(len(ep.RouteTableIDs))},
		},
	}
}

func NewPeeringDetailView(client VPCClient, router plugin.Router, p awsvpc.VPCPeeringInfo) *SubDetailView {
	return &SubDetailView{
		client: client,
		router: router,
		kind:   "peering",
		title:  fmt.Sprintf("Peering: %s", nameOrID(p.Name, p.PeeringID)),
		kvRows: []ui.KV{
			{"Peering ID", p.PeeringID},
			{"Name", p.Name},
			{"Status", p.Status},
			{"Requester VPC", p.RequesterVPC},
			{"Requester CIDR", p.RequesterCIDR},
			{"Accepter VPC", p.AccepterVPC},
			{"Accepter CIDR", p.AccepterCIDR},
		},
	}
}

func NewNACLDetailView(client VPCClient, router plugin.Router, n awsvpc.NetworkACLInfo) *SubDetailView {
	isDefault := "No"
	if n.IsDefault {
		isDefault = "Yes"
	}
	return &SubDetailView{
		client:  client,
		router:  router,
		kind:    "nacl",
		title:   fmt.Sprintf("NACL: %s", nameOrID(n.Name, n.NACLID)),
		naclID:  n.NACLID,
		tabs:    ui.NewTabController([]string{"Overview", "Inbound", "Outbound"}),
		loading: true,
		kvRows: []ui.KV{
			{"NACL ID", n.NACLID},
			{"Name", n.Name},
			{"Default", isDefault},
			{"Inbound Rules", strconv.Itoa(n.Inbound)},
			{"Outbound Rules", strconv.Itoa(n.Outbound)},
		},
	}
}

func NewFlowLogDetailView(client VPCClient, router plugin.Router, fl awsvpc.FlowLogInfo) *SubDetailView {
	return &SubDetailView{
		client: client,
		router: router,
		kind:   "flowlog",
		title:  fmt.Sprintf("Flow Log: %s", fl.FlowLogID),
		kvRows: []ui.KV{
			{"Flow Log ID", fl.FlowLogID},
			{"Status", fl.Status},
			{"Traffic Type", fl.TrafficType},
			{"Destination", fl.LogDestination},
			{"Log Format", fl.LogFormat},
		},
	}
}

func nameOrID(name, id string) string {
	if name != "" {
		return name
	}
	return id
}

// --- tea.Model ---

func (v *SubDetailView) Init() tea.Cmd {
	switch v.kind {
	case "sg":
		return v.fetchSGRules()
	case "nacl":
		return v.fetchNACLEntries()
	}
	return nil
}

func (v *SubDetailView) fetchSGRules() tea.Cmd {
	client := v.client
	groupID := v.sgGroupID
	return func() tea.Msg {
		rules, err := client.ListSecurityGroupRules(context.TODO(), groupID)
		return sgRulesMsg{rules: rules, err: err}
	}
}

func (v *SubDetailView) fetchNACLEntries() tea.Cmd {
	client := v.client
	naclID := v.naclID
	return func() tea.Msg {
		entries, err := client.ListNetworkACLEntries(context.TODO(), naclID)
		return naclEntriesMsg{entries: entries, err: err}
	}
}

func (v *SubDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sgRulesMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.sgRules = msg.rules
		return v, nil

	case naclEntriesMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.naclEntries = msg.entries
		return v, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			v.router.Pop()
			return v, nil
		}
	}

	if v.tabs.Count() > 0 {
		var cmd tea.Cmd
		v.tabs, cmd = v.tabs.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *SubDetailView) View() tea.View {
	if v.loading {
		skel := ui.NewSkeleton(60, 6)
		return tea.NewView(skel.View())
	}
	if v.err != nil {
		return tea.NewView("Error: " + v.err.Error())
	}

	var b strings.Builder

	hasTabs := v.tabs.Count() > 0
	if hasTabs {
		b.WriteString(v.tabs.View())
		b.WriteString("\n\n")
	}

	switch v.kind {
	case "sg":
		b.WriteString(v.renderSG())
	case "rt":
		b.WriteString(v.renderRouteTable())
	case "nacl":
		b.WriteString(v.renderNACL())
	case "endpoint":
		b.WriteString(v.renderEndpoint())
	default:
		// Simple KV-only views
		b.WriteString(ui.RenderKV(v.kvRows, 20, 0))
	}

	return tea.NewView(b.String())
}

// --- render helpers ---

func (v *SubDetailView) renderSG() string {
	switch v.tabs.Active() {
	case 0:
		return ui.RenderKV(v.kvRows, 20, 0)
	case 1:
		return v.renderSGRules("inbound")
	case 2:
		return v.renderSGRules("outbound")
	}
	return ""
}

func (v *SubDetailView) renderSGRules(direction string) string {
	var filtered []awsvpc.SecurityGroupRule
	for _, r := range v.sgRules {
		if r.Direction == direction {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return fmt.Sprintf("No %s rules.", direction)
	}

	cols := []ui.Column[awsvpc.SecurityGroupRule]{
		{Title: "Protocol", Width: 10, Field: func(r awsvpc.SecurityGroupRule) string { return r.Protocol }},
		{Title: "Port", Width: 12, Field: func(r awsvpc.SecurityGroupRule) string { return r.PortRange }},
		{Title: "Source/Dest", Width: 24, Field: func(r awsvpc.SecurityGroupRule) string { return r.Source }},
		{Title: "Description", Width: 30, Field: func(r awsvpc.SecurityGroupRule) string { return r.Description }},
	}
	tv := ui.NewTableView(cols, filtered, func(r awsvpc.SecurityGroupRule) string {
		return r.Protocol + r.PortRange + r.Source
	})
	return tv.View()
}

func (v *SubDetailView) renderRouteTable() string {
	switch v.tabs.Active() {
	case 0:
		return ui.RenderKV(v.kvRows, 20, 0)
	case 1:
		return v.renderRoutes()
	case 2:
		return v.renderAssociations()
	}
	return ""
}

func (v *SubDetailView) renderRoutes() string {
	if len(v.routes) == 0 {
		return "No routes."
	}
	cols := []ui.Column[awsvpc.RouteEntry]{
		{Title: "Destination", Width: 24, Field: func(r awsvpc.RouteEntry) string { return r.Destination }},
		{Title: "Target", Width: 28, Field: func(r awsvpc.RouteEntry) string { return r.Target }},
		{Title: "Status", Width: 12, Field: func(r awsvpc.RouteEntry) string { return r.Status }},
		{Title: "Origin", Width: 24, Field: func(r awsvpc.RouteEntry) string { return r.Origin }},
	}
	tv := ui.NewTableView(cols, v.routes, func(r awsvpc.RouteEntry) string {
		return r.Destination
	})
	return tv.View()
}

func (v *SubDetailView) renderAssociations() string {
	if len(v.associations) == 0 {
		return "No subnet associations."
	}
	cols := []ui.Column[awsvpc.RouteTableAssociation]{
		{Title: "Subnet ID", Width: 28, Field: func(a awsvpc.RouteTableAssociation) string {
			if a.SubnetID == "" {
				return "(main)"
			}
			return a.SubnetID
		}},
		{Title: "Main", Width: 6, Field: func(a awsvpc.RouteTableAssociation) string {
			if a.IsMain {
				return "Yes"
			}
			return "No"
		}},
	}
	tv := ui.NewTableView(cols, v.associations, func(a awsvpc.RouteTableAssociation) string {
		return a.SubnetID
	})
	return tv.View()
}

func (v *SubDetailView) renderNACL() string {
	switch v.tabs.Active() {
	case 0:
		return ui.RenderKV(v.kvRows, 20, 0)
	case 1:
		return v.renderNACLEntries("inbound")
	case 2:
		return v.renderNACLEntries("outbound")
	}
	return ""
}

func (v *SubDetailView) renderNACLEntries(direction string) string {
	var filtered []awsvpc.NetworkACLEntry
	for _, e := range v.naclEntries {
		if e.Direction == direction {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		return fmt.Sprintf("No %s entries.", direction)
	}

	cols := []ui.Column[awsvpc.NetworkACLEntry]{
		{Title: "#", Width: 8, Field: func(e awsvpc.NetworkACLEntry) string { return strconv.Itoa(e.RuleNumber) }},
		{Title: "Protocol", Width: 10, Field: func(e awsvpc.NetworkACLEntry) string { return e.Protocol }},
		{Title: "Port", Width: 12, Field: func(e awsvpc.NetworkACLEntry) string { return e.PortRange }},
		{Title: "CIDR", Width: 20, Field: func(e awsvpc.NetworkACLEntry) string { return e.CIDRBlock }},
		{Title: "Action", Width: 8, Field: func(e awsvpc.NetworkACLEntry) string { return e.Action }},
	}
	tv := ui.NewTableView(cols, filtered, func(e awsvpc.NetworkACLEntry) string {
		return strconv.Itoa(e.RuleNumber)
	})
	return tv.View()
}

func (v *SubDetailView) renderEndpoint() string {
	switch v.tabs.Active() {
	case 0:
		return ui.RenderKV(v.kvRows, 20, 0)
	case 1:
		if len(v.endpointSubnets) == 0 {
			return "No subnets."
		}
		var b strings.Builder
		for _, s := range v.endpointSubnets {
			b.WriteString("  " + s + "\n")
		}
		return b.String()
	case 2:
		if len(v.endpointRouteTables) == 0 {
			return "No route tables."
		}
		var b strings.Builder
		for _, rt := range v.endpointRouteTables {
			b.WriteString("  " + rt + "\n")
		}
		return b.String()
	}
	return ""
}

func (v *SubDetailView) Title() string { return v.title }

func (v *SubDetailView) KeyHints() []plugin.KeyHint {
	hints := []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
	}
	if v.tabs.Count() > 0 {
		hints = append(hints, plugin.KeyHint{Key: "[/]", Desc: "switch tab"})
	}
	return hints
}

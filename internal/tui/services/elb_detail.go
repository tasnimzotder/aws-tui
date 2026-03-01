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
	"charm.land/lipgloss/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awselb "tasnim.dev/aws-tui/internal/aws/elb"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type elbNavigateVPCMsg struct {
	vpc awsvpc.VPCInfo
}

type elbNavigateVPCErrMsg struct {
	err error
}

type elbHealthSummaryMsg struct {
	healthy   int
	unhealthy int
}

// ---------------------------------------------------------------------------
// ELB Detail View
// ---------------------------------------------------------------------------

type ELBDetailView struct {
	client *awsclient.ServiceClient
	lb     awselb.ELBLoadBalancer

	activeTab int
	tabNames  []string
	tabViews  []View

	healthy   int
	unhealthy int
	healthOK  bool

	spinner spinner.Model
	width   int
	height  int
}

func NewELBDetailView(client *awsclient.ServiceClient, lb awselb.ELBLoadBalancer) *ELBDetailView {
	return &ELBDetailView{
		client:   client,
		lb:       lb,
		tabNames: []string{"Listeners", "Target Groups", "Rules", "Attributes", "Tags"},
		tabViews: make([]View, 5),
		spinner:  theme.NewSpinner(),
	}
}

func (v *ELBDetailView) Title() string { return v.lb.Name }

func (v *ELBDetailView) HelpContext() *HelpContext {
	ctx := HelpContextELBDetail
	return &ctx
}

func (v *ELBDetailView) Init() tea.Cmd {
	v.initTab(0)
	var cmds []tea.Cmd
	if v.tabViews[0] != nil {
		cmds = append(cmds, v.tabViews[0].Init())
	}
	cmds = append(cmds, v.fetchHealthSummary())
	return tea.Batch(cmds...)
}

func (v *ELBDetailView) fetchHealthSummary() tea.Cmd {
	client := v.client
	lbARN := v.lb.ARN
	return func() tea.Msg {
		tgs, err := client.ELB.ListTargetGroups(context.Background(), lbARN)
		if err != nil {
			return elbHealthSummaryMsg{}
		}
		var h, u int
		for _, tg := range tgs {
			h += tg.HealthyCount
			u += tg.UnhealthyCount
		}
		return elbHealthSummaryMsg{healthy: h, unhealthy: u}
	}
}

func (v *ELBDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case elbNavigateVPCMsg:
		return v, pushView(NewVPCDetailView(v.client, msg.vpc))

	case elbNavigateVPCErrMsg:
		return v, nil

	case elbHealthSummaryMsg:
		v.healthy = msg.healthy
		v.unhealthy = msg.unhealthy
		v.healthOK = true
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
		case "v":
			if v.lb.VPCID != "" {
				return v, v.navigateToVPC()
			}
		default:
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
		if v.tabViews[v.activeTab] != nil {
			updated, cmd := v.tabViews[v.activeTab].Update(msg)
			v.tabViews[v.activeTab] = updated
			return v, cmd
		}
		return v, nil

	default:
		if v.tabViews[v.activeTab] != nil {
			updated, cmd := v.tabViews[v.activeTab].Update(msg)
			v.tabViews[v.activeTab] = updated
			return v, cmd
		}
	}

	return v, nil
}

func (v *ELBDetailView) navigateToVPC() tea.Cmd {
	client := v.client
	vpcID := v.lb.VPCID
	return func() tea.Msg {
		vpcs, err := client.VPC.ListVPCs(context.Background())
		if err != nil {
			return elbNavigateVPCErrMsg{err: err}
		}
		for _, vpc := range vpcs {
			if vpc.VPCID == vpcID {
				return elbNavigateVPCMsg{vpc: vpc}
			}
		}
		return elbNavigateVPCErrMsg{err: fmt.Errorf("VPC %s not found", vpcID)}
	}
}

func (v *ELBDetailView) switchTab(idx int) tea.Cmd {
	v.activeTab = idx
	v.initTab(idx)
	v.resizeActiveTab()
	if v.tabViews[idx] != nil {
		return v.tabViews[idx].Init()
	}
	return nil
}

func (v *ELBDetailView) initTab(idx int) {
	if v.tabViews[idx] != nil {
		return
	}
	switch idx {
	case 0:
		v.tabViews[idx] = newELBListenersTab(v.client, v.lb.ARN)
	case 1:
		v.tabViews[idx] = newELBTargetGroupsTab(v.client, v.lb.ARN)
	case 2:
		v.tabViews[idx] = newELBAllRulesTab(v.client, v.lb.ARN)
	case 3:
		v.tabViews[idx] = newELBAttributesTab(v.client, v.lb.ARN)
	case 4:
		v.tabViews[idx] = newELBTagsTab(v.client, v.lb.ARN)
	}
}

func (v *ELBDetailView) resizeActiveTab() {
	if v.tabViews[v.activeTab] == nil {
		return
	}
	if rv, ok := v.tabViews[v.activeTab].(ResizableView); ok {
		rv.SetSize(v.width, v.contentHeight())
	}
}

func (v *ELBDetailView) contentHeight() int {
	h := v.height - 8
	if h < 3 {
		h = 3
	}
	return h
}

func (v *ELBDetailView) View() string {
	var sections []string
	sections = append(sections, v.renderDashboard())
	sections = append(sections, v.renderTabBar())

	if v.tabViews[v.activeTab] != nil {
		sections = append(sections, v.tabViews[v.activeTab].View())
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *ELBDetailView) renderDashboard() string {
	lb := v.lb

	stateStyle := theme.MutedStyle
	stateIcon := "●"
	if lb.State == "active" {
		stateStyle = theme.SuccessStyle
	}

	line1 := fmt.Sprintf("%s  %s  %s",
		lb.Name,
		lb.Type,
		stateStyle.Render(stateIcon+" "+lb.State))

	line2 := fmt.Sprintf("Scheme: %s  DNS: %s", lb.Scheme, lb.DNSName)

	line3Parts := []string{}
	if lb.VPCID != "" {
		line3Parts = append(line3Parts, "VPC: "+lb.VPCID)
	}
	if !lb.CreatedAt.IsZero() {
		line3Parts = append(line3Parts, "Created: "+lb.CreatedAt.Format("2006-01-02"))
	}
	line3 := strings.Join(line3Parts, "  ")

	if v.healthOK {
		healthStr := fmt.Sprintf("Healthy: %s  Unhealthy: %s",
			theme.SuccessStyle.Render(fmt.Sprintf("%d", v.healthy)),
			theme.ErrorStyle.Render(fmt.Sprintf("%d", v.unhealthy)))
		line3 += "  " + healthStr
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 1)

	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4)
	}

	content := line1 + "\n" + line2
	if line3 != "" {
		content += "\n" + line3
	}

	return boxStyle.Render(content)
}

func (v *ELBDetailView) renderTabBar() string {
	var tabs []string
	for i, name := range v.tabNames {
		label := fmt.Sprintf("%d:%s", i+1, name)
		if i == v.activeTab {
			tabs = append(tabs, theme.TabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, theme.TabInactiveStyle.Render(label))
		}
	}
	return theme.TabBarStyle.Render(strings.Join(tabs, ""))
}

func (v *ELBDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.resizeActiveTab()
}

func (v *ELBDetailView) CopyID() string  { return v.lb.Name }
func (v *ELBDetailView) CopyARN() string { return v.lb.ARN }

// ---------------------------------------------------------------------------
// Tab 0: Listeners
// ---------------------------------------------------------------------------

func newELBListenersTab(client *awsclient.ServiceClient, lbARN string) *TableView[awselb.ELBListener] {
	return NewTableView(TableViewConfig[awselb.ELBListener]{
		Title:       "Listeners",
		LoadingText: "Loading listeners...",
		Columns: []table.Column{
			{Title: "Port", Width: 8},
			{Title: "Protocol", Width: 10},
			{Title: "SSL Policy", Width: 35},
			{Title: "Default Action", Width: 40},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBListener, error) {
			return client.ELB.ListListeners(ctx, lbARN)
		},
		RowMapper: func(l awselb.ELBListener) table.Row {
			ssl := l.SSLPolicy
			if ssl == "" {
				ssl = "—"
			}
			return table.Row{fmt.Sprintf("%d", l.Port), l.Protocol, ssl, l.DefaultAction}
		},
		CopyIDFunc:  func(l awselb.ELBListener) string { return fmt.Sprintf("%d", l.Port) },
		CopyARNFunc: func(l awselb.ELBListener) string { return l.ARN },
		OnEnter: func(l awselb.ELBListener) tea.Cmd {
			return pushView(NewELBListenerRulesView(client, l.ARN, fmt.Sprintf(":%d Rules", l.Port)))
		},
	})
}

// ---------------------------------------------------------------------------
// Tab 1: Target Groups
// ---------------------------------------------------------------------------

func newELBTargetGroupsTab(client *awsclient.ServiceClient, lbARN string) *TableView[awselb.ELBTargetGroup] {
	return NewTableView(TableViewConfig[awselb.ELBTargetGroup]{
		Title:       "Target Groups",
		LoadingText: "Loading target groups...",
		Columns: []table.Column{
			{Title: "Name", Width: 25},
			{Title: "Protocol", Width: 10},
			{Title: "Port", Width: 8},
			{Title: "Type", Width: 12},
			{Title: "Healthy", Width: 8},
			{Title: "Unhealthy", Width: 10},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBTargetGroup, error) {
			return client.ELB.ListTargetGroups(ctx, lbARN)
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
		OnEnter: func(tg awselb.ELBTargetGroup) tea.Cmd {
			return pushView(NewELBTargetsView(client, tg.ARN, tg.Name+" Targets"))
		},
	})
}

// ---------------------------------------------------------------------------
// Tab 2: All Rules (across all listeners)
// ---------------------------------------------------------------------------

func newELBAllRulesTab(client *awsclient.ServiceClient, lbARN string) *TableView[awselb.ELBListenerRule] {
	return NewTableView(TableViewConfig[awselb.ELBListenerRule]{
		Title:       "Rules",
		LoadingText: "Loading rules...",
		Columns: []table.Column{
			{Title: "Priority", Width: 10},
			{Title: "Conditions", Width: 40},
			{Title: "Actions", Width: 40},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBListenerRule, error) {
			listeners, err := client.ELB.ListListeners(ctx, lbARN)
			if err != nil {
				return nil, err
			}
			var allRules []awselb.ELBListenerRule
			for _, l := range listeners {
				rules, err := client.ELB.ListListenerRules(ctx, l.ARN)
				if err != nil {
					return nil, err
				}
				allRules = append(allRules, rules...)
			}
			return allRules, nil
		},
		RowMapper: func(r awselb.ELBListenerRule) table.Row {
			conds := strings.Join(r.Conditions, "; ")
			if conds == "" {
				conds = "—"
			}
			acts := strings.Join(r.Actions, "; ")
			if acts == "" {
				acts = "—"
			}
			return table.Row{r.Priority, conds, acts}
		},
		CopyIDFunc: func(r awselb.ELBListenerRule) string { return r.Priority },
	})
}

// ---------------------------------------------------------------------------
// Tab 3: Attributes (viewport)
// ---------------------------------------------------------------------------

type elbAttributesTab struct {
	client *awsclient.ServiceClient
	lbARN  string

	attrs         []awselb.ELBAttribute
	viewport      viewport.Model
	vpReady       bool
	loading       bool
	err           error
	spinner       spinner.Model
	width, height int
}

func newELBAttributesTab(client *awsclient.ServiceClient, lbARN string) *elbAttributesTab {
	return &elbAttributesTab{
		client:  client,
		lbARN:   lbARN,
		spinner: theme.NewSpinner(),
	}
}

type elbAttrsMsg struct{ attrs []awselb.ELBAttribute }

func (v *elbAttributesTab) Title() string { return "Attributes" }

func (v *elbAttributesTab) Init() tea.Cmd {
	v.loading = true
	return tea.Batch(v.spinner.Tick, v.fetch())
}

func (v *elbAttributesTab) fetch() tea.Cmd {
	client := v.client
	arn := v.lbARN
	return func() tea.Msg {
		attrs, err := client.ELB.GetLoadBalancerAttributes(context.Background(), arn)
		if err != nil {
			return errViewMsg{err: err}
		}
		return elbAttrsMsg{attrs: attrs}
	}
}

func (v *elbAttributesTab) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case elbAttrsMsg:
		v.attrs = msg.attrs
		v.loading = false
		v.initViewport()
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
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
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
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

func (v *elbAttributesTab) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 10
	}
	w := v.width
	if w < 20 {
		w = 80
	}
	vp := viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	vp.SetContent(v.renderContent())
	v.viewport = vp
	v.vpReady = true
}

func (v *elbAttributesTab) renderContent() string {
	if len(v.attrs) == 0 {
		return "No attributes"
	}

	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	b.WriteString(bold.Render("Load Balancer Attributes"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	maxKeyLen := 0
	for _, a := range v.attrs {
		if len(a.Key) > maxKeyLen {
			maxKeyLen = len(a.Key)
		}
	}

	for _, a := range v.attrs {
		b.WriteString(fmt.Sprintf("  %-*s = %s\n", maxKeyLen, a.Key, a.Value))
	}
	return b.String()
}

func (v *elbAttributesTab) View() string {
	if v.loading {
		return v.spinner.View() + " Loading attributes..."
	}
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.vpReady {
		return ""
	}
	return v.viewport.View()
}

func (v *elbAttributesTab) SetSize(width, height int) {
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
// Tab 4: Tags (viewport, async)
// ---------------------------------------------------------------------------

type elbTagsTab struct {
	client *awsclient.ServiceClient
	lbARN  string

	tags          map[string]string
	viewport      viewport.Model
	vpReady       bool
	loading       bool
	err           error
	spinner       spinner.Model
	width, height int
}

func newELBTagsTab(client *awsclient.ServiceClient, lbARN string) *elbTagsTab {
	return &elbTagsTab{
		client:  client,
		lbARN:   lbARN,
		spinner: theme.NewSpinner(),
	}
}

type elbTagsMsg struct{ tags map[string]string }

func (v *elbTagsTab) Title() string { return "Tags" }

func (v *elbTagsTab) Init() tea.Cmd {
	v.loading = true
	return tea.Batch(v.spinner.Tick, v.fetch())
}

func (v *elbTagsTab) fetch() tea.Cmd {
	client := v.client
	arn := v.lbARN
	return func() tea.Msg {
		result, err := client.ELB.GetResourceTags(context.Background(), []string{arn})
		if err != nil {
			return errViewMsg{err: err}
		}
		tags := result[arn]
		if tags == nil {
			tags = map[string]string{}
		}
		return elbTagsMsg{tags: tags}
	}
}

func (v *elbTagsTab) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case elbTagsMsg:
		v.tags = msg.tags
		v.loading = false
		v.initViewport()
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
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
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
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

func (v *elbTagsTab) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 10
	}
	w := v.width
	if w < 20 {
		w = 80
	}
	vp := viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	vp.SetContent(v.renderContent())
	v.viewport = vp
	v.vpReady = true
}

func (v *elbTagsTab) renderContent() string {
	if len(v.tags) == 0 {
		return "No tags"
	}

	keys := make([]string, 0, len(v.tags))
	for k := range v.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	b.WriteString(bold.Render("Tags"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	for _, k := range keys {
		b.WriteString(fmt.Sprintf("  %-*s = %s\n", maxKeyLen, k, v.tags[k]))
	}
	return b.String()
}

func (v *elbTagsTab) View() string {
	if v.loading {
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

func (v *elbTagsTab) SetSize(width, height int) {
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
// Sub-view: Listener Rules
// ---------------------------------------------------------------------------

func NewELBListenerRulesView(client *awsclient.ServiceClient, listenerARN, title string) *TableView[awselb.ELBListenerRule] {
	return NewTableView(TableViewConfig[awselb.ELBListenerRule]{
		Title:       title,
		LoadingText: "Loading rules...",
		Columns: []table.Column{
			{Title: "Priority", Width: 10},
			{Title: "Conditions", Width: 40},
			{Title: "Actions", Width: 40},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBListenerRule, error) {
			return client.ELB.ListListenerRules(ctx, listenerARN)
		},
		RowMapper: func(r awselb.ELBListenerRule) table.Row {
			conds := strings.Join(r.Conditions, "; ")
			if conds == "" {
				conds = "—"
			}
			acts := strings.Join(r.Actions, "; ")
			if acts == "" {
				acts = "—"
			}
			return table.Row{r.Priority, conds, acts}
		},
		CopyIDFunc: func(r awselb.ELBListenerRule) string { return r.Priority },
	})
}

// ---------------------------------------------------------------------------
// Sub-view: Targets
// ---------------------------------------------------------------------------

func NewELBTargetsView(client *awsclient.ServiceClient, tgARN, title string) *TableView[awselb.ELBTarget] {
	return NewTableView(TableViewConfig[awselb.ELBTarget]{
		Title:       title,
		LoadingText: "Loading targets...",
		Columns: []table.Column{
			{Title: "ID", Width: 22},
			{Title: "Port", Width: 8},
			{Title: "AZ", Width: 14},
			{Title: "Health", Width: 12},
			{Title: "Reason", Width: 25},
			{Title: "Description", Width: 30},
		},
		FetchFunc: func(ctx context.Context) ([]awselb.ELBTarget, error) {
			return client.ELB.ListTargets(ctx, tgARN)
		},
		RowMapper: func(t awselb.ELBTarget) table.Row {
			return table.Row{
				t.ID,
				fmt.Sprintf("%d", t.Port),
				t.AZ,
				t.HealthState,
				t.HealthReason,
				t.HealthDesc,
			}
		},
		CopyIDFunc: func(t awselb.ELBTarget) string { return t.ID },
	})
}

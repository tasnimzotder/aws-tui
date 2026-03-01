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
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

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

	tabs *TabController

	healthy   int
	unhealthy int
	healthOK  bool

	width  int
	height int
}

func NewELBDetailView(client *awsclient.ServiceClient, lb awselb.ELBLoadBalancer) *ELBDetailView {
	v := &ELBDetailView{
		client: client,
		lb:     lb,
	}
	v.tabs = NewTabController(
		[]string{"Listeners", "Target Groups", "Rules", "Attributes", "Tags"},
		v.createTab,
	)
	return v
}

func (v *ELBDetailView) createTab(idx int) View {
	switch idx {
	case 0:
		return newELBListenersTab(v.client, v.lb.ARN)
	case 1:
		return newELBTargetGroupsTab(v.client, v.lb.ARN)
	case 2:
		return newELBAllRulesTab(v.client, v.lb.ARN)
	case 3:
		return newELBAttributesTab(v.client, v.lb.ARN)
	case 4:
		return newELBTagsTab(v.client, v.lb.ARN)
	}
	return nil
}

func (v *ELBDetailView) Title() string { return v.lb.Name }

func (v *ELBDetailView) HelpContext() *HelpContext {
	ctx := HelpContextELBDetail
	return &ctx
}

func (v *ELBDetailView) Init() tea.Cmd {
	cmd := v.tabs.SwitchTab(0)
	v.tabs.ResizeActive(v.width, v.contentHeight())
	return tea.Batch(cmd, v.fetchHealthSummary())
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
	case navigateVPCMsg:
		return v, pushView(NewVPCDetailView(v.client, msg.vpc))

	case navigateVPCErrMsg:
		return v, nil

	case elbHealthSummaryMsg:
		v.healthy = msg.healthy
		v.unhealthy = msg.unhealthy
		v.healthOK = true
		return v, nil

	case tea.KeyPressMsg:
		key := msg.String()
		if handled, cmd := v.tabs.HandleKey(key); handled {
			v.tabs.ResizeActive(v.width, v.contentHeight())
			return v, cmd
		}
		switch key {
		case "v":
			if v.lb.VPCID != "" {
				return v, NavigateToVPC(v.client.VPC, v.lb.VPCID)
			}
		}
		return v, v.tabs.DelegateUpdate(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.tabs.ResizeActive(v.width, v.contentHeight())
		return v, nil

	default:
		return v, v.tabs.DelegateUpdate(msg)
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
	sections = append(sections, v.tabs.RenderTabBar())

	if av := v.tabs.ActiveView(); av != nil {
		sections = append(sections, av.View())
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *ELBDetailView) renderDashboard() string {
	lb := v.lb
	label := theme.MutedStyle

	// Title: Name + type badge + status badge
	title := theme.DashboardTitleStyle.Render(lb.Name) +
		"  " + theme.MutedStyle.Render("["+lb.Type+"]") +
		"  " + theme.RenderStatus(lb.State)

	line1 := label.Render("Scheme: ") + lb.Scheme + label.Render("  DNS: ") + lb.DNSName

	line2Parts := []string{}
	if lb.VPCID != "" {
		line2Parts = append(line2Parts, label.Render("VPC: ")+lb.VPCID)
	}
	if !lb.CreatedAt.IsZero() {
		line2Parts = append(line2Parts, label.Render("Created: ")+lb.CreatedAt.Format("2006-01-02"))
	}
	if v.healthOK {
		line2Parts = append(line2Parts,
			label.Render("Healthy: ")+theme.SuccessStyle.Render(fmt.Sprintf("%d", v.healthy))+
				label.Render("  Unhealthy: ")+theme.ErrorStyle.Render(fmt.Sprintf("%d", v.unhealthy)))
	}
	line2 := strings.Join(line2Parts, "  ")

	boxStyle := theme.DashboardBoxStyle
	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4)
	}

	content := title + "\n" + line1
	if line2 != "" {
		content += "\n" + line2
	}

	return boxStyle.Render(content)
}

func (v *ELBDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.tabs.ResizeActive(v.width, v.contentHeight())
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
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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

package services

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// eksK8sReadyMsg is sent when the K8s client is connected and node health is fetched.
type eksK8sReadyMsg struct {
	k8sClient      *awseks.K8sClient
	nodeHealth     string
	nodeConditions string
}

// eksK8sErrorMsg is sent when K8s connection fails.
type eksK8sErrorMsg struct {
	err error
}

// eksNamespacesMsg carries the fetched list of namespaces.
type eksNamespacesMsg struct {
	namespaces []string
}

// eksNamespaceItem is used with the list.Model namespace picker.
type eksNamespaceItem struct{ name string }

func (i eksNamespaceItem) Title() string       { return i.name }
func (i eksNamespaceItem) Description() string { return "" }
func (i eksNamespaceItem) FilterValue() string { return i.name }

// EKSClusterDetailView shows a pinned dashboard at the top and a horizontal
// tab bar with switchable sub-resource tables below.
type EKSClusterDetailView struct {
	client    *awsclient.ServiceClient
	cluster   awseks.EKSCluster
	k8sClient *awseks.K8sClient
	region    string

	// Dashboard
	nodeHealth     string
	nodeConditions string

	// Tabs
	activeTab int
	tabNames  []string
	tabViews  []View // lazily initialized, one per tab

	// Namespace filtering (for K8s tabs 4-6)
	namespace          string
	showNamespacePicker bool
	namespacePicker    list.Model

	// Port forward manager (persists across tab switches)
	pfManager *portForwardManager

	// State
	loading  bool
	k8sReady bool
	err      error
	width    int
	height   int

	spinner spinner.Model
}

// NewEKSClusterDetailView creates a new cluster detail view with dashboard and tabs.
func NewEKSClusterDetailView(client *awsclient.ServiceClient, cluster awseks.EKSCluster, region string) *EKSClusterDetailView {
	return &EKSClusterDetailView{
		client:  client,
		cluster: cluster,
		region:  region,
		tabNames: []string{
			"Node Groups", "Add-ons", "Fargate", "Access",
			"Pods", "Services", "Deployments",
		},
		tabViews:  make([]View, 7),
		pfManager: newPortForwardManager(),
		loading:   true,
		spinner:   theme.NewSpinner(),
	}
}

func (v *EKSClusterDetailView) Title() string { return v.cluster.Name }

func (v *EKSClusterDetailView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.connectK8s())
}

func (v *EKSClusterDetailView) connectK8s() tea.Cmd {
	cluster := v.cluster
	cfg := v.client.Cfg
	return func() tea.Msg {
		tokenProvider := awseks.NewTokenProvider(cfg, cluster.Name)
		k8sClient, err := awseks.NewK8sClient(cluster.Endpoint, cluster.CertAuthority, tokenProvider)
		if err != nil {
			return eksK8sErrorMsg{err: err}
		}
		k8sClient.ClusterName = cluster.Name

		ctx := context.Background()
		nodes, err := k8sClient.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return eksK8sErrorMsg{err: err}
		}

		readyCount := 0
		totalCount := len(nodes.Items)
		memoryPressure := 0
		diskPressure := 0
		pidPressure := 0

		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				switch cond.Type {
				case "Ready":
					if cond.Status == "True" {
						readyCount++
					}
				case "MemoryPressure":
					if cond.Status == "True" {
						memoryPressure++
					}
				case "DiskPressure":
					if cond.Status == "True" {
						diskPressure++
					}
				case "PIDPressure":
					if cond.Status == "True" {
						pidPressure++
					}
				}
			}
		}

		nodeHealth := fmt.Sprintf("%d/%d Ready", readyCount, totalCount)
		nodeConditions := fmt.Sprintf("● %d MemoryPressure  ● %d DiskPressure  ● %d PIDPressure",
			memoryPressure, diskPressure, pidPressure)

		return eksK8sReadyMsg{
			k8sClient:      k8sClient,
			nodeHealth:     nodeHealth,
			nodeConditions: nodeConditions,
		}
	}
}

func (v *EKSClusterDetailView) fetchNamespaces() tea.Cmd {
	k8s := v.k8sClient
	return func() tea.Msg {
		ctx := context.Background()
		nsList, err := k8s.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return eksK8sErrorMsg{err: err}
		}
		names := make([]string, 0, len(nsList.Items))
		for _, ns := range nsList.Items {
			names = append(names, ns.Name)
		}
		return eksNamespacesMsg{namespaces: names}
	}
}

// reinitK8sTab forces re-creation of the current K8s tab with the current namespace.
func (v *EKSClusterDetailView) reinitK8sTab() tea.Cmd {
	idx := v.activeTab
	if idx < 4 || v.k8sClient == nil {
		return nil
	}
	// Force re-creation by clearing the slot
	v.tabViews[idx] = nil
	switch idx {
	case 4:
		v.tabViews[idx] = NewK8sPodsTableViewWithPF(v.k8sClient, v.namespace, v.pfManager)
	case 5:
		v.tabViews[idx] = NewK8sServicesTableViewWithPF(v.k8sClient, v.namespace, v.pfManager)
	case 6:
		v.tabViews[idx] = NewK8sDeploymentsTableView(v.k8sClient, v.namespace)
	}
	v.resizeActiveTab()
	if v.tabViews[idx] != nil {
		return v.tabViews[idx].Init()
	}
	return nil
}

func (v *EKSClusterDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	// Handle namespace picker interactions when it's shown
	if v.showNamespacePicker {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "enter":
				selected, ok := v.namespacePicker.SelectedItem().(eksNamespaceItem)
				if ok {
					v.namespace = selected.name
				}
				v.showNamespacePicker = false
				return v, v.reinitK8sTab()
			case "esc":
				v.showNamespacePicker = false
				return v, nil
			default:
				var cmd tea.Cmd
				v.namespacePicker, cmd = v.namespacePicker.Update(msg)
				return v, cmd
			}
		default:
			var cmd tea.Cmd
			v.namespacePicker, cmd = v.namespacePicker.Update(msg)
			return v, cmd
		}
	}

	switch msg := msg.(type) {
	case eksNamespacesMsg:
		items := make([]list.Item, len(msg.namespaces))
		for i, ns := range msg.namespaces {
			items[i] = eksNamespaceItem{name: ns}
		}
		l := list.New(items, list.NewDefaultDelegate(), 40, 14)
		l.SetShowTitle(true)
		l.Title = "Select Namespace"
		l.SetShowStatusBar(false)
		l.SetShowHelp(false)
		l.SetFilteringEnabled(true)
		v.namespacePicker = l
		v.showNamespacePicker = true
		return v, nil

	case eksK8sReadyMsg:
		v.k8sClient = msg.k8sClient
		v.nodeHealth = msg.nodeHealth
		v.nodeConditions = msg.nodeConditions
		v.k8sReady = true
		v.loading = false
		// Initialize first tab
		v.initTab(0)
		if v.tabViews[0] != nil {
			cmd := v.tabViews[0].Init()
			return v, cmd
		}
		return v, nil

	case eksK8sErrorMsg:
		v.err = msg.err
		v.loading = false
		// Still allow browsing AWS-side tabs even if K8s connection failed
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
		case "N":
			// Namespace toggle: only on K8s tabs (4, 5, 6)
			if v.activeTab >= 4 && v.k8sClient != nil {
				if v.namespace != "" {
					// Clear namespace filter
					v.namespace = ""
					// Also clear other K8s tabs so they reinit with new namespace
					for i := 4; i <= 6; i++ {
						if i != v.activeTab {
							v.tabViews[i] = nil
						}
					}
					return v, v.reinitK8sTab()
				}
				// Fetch namespaces and show picker
				return v, v.fetchNamespaces()
			}
			// Fall through to delegate
			if v.tabViews[v.activeTab] != nil {
				updated, cmd := v.tabViews[v.activeTab].Update(msg)
				v.tabViews[v.activeTab] = updated
				return v, cmd
			}
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

func (v *EKSClusterDetailView) switchTab(idx int) tea.Cmd {
	v.activeTab = idx
	// When switching to a K8s tab, ensure it uses the current namespace.
	// Clear stale K8s tab views that may have been created with a different namespace.
	if idx >= 4 && v.tabViews[idx] != nil {
		// Force reinit if needed (namespace may have changed since last init)
		v.tabViews[idx] = nil
	}
	v.initTab(idx)
	v.resizeActiveTab()
	if v.tabViews[idx] != nil {
		return v.tabViews[idx].Init()
	}
	return nil
}

func (v *EKSClusterDetailView) initTab(idx int) {
	if v.tabViews[idx] != nil {
		return
	}
	switch idx {
	case 0:
		v.tabViews[idx] = NewEKSNodeGroupsTableView(v.client, v.cluster.Name)
	case 1:
		v.tabViews[idx] = NewEKSAddonsTableView(v.client, v.cluster.Name)
	case 2:
		v.tabViews[idx] = NewEKSFargateTableView(v.client, v.cluster.Name)
	case 3:
		v.tabViews[idx] = NewEKSAccessEntriesTableView(v.client, v.cluster.Name)
	case 4:
		if v.k8sClient != nil {
			v.tabViews[idx] = NewK8sPodsTableViewWithPF(v.k8sClient, v.namespace, v.pfManager)
		} else {
			v.tabViews[idx] = newEKSK8sPlaceholderView("Pods", v.k8sReady)
		}
	case 5:
		if v.k8sClient != nil {
			v.tabViews[idx] = NewK8sServicesTableViewWithPF(v.k8sClient, v.namespace, v.pfManager)
		} else {
			v.tabViews[idx] = newEKSK8sPlaceholderView("Services", v.k8sReady)
		}
	case 6:
		if v.k8sClient != nil {
			v.tabViews[idx] = NewK8sDeploymentsTableView(v.k8sClient, v.namespace)
		} else {
			v.tabViews[idx] = newEKSK8sPlaceholderView("Deployments", v.k8sReady)
		}
	}
}

func (v *EKSClusterDetailView) resizeActiveTab() {
	if v.tabViews[v.activeTab] == nil {
		return
	}
	if rv, ok := v.tabViews[v.activeTab].(ResizableView); ok {
		contentHeight := v.contentHeight()
		rv.SetSize(v.width, contentHeight)
	}
}

// contentHeight calculates the height available for the tab content area.
// Dashboard box takes ~6 lines, tab bar takes ~2 lines, namespace line ~1 line on K8s tabs.
func (v *EKSClusterDetailView) contentHeight() int {
	h := v.height - 8 // dashboard (6) + tab bar (2)
	if v.activeTab >= 4 {
		h-- // namespace indicator line
	}
	if h < 3 {
		h = 3
	}
	return h
}

func (v *EKSClusterDetailView) View() string {
	var sections []string

	// 1. Dashboard box
	sections = append(sections, v.renderDashboard())

	// 2. Tab bar
	sections = append(sections, v.renderTabBar())

	// 3. Namespace indicator (only on K8s tabs)
	if v.activeTab >= 4 {
		nsText := "All"
		nsStyle := theme.MutedStyle
		if v.namespace != "" {
			nsText = v.namespace
			nsStyle = lipgloss.NewStyle().Foreground(theme.Primary)
		}
		sections = append(sections, nsStyle.Render("Namespace: "+nsText))
	}

	// 4. Namespace picker overlay or tab content
	if v.showNamespacePicker {
		sections = append(sections, v.namespacePicker.View())
	} else if v.tabViews[v.activeTab] != nil {
		sections = append(sections, v.tabViews[v.activeTab].View())
	} else if v.loading {
		sections = append(sections, v.spinner.View()+" Connecting to cluster...")
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *EKSClusterDetailView) renderDashboard() string {
	cl := v.cluster

	// Status dot color
	statusDot := v.statusDot(cl.Status)

	// Info lines
	line1 := fmt.Sprintf("K8s: %s  Platform: %s  Region: %s", cl.Version, cl.PlatformVersion, v.region)

	var line2 string
	if v.loading {
		line2 = "Nodes: connecting..."
	} else if v.err != nil && !v.k8sReady {
		line2 = fmt.Sprintf("Nodes: error (%s)", v.err.Error())
	} else {
		line2 = fmt.Sprintf("Nodes: %s", v.nodeHealth)
	}

	var line3 string
	if v.k8sReady {
		line3 = fmt.Sprintf("  %s", v.nodeConditions)
	}

	endpoint := "Private"
	if cl.EndpointPublic {
		endpoint = "Public"
	}
	if cl.EndpointPublic && cl.EndpointPrivate {
		endpoint = "Public + Private"
	}
	line4 := fmt.Sprintf("Endpoint: %s  Created: %s", endpoint, utils.TimeOrDash(cl.CreatedAt, utils.DateOnly))

	content := line1 + "\n" + line2
	if line3 != "" {
		content += "\n" + line3
	}
	content += "\n" + line4

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 1)

	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4) // account for border + padding
	}

	// Render title and content
	titleStyle := lipgloss.NewStyle().Bold(true)
	statusStyle := lipgloss.NewStyle().Bold(true)

	header := titleStyle.Render(cl.Name) + "  " + statusStyle.Render(statusDot+" "+cl.Status)
	return boxStyle.Render(header + "\n" + content)
}

func (v *EKSClusterDetailView) statusDot(status string) string {
	switch strings.ToUpper(status) {
	case "ACTIVE":
		return lipgloss.NewStyle().Foreground(theme.Success).Render("●")
	case "UPDATING", "CREATING":
		return lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
	case "FAILED", "DELETING":
		return lipgloss.NewStyle().Foreground(theme.Error).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("●")
	}
}

func (v *EKSClusterDetailView) renderTabBar() string {
	var tabs []string
	for i, name := range v.tabNames {
		label := fmt.Sprintf("[%s]", name)
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
func (v *EKSClusterDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.resizeActiveTab()
}

// --- K8s placeholder view for tabs 4-6 ---

type eksK8sPlaceholderView struct {
	name    string
	k8sOK   bool
}

func newEKSK8sPlaceholderView(name string, k8sReady bool) *eksK8sPlaceholderView {
	return &eksK8sPlaceholderView{name: name, k8sOK: k8sReady}
}

func (p *eksK8sPlaceholderView) Title() string { return p.name }
func (p *eksK8sPlaceholderView) Init() tea.Cmd  { return nil }
func (p *eksK8sPlaceholderView) Update(msg tea.Msg) (View, tea.Cmd) { return p, nil }

func (p *eksK8sPlaceholderView) View() string {
	if !p.k8sOK {
		return theme.MutedStyle.Render("Connecting to cluster...")
	}
	return theme.MutedStyle.Render(fmt.Sprintf("%s — will be implemented in a future update", p.name))
}

func (p *eksK8sPlaceholderView) SetSize(width, height int) {}

// --- EKS resource tab view constructors (stubs for tabs 0-3) ---

func NewEKSNodeGroupsTableView(client *awsclient.ServiceClient, clusterName string) View {
	return NewTableView(TableViewConfig[awseks.EKSNodeGroup]{
		Title:       "Node Groups",
		LoadingText: "Loading node groups...",
		Columns: []table.Column{
			{Title: "Name", Width: 28},
			{Title: "Status", Width: 12},
			{Title: "Instance Types", Width: 20},
			{Title: "AMI Type", Width: 16},
			{Title: "Desired", Width: 8},
			{Title: "Min", Width: 6},
			{Title: "Max", Width: 6},
		},
		FetchFunc: func(ctx context.Context) ([]awseks.EKSNodeGroup, error) {
			return client.EKS.ListNodeGroups(ctx, clusterName)
		},
		RowMapper: func(ng awseks.EKSNodeGroup) table.Row {
			return table.Row{
				ng.Name,
				ng.Status,
				strings.Join(ng.InstanceTypes, ", "),
				ng.AMIType,
				fmt.Sprintf("%d", ng.DesiredSize),
				fmt.Sprintf("%d", ng.MinSize),
				fmt.Sprintf("%d", ng.MaxSize),
			}
		},
		CopyIDFunc:  func(ng awseks.EKSNodeGroup) string { return ng.Name },
		CopyARNFunc: func(ng awseks.EKSNodeGroup) string { return ng.ARN },
		OnEnter: func(ng awseks.EKSNodeGroup) tea.Cmd {
			return pushView(NewEKSNodeGroupDetailView(ng))
		},
	})
}

func NewEKSAddonsTableView(client *awsclient.ServiceClient, clusterName string) View {
	return NewTableView(TableViewConfig[awseks.EKSAddon]{
		Title:       "Add-ons",
		LoadingText: "Loading add-ons...",
		Columns: []table.Column{
			{Title: "Name", Width: 28},
			{Title: "Version", Width: 24},
			{Title: "Status", Width: 12},
			{Title: "Health", Width: 20},
		},
		FetchFunc: func(ctx context.Context) ([]awseks.EKSAddon, error) {
			return client.EKS.ListAddons(ctx, clusterName)
		},
		RowMapper: func(a awseks.EKSAddon) table.Row {
			return table.Row{a.Name, a.Version, a.Status, a.Health}
		},
		CopyIDFunc:  func(a awseks.EKSAddon) string { return a.Name },
		CopyARNFunc: func(a awseks.EKSAddon) string { return a.ARN },
		OnEnter: func(a awseks.EKSAddon) tea.Cmd {
			return pushView(NewEKSAddonDetailView(a))
		},
	})
}

func NewEKSFargateTableView(client *awsclient.ServiceClient, clusterName string) View {
	return NewTableView(TableViewConfig[awseks.EKSFargateProfile]{
		Title:       "Fargate Profiles",
		LoadingText: "Loading Fargate profiles...",
		Columns: []table.Column{
			{Title: "Name", Width: 28},
			{Title: "Status", Width: 12},
			{Title: "Selectors", Width: 40},
		},
		FetchFunc: func(ctx context.Context) ([]awseks.EKSFargateProfile, error) {
			return client.EKS.ListFargateProfiles(ctx, clusterName)
		},
		RowMapper: func(fp awseks.EKSFargateProfile) table.Row {
			var sels []string
			for _, s := range fp.Selectors {
				sels = append(sels, s.Namespace)
			}
			return table.Row{fp.Name, fp.Status, strings.Join(sels, ", ")}
		},
		CopyIDFunc:  func(fp awseks.EKSFargateProfile) string { return fp.Name },
		CopyARNFunc: func(fp awseks.EKSFargateProfile) string { return fp.ARN },
		OnEnter: func(fp awseks.EKSFargateProfile) tea.Cmd {
			return pushView(NewEKSFargateDetailView(fp))
		},
	})
}

func NewEKSAccessEntriesTableView(client *awsclient.ServiceClient, clusterName string) View {
	return NewTableView(TableViewConfig[awseks.EKSAccessEntry]{
		Title:       "Access Entries",
		LoadingText: "Loading access entries...",
		Columns: []table.Column{
			{Title: "Principal ARN", Width: 50},
			{Title: "Type", Width: 16},
			{Title: "Username", Width: 24},
			{Title: "Created", Width: 16},
		},
		FetchFunc: func(ctx context.Context) ([]awseks.EKSAccessEntry, error) {
			return client.EKS.ListAccessEntries(ctx, clusterName)
		},
		RowMapper: func(ae awseks.EKSAccessEntry) table.Row {
			return table.Row{ae.PrincipalARN, ae.Type, ae.Username, utils.TimeOrDash(ae.CreatedAt, utils.DateOnly)}
		},
		CopyIDFunc: func(ae awseks.EKSAccessEntry) string { return ae.PrincipalARN },
		OnEnter: func(ae awseks.EKSAccessEntry) tea.Cmd {
			return pushView(NewEKSAccessEntryDetailView(ae))
		},
	})
}

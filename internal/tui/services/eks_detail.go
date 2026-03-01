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
	tabs *TabController

	// Namespace filtering (for K8s tabs 4-8)
	namespace          string
	lastNamespace      string
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
	v := &EKSClusterDetailView{
		client:    client,
		cluster:   cluster,
		region:    region,
		pfManager: newPortForwardManager(),
		loading:   true,
		spinner:   theme.NewSpinner(),
	}
	v.tabs = NewTabController(
		[]string{
			"Node Groups", "Add-ons", "Fargate", "Access",
			"Pods", "Services", "Deployments", "Svc Accounts", "Ingresses",
		},
		v.createTab,
	)
	v.tabs.BeforeSwitch = func(idx int) {
		if idx >= 4 && v.lastNamespace != v.namespace {
			for i := 4; i < len(v.tabs.TabViews); i++ {
				v.tabs.TabViews[i] = nil
			}
			v.lastNamespace = v.namespace
		}
	}
	return v
}

func (v *EKSClusterDetailView) Title() string { return v.cluster.Name }

func (v *EKSClusterDetailView) HelpContext() *HelpContext {
	ctx := HelpContextEKSDetail
	return &ctx
}

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

func (v *EKSClusterDetailView) createTab(idx int) View {
	switch idx {
	case 0:
		return NewEKSNodeGroupsTableView(v.client, v.cluster.Name, v.k8sClient)
	case 1:
		return NewEKSAddonsTableView(v.client, v.cluster.Name)
	case 2:
		return NewEKSFargateTableView(v.client, v.cluster.Name)
	case 3:
		return NewEKSAccessEntriesTableView(v.client, v.cluster.Name)
	case 4:
		if v.k8sClient != nil {
			return NewK8sPodsTableViewWithPF(v.k8sClient, v.namespace, v.pfManager)
		}
		return newEKSK8sPlaceholderView("Pods", v.k8sReady)
	case 5:
		if v.k8sClient != nil {
			return NewK8sServicesTableViewWithPF(v.k8sClient, v.namespace, v.pfManager)
		}
		return newEKSK8sPlaceholderView("Services", v.k8sReady)
	case 6:
		if v.k8sClient != nil {
			return NewK8sDeploymentsTableView(v.k8sClient, v.namespace)
		}
		return newEKSK8sPlaceholderView("Deployments", v.k8sReady)
	case 7:
		if v.k8sClient != nil {
			return NewK8sServiceAccountsTableView(v.k8sClient, v.namespace)
		}
		return newEKSK8sPlaceholderView("Svc Accounts", v.k8sReady)
	case 8:
		if v.k8sClient != nil {
			return NewK8sIngressesTableView(v.k8sClient, v.namespace, v.pfManager)
		}
		return newEKSK8sPlaceholderView("Ingresses", v.k8sReady)
	}
	return nil
}

// reinitK8sTab forces re-creation of the current K8s tab with the current namespace.
func (v *EKSClusterDetailView) reinitK8sTab() tea.Cmd {
	idx := v.tabs.ActiveTab
	if idx < 4 || v.k8sClient == nil {
		return nil
	}
	v.tabs.TabViews[idx] = nil
	v.tabs.TabViews[idx] = v.createTab(idx)
	v.tabs.ResizeActive(v.width, v.contentHeight())
	if v.tabs.TabViews[idx] != nil {
		return v.tabs.TabViews[idx].Init()
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
		// Reinit tab 0 (node groups) now that k8sClient is available
		v.tabs.TabViews[0] = nil
		cmd := v.tabs.SwitchTab(0)
		v.tabs.ResizeActive(v.width, v.contentHeight())
		return v, cmd

	case eksK8sErrorMsg:
		v.err = msg.err
		v.loading = false
		// Still allow browsing AWS-side tabs even if K8s connection failed
		cmd := v.tabs.SwitchTab(0)
		v.tabs.ResizeActive(v.width, v.contentHeight())
		return v, cmd

	case tea.KeyPressMsg:
		key := msg.String()
		if handled, cmd := v.tabs.HandleKey(key); handled {
			v.tabs.ResizeActive(v.width, v.contentHeight())
			return v, cmd
		}
		switch key {
		case "N":
			if v.tabs.ActiveTab >= 4 && v.k8sClient != nil {
				if v.namespace != "" {
					v.namespace = ""
					v.lastNamespace = ""
					for i := 4; i < len(v.tabs.TabViews); i++ {
						if i != v.tabs.ActiveTab {
							v.tabs.TabViews[i] = nil
						}
					}
					return v, v.reinitK8sTab()
				}
				return v, v.fetchNamespaces()
			}
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
// Dashboard box takes ~6 lines, tab bar takes ~2 lines, namespace line ~1 line on K8s tabs.
func (v *EKSClusterDetailView) contentHeight() int {
	h := v.height - 8 // dashboard (6) + tab bar (2)
	if v.tabs.ActiveTab >= 4 {
		h-- // namespace indicator line
	}
	if h < 3 {
		h = 3
	}
	return h
}

func (v *EKSClusterDetailView) View() string {
	var sections []string

	sections = append(sections, v.renderDashboard())
	sections = append(sections, v.tabs.RenderTabBar())

	if v.tabs.ActiveTab >= 4 {
		nsText := "All"
		nsStyle := theme.MutedStyle
		if v.namespace != "" {
			nsText = v.namespace
			nsStyle = lipgloss.NewStyle().Foreground(theme.Primary)
		}
		sections = append(sections, nsStyle.Render("Namespace: "+nsText))
	}

	if v.showNamespacePicker {
		sections = append(sections, v.namespacePicker.View())
	} else if av := v.tabs.ActiveView(); av != nil {
		sections = append(sections, av.View())
	} else if v.loading {
		sections = append(sections, v.spinner.View()+" Connecting to cluster...")
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *EKSClusterDetailView) renderDashboard() string {
	cl := v.cluster
	label := theme.MutedStyle

	// Title: cluster name + status badge
	title := theme.DashboardTitleStyle.Render(cl.Name) + "  " + theme.RenderStatus(cl.Status)

	line1 := label.Render("K8s: ") + cl.Version +
		label.Render("  Platform: ") + cl.PlatformVersion +
		label.Render("  Region: ") + v.region

	var line2 string
	if v.loading {
		line2 = label.Render("Nodes: ") + theme.MutedStyle.Render("connecting...")
	} else if v.err != nil && !v.k8sReady {
		line2 = label.Render("Nodes: ") + theme.ErrorStyle.Render(v.err.Error())
	} else {
		line2 = label.Render("Nodes: ") + v.nodeHealth
	}

	if v.k8sReady && v.nodeConditions != "" {
		line2 += "  " + v.nodeConditions
	}

	endpoint := "Private"
	if cl.EndpointPublic {
		endpoint = "Public"
	}
	if cl.EndpointPublic && cl.EndpointPrivate {
		endpoint = "Public + Private"
	}
	line3 := label.Render("Endpoint: ") + endpoint +
		label.Render("  Created: ") + utils.TimeOrDash(cl.CreatedAt, utils.DateOnly)

	boxStyle := theme.DashboardBoxStyle
	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4)
	}

	return boxStyle.Render(title + "\n" + line1 + "\n" + line2 + "\n" + line3)
}

// SetSize implements ResizableView.
func (v *EKSClusterDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.tabs.ResizeActive(v.width, v.contentHeight())
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

func NewEKSNodeGroupsTableView(client *awsclient.ServiceClient, clusterName string, k8sClient *awseks.K8sClient) View {
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
				theme.RenderStatus(ng.Status),
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
			if k8sClient != nil {
				return pushView(NewK8sNodesTableView(k8sClient, ng.Name))
			}
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
			return table.Row{a.Name, a.Version, theme.RenderStatus(a.Status), a.Health}
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
			return table.Row{fp.Name, theme.RenderStatus(fp.Status), strings.Join(sels, ", ")}
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

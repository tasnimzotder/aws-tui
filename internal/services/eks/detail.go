package eks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

type eksExecFinishedMsg struct{ err error }

// Detail data messages.
type clusterDetailMsg struct {
	cluster awseks.EKSCluster
	err     error
}

type nodeGroupsMsg struct {
	nodeGroups []awseks.EKSNodeGroup
	err        error
}

type addonsMsg struct {
	addons []awseks.EKSAddon
	err    error
}

type fargateProfilesMsg struct {
	profiles []awseks.EKSFargateProfile
	err      error
}

type accessEntriesMsg struct {
	entries []awseks.EKSAccessEntry
	err     error
}

// DetailView shows detailed information for a single EKS cluster.
type DetailView struct {
	client      *awseks.Client
	router      plugin.Router
	clusterName string
	cluster     *awseks.EKSCluster
	nodeGroups  []awseks.EKSNodeGroup
	addons      []awseks.EKSAddon
	fargate     []awseks.EKSFargateProfile
	access      []awseks.EKSAccessEntry
	tabs        ui.TabController
	loading     bool
	tabLoading  bool
	err         error
	tabErr      error
	lastTab     int
	region      string
	profile     string
}

// NewDetailView creates a DetailView for the given cluster name.
func NewDetailView(client *awseks.Client, router plugin.Router, clusterName, region, profile string) *DetailView {
	return &DetailView{
		client:      client,
		router:      router,
		clusterName: clusterName,
		tabs:        ui.NewTabController([]string{"Overview", "Node Groups", "Addons", "Fargate Profiles", "Access Entries"}),
		loading:     true,
		lastTab:     -1,
		region:      region,
		profile:     profile,
	}
}

func (dv *DetailView) loadCluster() tea.Cmd {
	client := dv.client
	name := dv.clusterName
	return func() tea.Msg {
		cluster, err := client.DescribeCluster(context.TODO(), name)
		return clusterDetailMsg{cluster: cluster, err: err}
	}
}

func (dv *DetailView) loadNodeGroups() tea.Cmd {
	client := dv.client
	name := dv.clusterName
	return func() tea.Msg {
		ngs, err := client.ListNodeGroups(context.TODO(), name)
		return nodeGroupsMsg{nodeGroups: ngs, err: err}
	}
}

func (dv *DetailView) loadAddons() tea.Cmd {
	client := dv.client
	name := dv.clusterName
	return func() tea.Msg {
		addons, err := client.ListAddons(context.TODO(), name)
		return addonsMsg{addons: addons, err: err}
	}
}

func (dv *DetailView) loadFargateProfiles() tea.Cmd {
	client := dv.client
	name := dv.clusterName
	return func() tea.Msg {
		profiles, err := client.ListFargateProfiles(context.TODO(), name)
		return fargateProfilesMsg{profiles: profiles, err: err}
	}
}

func (dv *DetailView) loadAccessEntries() tea.Cmd {
	client := dv.client
	name := dv.clusterName
	return func() tea.Msg {
		entries, err := client.ListAccessEntries(context.TODO(), name)
		return accessEntriesMsg{entries: entries, err: err}
	}
}

func (dv *DetailView) loadTabData() tea.Cmd {
	switch dv.tabs.Active() {
	case 1:
		return dv.loadNodeGroups()
	case 2:
		return dv.loadAddons()
	case 3:
		return dv.loadFargateProfiles()
	case 4:
		return dv.loadAccessEntries()
	default:
		return nil
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.loadCluster()
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clusterDetailMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.cluster = &msg.cluster
		dv.lastTab = 0
		dv.tabLoading = true
		return dv, dv.loadTabData()

	case nodeGroupsMsg:
		dv.tabLoading = false
		if msg.err != nil {
			dv.tabErr = msg.err
			return dv, nil
		}
		dv.nodeGroups = msg.nodeGroups
		return dv, nil

	case addonsMsg:
		dv.tabLoading = false
		if msg.err != nil {
			dv.tabErr = msg.err
			return dv, nil
		}
		dv.addons = msg.addons
		return dv, nil

	case fargateProfilesMsg:
		dv.tabLoading = false
		if msg.err != nil {
			dv.tabErr = msg.err
			return dv, nil
		}
		dv.fargate = msg.profiles
		return dv, nil

	case accessEntriesMsg:
		dv.tabLoading = false
		if msg.err != nil {
			dv.tabErr = msg.err
			return dv, nil
		}
		dv.access = msg.entries
		return dv, nil

	case eksExecFinishedMsg:
		if msg.err != nil {
			dv.router.Toast(plugin.ToastError, "kubectl failed: "+msg.err.Error())
		}
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		case "x":
			if dv.cluster != nil && dv.cluster.Status == "ACTIVE" {
				return dv, dv.execKubectl()
			}
			return dv, nil
		}
	}

	prevTab := dv.tabs.Active()
	var cmd tea.Cmd
	dv.tabs, cmd = dv.tabs.Update(msg)

	// If tab changed, fetch data for the new tab.
	if dv.tabs.Active() != prevTab && dv.cluster != nil {
		dv.lastTab = dv.tabs.Active()
		dv.tabLoading = true
		dv.tabErr = nil
		return dv, tea.Batch(cmd, dv.loadTabData())
	}

	return dv, cmd
}


func (dv *DetailView) execKubectl() tea.Cmd {
	// Update kubeconfig for this cluster, then open an interactive kubectl shell.
	args := []string{"eks", "update-kubeconfig", "--name", dv.clusterName}
	if dv.region != "" {
		args = append(args, "--region", dv.region)
	}
	if dv.profile != "" {
		args = append(args, "--profile", dv.profile)
	}
	// Chain: update kubeconfig then drop into interactive shell with kubectl available.
	c := exec.Command("sh", "-c",
		fmt.Sprintf("aws %s && echo 'Kubeconfig updated for %s. Use kubectl to interact.' && exec $SHELL",
			strings.Join(args, " "), dv.clusterName))
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return eksExecFinishedMsg{err: err}
	})
}

func (dv *DetailView) View() tea.View {
	if dv.loading {
		skel := ui.NewSkeleton(60, 8)
		return tea.NewView(skel.View())
	}
	if dv.err != nil {
		return tea.NewView("Error: " + dv.err.Error())
	}

	var b strings.Builder
	b.WriteString(dv.tabs.View())
	b.WriteString("\n\n")

	if dv.tabLoading {
		skel := ui.NewSkeleton(60, 4)
		b.WriteString(skel.View())
	} else if dv.tabErr != nil {
		b.WriteString("Error: " + dv.tabErr.Error())
	} else {
		switch dv.tabs.Active() {
		case 0:
			b.WriteString(dv.renderOverview())
		case 1:
			b.WriteString(dv.renderNodeGroups())
		case 2:
			b.WriteString(dv.renderAddons())
		case 3:
			b.WriteString(dv.renderFargateProfiles())
		case 4:
			b.WriteString(dv.renderAccessEntries())
		}
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderOverview() string {
	c := dv.cluster

	endpointAccess := "Public"
	if c.EndpointPrivate && c.EndpointPublic {
		endpointAccess = "Public + Private"
	} else if c.EndpointPrivate {
		endpointAccess = "Private"
	}

	rows := []ui.KV{
		{K: "Name", V: c.Name},
		{K: "ARN", V: c.ARN},
		{K: "Status", V: c.Status},
		{K: "Version", V: c.Version},
		{K: "Platform Version", V: c.PlatformVersion},
		{K: "Endpoint", V: c.Endpoint},
		{K: "Endpoint Access", V: endpointAccess},
		{K: "VPC ID", V: c.VPCID},
		{K: "Role ARN", V: c.RoleARN},
		{K: "Created At", V: c.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
	}
	return ui.RenderKV(rows, 22, 0)
}

func (dv *DetailView) renderNodeGroups() string {
	if len(dv.nodeGroups) == 0 {
		return "No node groups found."
	}

	cols := []ui.Column[awseks.EKSNodeGroup]{
		{Title: "Status", Width: 3, Field: func(ng awseks.EKSNodeGroup) string {
			return nodeGroupStatusDot(ng.Status)
		}},
		{Title: "Name", Width: 24, Field: func(ng awseks.EKSNodeGroup) string { return ng.Name }},
		{Title: "Instance Types", Width: 20, Field: func(ng awseks.EKSNodeGroup) string {
			return strings.Join(ng.InstanceTypes, ", ")
		}},
		{Title: "AMI Type", Width: 16, Field: func(ng awseks.EKSNodeGroup) string { return ng.AMIType }},
		{Title: "Desired", Width: 8, Field: func(ng awseks.EKSNodeGroup) string {
			return fmt.Sprintf("%d", ng.DesiredSize)
		}},
		{Title: "Min", Width: 5, Field: func(ng awseks.EKSNodeGroup) string {
			return fmt.Sprintf("%d", ng.MinSize)
		}},
		{Title: "Max", Width: 5, Field: func(ng awseks.EKSNodeGroup) string {
			return fmt.Sprintf("%d", ng.MaxSize)
		}},
	}

	tv := ui.NewTableView(cols, dv.nodeGroups, func(ng awseks.EKSNodeGroup) string { return ng.Name })
	return tv.View()
}

func nodeGroupStatusDot(status string) string {
	switch status {
	case "ACTIVE":
		return greenDot
	case "CREATING", "UPDATING":
		return yellowDot
	case "DELETING", "DELETE_FAILED", "DEGRADED":
		return redDot
	default:
		return grayDot
	}
}

func (dv *DetailView) renderAddons() string {
	if len(dv.addons) == 0 {
		return "No addons found."
	}

	cols := []ui.Column[awseks.EKSAddon]{
		{Title: "Status", Width: 3, Field: func(a awseks.EKSAddon) string {
			return addonStatusDot(a.Status)
		}},
		{Title: "Name", Width: 28, Field: func(a awseks.EKSAddon) string { return a.Name }},
		{Title: "Version", Width: 30, Field: func(a awseks.EKSAddon) string { return a.Version }},
		{Title: "Health", Width: 20, Field: func(a awseks.EKSAddon) string { return a.Health }},
	}

	tv := ui.NewTableView(cols, dv.addons, func(a awseks.EKSAddon) string { return a.Name })
	return tv.View()
}

func addonStatusDot(status string) string {
	switch status {
	case "ACTIVE":
		return greenDot
	case "CREATING", "UPDATING":
		return yellowDot
	case "DELETING", "DELETE_FAILED", "DEGRADED":
		return redDot
	default:
		return grayDot
	}
}

func (dv *DetailView) renderFargateProfiles() string {
	if len(dv.fargate) == 0 {
		return "No Fargate profiles found."
	}

	cols := []ui.Column[awseks.EKSFargateProfile]{
		{Title: "Status", Width: 3, Field: func(fp awseks.EKSFargateProfile) string {
			return fargateStatusDot(fp.Status)
		}},
		{Title: "Name", Width: 24, Field: func(fp awseks.EKSFargateProfile) string { return fp.Name }},
		{Title: "Selectors", Width: 40, Field: func(fp awseks.EKSFargateProfile) string {
			var parts []string
			for _, s := range fp.Selectors {
				parts = append(parts, s.Namespace)
			}
			return strings.Join(parts, ", ")
		}},
		{Title: "Subnets", Width: 12, Field: func(fp awseks.EKSFargateProfile) string {
			return fmt.Sprintf("%d", len(fp.Subnets))
		}},
	}

	tv := ui.NewTableView(cols, dv.fargate, func(fp awseks.EKSFargateProfile) string { return fp.Name })
	return tv.View()
}

func fargateStatusDot(status string) string {
	switch status {
	case "ACTIVE":
		return greenDot
	case "CREATING":
		return yellowDot
	case "DELETING", "DELETE_FAILED":
		return redDot
	default:
		return grayDot
	}
}

func (dv *DetailView) renderAccessEntries() string {
	if len(dv.access) == 0 {
		return "No access entries found."
	}

	cols := []ui.Column[awseks.EKSAccessEntry]{
		{Title: "Principal ARN", Width: 50, Field: func(ae awseks.EKSAccessEntry) string { return ae.PrincipalARN }},
		{Title: "Type", Width: 16, Field: func(ae awseks.EKSAccessEntry) string { return ae.Type }},
		{Title: "Username", Width: 24, Field: func(ae awseks.EKSAccessEntry) string { return ae.Username }},
		{Title: "Groups", Width: 30, Field: func(ae awseks.EKSAccessEntry) string {
			return strings.Join(ae.Groups, ", ")
		}},
	}

	tv := ui.NewTableView(cols, dv.access, func(ae awseks.EKSAccessEntry) string { return ae.PrincipalARN })
	return tv.View()
}

func (dv *DetailView) Title() string {
	if dv.cluster != nil {
		return dv.cluster.Name
	}
	return dv.clusterName
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	hints := []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "[/]", Desc: "switch tab"},
		{Key: "1-5", Desc: "jump to tab"},
	}
	if dv.cluster != nil && dv.cluster.Status == "ACTIVE" {
		hints = append(hints, plugin.KeyHint{Key: "x", Desc: "kubectl shell"})
	}
	return hints
}

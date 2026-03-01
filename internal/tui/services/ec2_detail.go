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
	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	awsvpc "tasnim.dev/aws-tui/internal/aws/vpc"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type ec2NavigateVPCMsg struct {
	vpc awsvpc.VPCInfo
}

type ec2NavigateVPCErrMsg struct {
	err error
}

// ---------------------------------------------------------------------------
// EC2 Detail View
// ---------------------------------------------------------------------------

type EC2DetailView struct {
	client   *awsclient.ServiceClient
	instance awsec2.EC2Instance
	profile  string
	region   string

	// Tabs
	activeTab int
	tabNames  []string
	tabViews  []View

	spinner spinner.Model
	width   int
	height  int
}

func NewEC2DetailView(client *awsclient.ServiceClient, instance awsec2.EC2Instance, profile, region string) *EC2DetailView {
	return &EC2DetailView{
		client:   client,
		instance: instance,
		profile:  profile,
		region:   region,
		tabNames: []string{"Details", "Security Groups", "Volumes", "Tags"},
		tabViews: make([]View, 4),
		spinner:  theme.NewSpinner(),
	}
}

func (v *EC2DetailView) Title() string {
	if v.instance.Name != "" {
		return v.instance.Name
	}
	return v.instance.InstanceID
}

func (v *EC2DetailView) HelpContext() *HelpContext {
	ctx := HelpContextEC2Detail
	return &ctx
}

func (v *EC2DetailView) Init() tea.Cmd {
	v.initTab(0)
	if v.tabViews[0] != nil {
		return v.tabViews[0].Init()
	}
	return nil
}

func (v *EC2DetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ec2NavigateVPCMsg:
		return v, pushView(NewVPCDetailView(v.client, msg.vpc))

	case ec2NavigateVPCErrMsg:
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
		case "x":
			return v, pushView(newSSMInputView(v.instance, v.profile, v.region))
		case "v":
			if v.instance.VpcID != "" {
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

func (v *EC2DetailView) navigateToVPC() tea.Cmd {
	client := v.client
	vpcID := v.instance.VpcID
	return func() tea.Msg {
		vpcs, err := client.VPC.ListVPCs(context.Background())
		if err != nil {
			return ec2NavigateVPCErrMsg{err: err}
		}
		for _, vpc := range vpcs {
			if vpc.VPCID == vpcID {
				return ec2NavigateVPCMsg{vpc: vpc}
			}
		}
		return ec2NavigateVPCErrMsg{err: fmt.Errorf("VPC %s not found", vpcID)}
	}
}

func (v *EC2DetailView) switchTab(idx int) tea.Cmd {
	v.activeTab = idx
	v.initTab(idx)
	v.resizeActiveTab()
	if v.tabViews[idx] != nil {
		return v.tabViews[idx].Init()
	}
	return nil
}

func (v *EC2DetailView) initTab(idx int) {
	if v.tabViews[idx] != nil {
		return
	}
	switch idx {
	case 0:
		v.tabViews[idx] = newEC2DetailsTab(v.instance)
	case 1:
		v.tabViews[idx] = newEC2SGTab(v.client, v.instance.SecurityGroups)
	case 2:
		v.tabViews[idx] = newEC2VolumesTab(v.client, v.instance.Volumes)
	case 3:
		v.tabViews[idx] = newEC2TagsTab(v.instance.Tags)
	}
}

func (v *EC2DetailView) resizeActiveTab() {
	if v.tabViews[v.activeTab] == nil {
		return
	}
	if rv, ok := v.tabViews[v.activeTab].(ResizableView); ok {
		rv.SetSize(v.width, v.contentHeight())
	}
}

func (v *EC2DetailView) contentHeight() int {
	h := v.height - 8 // dashboard (~6) + tab bar (2)
	if h < 3 {
		h = 3
	}
	return h
}

func (v *EC2DetailView) View() string {
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

func (v *EC2DetailView) renderDashboard() string {
	inst := v.instance

	stateStyle := theme.MutedStyle
	stateIcon := "●"
	if inst.State == "running" {
		stateStyle = theme.SuccessStyle
	} else if inst.State == "stopped" {
		stateStyle = lipgloss.NewStyle().Foreground(theme.Warning)
	}

	line1 := fmt.Sprintf("%s  %s  %s",
		inst.InstanceID,
		inst.Name,
		stateStyle.Render(stateIcon+" "+inst.State))

	line2 := fmt.Sprintf("Type: %s  AZ: %s  Arch: %s",
		inst.Type, inst.AZ, inst.Architecture)

	line3Parts := []string{}
	if inst.ImageID != "" {
		line3Parts = append(line3Parts, "AMI: "+inst.ImageID)
	}
	if inst.KeyName != "" {
		line3Parts = append(line3Parts, "Key: "+inst.KeyName)
	}
	if inst.IAMProfile != "" {
		line3Parts = append(line3Parts, "IAM: "+inst.IAMProfile)
	}
	line3 := strings.Join(line3Parts, "  ")

	line4Parts := []string{}
	if inst.VpcID != "" {
		line4Parts = append(line4Parts, "VPC: "+inst.VpcID)
	}
	if inst.SubnetID != "" {
		line4Parts = append(line4Parts, "Subnet: "+inst.SubnetID)
	}
	if !inst.LaunchTime.IsZero() {
		line4Parts = append(line4Parts, "Launch: "+inst.LaunchTime.Format("2006-01-02"))
	}
	line4 := strings.Join(line4Parts, "  ")

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
	if line4 != "" {
		content += "\n" + line4
	}

	return boxStyle.Render(content)
}

func (v *EC2DetailView) renderTabBar() string {
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

func (v *EC2DetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.resizeActiveTab()
}

// ---------------------------------------------------------------------------
// Tab 0: Details (viewport)
// ---------------------------------------------------------------------------

type ec2DetailsTab struct {
	instance      awsec2.EC2Instance
	viewport      viewport.Model
	vpReady       bool
	width, height int
}

func newEC2DetailsTab(inst awsec2.EC2Instance) *ec2DetailsTab {
	return &ec2DetailsTab{instance: inst}
}

func (v *ec2DetailsTab) Title() string { return "Details" }

func (v *ec2DetailsTab) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *ec2DetailsTab) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 10
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

func (v *ec2DetailsTab) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	inst := v.instance

	b.WriteString(bold.Render("Instance Properties"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	props := []struct{ k, v string }{
		{"Instance ID", inst.InstanceID},
		{"Name", inst.Name},
		{"Type", inst.Type},
		{"State", inst.State},
		{"Availability Zone", inst.AZ},
		{"Architecture", inst.Architecture},
		{"Platform", inst.Platform},
		{"AMI ID", inst.ImageID},
		{"Key Name", inst.KeyName},
		{"IAM Profile", inst.IAMProfile},
		{"VPC ID", inst.VpcID},
		{"Subnet ID", inst.SubnetID},
		{"Private IP", inst.PrivateIP},
		{"Public IP", inst.PublicIP},
	}

	if !inst.LaunchTime.IsZero() {
		props = append(props, struct{ k, v string }{"Launch Time", inst.LaunchTime.Format("2006-01-02 15:04:05 UTC")})
	}

	for _, p := range props {
		if p.v == "" {
			p.v = "—"
		}
		b.WriteString(fmt.Sprintf("  %-20s %s\n", p.k+":", p.v))
	}

	// Security groups summary
	if len(inst.SecurityGroups) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Security Groups"))
		b.WriteString("\n")
		for _, sg := range inst.SecurityGroups {
			b.WriteString(fmt.Sprintf("  %s (%s)\n", sg.GroupID, sg.GroupName))
		}
	}

	// Block devices summary
	if len(inst.Volumes) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Block Devices"))
		b.WriteString("\n")
		for _, vol := range inst.Volumes {
			del := "keep"
			if vol.DeleteOnTermination {
				del = "delete"
			}
			b.WriteString(fmt.Sprintf("  %s → %s (%s, %s)\n", vol.DeviceName, vol.VolumeID, vol.Status, del))
		}
	}

	return b.String()
}

func (v *ec2DetailsTab) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *ec2DetailsTab) View() string {
	if !v.vpReady {
		return ""
	}
	return v.viewport.View()
}

func (v *ec2DetailsTab) SetSize(width, height int) {
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
// Tab 1: Security Groups (table)
// ---------------------------------------------------------------------------

func newEC2SGTab(client *awsclient.ServiceClient, sgs []awsec2.EC2SecurityGroup) *TableView[awsec2.EC2SecurityGroup] {
	return NewTableView(TableViewConfig[awsec2.EC2SecurityGroup]{
		Title:       "Security Groups",
		LoadingText: "Loading security groups...",
		Columns: []table.Column{
			{Title: "Group ID", Width: 24},
			{Title: "Name", Width: 30},
		},
		FetchFunc: func(ctx context.Context) ([]awsec2.EC2SecurityGroup, error) {
			return sgs, nil
		},
		RowMapper: func(sg awsec2.EC2SecurityGroup) table.Row {
			return table.Row{sg.GroupID, sg.GroupName}
		},
		CopyIDFunc: func(sg awsec2.EC2SecurityGroup) string { return sg.GroupID },
		OnEnter: func(sg awsec2.EC2SecurityGroup) tea.Cmd {
			sgInfo := awsvpc.SecurityGroupInfo{
				GroupID: sg.GroupID,
				Name:    sg.GroupName,
			}
			return pushView(NewSecurityGroupDetailView(client, sgInfo))
		},
	})
}

// ---------------------------------------------------------------------------
// Tab 2: Volumes (table with async fetch)
// ---------------------------------------------------------------------------

func newEC2VolumesTab(client *awsclient.ServiceClient, blockDevices []awsec2.EC2BlockDevice) *TableView[awsec2.EBSVolume] {
	volumeIDs := make([]string, 0, len(blockDevices))
	for _, bd := range blockDevices {
		if bd.VolumeID != "" {
			volumeIDs = append(volumeIDs, bd.VolumeID)
		}
	}

	return NewTableView(TableViewConfig[awsec2.EBSVolume]{
		Title:       "Volumes",
		LoadingText: "Loading EBS volumes...",
		Columns: []table.Column{
			{Title: "Volume ID", Width: 24},
			{Title: "Size (GB)", Width: 10},
			{Title: "Type", Width: 8},
			{Title: "State", Width: 12},
			{Title: "IOPS", Width: 8},
			{Title: "Encrypted", Width: 10},
			{Title: "AZ", Width: 14},
		},
		FetchFunc: func(ctx context.Context) ([]awsec2.EBSVolume, error) {
			return client.EC2.GetInstanceVolumes(ctx, volumeIDs)
		},
		RowMapper: func(vol awsec2.EBSVolume) table.Row {
			enc := "No"
			if vol.Encrypted {
				enc = "Yes"
			}
			return table.Row{
				vol.VolumeID,
				fmt.Sprintf("%d", vol.Size),
				vol.VolumeType,
				vol.State,
				fmt.Sprintf("%d", vol.IOPS),
				enc,
				vol.AZ,
			}
		},
		CopyIDFunc: func(vol awsec2.EBSVolume) string { return vol.VolumeID },
	})
}

// ---------------------------------------------------------------------------
// Tab 3: Tags (viewport)
// ---------------------------------------------------------------------------

type ec2TagsTab struct {
	tags          map[string]string
	viewport      viewport.Model
	vpReady       bool
	width, height int
}

func newEC2TagsTab(tags map[string]string) *ec2TagsTab {
	return &ec2TagsTab{tags: tags}
}

func (v *ec2TagsTab) Title() string { return "Tags" }

func (v *ec2TagsTab) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *ec2TagsTab) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 10
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

func (v *ec2TagsTab) renderContent() string {
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

func (v *ec2TagsTab) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *ec2TagsTab) View() string {
	if !v.vpReady {
		return ""
	}
	return v.viewport.View()
}

func (v *ec2TagsTab) SetSize(width, height int) {
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

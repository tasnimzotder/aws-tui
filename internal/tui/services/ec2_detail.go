package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
// EC2 Detail View
// ---------------------------------------------------------------------------

type EC2DetailView struct {
	client   *awsclient.ServiceClient
	instance awsec2.EC2Instance
	profile  string
	region   string

	tabs   *TabController
	width  int
	height  int
}

func NewEC2DetailView(client *awsclient.ServiceClient, instance awsec2.EC2Instance, profile, region string) *EC2DetailView {
	v := &EC2DetailView{
		client:   client,
		instance: instance,
		profile:  profile,
		region:   region,
	}
	v.tabs = NewTabController(
		[]string{"Details", "Security Groups", "Volumes", "Tags"},
		v.createTab,
	)
	return v
}

func (v *EC2DetailView) createTab(idx int) View {
	switch idx {
	case 0:
		return newEC2DetailsTab(v.instance)
	case 1:
		return newEC2SGTab(v.client, v.instance.SecurityGroups)
	case 2:
		return newEC2VolumesTab(v.client, v.instance.Volumes)
	case 3:
		return newEC2TagsTab(v.instance.Tags)
	}
	return nil
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
	cmd := v.tabs.SwitchTab(0)
	v.tabs.ResizeActive(v.width, v.contentHeight())
	return cmd
}

func (v *EC2DetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case navigateVPCMsg:
		return v, pushView(NewVPCDetailView(v.client, msg.vpc))

	case navigateVPCErrMsg:
		return v, nil

	case tea.KeyPressMsg:
		key := msg.String()
		if handled, cmd := v.tabs.HandleKey(key); handled {
			v.tabs.ResizeActive(v.width, v.contentHeight())
			return v, cmd
		}
		switch key {
		case "x":
			return v, pushView(newSSMInputView(v.instance, v.profile, v.region))
		case "v":
			if v.instance.VpcID != "" {
				return v, NavigateToVPC(v.client.VPC, v.instance.VpcID)
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
	sections = append(sections, v.tabs.RenderTabBar())

	if av := v.tabs.ActiveView(); av != nil {
		sections = append(sections, av.View())
	} else {
		sections = append(sections, theme.MutedStyle.Render("No data"))
	}

	return strings.Join(sections, "\n")
}

func (v *EC2DetailView) renderDashboard() string {
	inst := v.instance
	label := theme.MutedStyle

	// Title: ID + Name + status badge
	title := theme.DashboardTitleStyle.Render(inst.InstanceID)
	if inst.Name != "" {
		title += "  " + inst.Name
	}
	title += "  " + theme.RenderStatus(inst.State)

	line1 := label.Render("Type: ") + inst.Type +
		label.Render("  AZ: ") + inst.AZ +
		label.Render("  Arch: ") + inst.Architecture

	line2Parts := []string{}
	if inst.ImageID != "" {
		line2Parts = append(line2Parts, label.Render("AMI: ")+inst.ImageID)
	}
	if inst.KeyName != "" {
		line2Parts = append(line2Parts, label.Render("Key: ")+inst.KeyName)
	}
	if inst.IAMProfile != "" {
		line2Parts = append(line2Parts, label.Render("IAM: ")+inst.IAMProfile)
	}
	line2 := strings.Join(line2Parts, "  ")

	line3Parts := []string{}
	if inst.VpcID != "" {
		line3Parts = append(line3Parts, label.Render("VPC: ")+inst.VpcID)
	}
	if inst.SubnetID != "" {
		line3Parts = append(line3Parts, label.Render("Subnet: ")+inst.SubnetID)
	}
	if !inst.LaunchTime.IsZero() {
		line3Parts = append(line3Parts, label.Render("Launch: ")+inst.LaunchTime.Format("2006-01-02"))
	}
	line3 := strings.Join(line3Parts, "  ")

	boxStyle := theme.DashboardBoxStyle
	if v.width > 0 {
		boxStyle = boxStyle.Width(v.width - 4)
	}

	content := title + "\n" + line1
	if line2 != "" {
		content += "\n" + line2
	}
	if line3 != "" {
		content += "\n" + line3
	}

	return boxStyle.Render(content)
}

func (v *EC2DetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.tabs.ResizeActive(v.width, v.contentHeight())
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
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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

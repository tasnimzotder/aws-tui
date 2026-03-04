package ec2

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awsec2 "tasnim.dev/aws-tui/internal/aws/ec2"
	"tasnim.dev/aws-tui/internal/plugin"
	"tasnim.dev/aws-tui/internal/ui"
)

type execFinishedMsg struct{ err error }

// detailLoadedMsg carries the result of loading instance detail.
type detailLoadedMsg struct {
	instance awsec2.EC2Instance
	volumes  []awsec2.EBSVolume
	err      error
}

// DetailView shows detailed information for a single EC2 instance.
type DetailView struct {
	client     EC2Client
	router     plugin.Router
	instanceID string
	instance   *awsec2.EC2Instance
	volumes    []awsec2.EBSVolume
	tabs       ui.TabController
	loading    bool
	err        error
	width      int
	region     string
	profile    string
}

// NewDetailView creates a DetailView for the given instance ID.
func NewDetailView(client EC2Client, router plugin.Router, instanceID, region, profile string) *DetailView {
	return &DetailView{
		client:     client,
		router:     router,
		instanceID: instanceID,
		tabs:       ui.NewTabController([]string{"Overview", "Security Groups", "Volumes", "Tags"}),
		loading:    true,
		region:     region,
		profile:    profile,
	}
}

func (dv *DetailView) loadInstance() tea.Cmd {
	client := dv.client
	instanceID := dv.instanceID
	return func() tea.Msg {
		ctx := context.TODO()
		instances, _, err := client.ListInstances(ctx)
		if err != nil {
			return detailLoadedMsg{err: err}
		}

		var found *awsec2.EC2Instance
		for _, inst := range instances {
			if inst.InstanceID == instanceID {
				found = &inst
				break
			}
		}
		if found == nil {
			return detailLoadedMsg{err: fmt.Errorf("instance %s not found", instanceID)}
		}

		volumes := loadVolumes(ctx, client, found.Volumes)
		return detailLoadedMsg{instance: *found, volumes: volumes}
	}
}

func loadVolumes(ctx context.Context, client EC2Client, blockDevices []awsec2.EC2BlockDevice) []awsec2.EBSVolume {
	var volIDs []string
	for _, bd := range blockDevices {
		if bd.VolumeID != "" {
			volIDs = append(volIDs, bd.VolumeID)
		}
	}
	if len(volIDs) == 0 {
		return nil
	}
	vols, err := client.GetInstanceVolumes(ctx, volIDs)
	if err != nil {
		return nil
	}
	return vols
}

func (dv *DetailView) Init() tea.Cmd {
	return dv.loadInstance()
}

func (dv *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detailLoadedMsg:
		dv.loading = false
		if msg.err != nil {
			dv.err = msg.err
			return dv, nil
		}
		dv.instance = &msg.instance
		dv.volumes = msg.volumes
		return dv, nil

	case tea.WindowSizeMsg:
		dv.width = msg.Width
		return dv, nil

	case execFinishedMsg:
		if msg.err != nil {
			dv.router.Toast(plugin.ToastError, "SSM session failed: "+msg.err.Error())
		}
		return dv, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			dv.router.Pop()
			return dv, nil
		case "x":
			if dv.instance != nil && dv.instance.State == "running" {
				return dv, dv.execSSM()
			}
			return dv, nil
		}
	}

	var cmd tea.Cmd
	dv.tabs, cmd = dv.tabs.Update(msg)
	return dv, cmd
}

func (dv *DetailView) execSSM() tea.Cmd {
	args := []string{"ssm", "start-session", "--target", dv.instance.InstanceID}
	if dv.region != "" {
		args = append(args, "--region", dv.region)
	}
	if dv.profile != "" {
		args = append(args, "--profile", dv.profile)
	}
	c := exec.Command("aws", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execFinishedMsg{err: err}
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

	switch dv.tabs.Active() {
	case 0:
		b.WriteString(dv.renderOverview())
	case 1:
		b.WriteString(dv.renderSecurityGroups())
	case 2:
		b.WriteString(dv.renderVolumes())
	case 3:
		b.WriteString(dv.renderTags())
	}

	return tea.NewView(b.String())
}

func (dv *DetailView) renderOverview() string {
	inst := dv.instance
	rows := []ui.KV{
		{K: "Instance ID", V: inst.InstanceID},
		{K: "Name", V: inst.Name},
		{K: "State", V: inst.State},
		{K: "Type", V: inst.Type},
		{K: "AZ", V: inst.AZ},
		{K: "Platform", V: inst.Platform},
		{K: "Architecture", V: inst.Architecture},
		{K: "AMI", V: inst.ImageID},
		{K: "Key Name", V: inst.KeyName},
		{K: "IAM Profile", V: inst.IAMProfile},
		{K: "VPC", V: inst.VpcID},
		{K: "Subnet", V: inst.SubnetID},
		{K: "Private IP", V: inst.PrivateIP},
		{K: "Public IP", V: inst.PublicIP},
		{K: "Launch Time", V: inst.LaunchTime.Format("2006-01-02 15:04:05 UTC")},
	}
	valWidth := dv.width - 22
	if valWidth < 40 {
		valWidth = 40
	}
	return ui.RenderKV(rows, 20, valWidth)
}

func (dv *DetailView) renderSecurityGroups() string {
	inst := dv.instance
	if len(inst.SecurityGroups) == 0 {
		return "No security groups attached."
	}

	var b strings.Builder
	header := fmt.Sprintf("%-24s %s", "Group ID", "Name")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(header))
	b.WriteString("\n")
	for _, sg := range inst.SecurityGroups {
		b.WriteString(fmt.Sprintf("%-24s %s\n", sg.GroupID, sg.GroupName))
	}
	return b.String()
}

func (dv *DetailView) renderVolumes() string {
	if len(dv.volumes) == 0 {
		return "No volumes found."
	}

	var b strings.Builder
	header := fmt.Sprintf("%-24s %-8s %-10s %-12s %-8s %s", "Volume ID", "Size", "Type", "State", "IOPS", "Encrypted")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(header))
	b.WriteString("\n")
	for _, v := range dv.volumes {
		enc := "no"
		if v.Encrypted {
			enc = "yes"
		}
		b.WriteString(fmt.Sprintf("%-24s %-8s %-10s %-12s %-8d %s\n",
			v.VolumeID,
			fmt.Sprintf("%dGB", v.Size),
			v.VolumeType,
			v.State,
			v.IOPS,
			enc,
		))
	}
	return b.String()
}

func (dv *DetailView) renderTags() string {
	inst := dv.instance
	if len(inst.Tags) == 0 {
		return "No tags."
	}

	keys := make([]string, 0, len(inst.Tags))
	for k := range inst.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	valWidth := dv.width - 22
	if valWidth < 40 {
		valWidth = 40
	}
	var rows []ui.KV
	for _, k := range keys {
		rows = append(rows, ui.KV{K: k, V: inst.Tags[k]})
	}
	return ui.RenderKV(rows, 20, valWidth)
}

func (dv *DetailView) Title() string {
	if dv.instance != nil && dv.instance.Name != "" {
		return dv.instance.Name
	}
	return dv.instanceID
}

func (dv *DetailView) KeyHints() []plugin.KeyHint {
	hints := []plugin.KeyHint{
		{Key: "esc", Desc: "back"},
		{Key: "[/]", Desc: "switch tab"},
		{Key: "1-4", Desc: "jump to tab"},
	}
	if dv.instance != nil && dv.instance.State == "running" {
		hints = append(hints, plugin.KeyHint{Key: "x", Desc: "SSM session"})
	}
	return hints
}

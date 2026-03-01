package services

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// eksStatusDot returns a colored status dot based on an EKS resource status string.
func eksStatusDot(status string) string {
	switch strings.ToUpper(status) {
	case "ACTIVE":
		return lipgloss.NewStyle().Foreground(theme.Success).Render("●")
	case "UPDATING", "CREATING":
		return lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
	case "FAILED", "DELETING", "DELETE_FAILED", "CREATE_FAILED":
		return lipgloss.NewStyle().Foreground(theme.Error).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("●")
	}
}

// ---------------------------------------------------------------------------
// Node Group Detail
// ---------------------------------------------------------------------------

// EKSNodeGroupDetailView shows detailed info for a single EKS node group.
type EKSNodeGroupDetailView struct {
	ng            awseks.EKSNodeGroup
	width, height int
	viewport      viewport.Model
	vpReady       bool
}

// NewEKSNodeGroupDetailView creates a detail view for the given node group.
func NewEKSNodeGroupDetailView(ng awseks.EKSNodeGroup) *EKSNodeGroupDetailView {
	return &EKSNodeGroupDetailView{ng: ng}
}

func (v *EKSNodeGroupDetailView) Title() string { return "Node Group: " + v.ng.Name }

func (v *EKSNodeGroupDetailView) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *EKSNodeGroupDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *EKSNodeGroupDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	ng := v.ng

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("Node Group: %s", ng.Name)))
	b.WriteString(fmt.Sprintf("   Status: %s %s\n", eksStatusDot(ng.Status), ng.Status))
	b.WriteString(strings.Repeat("─", 45))
	b.WriteString("\n")

	// Instance types
	b.WriteString(fmt.Sprintf("Instance Types: %s\n", strings.Join(ng.InstanceTypes, ", ")))
	b.WriteString(fmt.Sprintf("AMI Type: %s\n", ng.AMIType))
	if ng.LaunchTemplate != "" {
		b.WriteString(fmt.Sprintf("Launch Template: %s\n", ng.LaunchTemplate))
	}

	// Scaling
	b.WriteString("\n")
	b.WriteString(bold.Render("Scaling:"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Min: %d  Max: %d  Desired: %d\n", ng.MinSize, ng.MaxSize, ng.DesiredSize))

	// Labels
	if len(ng.Labels) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Labels:"))
		b.WriteString("\n")
		keys := make([]string, 0, len(ng.Labels))
		for k := range ng.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, ng.Labels[k]))
		}
	}

	// Taints
	if len(ng.Taints) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Taints:"))
		b.WriteString("\n")
		for _, t := range ng.Taints {
			b.WriteString(fmt.Sprintf("  %s=%s:%s\n", t.Key, t.Value, t.Effect))
		}
	}

	// Subnets
	if len(ng.Subnets) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Subnets:"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s\n", strings.Join(ng.Subnets, ", ")))
	}

	return b.String()
}

func (v *EKSNodeGroupDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *EKSNodeGroupDetailView) View() string {
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *EKSNodeGroupDetailView) SetSize(width, height int) {
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
// Add-on Detail
// ---------------------------------------------------------------------------

// EKSAddonDetailView shows detailed info for a single EKS add-on.
type EKSAddonDetailView struct {
	addon         awseks.EKSAddon
	width, height int
	viewport      viewport.Model
	vpReady       bool
}

// NewEKSAddonDetailView creates a detail view for the given add-on.
func NewEKSAddonDetailView(addon awseks.EKSAddon) *EKSAddonDetailView {
	return &EKSAddonDetailView{addon: addon}
}

func (v *EKSAddonDetailView) Title() string { return "Add-on: " + v.addon.Name }

func (v *EKSAddonDetailView) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *EKSAddonDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *EKSAddonDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	a := v.addon

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("Add-on: %s", a.Name)))
	b.WriteString(fmt.Sprintf("   Status: %s %s\n", eksStatusDot(a.Status), a.Status))
	b.WriteString(strings.Repeat("─", 45))
	b.WriteString("\n")

	// Version
	b.WriteString(fmt.Sprintf("Version: %s\n", a.Version))

	// Health
	health := a.Health
	if health == "" {
		health = "(healthy)"
	}
	b.WriteString(fmt.Sprintf("Health: %s\n", health))

	// Service account role
	if a.ServiceAccountRole != "" {
		b.WriteString(fmt.Sprintf("Service Account Role: %s\n", a.ServiceAccountRole))
	}

	// Configuration
	if a.ConfigurationValues != "" {
		b.WriteString("\n")
		b.WriteString(bold.Render("Configuration:"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s\n", a.ConfigurationValues))
	}

	return b.String()
}

func (v *EKSAddonDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *EKSAddonDetailView) View() string {
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *EKSAddonDetailView) SetSize(width, height int) {
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
// Fargate Profile Detail
// ---------------------------------------------------------------------------

// EKSFargateDetailView shows detailed info for a single EKS Fargate profile.
type EKSFargateDetailView struct {
	profile       awseks.EKSFargateProfile
	width, height int
	viewport      viewport.Model
	vpReady       bool
}

// NewEKSFargateDetailView creates a detail view for the given Fargate profile.
func NewEKSFargateDetailView(profile awseks.EKSFargateProfile) *EKSFargateDetailView {
	return &EKSFargateDetailView{profile: profile}
}

func (v *EKSFargateDetailView) Title() string { return "Fargate Profile: " + v.profile.Name }

func (v *EKSFargateDetailView) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *EKSFargateDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *EKSFargateDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	fp := v.profile

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("Fargate Profile: %s", fp.Name)))
	b.WriteString(fmt.Sprintf("   Status: %s %s\n", eksStatusDot(fp.Status), fp.Status))
	b.WriteString(strings.Repeat("─", 45))
	b.WriteString("\n")

	// Pod execution role
	if fp.PodExecutionRole != "" {
		b.WriteString(fmt.Sprintf("Pod Execution Role: %s\n", fp.PodExecutionRole))
	}

	// Selectors
	if len(fp.Selectors) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Selectors:"))
		b.WriteString("\n")
		for i, sel := range fp.Selectors {
			b.WriteString(fmt.Sprintf("  %d. Namespace: %s\n", i+1, sel.Namespace))
			if len(sel.Labels) > 0 {
				keys := make([]string, 0, len(sel.Labels))
				for k := range sel.Labels {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				var pairs []string
				for _, k := range keys {
					pairs = append(pairs, fmt.Sprintf("%s=%s", k, sel.Labels[k]))
				}
				b.WriteString(fmt.Sprintf("     Labels: %s\n", strings.Join(pairs, ", ")))
			}
		}
	}

	// Subnets
	if len(fp.Subnets) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Subnets:"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s\n", strings.Join(fp.Subnets, ", ")))
	}

	return b.String()
}

func (v *EKSFargateDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *EKSFargateDetailView) View() string {
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *EKSFargateDetailView) SetSize(width, height int) {
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
// Access Entry Detail
// ---------------------------------------------------------------------------

// EKSAccessEntryDetailView shows detailed info for a single EKS access entry.
type EKSAccessEntryDetailView struct {
	entry         awseks.EKSAccessEntry
	width, height int
	viewport      viewport.Model
	vpReady       bool
}

// NewEKSAccessEntryDetailView creates a detail view for the given access entry.
func NewEKSAccessEntryDetailView(entry awseks.EKSAccessEntry) *EKSAccessEntryDetailView {
	return &EKSAccessEntryDetailView{entry: entry}
}

func (v *EKSAccessEntryDetailView) Title() string { return "Access Entry: " + v.entry.PrincipalARN }

func (v *EKSAccessEntryDetailView) Init() tea.Cmd {
	v.initViewport()
	return nil
}

func (v *EKSAccessEntryDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *EKSAccessEntryDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	ae := v.entry

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("Access Entry: %s", ae.PrincipalARN)))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 45))
	b.WriteString("\n")

	// Type
	b.WriteString(fmt.Sprintf("Type: %s\n", ae.Type))

	// Username
	if ae.Username != "" {
		b.WriteString(fmt.Sprintf("Username: %s\n", ae.Username))
	}

	// Created
	b.WriteString(fmt.Sprintf("Created: %s\n", utils.TimeOrDash(ae.CreatedAt, utils.DateOnly)))

	// Kubernetes Groups
	if len(ae.Groups) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Kubernetes Groups:"))
		b.WriteString("\n")
		for _, g := range ae.Groups {
			b.WriteString(fmt.Sprintf("  %s\n", g))
		}
	}

	return b.String()
}

func (v *EKSAccessEntryDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
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

func (v *EKSAccessEntryDetailView) View() string {
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *EKSAccessEntryDetailView) SetSize(width, height int) {
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

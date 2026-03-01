package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

type k8sNodeDetailMsg struct {
	conditions  []NodeCondition
	capacity    map[string]string
	allocatable map[string]string
	labels      map[string]string
	taints      []string
	addresses   []string
	systemInfo  map[string]string
}

type NodeCondition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

type K8sNodeDetailView struct {
	k8s  *awseks.K8sClient
	node K8sNode

	conditions  []NodeCondition
	capacity    map[string]string
	allocatable map[string]string
	labels      map[string]string
	taints      []string
	addresses   []string
	systemInfo  map[string]string
	loaded      bool
	err         error

	width, height int
	viewport      viewport.Model
	vpReady       bool
}

func NewK8sNodeDetailView(k8s *awseks.K8sClient, node K8sNode) *K8sNodeDetailView {
	return &K8sNodeDetailView{
		k8s:  k8s,
		node: node,
	}
}

func (v *K8sNodeDetailView) Title() string { return "Node: " + v.node.Name }

func (v *K8sNodeDetailView) Init() tea.Cmd {
	return v.fetchNodeDetail()
}

func (v *K8sNodeDetailView) fetchNodeDetail() tea.Cmd {
	k8s := v.k8s
	nodeName := v.node.Name
	return func() tea.Msg {
		ctx := context.Background()

		n, err := k8s.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		// Conditions
		conditions := make([]NodeCondition, 0, len(n.Status.Conditions))
		for _, c := range n.Status.Conditions {
			conditions = append(conditions, NodeCondition{
				Type:    string(c.Type),
				Status:  string(c.Status),
				Reason:  c.Reason,
				Message: c.Message,
			})
		}

		// Capacity & Allocatable
		capacity := make(map[string]string)
		for k, v := range n.Status.Capacity {
			capacity[string(k)] = v.String()
		}
		allocatable := make(map[string]string)
		for k, v := range n.Status.Allocatable {
			allocatable[string(k)] = v.String()
		}

		// Labels
		labels := make(map[string]string, len(n.Labels))
		for k, v := range n.Labels {
			labels[k] = v
		}

		// Taints
		taints := make([]string, 0, len(n.Spec.Taints))
		for _, t := range n.Spec.Taints {
			taints = append(taints, fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect))
		}

		// Addresses
		addresses := make([]string, 0, len(n.Status.Addresses))
		for _, a := range n.Status.Addresses {
			addresses = append(addresses, fmt.Sprintf("%s: %s", a.Type, a.Address))
		}

		// System info
		info := n.Status.NodeInfo
		systemInfo := map[string]string{
			"OS":                info.OperatingSystem + "/" + info.Architecture,
			"OS Image":         info.OSImage,
			"Kernel":           info.KernelVersion,
			"Container Runtime": info.ContainerRuntimeVersion,
			"Kubelet":          info.KubeletVersion,
			"Kube-Proxy":       info.KubeProxyVersion,
		}

		return k8sNodeDetailMsg{
			conditions:  conditions,
			capacity:    capacity,
			allocatable: allocatable,
			labels:      labels,
			taints:      taints,
			addresses:   addresses,
			systemInfo:  systemInfo,
		}
	}
}

func (v *K8sNodeDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case k8sNodeDetailMsg:
		v.conditions = msg.conditions
		v.capacity = msg.capacity
		v.allocatable = msg.allocatable
		v.labels = msg.labels
		v.taints = msg.taints
		v.addresses = msg.addresses
		v.systemInfo = msg.systemInfo
		v.loaded = true
		v.initViewport()
		return v, nil

	case k8sDetailErrorMsg:
		v.err = msg.err
		v.loaded = true
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
	}

	if v.vpReady {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *K8sNodeDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *K8sNodeDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	muted := theme.MutedStyle

	// Header
	statusDot := v.statusDot(v.node.Status)
	b.WriteString(bold.Render(fmt.Sprintf("Node: %s", v.node.Name)))
	b.WriteString(fmt.Sprintf("   Status: %s %s\n", statusDot, v.node.Status))
	b.WriteString(fmt.Sprintf("Roles: %s  Version: %s  Instance: %s\n",
		v.node.Roles, v.node.Version, v.node.InstanceType))

	// Addresses
	if len(v.addresses) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Addresses:"))
		b.WriteString("\n")
		for _, a := range v.addresses {
			b.WriteString(fmt.Sprintf(" %s\n", a))
		}
	}

	// Conditions
	if len(v.conditions) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Conditions:"))
		b.WriteString("\n")
		b.WriteString(muted.Render(fmt.Sprintf(" %-20s %-8s %-20s %s", "TYPE", "STATUS", "REASON", "MESSAGE")))
		b.WriteString("\n")
		for _, c := range v.conditions {
			b.WriteString(fmt.Sprintf(" %-20s %-8s %-20s %s\n", c.Type, c.Status, c.Reason, c.Message))
		}
	}

	// Capacity vs Allocatable
	if len(v.capacity) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Resources (Capacity / Allocatable):"))
		b.WriteString("\n")
		keys := make([]string, 0, len(v.capacity))
		for k := range v.capacity {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			alloc := v.allocatable[k]
			if alloc == "" {
				alloc = "-"
			}
			b.WriteString(fmt.Sprintf(" %-28s %s / %s\n", k, v.capacity[k], alloc))
		}
	}

	// System Info
	if len(v.systemInfo) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("System Info:"))
		b.WriteString("\n")
		infoKeys := []string{"OS", "OS Image", "Kernel", "Container Runtime", "Kubelet", "Kube-Proxy"}
		for _, k := range infoKeys {
			if val, ok := v.systemInfo[k]; ok {
				b.WriteString(fmt.Sprintf(" %-22s %s\n", k+":", val))
			}
		}
	}

	// Taints
	b.WriteString("\n")
	b.WriteString(bold.Render("Taints:"))
	b.WriteString("\n")
	if len(v.taints) > 0 {
		for _, t := range v.taints {
			b.WriteString(fmt.Sprintf(" %s\n", t))
		}
	} else {
		b.WriteString(muted.Render(" <none>") + "\n")
	}

	// Labels
	if len(v.labels) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Labels:"))
		b.WriteString("\n")
		keys := make([]string, 0, len(v.labels))
		for k := range v.labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf(" %s=%s\n", k, v.labels[k]))
		}
	}

	return b.String()
}

func (v *K8sNodeDetailView) statusDot(status string) string {
	switch strings.ToLower(status) {
	case "ready":
		return lipgloss.NewStyle().Foreground(theme.Success).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(theme.Error).Render("●")
	}
}

func (v *K8sNodeDetailView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.loaded {
		return theme.MutedStyle.Render("Loading node details...")
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *K8sNodeDetailView) SetSize(width, height int) {
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

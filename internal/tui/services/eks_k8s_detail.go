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

// ---------------------------------------------------------------------------
// Shared messages for K8s detail views
// ---------------------------------------------------------------------------

type k8sPodDetailMsg struct {
	containers []PodContainerDetail
	conditions []PodCondition
	events     []PodEvent
	node       string
	podIP      string
}

type k8sServiceDetailMsg struct {
	endpoints []ServiceEndpoint
	selector  string
	clusterIP string
	externalIP string
	svcType   string
}

type k8sDeploymentDetailMsg struct {
	revisions   []DeploymentRevision
	strategy    string
	maxSurge    string
	maxUnavail  string
}

type k8sDetailErrorMsg struct {
	err error
}

// ---------------------------------------------------------------------------
// Pod Detail
// ---------------------------------------------------------------------------

type PodContainerDetail struct {
	Name     string
	Image    string
	Status   string
	Restarts int
}

type PodCondition struct {
	Type   string
	Status string
}

type PodEvent struct {
	Reason  string
	Message string
	Age     string
}

type K8sPodDetailView struct {
	k8s  *awseks.K8sClient
	pod  K8sPod

	containers []PodContainerDetail
	conditions []PodCondition
	events     []PodEvent
	node       string
	podIP      string
	loaded     bool
	err        error

	width, height int
	viewport      viewport.Model
	vpReady       bool
}

func NewK8sPodDetailView(k8s *awseks.K8sClient, pod K8sPod) *K8sPodDetailView {
	return &K8sPodDetailView{
		k8s: k8s,
		pod: pod,
	}
}

func (v *K8sPodDetailView) Title() string { return "Pod: " + v.pod.Name }

func (v *K8sPodDetailView) Init() tea.Cmd {
	return v.fetchPodDetail()
}

func (v *K8sPodDetailView) fetchPodDetail() tea.Cmd {
	k8s := v.k8s
	pod := v.pod
	return func() tea.Msg {
		ctx := context.Background()

		// Fetch pod details
		p, err := k8s.Clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		// Build container details
		containers := make([]PodContainerDetail, 0, len(p.Spec.Containers))
		for _, c := range p.Spec.Containers {
			status := "Waiting"
			restarts := 0
			for _, cs := range p.Status.ContainerStatuses {
				if cs.Name == c.Name {
					restarts = int(cs.RestartCount)
					if cs.State.Running != nil {
						status = "Running"
					} else if cs.State.Terminated != nil {
						status = cs.State.Terminated.Reason
						if status == "" {
							status = "Terminated"
						}
					} else if cs.State.Waiting != nil {
						status = cs.State.Waiting.Reason
						if status == "" {
							status = "Waiting"
						}
					}
					break
				}
			}
			containers = append(containers, PodContainerDetail{
				Name:     c.Name,
				Image:    c.Image,
				Status:   status,
				Restarts: restarts,
			})
		}

		// Build conditions
		conditions := make([]PodCondition, 0, len(p.Status.Conditions))
		for _, c := range p.Status.Conditions {
			conditions = append(conditions, PodCondition{
				Type:   string(c.Type),
				Status: string(c.Status),
			})
		}

		// Fetch events
		events, err := k8s.Clientset.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + pod.Name,
		})
		var podEvents []PodEvent
		if err == nil {
			for _, e := range events.Items {
				podEvents = append(podEvents, PodEvent{
					Reason:  e.Reason,
					Message: e.Message,
					Age:     formatAge(e.LastTimestamp.Time),
				})
			}
		}

		return k8sPodDetailMsg{
			containers: containers,
			conditions: conditions,
			events:     podEvents,
			node:       p.Spec.NodeName,
			podIP:      p.Status.PodIP,
		}
	}
}

func (v *K8sPodDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case k8sPodDetailMsg:
		v.containers = msg.containers
		v.conditions = msg.conditions
		v.events = msg.events
		v.node = msg.node
		v.podIP = msg.podIP
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

func (v *K8sPodDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
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

func (v *K8sPodDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	muted := theme.MutedStyle

	// Header
	statusDot := v.statusDot(v.pod.Status)
	b.WriteString(bold.Render(fmt.Sprintf("Pod: %s", v.pod.Name)))
	b.WriteString(fmt.Sprintf("   Status: %s %s\n", statusDot, v.pod.Status))
	b.WriteString(fmt.Sprintf("Namespace: %s  Node: %s  IP: %s\n", v.pod.Namespace, v.node, v.podIP))

	// Containers
	b.WriteString("\n")
	b.WriteString(bold.Render("Containers:"))
	b.WriteString("\n")
	b.WriteString(muted.Render(fmt.Sprintf(" %-20s %-30s %-12s %s", "NAME", "IMAGE", "STATUS", "RESTARTS")))
	b.WriteString("\n")
	for _, c := range v.containers {
		b.WriteString(fmt.Sprintf(" %-20s %-30s %-12s %d\n", c.Name, c.Image, c.Status, c.Restarts))
	}

	// Conditions
	b.WriteString("\n")
	b.WriteString(bold.Render("Conditions:"))
	b.WriteString("\n ")
	for i, c := range v.conditions {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(fmt.Sprintf("%s: %s", c.Type, c.Status))
	}
	b.WriteString("\n")

	// Events
	if len(v.events) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Recent Events:"))
		b.WriteString("\n")
		for _, e := range v.events {
			b.WriteString(fmt.Sprintf(" %-10s %-50s %s\n", e.Reason, e.Message, e.Age))
		}
	}

	return b.String()
}

func (v *K8sPodDetailView) statusDot(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return lipgloss.NewStyle().Foreground(theme.Success).Render("●")
	case "pending":
		return lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
	case "succeeded":
		return lipgloss.NewStyle().Foreground(theme.Primary).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(theme.Error).Render("●")
	}
}

func (v *K8sPodDetailView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.loaded {
		return theme.MutedStyle.Render("Loading pod details...")
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *K8sPodDetailView) SetSize(width, height int) {
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
// Service Detail
// ---------------------------------------------------------------------------

type ServiceEndpoint struct {
	IP      string
	Port    int32
	PodName string
}

type K8sServiceDetailView struct {
	k8s     *awseks.K8sClient
	service K8sService

	endpoints  []ServiceEndpoint
	loaded     bool
	err        error

	width, height int
	viewport      viewport.Model
	vpReady       bool
}

func NewK8sServiceDetailView(k8s *awseks.K8sClient, service K8sService) *K8sServiceDetailView {
	return &K8sServiceDetailView{
		k8s:     k8s,
		service: service,
	}
}

func (v *K8sServiceDetailView) Title() string { return "Service: " + v.service.Name }

func (v *K8sServiceDetailView) Init() tea.Cmd {
	return v.fetchServiceDetail()
}

func (v *K8sServiceDetailView) fetchServiceDetail() tea.Cmd {
	k8s := v.k8s
	svc := v.service
	return func() tea.Msg {
		ctx := context.Background()

		// Fetch endpoints
		ep, err := k8s.Clientset.CoreV1().Endpoints(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		var endpoints []ServiceEndpoint
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				podName := ""
				if addr.TargetRef != nil {
					podName = addr.TargetRef.Name
				}
				for _, port := range subset.Ports {
					endpoints = append(endpoints, ServiceEndpoint{
						IP:      addr.IP,
						Port:    port.Port,
						PodName: podName,
					})
				}
			}
		}

		return k8sServiceDetailMsg{
			endpoints: endpoints,
		}
	}
}

func (v *K8sServiceDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case k8sServiceDetailMsg:
		v.endpoints = msg.endpoints
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

func (v *K8sServiceDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
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

func (v *K8sServiceDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("Service: %s", v.service.Name)))
	b.WriteString(fmt.Sprintf("   Type: %s\n", v.service.Type))
	b.WriteString(fmt.Sprintf("Namespace: %s\n", v.service.Namespace))

	// Selector
	if len(v.service.Selector) > 0 {
		var parts []string
		for k, val := range v.service.Selector {
			parts = append(parts, k+"="+val)
		}
		b.WriteString(fmt.Sprintf("Selector: %s\n", strings.Join(parts, ", ")))
	}

	b.WriteString(fmt.Sprintf("Cluster IP: %s\n", v.service.ClusterIP))
	if v.service.ExternalIP != "<none>" && v.service.ExternalIP != "" {
		b.WriteString(fmt.Sprintf("External: %s\n", v.service.ExternalIP))
	}

	if v.service.Ports != "" {
		b.WriteString(fmt.Sprintf("Ports: %s\n", v.service.Ports))
	}

	// Endpoints
	if len(v.endpoints) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Endpoints:"))
		b.WriteString("\n")
		for _, ep := range v.endpoints {
			line := fmt.Sprintf(" %s:%d", ep.IP, ep.Port)
			if ep.PodName != "" {
				line += fmt.Sprintf("   (%s)", ep.PodName)
			}
			b.WriteString(line + "\n")
		}
	} else {
		b.WriteString("\n")
		b.WriteString(theme.MutedStyle.Render("No endpoints"))
		b.WriteString("\n")
	}

	return b.String()
}

func (v *K8sServiceDetailView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.loaded {
		return theme.MutedStyle.Render("Loading service details...")
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *K8sServiceDetailView) SetSize(width, height int) {
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
// Deployment Detail
// ---------------------------------------------------------------------------

type DeploymentRevision struct {
	Revision    string
	Image       string
	ChangeCause string
}

type K8sDeploymentDetailView struct {
	k8s        *awseks.K8sClient
	deployment K8sDeployment

	revisions   []DeploymentRevision
	strategy    string
	maxSurge    string
	maxUnavail  string
	loaded      bool
	err         error

	width, height int
	viewport      viewport.Model
	vpReady       bool
}

func NewK8sDeploymentDetailView(k8s *awseks.K8sClient, deployment K8sDeployment) *K8sDeploymentDetailView {
	return &K8sDeploymentDetailView{
		k8s:        k8s,
		deployment: deployment,
	}
}

func (v *K8sDeploymentDetailView) Title() string { return "Deployment: " + v.deployment.Name }

func (v *K8sDeploymentDetailView) Init() tea.Cmd {
	return v.fetchDeploymentDetail()
}

func (v *K8sDeploymentDetailView) fetchDeploymentDetail() tea.Cmd {
	k8s := v.k8s
	dep := v.deployment
	return func() tea.Msg {
		ctx := context.Background()

		// Fetch deployment for strategy details
		d, err := k8s.Clientset.AppsV1().Deployments(dep.Namespace).Get(ctx, dep.Name, metav1.GetOptions{})
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		strategy := string(d.Spec.Strategy.Type)
		maxSurge := ""
		maxUnavail := ""
		if d.Spec.Strategy.RollingUpdate != nil {
			if d.Spec.Strategy.RollingUpdate.MaxSurge != nil {
				maxSurge = d.Spec.Strategy.RollingUpdate.MaxSurge.String()
			}
			if d.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
				maxUnavail = d.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
			}
		}

		// Fetch ReplicaSets for revision history
		selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		rsList, err := k8s.Clientset.AppsV1().ReplicaSets(dep.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		var revisions []DeploymentRevision
		for _, rs := range rsList.Items {
			rev := rs.Annotations["deployment.kubernetes.io/revision"]
			if rev == "" {
				continue
			}

			// Get first container image
			image := ""
			if len(rs.Spec.Template.Spec.Containers) > 0 {
				image = rs.Spec.Template.Spec.Containers[0].Image
			}

			changeCause := rs.Annotations["kubernetes.io/change-cause"]

			revisions = append(revisions, DeploymentRevision{
				Revision:    rev,
				Image:       image,
				ChangeCause: changeCause,
			})
		}

		// Sort by revision descending
		sort.Slice(revisions, func(i, j int) bool {
			return revisions[i].Revision > revisions[j].Revision
		})

		return k8sDeploymentDetailMsg{
			revisions:  revisions,
			strategy:   strategy,
			maxSurge:   maxSurge,
			maxUnavail: maxUnavail,
		}
	}
}

func (v *K8sDeploymentDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case k8sDeploymentDetailMsg:
		v.revisions = msg.revisions
		v.strategy = msg.strategy
		v.maxSurge = msg.maxSurge
		v.maxUnavail = msg.maxUnavail
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

func (v *K8sDeploymentDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
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

func (v *K8sDeploymentDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	muted := theme.MutedStyle

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("Deployment: %s", v.deployment.Name)))
	b.WriteString(fmt.Sprintf("   Replicas: %s\n", v.deployment.Ready))
	b.WriteString(fmt.Sprintf("Namespace: %s\n", v.deployment.Namespace))

	// Strategy
	stratLine := fmt.Sprintf("Strategy: %s", v.strategy)
	if v.maxSurge != "" {
		stratLine += fmt.Sprintf("  MaxSurge: %s", v.maxSurge)
	}
	if v.maxUnavail != "" {
		stratLine += fmt.Sprintf("  MaxUnavailable: %s", v.maxUnavail)
	}
	b.WriteString(stratLine + "\n")

	// Revision history
	if len(v.revisions) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Revision History:"))
		b.WriteString("\n")
		b.WriteString(muted.Render(fmt.Sprintf(" %-6s %-40s %s", "REV", "IMAGE", "CHANGE CAUSE")))
		b.WriteString("\n")
		for _, r := range v.revisions {
			cause := r.ChangeCause
			if cause == "" {
				cause = "<none>"
			}
			b.WriteString(fmt.Sprintf(" %-6s %-40s %s\n", r.Revision, r.Image, cause))
		}
	} else {
		b.WriteString("\n")
		b.WriteString(muted.Render("No revision history"))
		b.WriteString("\n")
	}

	return b.String()
}

func (v *K8sDeploymentDetailView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.loaded {
		return theme.MutedStyle.Render("Loading deployment details...")
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *K8sDeploymentDetailView) SetSize(width, height int) {
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

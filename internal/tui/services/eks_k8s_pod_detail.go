package services

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

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
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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

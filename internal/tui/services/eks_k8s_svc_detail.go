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
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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

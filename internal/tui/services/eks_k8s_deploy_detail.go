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
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
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

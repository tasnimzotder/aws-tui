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

type K8sServiceAccountDetailView struct {
	k8s *awseks.K8sClient
	sa  K8sServiceAccount

	annotations map[string]string
	labels      map[string]string
	secrets     []string
	automount   string
	loaded      bool
	err         error

	width, height int
	viewport      viewport.Model
	vpReady       bool
}

func NewK8sServiceAccountDetailView(k8s *awseks.K8sClient, sa K8sServiceAccount) *K8sServiceAccountDetailView {
	return &K8sServiceAccountDetailView{
		k8s: k8s,
		sa:  sa,
	}
}

func (v *K8sServiceAccountDetailView) Title() string {
	return "ServiceAccount: " + v.sa.Name
}

func (v *K8sServiceAccountDetailView) Init() tea.Cmd {
	return v.fetchDetail()
}

func (v *K8sServiceAccountDetailView) fetchDetail() tea.Cmd {
	k8s := v.k8s
	sa := v.sa
	return func() tea.Msg {
		ctx := context.Background()
		obj, err := k8s.Clientset.CoreV1().ServiceAccounts(sa.Namespace).Get(ctx, sa.Name, metav1.GetOptions{})
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		annotations := make(map[string]string, len(obj.Annotations))
		for k, val := range obj.Annotations {
			annotations[k] = val
		}

		labels := make(map[string]string, len(obj.Labels))
		for k, val := range obj.Labels {
			labels[k] = val
		}

		secrets := make([]string, 0, len(obj.Secrets))
		for _, s := range obj.Secrets {
			secrets = append(secrets, s.Name)
		}

		automount := "true (default)"
		if obj.AutomountServiceAccountToken != nil {
			if *obj.AutomountServiceAccountToken {
				automount = "true"
			} else {
				automount = "false"
			}
		}

		return k8sServiceAccountDetailMsg{
			annotations: annotations,
			labels:      labels,
			secrets:     secrets,
			automount:   automount,
		}
	}
}

func (v *K8sServiceAccountDetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case k8sServiceAccountDetailMsg:
		v.annotations = msg.annotations
		v.labels = msg.labels
		v.secrets = msg.secrets
		v.automount = msg.automount
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

func (v *K8sServiceAccountDetailView) initViewport() {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	v.viewport = NewStyledViewport(v.width, h)
	v.viewport.SetContent(v.renderContent())
	v.vpReady = true
}

func (v *K8sServiceAccountDetailView) renderContent() string {
	var b strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	muted := theme.MutedStyle
	highlight := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)

	// Header
	b.WriteString(bold.Render(fmt.Sprintf("ServiceAccount: %s", v.sa.Name)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Namespace: %s  Age: %s\n", v.sa.Namespace, v.sa.Age))
	b.WriteString(fmt.Sprintf("Automount Token: %s\n", v.automount))

	// IAM Role (highlighted — key EKS-specific info)
	iamRole := v.annotations["eks.amazonaws.com/role-arn"]
	if iamRole != "" {
		b.WriteString("\n")
		b.WriteString(bold.Render("IAM Role (IRSA):"))
		b.WriteString("\n")
		b.WriteString(" " + highlight.Render(iamRole) + "\n")
	}

	// Secrets
	b.WriteString("\n")
	b.WriteString(bold.Render("Secrets:"))
	b.WriteString("\n")
	if len(v.secrets) > 0 {
		for _, s := range v.secrets {
			b.WriteString(fmt.Sprintf(" %s\n", s))
		}
	} else {
		b.WriteString(muted.Render(" <none>") + "\n")
	}

	// Annotations
	if len(v.annotations) > 0 {
		b.WriteString("\n")
		b.WriteString(bold.Render("Annotations:"))
		b.WriteString("\n")
		keys := make([]string, 0, len(v.annotations))
		for k := range v.annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf(" %s=%s\n", k, v.annotations[k]))
		}
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

func (v *K8sServiceAccountDetailView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if !v.loaded {
		return theme.MutedStyle.Render("Loading service account details...")
	}
	if !v.vpReady {
		return ""
	}
	status := theme.MutedStyle.Render("  Esc back  ↑/↓ scroll")
	return v.viewport.View() + "\n" + status
}

func (v *K8sServiceAccountDetailView) SetSize(width, height int) {
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

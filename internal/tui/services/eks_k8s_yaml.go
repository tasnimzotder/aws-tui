package services

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	"tasnim.dev/aws-tui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type k8sYAMLReadyMsg struct {
	yaml []byte
}

// ---------------------------------------------------------------------------
// K8sYAMLSpecView — async loads a K8s resource and shows its YAML spec
// ---------------------------------------------------------------------------

type K8sYAMLSpecView struct {
	title    string
	fetchFn  func(ctx context.Context) (any, error)
	textView *TextView

	loaded  bool
	err     error
	spinner spinner.Model
	width   int
	height  int
}

func newK8sYAMLSpecView(title string, fetchFn func(ctx context.Context) (any, error)) *K8sYAMLSpecView {
	return &K8sYAMLSpecView{
		title:   title,
		fetchFn: fetchFn,
		spinner: theme.NewSpinner(),
	}
}

func (v *K8sYAMLSpecView) Title() string { return v.title }

func (v *K8sYAMLSpecView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchYAML())
}

func (v *K8sYAMLSpecView) fetchYAML() tea.Cmd {
	fn := v.fetchFn
	return func() tea.Msg {
		ctx := context.Background()
		obj, err := fn(ctx)
		if err != nil {
			return k8sDetailErrorMsg{err: err}
		}

		// Marshal to JSON first, then convert to YAML
		jsonBytes, err := json.Marshal(obj)
		if err != nil {
			return k8sDetailErrorMsg{err: fmt.Errorf("marshal json: %w", err)}
		}

		yamlBytes, err := sigsyaml.JSONToYAML(jsonBytes)
		if err != nil {
			return k8sDetailErrorMsg{err: fmt.Errorf("convert to yaml: %w", err)}
		}

		return k8sYAMLReadyMsg{yaml: yamlBytes}
	}
}

func (v *K8sYAMLSpecView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case k8sYAMLReadyMsg:
		v.loaded = true
		v.textView = NewTextView(v.title, msg.yaml, "spec.yaml")
		if v.width > 0 {
			v.textView.SetSize(v.width, v.height)
		}
		return v, v.textView.Init()

	case k8sDetailErrorMsg:
		v.err = msg.err
		v.loaded = true
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if v.textView != nil {
			v.textView.SetSize(v.width, v.height)
		}
		return v, nil

	case spinner.TickMsg:
		if !v.loaded {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
	}

	// Delegate to inner text view once loaded
	if v.textView != nil {
		updated, cmd := v.textView.Update(msg)
		if tv, ok := updated.(*TextView); ok {
			v.textView = tv
		}
		return v, cmd
	}
	return v, nil
}

func (v *K8sYAMLSpecView) View() string {
	if v.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err)) +
			"\n\n" + theme.MutedStyle.Render("Press Esc to go back")
	}
	if v.textView != nil {
		return v.textView.View()
	}
	return v.spinner.View() + " Loading YAML spec..."
}

func (v *K8sYAMLSpecView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.textView != nil {
		v.textView.SetSize(width, height)
	}
}

// ---------------------------------------------------------------------------
// Constructors for each K8s resource type
// ---------------------------------------------------------------------------

func NewK8sPodYAMLView(k8s *awseks.K8sClient, pod K8sPod) *K8sYAMLSpecView {
	return newK8sYAMLSpecView("YAML: "+pod.Name, func(ctx context.Context) (any, error) {
		return k8s.Clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	})
}

func NewK8sServiceYAMLView(k8s *awseks.K8sClient, svc K8sService) *K8sYAMLSpecView {
	return newK8sYAMLSpecView("YAML: "+svc.Name, func(ctx context.Context) (any, error) {
		return k8s.Clientset.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	})
}

func NewK8sDeploymentYAMLView(k8s *awseks.K8sClient, dep K8sDeployment) *K8sYAMLSpecView {
	return newK8sYAMLSpecView("YAML: "+dep.Name, func(ctx context.Context) (any, error) {
		return k8s.Clientset.AppsV1().Deployments(dep.Namespace).Get(ctx, dep.Name, metav1.GetOptions{})
	})
}

func NewK8sNodeYAMLView(k8s *awseks.K8sClient, node K8sNode) *K8sYAMLSpecView {
	return newK8sYAMLSpecView("YAML: "+node.Name, func(ctx context.Context) (any, error) {
		return k8s.Clientset.CoreV1().Nodes().Get(ctx, node.Name, metav1.GetOptions{})
	})
}

func NewK8sServiceAccountYAMLView(k8s *awseks.K8sClient, sa K8sServiceAccount) *K8sYAMLSpecView {
	return newK8sYAMLSpecView("YAML: "+sa.Name, func(ctx context.Context) (any, error) {
		return k8s.Clientset.CoreV1().ServiceAccounts(sa.Namespace).Get(ctx, sa.Name, metav1.GetOptions{})
	})
}

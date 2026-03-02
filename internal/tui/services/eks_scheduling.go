package services

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
)

// --- Data types ---

// K8sTaint represents a taint on a Kubernetes node.
type K8sTaint struct {
	Node   string
	Key    string
	Value  string
	Effect string
}

// K8sToleration represents a toleration on a Kubernetes pod.
type K8sToleration struct {
	Namespace string
	Pod       string
	Key       string
	Operator  string
	Value     string
	Effect    string
}

// K8sNodeAffinity represents a node affinity rule on a Kubernetes pod.
type K8sNodeAffinity struct {
	Namespace string
	Pod       string
	RuleType  string
	Key       string
	Operator  string
	Values    string
}

// --- Scheduling selector list ---

type schedulingItem struct {
	title string
	desc  string
}

func (i schedulingItem) Title() string       { return i.title }
func (i schedulingItem) Description() string { return i.desc }
func (i schedulingItem) FilterValue() string { return i.title }

// eksSchedulingView is the tab-0 content showing a 3-item selector list.
type eksSchedulingView struct {
	k8sClient *awseks.K8sClient
	list      list.Model
	width     int
	height    int
}

// NewEKSSchedulingView creates the scheduling category picker.
func NewEKSSchedulingView(k8sClient *awseks.K8sClient) *eksSchedulingView {
	items := []list.Item{
		schedulingItem{title: "Taints", desc: "Node taints across all cluster nodes"},
		schedulingItem{title: "Tolerations", desc: "Pod tolerations across all namespaces"},
		schedulingItem{title: "Node Affinity", desc: "Pod node affinity rules across all namespaces"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 60, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &eksSchedulingView{
		k8sClient: k8sClient,
		list:      l,
	}
}

func (v *eksSchedulingView) Title() string { return "Scheduling" }
func (v *eksSchedulingView) Init() tea.Cmd  { return nil }

func (v *eksSchedulingView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			selected, ok := v.list.SelectedItem().(schedulingItem)
			if !ok {
				return v, nil
			}
			switch selected.title {
			case "Taints":
				return v, pushView(NewK8sTaintsTableView(v.k8sClient))
			case "Tolerations":
				return v, pushView(NewK8sTolerationsTableView(v.k8sClient))
			case "Node Affinity":
				return v, pushView(NewK8sNodeAffinityTableView(v.k8sClient))
			}
			return v, nil
		}
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.list.SetSize(msg.Width, msg.Height)
		return v, nil
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *eksSchedulingView) View() string {
	return v.list.View()
}

func (v *eksSchedulingView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width, height)
}

// --- Taints table ---

func NewK8sTaintsTableView(k8s *awseks.K8sClient) View {
	return NewTableView(TableViewConfig[K8sTaint]{
		Title:       "Taints",
		LoadingText: "Loading node taints...",
		Columns: []table.Column{
			{Title: "Node", Width: 36},
			{Title: "Key", Width: 28},
			{Title: "Value", Width: 20},
			{Title: "Effect", Width: 14},
		},
		FetchFunc: func(ctx context.Context) ([]K8sTaint, error) {
			nodes, err := k8s.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("list nodes: %w", err)
			}
			var taints []K8sTaint
			for _, node := range nodes.Items {
				for _, t := range node.Spec.Taints {
					taints = append(taints, K8sTaint{
						Node:   node.Name,
						Key:    t.Key,
						Value:  t.Value,
						Effect: string(t.Effect),
					})
				}
			}
			return taints, nil
		},
		RowMapper: func(t K8sTaint) table.Row {
			return table.Row{t.Node, t.Key, t.Value, t.Effect}
		},
		CopyIDFunc: func(t K8sTaint) string { return t.Node + "/" + t.Key },
		OnEnter: func(t K8sTaint) tea.Cmd {
			return pushView(NewK8sNodeDetailView(k8s, K8sNode{Name: t.Node}))
		},
	})
}

// --- Tolerations table ---

func NewK8sTolerationsTableView(k8s *awseks.K8sClient) View {
	return NewTableView(TableViewConfig[K8sToleration]{
		Title:       "Tolerations",
		LoadingText: "Loading pod tolerations...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 14},
			{Title: "Pod", Width: 24},
			{Title: "Key", Width: 20},
			{Title: "Operator", Width: 10},
			{Title: "Value", Width: 16},
			{Title: "Effect", Width: 14},
		},
		FetchFunc: func(ctx context.Context) ([]K8sToleration, error) {
			pods, err := k8s.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("list pods: %w", err)
			}
			var tolerations []K8sToleration
			for _, pod := range pods.Items {
				for _, t := range pod.Spec.Tolerations {
					tolerations = append(tolerations, K8sToleration{
						Namespace: pod.Namespace,
						Pod:       pod.Name,
						Key:       t.Key,
						Operator:  string(t.Operator),
						Value:     t.Value,
						Effect:    string(t.Effect),
					})
				}
			}
			return tolerations, nil
		},
		RowMapper: func(t K8sToleration) table.Row {
			return table.Row{t.Namespace, t.Pod, t.Key, t.Operator, t.Value, t.Effect}
		},
		CopyIDFunc: func(t K8sToleration) string { return t.Namespace + "/" + t.Pod + "/" + t.Key },
		OnEnter: func(t K8sToleration) tea.Cmd {
			return pushView(NewK8sPodDetailView(k8s, K8sPod{Namespace: t.Namespace, Name: t.Pod}))
		},
	})
}

// --- Node Affinity table ---

func NewK8sNodeAffinityTableView(k8s *awseks.K8sClient) View {
	return NewTableView(TableViewConfig[K8sNodeAffinity]{
		Title:       "Node Affinity",
		LoadingText: "Loading node affinity rules...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 14},
			{Title: "Pod", Width: 24},
			{Title: "Type", Width: 12},
			{Title: "Key", Width: 20},
			{Title: "Operator", Width: 12},
			{Title: "Values", Width: 24},
		},
		FetchFunc: func(ctx context.Context) ([]K8sNodeAffinity, error) {
			pods, err := k8s.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("list pods: %w", err)
			}
			var rules []K8sNodeAffinity
			for _, pod := range pods.Items {
				if pod.Spec.Affinity == nil || pod.Spec.Affinity.NodeAffinity == nil {
					continue
				}
				na := pod.Spec.Affinity.NodeAffinity

				// Required rules
				if na.RequiredDuringSchedulingIgnoredDuringExecution != nil {
					for _, term := range na.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
						for _, expr := range term.MatchExpressions {
							rules = append(rules, K8sNodeAffinity{
								Namespace: pod.Namespace,
								Pod:       pod.Name,
								RuleType:  "Required",
								Key:       expr.Key,
								Operator:  string(expr.Operator),
								Values:    strings.Join(expr.Values, ", "),
							})
						}
					}
				}

				// Preferred rules
				for _, pref := range na.PreferredDuringSchedulingIgnoredDuringExecution {
					for _, expr := range pref.Preference.MatchExpressions {
						rules = append(rules, K8sNodeAffinity{
							Namespace: pod.Namespace,
							Pod:       pod.Name,
							RuleType:  "Preferred",
							Key:       expr.Key,
							Operator:  string(expr.Operator),
							Values:    strings.Join(expr.Values, ", "),
						})
					}
				}
			}
			return rules, nil
		},
		RowMapper: func(a K8sNodeAffinity) table.Row {
			return table.Row{a.Namespace, a.Pod, a.RuleType, a.Key, a.Operator, a.Values}
		},
		CopyIDFunc: func(a K8sNodeAffinity) string { return a.Namespace + "/" + a.Pod + "/" + a.Key },
		OnEnter: func(a K8sNodeAffinity) tea.Cmd {
			return pushView(NewK8sPodDetailView(k8s, K8sPod{Namespace: a.Namespace, Name: a.Pod}))
		},
	})
}

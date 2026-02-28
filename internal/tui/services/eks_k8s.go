package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awseks "tasnim.dev/aws-tui/internal/aws/eks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ---------------------------------------------------------------------------
// K8s TUI-friendly resource types
// ---------------------------------------------------------------------------

// K8sPod represents a pod for TUI display.
type K8sPod struct {
	Namespace  string
	Name       string
	Status     string
	Ready      string   // e.g., "1/1" or "2/3"
	Restarts   int
	Age        string   // e.g., "2d", "5h", "30m"
	Containers []string // container names (for log/exec picker)
	NodeName   string
	PodIP      string
}

// K8sService represents a service for TUI display.
type K8sService struct {
	Namespace  string
	Name       string
	Type       string // ClusterIP, LoadBalancer, NodePort, ExternalName
	ClusterIP  string
	ExternalIP string // or "<none>"
	Ports      string // e.g., "80:31234/TCP, 443:31235/TCP"
	Age        string
	Selector   map[string]string
}

// K8sDeployment represents a deployment for TUI display.
type K8sDeployment struct {
	Namespace string
	Name      string
	Ready     string // e.g., "3/3"
	UpToDate  int
	Available int
	Age       string
	Strategy  string
}

// ---------------------------------------------------------------------------
// Data-fetching helpers
// ---------------------------------------------------------------------------

// listPods fetches pods from K8s API and maps them to TUI-friendly structs.
func listPods(ctx context.Context, k8s *awseks.K8sClient, namespace string) ([]K8sPod, error) {
	podList, err := k8s.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	pods := make([]K8sPod, 0, len(podList.Items))
	for _, pod := range podList.Items {
		// Determine status: check container statuses for waiting reasons first.
		status := string(pod.Status.Phase)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				status = cs.State.Waiting.Reason
				break
			}
		}

		// Ready count
		readyCount := 0
		totalCount := len(pod.Spec.Containers)
		restarts := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
			restarts += int(cs.RestartCount)
		}

		// Container names
		containers := make([]string, 0, len(pod.Spec.Containers))
		for _, c := range pod.Spec.Containers {
			containers = append(containers, c.Name)
		}

		pods = append(pods, K8sPod{
			Namespace:  pod.Namespace,
			Name:       pod.Name,
			Status:     status,
			Ready:      fmt.Sprintf("%d/%d", readyCount, totalCount),
			Restarts:   restarts,
			Age:        formatAge(pod.CreationTimestamp.Time),
			Containers: containers,
			NodeName:   pod.Spec.NodeName,
			PodIP:      pod.Status.PodIP,
		})
	}
	return pods, nil
}

// listServices fetches services from K8s API and maps them to TUI-friendly structs.
func listServices(ctx context.Context, k8s *awseks.K8sClient, namespace string) ([]K8sService, error) {
	svcList, err := k8s.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	services := make([]K8sService, 0, len(svcList.Items))
	for _, svc := range svcList.Items {
		// External IP
		externalIP := "<none>"
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			var ips []string
			for _, ing := range svc.Status.LoadBalancer.Ingress {
				if ing.IP != "" {
					ips = append(ips, ing.IP)
				} else if ing.Hostname != "" {
					ips = append(ips, ing.Hostname)
				}
			}
			if len(ips) > 0 {
				externalIP = strings.Join(ips, ", ")
			}
		}

		// Ports
		var ports []string
		for _, p := range svc.Spec.Ports {
			portStr := fmt.Sprintf("%d", p.Port)
			if p.NodePort != 0 {
				portStr += fmt.Sprintf(":%d", p.NodePort)
			}
			portStr += "/" + string(p.Protocol)
			ports = append(ports, portStr)
		}

		services = append(services, K8sService{
			Namespace:  svc.Namespace,
			Name:       svc.Name,
			Type:       string(svc.Spec.Type),
			ClusterIP:  svc.Spec.ClusterIP,
			ExternalIP: externalIP,
			Ports:      strings.Join(ports, ", "),
			Age:        formatAge(svc.CreationTimestamp.Time),
			Selector:   svc.Spec.Selector,
		})
	}
	return services, nil
}

// listDeployments fetches deployments from K8s API and maps them to TUI-friendly structs.
func listDeployments(ctx context.Context, k8s *awseks.K8sClient, namespace string) ([]K8sDeployment, error) {
	depList, err := k8s.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	deployments := make([]K8sDeployment, 0, len(depList.Items))
	for _, dep := range depList.Items {
		var replicas int32
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}

		strategy := ""
		if dep.Spec.Strategy.Type != "" {
			strategy = string(dep.Spec.Strategy.Type)
		}

		deployments = append(deployments, K8sDeployment{
			Namespace: dep.Namespace,
			Name:      dep.Name,
			Ready:     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, replicas),
			UpToDate:  int(dep.Status.UpdatedReplicas),
			Available: int(dep.Status.AvailableReplicas),
			Age:       formatAge(dep.CreationTimestamp.Time),
			Strategy:  strategy,
		})
	}
	return deployments, nil
}

// formatAge converts a time to a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// ---------------------------------------------------------------------------
// TableView constructors
// ---------------------------------------------------------------------------

// NewK8sPodsTableView creates a table view for K8s pods.
func NewK8sPodsTableView(k8s *awseks.K8sClient, namespace string) *TableView[K8sPod] {
	return newK8sPodsTableView(k8s, namespace, nil)
}

// NewK8sPodsTableViewWithPF creates a table view for K8s pods with port-forward support.
func NewK8sPodsTableViewWithPF(k8s *awseks.K8sClient, namespace string, pfManager *portForwardManager) *TableView[K8sPod] {
	return newK8sPodsTableView(k8s, namespace, pfManager)
}

func newK8sPodsTableView(k8s *awseks.K8sClient, namespace string, pfManager *portForwardManager) *TableView[K8sPod] {
	keyHandlers := map[string]func(K8sPod) tea.Cmd{
		"l": func(p K8sPod) tea.Cmd {
			if len(p.Containers) > 1 {
				return pushView(NewContainerPickerView(k8s, p, "logs"))
			}
			return pushView(NewEKSLogView(k8s, p, ""))
		},
		"x": func(p K8sPod) tea.Cmd {
			if len(p.Containers) > 1 {
				return pushView(NewContainerPickerView(k8s, p, "exec"))
			}
			return execIntoPod(k8s, p, "")
		},
	}

	if pfManager != nil {
		keyHandlers["f"] = func(p K8sPod) tea.Cmd {
			return pushView(newPortForwardInputView(k8s, p, pfManager))
		}
		keyHandlers["F"] = func(_ K8sPod) tea.Cmd {
			return pushView(newPortForwardListView(pfManager))
		}
	}

	return NewTableView(TableViewConfig[K8sPod]{
		Title:       "Pods",
		LoadingText: "Loading pods...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 16},
			{Title: "Name", Width: 30},
			{Title: "Status", Width: 12},
			{Title: "Ready", Width: 8},
			{Title: "Restarts", Width: 10},
			{Title: "Age", Width: 8},
		},
		FetchFunc: func(ctx context.Context) ([]K8sPod, error) {
			return listPods(ctx, k8s, namespace)
		},
		RowMapper: func(p K8sPod) table.Row {
			return table.Row{p.Namespace, p.Name, p.Status, p.Ready,
				fmt.Sprintf("%d", p.Restarts), p.Age}
		},
		CopyIDFunc:  func(p K8sPod) string { return p.Namespace + "/" + p.Name },
		KeyHandlers: keyHandlers,
		OnEnter: func(p K8sPod) tea.Cmd {
			return pushView(NewK8sPodDetailView(k8s, p))
		},
	})
}

// NewK8sServicesTableView creates a table view for K8s services.
func NewK8sServicesTableView(k8s *awseks.K8sClient, namespace string) *TableView[K8sService] {
	return newK8sServicesTableView(k8s, namespace, nil)
}

// NewK8sServicesTableViewWithPF creates a table view for K8s services with port-forward support.
func NewK8sServicesTableViewWithPF(k8s *awseks.K8sClient, namespace string, pfManager *portForwardManager) *TableView[K8sService] {
	return newK8sServicesTableView(k8s, namespace, pfManager)
}

func newK8sServicesTableView(k8s *awseks.K8sClient, namespace string, pfManager *portForwardManager) *TableView[K8sService] {
	var keyHandlers map[string]func(K8sService) tea.Cmd
	if pfManager != nil {
		keyHandlers = map[string]func(K8sService) tea.Cmd{
			"F": func(_ K8sService) tea.Cmd {
				return pushView(newPortForwardListView(pfManager))
			},
		}
	}

	return NewTableView(TableViewConfig[K8sService]{
		Title:       "Services",
		LoadingText: "Loading services...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 16},
			{Title: "Name", Width: 24},
			{Title: "Type", Width: 14},
			{Title: "Cluster IP", Width: 16},
			{Title: "External IP", Width: 30},
			{Title: "Ports", Width: 24},
		},
		FetchFunc: func(ctx context.Context) ([]K8sService, error) {
			return listServices(ctx, k8s, namespace)
		},
		RowMapper: func(s K8sService) table.Row {
			return table.Row{s.Namespace, s.Name, s.Type, s.ClusterIP,
				s.ExternalIP, s.Ports}
		},
		CopyIDFunc:  func(s K8sService) string { return s.Namespace + "/" + s.Name },
		KeyHandlers: keyHandlers,
		OnEnter: func(s K8sService) tea.Cmd {
			return pushView(NewK8sServiceDetailView(k8s, s))
		},
	})
}

// NewK8sDeploymentsTableView creates a table view for K8s deployments.
func NewK8sDeploymentsTableView(k8s *awseks.K8sClient, namespace string) *TableView[K8sDeployment] {
	return NewTableView(TableViewConfig[K8sDeployment]{
		Title:       "Deployments",
		LoadingText: "Loading deployments...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 16},
			{Title: "Name", Width: 28},
			{Title: "Ready", Width: 8},
			{Title: "Up-to-date", Width: 12},
			{Title: "Available", Width: 10},
			{Title: "Age", Width: 8},
		},
		FetchFunc: func(ctx context.Context) ([]K8sDeployment, error) {
			return listDeployments(ctx, k8s, namespace)
		},
		RowMapper: func(d K8sDeployment) table.Row {
			return table.Row{d.Namespace, d.Name, d.Ready,
				fmt.Sprintf("%d", d.UpToDate),
				fmt.Sprintf("%d", d.Available), d.Age}
		},
		CopyIDFunc: func(d K8sDeployment) string { return d.Namespace + "/" + d.Name },
		OnEnter: func(d K8sDeployment) tea.Cmd {
			return pushView(NewK8sDeploymentDetailView(k8s, d))
		},
	})
}

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

// K8sServiceAccount represents a service account for TUI display.
type K8sServiceAccount struct {
	Namespace      string
	Name           string
	Secrets        int    // number of secret references
	IAMRole        string // eks.amazonaws.com/role-arn annotation
	AutomountToken string // "true"/"false"
	Age            string
}

// K8sIngress represents an ingress for TUI display.
type K8sIngress struct {
	Name      string
	Namespace string
	Class     string // spec.ingressClassName
	Hosts     string // comma-joined from rules[].host
	Address   string // from status.loadBalancer.ingress
	Ports     string // from TLS (443) or default (80)
	Age       string
}

// K8sNode represents a node for TUI display.
type K8sNode struct {
	Name         string
	Status       string // "Ready", "NotReady"
	Roles        string // "control-plane", "worker", etc.
	Age          string
	Version      string // kubelet version
	InternalIP   string
	ExternalIP   string
	OS           string // e.g. "linux/amd64"
	InstanceType string // from node label
	NodeGroup    string // from eks.amazonaws.com/nodegroup label
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

// listServiceAccounts fetches service accounts from K8s API.
func listServiceAccounts(ctx context.Context, k8s *awseks.K8sClient, namespace string) ([]K8sServiceAccount, error) {
	saList, err := k8s.Clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list service accounts: %w", err)
	}

	accounts := make([]K8sServiceAccount, 0, len(saList.Items))
	for _, sa := range saList.Items {
		iamRole := sa.Annotations["eks.amazonaws.com/role-arn"]

		automount := "true"
		if sa.AutomountServiceAccountToken != nil && !*sa.AutomountServiceAccountToken {
			automount = "false"
		}

		accounts = append(accounts, K8sServiceAccount{
			Namespace:      sa.Namespace,
			Name:           sa.Name,
			Secrets:        len(sa.Secrets),
			IAMRole:        iamRole,
			AutomountToken: automount,
			Age:            formatAge(sa.CreationTimestamp.Time),
		})
	}
	return accounts, nil
}

// listNodes fetches nodes from K8s API filtered by node group name.
func listNodes(ctx context.Context, k8s *awseks.K8sClient, nodeGroupName string) ([]K8sNode, error) {
	opts := metav1.ListOptions{}
	if nodeGroupName != "" {
		opts.LabelSelector = "eks.amazonaws.com/nodegroup=" + nodeGroupName
	}
	nodeList, err := k8s.Clientset.CoreV1().Nodes().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	nodes := make([]K8sNode, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		// Determine status from conditions
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
				break
			}
		}

		// Determine roles from labels
		var roles []string
		for label := range node.Labels {
			if role, ok := strings.CutPrefix(label, "node-role.kubernetes.io/"); ok && role != "" {
				roles = append(roles, role)
			}
		}
		roleStr := strings.Join(roles, ",")
		if roleStr == "" {
			roleStr = "<none>"
		}

		// Addresses
		internalIP := ""
		externalIP := ""
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case "InternalIP":
				internalIP = addr.Address
			case "ExternalIP":
				externalIP = addr.Address
			}
		}

		nodes = append(nodes, K8sNode{
			Name:         node.Name,
			Status:       status,
			Roles:        roleStr,
			Age:          formatAge(node.CreationTimestamp.Time),
			Version:      node.Status.NodeInfo.KubeletVersion,
			InternalIP:   internalIP,
			ExternalIP:   externalIP,
			OS:           node.Status.NodeInfo.OperatingSystem + "/" + node.Status.NodeInfo.Architecture,
			InstanceType: node.Labels["node.kubernetes.io/instance-type"],
			NodeGroup:    node.Labels["eks.amazonaws.com/nodegroup"],
		})
	}
	return nodes, nil
}

// listIngresses fetches ingresses from K8s API and maps them to TUI-friendly structs.
func listIngresses(ctx context.Context, k8s *awseks.K8sClient, namespace string) ([]K8sIngress, error) {
	ingList, err := k8s.Clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ingresses: %w", err)
	}

	ingresses := make([]K8sIngress, 0, len(ingList.Items))
	for _, ing := range ingList.Items {
		// Class
		class := ""
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}

		// Hosts
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		hostsStr := strings.Join(hosts, ", ")
		if hostsStr == "" {
			hostsStr = "*"
		}

		// Address from status
		var addrs []string
		for _, lb := range ing.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				addrs = append(addrs, lb.IP)
			} else if lb.Hostname != "" {
				addrs = append(addrs, lb.Hostname)
			}
		}
		address := strings.Join(addrs, ", ")

		// Ports: if TLS is configured show 443, otherwise 80
		ports := "80"
		if len(ing.Spec.TLS) > 0 {
			ports = "80, 443"
		}

		ingresses = append(ingresses, K8sIngress{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Class:     class,
			Hosts:     hostsStr,
			Address:   address,
			Ports:     ports,
			Age:       formatAge(ing.CreationTimestamp.Time),
		})
	}
	return ingresses, nil
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
			return pushView(newExecInputView(k8s, p, ""))
		},
		"e": func(p K8sPod) tea.Cmd {
			return pushView(NewK8sPodYAMLView(k8s, p))
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

	podsHelp := HelpContextK8sPods
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
		HelpCtx:     &podsHelp,
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
	keyHandlers := map[string]func(K8sService) tea.Cmd{
		"e": func(s K8sService) tea.Cmd {
			return pushView(NewK8sServiceYAMLView(k8s, s))
		},
	}
	if pfManager != nil {
		keyHandlers["F"] = func(_ K8sService) tea.Cmd {
			return pushView(newPortForwardListView(pfManager))
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

// NewK8sNodesTableView creates a table view for K8s nodes in a node group.
func NewK8sNodesTableView(k8s *awseks.K8sClient, nodeGroupName string) *TableView[K8sNode] {
	nodesHelp := HelpContextK8sNodes
	return NewTableView(TableViewConfig[K8sNode]{
		Title:       "Nodes: " + nodeGroupName,
		LoadingText: "Loading nodes...",
		Columns: []table.Column{
			{Title: "Name", Width: 36},
			{Title: "Status", Width: 10},
			{Title: "Version", Width: 14},
			{Title: "Instance Type", Width: 16},
			{Title: "Internal IP", Width: 16},
			{Title: "Age", Width: 8},
		},
		FetchFunc: func(ctx context.Context) ([]K8sNode, error) {
			return listNodes(ctx, k8s, nodeGroupName)
		},
		RowMapper: func(n K8sNode) table.Row {
			return table.Row{n.Name, n.Status, n.Version, n.InstanceType,
				n.InternalIP, n.Age}
		},
		CopyIDFunc: func(n K8sNode) string { return n.Name },
		KeyHandlers: map[string]func(K8sNode) tea.Cmd{
			"x": func(n K8sNode) tea.Cmd {
				return pushView(newNodeDebugInputView(k8s, n))
			},
			"e": func(n K8sNode) tea.Cmd {
				return pushView(NewK8sNodeYAMLView(k8s, n))
			},
		},
		HelpCtx: &nodesHelp,
		OnEnter: func(n K8sNode) tea.Cmd {
			return pushView(NewK8sNodeDetailView(k8s, n))
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
		KeyHandlers: map[string]func(K8sDeployment) tea.Cmd{
			"e": func(d K8sDeployment) tea.Cmd {
				return pushView(NewK8sDeploymentYAMLView(k8s, d))
			},
		},
		OnEnter: func(d K8sDeployment) tea.Cmd {
			return pushView(NewK8sDeploymentDetailView(k8s, d))
		},
	})
}

// NewK8sServiceAccountsTableView creates a table view for K8s service accounts.
func NewK8sServiceAccountsTableView(k8s *awseks.K8sClient, namespace string) *TableView[K8sServiceAccount] {
	return NewTableView(TableViewConfig[K8sServiceAccount]{
		Title:       "Service Accounts",
		LoadingText: "Loading service accounts...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 16},
			{Title: "Name", Width: 24},
			{Title: "IAM Role", Width: 44},
			{Title: "Secrets", Width: 8},
			{Title: "Automount", Width: 10},
			{Title: "Age", Width: 8},
		},
		FetchFunc: func(ctx context.Context) ([]K8sServiceAccount, error) {
			return listServiceAccounts(ctx, k8s, namespace)
		},
		RowMapper: func(sa K8sServiceAccount) table.Row {
			iamRole := sa.IAMRole
			if iamRole == "" {
				iamRole = "<none>"
			}
			return table.Row{sa.Namespace, sa.Name, iamRole,
				fmt.Sprintf("%d", sa.Secrets), sa.AutomountToken, sa.Age}
		},
		CopyIDFunc: func(sa K8sServiceAccount) string { return sa.Namespace + "/" + sa.Name },
		KeyHandlers: map[string]func(K8sServiceAccount) tea.Cmd{
			"e": func(sa K8sServiceAccount) tea.Cmd {
				return pushView(NewK8sServiceAccountYAMLView(k8s, sa))
			},
		},
		OnEnter: func(sa K8sServiceAccount) tea.Cmd {
			return pushView(NewK8sServiceAccountDetailView(k8s, sa))
		},
	})
}

// NewK8sIngressesTableView creates a table view for K8s ingresses.
func NewK8sIngressesTableView(k8s *awseks.K8sClient, namespace string, pfMgr *portForwardManager) *TableView[K8sIngress] {
	keyHandlers := map[string]func(K8sIngress) tea.Cmd{
		"e": func(ing K8sIngress) tea.Cmd {
			return pushView(NewK8sIngressYAMLView(k8s, ing))
		},
	}
	if pfMgr != nil {
		keyHandlers["F"] = func(_ K8sIngress) tea.Cmd {
			return pushView(newPortForwardListView(pfMgr))
		}
	}

	return NewTableView(TableViewConfig[K8sIngress]{
		Title:       "Ingresses",
		LoadingText: "Loading ingresses...",
		Columns: []table.Column{
			{Title: "Namespace", Width: 16},
			{Title: "Name", Width: 24},
			{Title: "Class", Width: 16},
			{Title: "Hosts", Width: 30},
			{Title: "Address", Width: 30},
			{Title: "Ports", Width: 10},
			{Title: "Age", Width: 8},
		},
		FetchFunc: func(ctx context.Context) ([]K8sIngress, error) {
			return listIngresses(ctx, k8s, namespace)
		},
		RowMapper: func(ing K8sIngress) table.Row {
			return table.Row{ing.Namespace, ing.Name, ing.Class,
				ing.Hosts, ing.Address, ing.Ports, ing.Age}
		},
		CopyIDFunc:  func(ing K8sIngress) string { return ing.Namespace + "/" + ing.Name },
		KeyHandlers: keyHandlers,
		OnEnter: func(ing K8sIngress) tea.Cmd {
			return pushView(NewK8sIngressYAMLView(k8s, ing))
		},
	})
}

package services

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	awsclient "tasnim.dev/aws-tui/internal/aws"
	awsecs "tasnim.dev/aws-tui/internal/aws/ecs"
	"tasnim.dev/aws-tui/internal/tui/theme"
	"tasnim.dev/aws-tui/internal/utils"
)

// --- Clusters View ---

func NewECSClustersView(client *awsclient.ServiceClient) *TableView[awsecs.ECSCluster] {
	return NewTableView(TableViewConfig[awsecs.ECSCluster]{
		Title:       "ECS",
		LoadingText: "Loading ECS clusters...",
		Columns: []table.Column{
			{Title: "Cluster", Width: 30},
			{Title: "Status", Width: 10},
			{Title: "Services", Width: 10},
			{Title: "Tasks", Width: 10},
		},
		FetchFunc: func(ctx context.Context) ([]awsecs.ECSCluster, error) {
			return client.ECS.ListClusters(ctx)
		},
		RowMapper: func(cl awsecs.ECSCluster) table.Row {
			return table.Row{cl.Name, theme.RenderStatus(cl.Status), fmt.Sprintf("%d", cl.ServiceCount), fmt.Sprintf("%d", cl.RunningTaskCount)}
		},
		CopyIDFunc:  func(cl awsecs.ECSCluster) string { return cl.Name },
		CopyARNFunc: func(cl awsecs.ECSCluster) string { return cl.ARN },
		OnEnter: func(cl awsecs.ECSCluster) tea.Cmd {
			return pushView(NewECSServicesView(client, cl.Name))
		},
	})
}

// --- Services View ---

func NewECSServicesView(client *awsclient.ServiceClient, clusterName string) *TableView[awsecs.ECSService] {
	return NewTableView(TableViewConfig[awsecs.ECSService]{
		Title:       clusterName,
		LoadingText: "Loading services...",
		Columns: []table.Column{
			{Title: "Service", Width: 28},
			{Title: "Status", Width: 10},
			{Title: "Desired", Width: 8},
			{Title: "Running", Width: 8},
			{Title: "Pending", Width: 8},
			{Title: "Task Def", Width: 28},
		},
		FetchFunc: func(ctx context.Context) ([]awsecs.ECSService, error) {
			return client.ECS.ListServices(ctx, clusterName)
		},
		RowMapper: func(svc awsecs.ECSService) table.Row {
			return table.Row{svc.Name, theme.RenderStatus(svc.Status), fmt.Sprintf("%d", svc.DesiredCount), fmt.Sprintf("%d", svc.RunningCount), fmt.Sprintf("%d", svc.PendingCount), svc.TaskDef}
		},
		CopyIDFunc:  func(svc awsecs.ECSService) string { return svc.Name },
		CopyARNFunc: func(svc awsecs.ECSService) string { return svc.ARN },
		OnEnter: func(svc awsecs.ECSService) tea.Cmd {
			return pushView(NewECSServiceSubMenuView(client, clusterName, svc.Name, svc.ARN))
		},
	})
}

// --- Service SubMenu View ---

type ecsServiceMenuItem struct {
	name string
	desc string
}

func (i ecsServiceMenuItem) Title() string       { return i.name }
func (i ecsServiceMenuItem) Description() string { return i.desc }
func (i ecsServiceMenuItem) FilterValue() string { return i.name }

type ECSServiceSubMenuView struct {
	client      *awsclient.ServiceClient
	clusterName string
	serviceName string
	serviceARN  string
	list        list.Model
}

func NewECSServiceSubMenuView(client *awsclient.ServiceClient, clusterName, serviceName, serviceARN string) *ECSServiceSubMenuView {
	items := []list.Item{
		ecsServiceMenuItem{name: "Tasks", desc: "View running tasks for this service"},
		ecsServiceMenuItem{name: "Deployments", desc: "View deployment history"},
		ecsServiceMenuItem{name: "Events", desc: "View recent service events"},
		ecsServiceMenuItem{name: "Load Balancers", desc: "View attached load balancer targets"},
		ecsServiceMenuItem{name: "Auto Scaling", desc: "View auto scaling configuration"},
		ecsServiceMenuItem{name: "Config", desc: "View service configuration and placement"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 14)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &ECSServiceSubMenuView{
		client:      client,
		clusterName: clusterName,
		serviceName: serviceName,
		serviceARN:  serviceARN,
		list:        l,
	}
}

func (v *ECSServiceSubMenuView) Title() string { return v.serviceName }

func (v *ECSServiceSubMenuView) HelpContext() *HelpContext {
	ctx := HelpContextRoot
	return &ctx
}

func (v *ECSServiceSubMenuView) Init() tea.Cmd { return nil }
func (v *ECSServiceSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(ecsServiceMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Tasks":
				return v, pushView(NewECSTasksView(v.client, v.clusterName, v.serviceName))
			case "Deployments":
				return v, pushView(NewECSDeploymentsView(v.client, v.clusterName, v.serviceName))
			case "Events":
				return v, pushView(NewECSEventsView(v.client, v.clusterName, v.serviceName))
			case "Load Balancers":
				return v, pushView(NewECSLoadBalancersView(v.client, v.clusterName, v.serviceName))
			case "Auto Scaling":
				return v, pushView(NewECSAutoScalingView(v.client, v.clusterName, v.serviceName))
			case "Config":
				return v, pushView(NewECSServiceConfigView(v.client, v.clusterName, v.serviceName, v.serviceARN))
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *ECSServiceSubMenuView) View() string { return v.list.View() }
func (v *ECSServiceSubMenuView) SetSize(width, height int) {
	v.list.SetSize(width, height)
}

func (v *ECSServiceSubMenuView) CopyID() string  { return v.serviceName }
func (v *ECSServiceSubMenuView) CopyARN() string { return v.serviceARN }

// --- Tasks View ---

func NewECSTasksView(client *awsclient.ServiceClient, clusterName, serviceName string) *TableView[awsecs.ECSTask] {
	return NewTableView(TableViewConfig[awsecs.ECSTask]{
		Title:       "Tasks",
		LoadingText: "Loading tasks...",
		Columns: []table.Column{
			{Title: "Task ID", Width: 38},
			{Title: "Status", Width: 10},
			{Title: "Task Def", Width: 25},
			{Title: "Started", Width: 20},
			{Title: "Health", Width: 10},
		},
		FetchFunc: func(ctx context.Context) ([]awsecs.ECSTask, error) {
			return client.ECS.ListTasks(ctx, clusterName, serviceName)
		},
		RowMapper: func(t awsecs.ECSTask) table.Row {
			return table.Row{t.TaskID, theme.RenderStatus(t.Status), t.TaskDef, utils.TimeOrDash(t.StartedAt, utils.DateTime), theme.RenderStatus(t.HealthStatus)}
		},
		CopyIDFunc:  func(t awsecs.ECSTask) string { return t.TaskID },
		CopyARNFunc: func(t awsecs.ECSTask) string { return t.ARN },
		OnEnter: func(t awsecs.ECSTask) tea.Cmd {
			return pushView(NewTaskDetailView(client, clusterName, t.ARN))
		},
	})
}

// --- Deployments View ---

func NewECSDeploymentsView(client *awsclient.ServiceClient, clusterName, serviceName string) *TableView[awsecs.ECSDeployment] {
	return NewTableView(TableViewConfig[awsecs.ECSDeployment]{
		Title:       "Deployments",
		LoadingText: "Loading deployments...",
		Columns: []table.Column{
			{Title: "ID", Width: 16},
			{Title: "Status", Width: 10},
			{Title: "Task Def", Width: 24},
			{Title: "Desired", Width: 8},
			{Title: "Running", Width: 8},
			{Title: "Pending", Width: 8},
			{Title: "Rollout", Width: 14},
			{Title: "Created", Width: 18},
		},
		FetchFunc: func(ctx context.Context) ([]awsecs.ECSDeployment, error) {
			detail, err := client.ECS.DescribeService(ctx, clusterName, serviceName)
			if err != nil {
				return nil, err
			}
			return detail.Deployments, nil
		},
		RowMapper: func(d awsecs.ECSDeployment) table.Row {
			return table.Row{d.ID, theme.RenderStatus(d.Status), d.TaskDef, fmt.Sprintf("%d", d.DesiredCount), fmt.Sprintf("%d", d.RunningCount), fmt.Sprintf("%d", d.PendingCount), theme.RenderStatus(d.RolloutState), utils.TimeOrDash(d.CreatedAt, utils.DateTime)}
		},
		CopyIDFunc: func(d awsecs.ECSDeployment) string { return d.ID },
	})
}

// --- Events View ---

func NewECSEventsView(client *awsclient.ServiceClient, clusterName, serviceName string) *TableView[awsecs.ECSServiceEvent] {
	return NewTableView(TableViewConfig[awsecs.ECSServiceEvent]{
		Title:       "Events",
		LoadingText: "Loading events...",
		Columns: []table.Column{
			{Title: "Time", Width: 20},
			{Title: "Message", Width: 90},
		},
		FetchFunc: func(ctx context.Context) ([]awsecs.ECSServiceEvent, error) {
			detail, err := client.ECS.DescribeService(ctx, clusterName, serviceName)
			if err != nil {
				return nil, err
			}
			return detail.Events, nil
		},
		RowMapper: func(e awsecs.ECSServiceEvent) table.Row {
			return table.Row{utils.TimeOrDash(e.CreatedAt, utils.DateTimeSec), e.Message}
		},
		CopyIDFunc: func(e awsecs.ECSServiceEvent) string { return e.ID },
	})
}

// --- Load Balancers View ---

func NewECSLoadBalancersView(client *awsclient.ServiceClient, clusterName, serviceName string) *TableView[awsecs.ECSLoadBalancerRef] {
	return NewTableView(TableViewConfig[awsecs.ECSLoadBalancerRef]{
		Title:       "Load Balancers",
		LoadingText: "Loading load balancers...",
		Columns: []table.Column{
			{Title: "Target Group ARN", Width: 60},
			{Title: "Container", Width: 20},
			{Title: "Port", Width: 8},
		},
		FetchFunc: func(ctx context.Context) ([]awsecs.ECSLoadBalancerRef, error) {
			detail, err := client.ECS.DescribeService(ctx, clusterName, serviceName)
			if err != nil {
				return nil, err
			}
			return detail.LoadBalancers, nil
		},
		RowMapper: func(lb awsecs.ECSLoadBalancerRef) table.Row {
			return table.Row{lb.TargetGroupARN, lb.ContainerName, fmt.Sprintf("%d", lb.ContainerPort)}
		},
		CopyIDFunc: func(lb awsecs.ECSLoadBalancerRef) string { return lb.TargetGroupARN },
	})
}


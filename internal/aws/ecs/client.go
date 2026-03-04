package ecs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ECSAPI defines the subset of the ECS SDK client used by this package.
type ECSAPI interface {
	ListClusters(ctx context.Context, params *awsecs.ListClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.ListClustersOutput, error)
	DescribeClusters(ctx context.Context, params *awsecs.DescribeClustersInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeClustersOutput, error)
	ListServices(ctx context.Context, params *awsecs.ListServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.ListServicesOutput, error)
	DescribeServices(ctx context.Context, params *awsecs.DescribeServicesInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeServicesOutput, error)
	ListTasks(ctx context.Context, params *awsecs.ListTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.ListTasksOutput, error)
	DescribeTasks(ctx context.Context, params *awsecs.DescribeTasksInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTasksOutput, error)
	DescribeTaskDefinition(ctx context.Context, params *awsecs.DescribeTaskDefinitionInput, optFns ...func(*awsecs.Options)) (*awsecs.DescribeTaskDefinitionOutput, error)
}

// Client wraps an ECSAPI for higher-level operations.
type Client struct {
	api ECSAPI
}

// NewClient creates a new ECS Client.
func NewClient(api ECSAPI) *Client {
	return &Client{api: api}
}

// shortName extracts the last segment after "/" from an ARN or path.
func shortName(arn string) string {
	if parts := strings.Split(arn, "/"); len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return arn
}

// --- Domain types ---

// ECSCluster represents a single ECS cluster.
type ECSCluster struct {
	Name             string
	ARN              string
	Status           string
	RunningTaskCount int
	ServiceCount     int
}

// ECSService represents a single ECS service.
type ECSService struct {
	Name         string
	ARN          string
	Status       string
	DesiredCount int
	RunningCount int
	PendingCount int
	TaskDef      string
}

// ECSServiceDetail holds extended information about an ECS service.
type ECSServiceDetail struct {
	Name                 string
	ARN                  string
	Status               string
	DesiredCount         int
	RunningCount         int
	PendingCount         int
	TaskDef              string
	LaunchType           string
	EnableExecuteCommand bool
	Events               []ECSServiceEvent
	Deployments          []ECSDeployment
	LoadBalancers        []ECSLoadBalancerRef
	PlacementConstraints []ECSPlacementConstraint
	PlacementStrategy    []ECSPlacementStrategy
}

// ECSServiceEvent represents a service event entry.
type ECSServiceEvent struct {
	ID        string
	CreatedAt time.Time
	Message   string
}

// ECSDeployment represents a single deployment.
type ECSDeployment struct {
	ID           string
	Status       string
	TaskDef      string
	DesiredCount int
	RunningCount int
	PendingCount int
	RolloutState string
	CreatedAt    time.Time
}

// ECSLoadBalancerRef holds a load balancer reference for a service.
type ECSLoadBalancerRef struct {
	TargetGroupARN string
	ContainerName  string
	ContainerPort  int
}

// ECSPlacementConstraint holds a placement constraint.
type ECSPlacementConstraint struct {
	Type       string
	Expression string
}

// ECSPlacementStrategy holds a placement strategy.
type ECSPlacementStrategy struct {
	Type  string
	Field string
}

// ECSTask represents a single ECS task.
type ECSTask struct {
	TaskID       string
	ARN          string
	Status       string
	TaskDef      string
	StartedAt    time.Time
	HealthStatus string
}

// ECSTaskDetail holds extended information about an ECS task.
type ECSTaskDetail struct {
	TaskID      string
	TaskARN     string
	Status      string
	TaskDef     string
	StartedAt   time.Time
	StoppedAt   time.Time
	StopCode    string
	StopReason  string
	CPU         string
	Memory      string
	Containers  []ECSContainerDetail
	NetworkMode string
	PrivateIP   string
	SubnetID    string
}

// EnvVar holds a key-value environment variable.
type EnvVar struct {
	Name  string
	Value string
}

// ECSContainerDetail holds details about a single container in a task.
type ECSContainerDetail struct {
	Name         string
	Image        string
	Status       string
	ExitCode     *int
	LogGroup     string
	LogStream    string
	CPU          int
	Memory       int
	HealthStatus string
	Environment  []EnvVar
}

// --- Methods ---

// ListClusters fetches all ECS clusters across all pages.
func (c *Client) ListClusters(ctx context.Context) ([]ECSCluster, error) {
	var allARNs []string
	var nextToken *string

	for {
		listOut, err := c.api.ListClusters(ctx, &awsecs.ListClustersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListClusters: %w", err)
		}
		allARNs = append(allARNs, listOut.ClusterArns...)
		if listOut.NextToken == nil {
			break
		}
		nextToken = listOut.NextToken
	}

	if len(allARNs) == 0 {
		return nil, nil
	}

	var clusters []ECSCluster
	for i := 0; i < len(allARNs); i += 100 {
		end := min(i+100, len(allARNs))
		descOut, err := c.api.DescribeClusters(ctx, &awsecs.DescribeClustersInput{
			Clusters: allARNs[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeClusters: %w", err)
		}
		for _, cl := range descOut.Clusters {
			clusters = append(clusters, ECSCluster{
				Name:             aws.ToString(cl.ClusterName),
				ARN:              aws.ToString(cl.ClusterArn),
				Status:           aws.ToString(cl.Status),
				RunningTaskCount: int(cl.RunningTasksCount),
				ServiceCount:     int(cl.ActiveServicesCount),
			})
		}
	}
	return clusters, nil
}

// ListServices fetches all ECS services for a cluster across all pages.
func (c *Client) ListServices(ctx context.Context, clusterName string) ([]ECSService, error) {
	var allARNs []string
	var nextToken *string

	for {
		listOut, err := c.api.ListServices(ctx, &awsecs.ListServicesInput{
			Cluster:   aws.String(clusterName),
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListServices: %w", err)
		}
		allARNs = append(allARNs, listOut.ServiceArns...)
		if listOut.NextToken == nil {
			break
		}
		nextToken = listOut.NextToken
	}

	if len(allARNs) == 0 {
		return nil, nil
	}

	var services []ECSService
	for i := 0; i < len(allARNs); i += 10 {
		end := min(i+10, len(allARNs))
		descOut, err := c.api.DescribeServices(ctx, &awsecs.DescribeServicesInput{
			Cluster:  aws.String(clusterName),
			Services: allARNs[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeServices: %w", err)
		}
		for _, svc := range descOut.Services {
			taskDef := shortName(aws.ToString(svc.TaskDefinition))
			services = append(services, ECSService{
				Name:         aws.ToString(svc.ServiceName),
				ARN:          aws.ToString(svc.ServiceArn),
				Status:       aws.ToString(svc.Status),
				DesiredCount: int(svc.DesiredCount),
				RunningCount: int(svc.RunningCount),
				PendingCount: int(svc.PendingCount),
				TaskDef:      taskDef,
			})
		}
	}
	return services, nil
}

// ListTasks fetches all ECS tasks for a service across all pages.
func (c *Client) ListTasks(ctx context.Context, clusterName, serviceName string) ([]ECSTask, error) {
	var allARNs []string
	var nextToken *string

	for {
		listOut, err := c.api.ListTasks(ctx, &awsecs.ListTasksInput{
			Cluster:     aws.String(clusterName),
			ServiceName: aws.String(serviceName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("ListTasks: %w", err)
		}
		allARNs = append(allARNs, listOut.TaskArns...)
		if listOut.NextToken == nil {
			break
		}
		nextToken = listOut.NextToken
	}

	if len(allARNs) == 0 {
		return nil, nil
	}

	var tasks []ECSTask
	for i := 0; i < len(allARNs); i += 100 {
		end := min(i+100, len(allARNs))
		descOut, err := c.api.DescribeTasks(ctx, &awsecs.DescribeTasksInput{
			Cluster: aws.String(clusterName),
			Tasks:   allARNs[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeTasks: %w", err)
		}
		for _, t := range descOut.Tasks {
			taskARN := aws.ToString(t.TaskArn)
			taskID := shortName(taskARN)
			taskDef := shortName(aws.ToString(t.TaskDefinitionArn))

			var startedAt time.Time
			if t.StartedAt != nil {
				startedAt = *t.StartedAt
			}

			tasks = append(tasks, ECSTask{
				TaskID:       taskID,
				ARN:          aws.ToString(t.TaskArn),
				Status:       aws.ToString(t.LastStatus),
				TaskDef:      taskDef,
				StartedAt:    startedAt,
				HealthStatus: string(t.HealthStatus),
			})
		}
	}
	return tasks, nil
}

// extractNetworkInfo populates network fields on the task detail from ECS attachments.
func extractNetworkInfo(detail *ECSTaskDetail, attachments []ecstypes.Attachment) {
	for _, att := range attachments {
		if aws.ToString(att.Type) == "ElasticNetworkInterface" {
			for _, kv := range att.Details {
				switch aws.ToString(kv.Name) {
				case "privateIPv4Address":
					detail.PrivateIP = aws.ToString(kv.Value)
				case "subnetId":
					detail.SubnetID = aws.ToString(kv.Value)
				case "networkInterfaceId":
					detail.NetworkMode = "awsvpc"
				}
			}
		}
	}
}

// buildContainerDetails constructs container detail entries from ECS containers and task definition metadata.
func buildContainerDetails(containers []ecstypes.Container, logConfigs map[string][2]string, envConfigs map[string][]EnvVar, taskID string) []ECSContainerDetail {
	details := make([]ECSContainerDetail, len(containers))
	for i, c := range containers {
		var exitCode *int
		if c.ExitCode != nil {
			ec := int(*c.ExitCode)
			exitCode = &ec
		}

		cd := ECSContainerDetail{
			Name:         aws.ToString(c.Name),
			Image:        aws.ToString(c.Image),
			Status:       aws.ToString(c.LastStatus),
			ExitCode:     exitCode,
			HealthStatus: string(c.HealthStatus),
		}

		if lc, ok := logConfigs[cd.Name]; ok {
			cd.LogGroup = lc[0]
			if lc[1] != "" {
				cd.LogStream = lc[1] + "/" + cd.Name + "/" + taskID
			}
		}
		if envs, ok := envConfigs[cd.Name]; ok {
			cd.Environment = envs
		}

		details[i] = cd
	}
	return details
}

// DescribeTask fetches extended details for a single ECS task.
func (c *Client) DescribeTask(ctx context.Context, clusterName, taskARN string) (*ECSTaskDetail, error) {
	descOut, err := c.api.DescribeTasks(ctx, &awsecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   []string{taskARN},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeTasks: %w", err)
	}
	if len(descOut.Tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskARN)
	}

	t := descOut.Tasks[0]

	taskID := shortName(aws.ToString(t.TaskArn))
	taskDef := aws.ToString(t.TaskDefinitionArn)
	taskDefShort := shortName(taskDef)

	var startedAt, stoppedAt time.Time
	if t.StartedAt != nil {
		startedAt = *t.StartedAt
	}
	if t.StoppedAt != nil {
		stoppedAt = *t.StoppedAt
	}

	detail := &ECSTaskDetail{
		TaskID:     taskID,
		TaskARN:    aws.ToString(t.TaskArn),
		Status:     aws.ToString(t.LastStatus),
		TaskDef:    taskDefShort,
		StartedAt:  startedAt,
		StoppedAt:  stoppedAt,
		StopCode:   string(t.StopCode),
		StopReason: aws.ToString(t.StoppedReason),
		CPU:        aws.ToString(t.Cpu),
		Memory:     aws.ToString(t.Memory),
	}

	extractNetworkInfo(detail, t.Attachments)

	// Get log configuration and environment from task definition
	logConfigs := map[string][2]string{}
	envConfigs := map[string][]EnvVar{}
	tdCtx, tdCancel := context.WithTimeout(ctx, 15*time.Second)
	defer tdCancel()
	tdOut, err := c.api.DescribeTaskDefinition(tdCtx, &awsecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDef),
	})
	if err == nil && tdOut.TaskDefinition != nil {
		for _, cd := range tdOut.TaskDefinition.ContainerDefinitions {
			name := aws.ToString(cd.Name)
			if cd.LogConfiguration != nil && cd.LogConfiguration.Options != nil {
				logConfigs[name] = [2]string{
					cd.LogConfiguration.Options["awslogs-group"],
					cd.LogConfiguration.Options["awslogs-stream-prefix"],
				}
			}
			for _, ev := range cd.Environment {
				envConfigs[name] = append(envConfigs[name], EnvVar{
					Name:  aws.ToString(ev.Name),
					Value: aws.ToString(ev.Value),
				})
			}
		}
	}

	detail.Containers = buildContainerDetails(t.Containers, logConfigs, envConfigs, taskID)

	return detail, nil
}

// DescribeService fetches extended details for a single ECS service.
func (c *Client) DescribeService(ctx context.Context, clusterName, serviceName string) (*ECSServiceDetail, error) {
	descOut, err := c.api.DescribeServices(ctx, &awsecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeServices: %w", err)
	}
	if len(descOut.Services) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	svc := descOut.Services[0]

	taskDef := shortName(aws.ToString(svc.TaskDefinition))

	detail := &ECSServiceDetail{
		Name:                 aws.ToString(svc.ServiceName),
		ARN:                  aws.ToString(svc.ServiceArn),
		Status:               aws.ToString(svc.Status),
		DesiredCount:         int(svc.DesiredCount),
		RunningCount:         int(svc.RunningCount),
		PendingCount:         int(svc.PendingCount),
		TaskDef:              taskDef,
		LaunchType:           string(svc.LaunchType),
		EnableExecuteCommand: svc.EnableExecuteCommand,
	}

	for _, e := range svc.Events {
		var createdAt time.Time
		if e.CreatedAt != nil {
			createdAt = *e.CreatedAt
		}
		detail.Events = append(detail.Events, ECSServiceEvent{
			ID:        aws.ToString(e.Id),
			CreatedAt: createdAt,
			Message:   aws.ToString(e.Message),
		})
	}

	for _, d := range svc.Deployments {
		depTaskDef := shortName(aws.ToString(d.TaskDefinition))
		var createdAt time.Time
		if d.CreatedAt != nil {
			createdAt = *d.CreatedAt
		}
		detail.Deployments = append(detail.Deployments, ECSDeployment{
			ID:           aws.ToString(d.Id),
			Status:       aws.ToString(d.Status),
			TaskDef:      depTaskDef,
			DesiredCount: int(d.DesiredCount),
			RunningCount: int(d.RunningCount),
			PendingCount: int(d.PendingCount),
			RolloutState: string(d.RolloutState),
			CreatedAt:    createdAt,
		})
	}

	for _, lb := range svc.LoadBalancers {
		detail.LoadBalancers = append(detail.LoadBalancers, ECSLoadBalancerRef{
			TargetGroupARN: aws.ToString(lb.TargetGroupArn),
			ContainerName:  aws.ToString(lb.ContainerName),
			ContainerPort:  int(aws.ToInt32(lb.ContainerPort)),
		})
	}

	for _, pc := range svc.PlacementConstraints {
		detail.PlacementConstraints = append(detail.PlacementConstraints, ECSPlacementConstraint{
			Type:       string(pc.Type),
			Expression: aws.ToString(pc.Expression),
		})
	}

	for _, ps := range svc.PlacementStrategy {
		detail.PlacementStrategy = append(detail.PlacementStrategy, ECSPlacementStrategy{
			Type:  string(ps.Type),
			Field: aws.ToString(ps.Field),
		})
	}

	return detail, nil
}

// ListClustersPage fetches a single page of ECS cluster ARNs and describes them.
func (c *Client) ListClustersPage(ctx context.Context, token *string) ([]ECSCluster, *string, error) {
	listOut, err := c.api.ListClusters(ctx, &awsecs.ListClustersInput{
		NextToken: token,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ListClusters: %w", err)
	}

	if len(listOut.ClusterArns) == 0 {
		return nil, nil, nil
	}

	descOut, err := c.api.DescribeClusters(ctx, &awsecs.DescribeClustersInput{
		Clusters: listOut.ClusterArns,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("DescribeClusters: %w", err)
	}

	clusters := make([]ECSCluster, 0, len(descOut.Clusters))
	for _, cl := range descOut.Clusters {
		clusters = append(clusters, ECSCluster{
			Name:             aws.ToString(cl.ClusterName),
			ARN:              aws.ToString(cl.ClusterArn),
			Status:           aws.ToString(cl.Status),
			RunningTaskCount: int(cl.RunningTasksCount),
			ServiceCount:     int(cl.ActiveServicesCount),
		})
	}
	return clusters, listOut.NextToken, nil
}

// ListServicesPage fetches a single page of ECS services and describes them.
func (c *Client) ListServicesPage(ctx context.Context, clusterName string, token *string) ([]ECSService, *string, error) {
	listOut, err := c.api.ListServices(ctx, &awsecs.ListServicesInput{
		Cluster:   aws.String(clusterName),
		NextToken: token,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ListServices: %w", err)
	}

	if len(listOut.ServiceArns) == 0 {
		return nil, nil, nil
	}

	// DescribeServices supports max 10 at a time
	var services []ECSService
	for i := 0; i < len(listOut.ServiceArns); i += 10 {
		end := min(i+10, len(listOut.ServiceArns))
		descOut, err := c.api.DescribeServices(ctx, &awsecs.DescribeServicesInput{
			Cluster:  aws.String(clusterName),
			Services: listOut.ServiceArns[i:end],
		})
		if err != nil {
			return nil, nil, fmt.Errorf("DescribeServices: %w", err)
		}
		for _, svc := range descOut.Services {
			taskDef := shortName(aws.ToString(svc.TaskDefinition))
			services = append(services, ECSService{
				Name:         aws.ToString(svc.ServiceName),
				ARN:          aws.ToString(svc.ServiceArn),
				Status:       aws.ToString(svc.Status),
				DesiredCount: int(svc.DesiredCount),
				RunningCount: int(svc.RunningCount),
				PendingCount: int(svc.PendingCount),
				TaskDef:      taskDef,
			})
		}
	}
	return services, listOut.NextToken, nil
}

// ListTasksPage fetches a single page of ECS tasks and describes them.
func (c *Client) ListTasksPage(ctx context.Context, clusterName, serviceName string, token *string) ([]ECSTask, *string, error) {
	listOut, err := c.api.ListTasks(ctx, &awsecs.ListTasksInput{
		Cluster:     aws.String(clusterName),
		ServiceName: aws.String(serviceName),
		NextToken:   token,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ListTasks: %w", err)
	}

	if len(listOut.TaskArns) == 0 {
		return nil, nil, nil
	}

	descOut, err := c.api.DescribeTasks(ctx, &awsecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   listOut.TaskArns,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("DescribeTasks: %w", err)
	}

	tasks := make([]ECSTask, 0, len(descOut.Tasks))
	for _, t := range descOut.Tasks {
		taskARN := aws.ToString(t.TaskArn)
		taskID := shortName(taskARN)
		taskDef := shortName(aws.ToString(t.TaskDefinitionArn))

		var startedAt time.Time
		if t.StartedAt != nil {
			startedAt = *t.StartedAt
		}

		tasks = append(tasks, ECSTask{
			TaskID:       taskID,
			ARN:          aws.ToString(t.TaskArn),
			Status:       aws.ToString(t.LastStatus),
			TaskDef:      taskDef,
			StartedAt:    startedAt,
			HealthStatus: string(t.HealthStatus),
		})
	}
	return tasks, listOut.NextToken, nil
}

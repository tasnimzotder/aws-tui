# Services Browser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `aws-utils services --profile <name> --region <region>` — a Bubble Tea TUI that lets you browse EC2, ECS, VPC, and ECR resources via a stack-based drill-down interface with breadcrumb navigation.

**Architecture:** A `View` interface defines each navigation level (Title, View, Update, Init). A `ViewStack` manages push/pop navigation. Each AWS service has its own client file (with interface for testing) and its own TUI view file. Data is lazy-loaded — only fetched when the user drills into a level.

**Tech Stack:** Go 1.25, cobra, bubbletea, bubbles (list, table, spinner), lipgloss, aws-sdk-go-v2 (ec2, ecs, ecr)

---

### Task 1: Install New AWS SDK Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Install EC2, ECS, and ECR service packages**

Run:
```bash
go get github.com/aws/aws-sdk-go-v2/service/ec2@latest
go get github.com/aws/aws-sdk-go-v2/service/ecs@latest
go get github.com/aws/aws-sdk-go-v2/service/ecr@latest
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add EC2, ECS, ECR SDK dependencies"
```

---

### Task 2: AWS Data Types for All Services

**Files:**
- Create: `internal/aws/service_types.go`

**Step 1: Create all service data types**

Create `internal/aws/service_types.go`:

```go
package aws

import "time"

// EC2

type EC2Instance struct {
	Name       string
	InstanceID string
	Type       string
	State      string
	PrivateIP  string
	PublicIP   string
}

type EC2Summary struct {
	Total   int
	Running int
	Stopped int
}

// ECS

type ECSCluster struct {
	Name             string
	Status           string
	RunningTaskCount int
	ServiceCount     int
}

type ECSService struct {
	Name         string
	Status       string
	DesiredCount int
	RunningCount int
	TaskDef      string
}

type ECSTask struct {
	TaskID       string
	Status       string
	TaskDef      string
	StartedAt    time.Time
	HealthStatus string
}

// VPC (uses EC2 API)

type VPCInfo struct {
	VPCID     string
	Name      string
	CIDR      string
	IsDefault bool
	State     string
}

type SubnetInfo struct {
	SubnetID     string
	Name         string
	CIDR         string
	AZ           string
	AvailableIPs int
}

type SecurityGroupInfo struct {
	GroupID       string
	Name          string
	Description   string
	InboundRules  int
	OutboundRules int
}

type InternetGatewayInfo struct {
	GatewayID string
	Name      string
	State     string
}

// ECR

type ECRRepo struct {
	Name       string
	URI        string
	ImageCount int
	CreatedAt  time.Time
}

type ECRImage struct {
	Tags     []string
	Digest   string
	SizeMB   float64
	PushedAt time.Time
}
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/aws/service_types.go
git commit -m "feat: add data types for EC2, ECS, VPC, ECR"
```

---

### Task 3: EC2 Client

**Files:**
- Create: `internal/aws/ec2.go`
- Create: `internal/aws/ec2_test.go`

**Step 1: Create EC2 client**

Create `internal/aws/ec2.go`:

```go
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type EC2API interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

type EC2Client struct {
	api EC2API
}

func NewEC2Client(api EC2API) *EC2Client {
	return &EC2Client{api: api}
}

func (c *EC2Client) ListInstances(ctx context.Context) ([]EC2Instance, EC2Summary, error) {
	out, err := c.api.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, EC2Summary{}, fmt.Errorf("DescribeInstances: %w", err)
	}

	var instances []EC2Instance
	var summary EC2Summary

	for _, reservation := range out.Reservations {
		for _, inst := range reservation.Instances {
			name := ""
			for _, tag := range inst.Tags {
				if aws.ToString(tag.Key) == "Name" {
					name = aws.ToString(tag.Value)
					break
				}
			}

			publicIP := "—"
			if inst.PublicIpAddress != nil {
				publicIP = aws.ToString(inst.PublicIpAddress)
			}

			state := string(inst.State.Name)
			instances = append(instances, EC2Instance{
				Name:       name,
				InstanceID: aws.ToString(inst.InstanceId),
				Type:       string(inst.InstanceType),
				State:      state,
				PrivateIP:  aws.ToString(inst.PrivateIpAddress),
				PublicIP:   publicIP,
			})

			summary.Total++
			switch inst.State.Name {
			case types.InstanceStateNameRunning:
				summary.Running++
			case types.InstanceStateNameStopped:
				summary.Stopped++
			}
		}
	}

	return instances, summary, nil
}
```

**Step 2: Write tests**

Create `internal/aws/ec2_test.go`:

```go
package aws

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2API struct {
	describeInstancesFunc func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

func (m *mockEC2API) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.describeInstancesFunc(ctx, params, optFns...)
}

func TestListInstances(t *testing.T) {
	mock := &mockEC2API{
		describeInstancesFunc: func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
			return &ec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{
					{
						Instances: []types.Instance{
							{
								InstanceId:       awssdk.String("i-abc123"),
								InstanceType:     types.InstanceTypeT3Medium,
								State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
								PrivateIpAddress: awssdk.String("10.0.1.50"),
								PublicIpAddress:   awssdk.String("54.21.3.100"),
								Tags: []types.Tag{
									{Key: awssdk.String("Name"), Value: awssdk.String("web-server")},
								},
							},
							{
								InstanceId:       awssdk.String("i-def456"),
								InstanceType:     types.InstanceTypeT3Large,
								State:            &types.InstanceState{Name: types.InstanceStateNameStopped},
								PrivateIpAddress: awssdk.String("10.0.2.30"),
								Tags: []types.Tag{
									{Key: awssdk.String("Name"), Value: awssdk.String("api-server")},
								},
							},
						},
					},
				},
			}, nil
		},
	}

	client := NewEC2Client(mock)
	instances, summary, err := client.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}
	if instances[0].Name != "web-server" {
		t.Errorf("instances[0].Name = %s, want web-server", instances[0].Name)
	}
	if instances[0].PublicIP != "54.21.3.100" {
		t.Errorf("instances[0].PublicIP = %s, want 54.21.3.100", instances[0].PublicIP)
	}
	if instances[1].PublicIP != "—" {
		t.Errorf("instances[1].PublicIP = %s, want —", instances[1].PublicIP)
	}
	if summary.Total != 2 {
		t.Errorf("summary.Total = %d, want 2", summary.Total)
	}
	if summary.Running != 1 {
		t.Errorf("summary.Running = %d, want 1", summary.Running)
	}
	if summary.Stopped != 1 {
		t.Errorf("summary.Stopped = %d, want 1", summary.Stopped)
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/aws/ -v -run TestListInstances`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/aws/ec2.go internal/aws/ec2_test.go
git commit -m "feat: add EC2 client with ListInstances"
```

---

### Task 4: ECS Client

**Files:**
- Create: `internal/aws/ecs.go`
- Create: `internal/aws/ecs_test.go`

**Step 1: Create ECS client**

Create `internal/aws/ecs.go`:

```go
package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ECSAPI interface {
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
	ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error)
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

type ECSClient struct {
	api ECSAPI
}

func NewECSClient(api ECSAPI) *ECSClient {
	return &ECSClient{api: api}
}

func (c *ECSClient) ListClusters(ctx context.Context) ([]ECSCluster, error) {
	listOut, err := c.api.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("ListClusters: %w", err)
	}

	if len(listOut.ClusterArns) == 0 {
		return nil, nil
	}

	descOut, err := c.api.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: listOut.ClusterArns,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeClusters: %w", err)
	}

	clusters := make([]ECSCluster, len(descOut.Clusters))
	for i, cl := range descOut.Clusters {
		clusters[i] = ECSCluster{
			Name:             awssdk.ToString(cl.ClusterName),
			Status:           awssdk.ToString(cl.Status),
			RunningTaskCount: int(cl.RunningTasksCount),
			ServiceCount:     int(cl.ActiveServicesCount),
		}
	}
	return clusters, nil
}

func (c *ECSClient) ListServices(ctx context.Context, clusterName string) ([]ECSService, error) {
	listOut, err := c.api.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: awssdk.String(clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("ListServices: %w", err)
	}

	if len(listOut.ServiceArns) == 0 {
		return nil, nil
	}

	descOut, err := c.api.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  awssdk.String(clusterName),
		Services: listOut.ServiceArns,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeServices: %w", err)
	}

	services := make([]ECSService, len(descOut.Services))
	for i, svc := range descOut.Services {
		taskDef := awssdk.ToString(svc.TaskDefinition)
		// Shorten ARN to family:revision
		if parts := strings.Split(taskDef, "/"); len(parts) > 1 {
			taskDef = parts[len(parts)-1]
		}
		services[i] = ECSService{
			Name:         awssdk.ToString(svc.ServiceName),
			Status:       awssdk.ToString(svc.Status),
			DesiredCount: int(svc.DesiredCount),
			RunningCount: int(svc.RunningCount),
			TaskDef:      taskDef,
		}
	}
	return services, nil
}

func (c *ECSClient) ListTasks(ctx context.Context, clusterName, serviceName string) ([]ECSTask, error) {
	listOut, err := c.api.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     awssdk.String(clusterName),
		ServiceName: awssdk.String(serviceName),
	})
	if err != nil {
		return nil, fmt.Errorf("ListTasks: %w", err)
	}

	if len(listOut.TaskArns) == 0 {
		return nil, nil
	}

	descOut, err := c.api.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: awssdk.String(clusterName),
		Tasks:   listOut.TaskArns,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeTasks: %w", err)
	}

	tasks := make([]ECSTask, len(descOut.Tasks))
	for i, t := range descOut.Tasks {
		taskARN := awssdk.ToString(t.TaskArn)
		taskID := taskARN
		if parts := strings.Split(taskARN, "/"); len(parts) > 2 {
			taskID = parts[len(parts)-1]
		}

		taskDef := awssdk.ToString(t.TaskDefinitionArn)
		if parts := strings.Split(taskDef, "/"); len(parts) > 1 {
			taskDef = parts[len(parts)-1]
		}

		var startedAt time.Time
		if t.StartedAt != nil {
			startedAt = *t.StartedAt
		}

		tasks[i] = ECSTask{
			TaskID:       taskID,
			Status:       awssdk.ToString(t.LastStatus),
			TaskDef:      taskDef,
			StartedAt:    startedAt,
			HealthStatus: string(t.HealthStatus),
		}
	}
	return tasks, nil
}
```

**Step 2: Write test for ListClusters**

Create `internal/aws/ecs_test.go`:

```go
package aws

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type mockECSAPI struct {
	listClustersFunc    func(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	describeClustersFunc func(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
	listServicesFunc    func(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error)
	describeServicesFunc func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	listTasksFunc       func(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	describeTasksFunc   func(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

func (m *mockECSAPI) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return m.listClustersFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
	return m.describeClustersFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) ListServices(ctx context.Context, params *ecs.ListServicesInput, optFns ...func(*ecs.Options)) (*ecs.ListServicesOutput, error) {
	return m.listServicesFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return m.describeServicesFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	return m.listTasksFunc(ctx, params, optFns...)
}
func (m *mockECSAPI) DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	return m.describeTasksFunc(ctx, params, optFns...)
}

func TestListClusters(t *testing.T) {
	mock := &mockECSAPI{
		listClustersFunc: func(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
			return &ecs.ListClustersOutput{
				ClusterArns: []string{"arn:aws:ecs:us-east-1:123456:cluster/prod"},
			}, nil
		},
		describeClustersFunc: func(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
			return &ecs.DescribeClustersOutput{
				Clusters: []ecstypes.Cluster{
					{
						ClusterName:       awssdk.String("prod"),
						Status:            awssdk.String("ACTIVE"),
						RunningTasksCount: 12,
						ActiveServicesCount: 5,
					},
				},
			}, nil
		},
	}

	client := NewECSClient(mock)
	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Name != "prod" {
		t.Errorf("Name = %s, want prod", clusters[0].Name)
	}
	if clusters[0].RunningTaskCount != 12 {
		t.Errorf("RunningTaskCount = %d, want 12", clusters[0].RunningTaskCount)
	}
	if clusters[0].ServiceCount != 5 {
		t.Errorf("ServiceCount = %d, want 5", clusters[0].ServiceCount)
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/aws/ -v -run TestListClusters`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/aws/ecs.go internal/aws/ecs_test.go
git commit -m "feat: add ECS client with ListClusters, ListServices, ListTasks"
```

---

### Task 5: VPC Client

**Files:**
- Create: `internal/aws/vpc.go`
- Create: `internal/aws/vpc_test.go`

**Step 1: Create VPC client**

Create `internal/aws/vpc.go`:

```go
package aws

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type VPCAPI interface {
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
}

type VPCClient struct {
	api VPCAPI
}

func NewVPCClient(api VPCAPI) *VPCClient {
	return &VPCClient{api: api}
}

func nameFromTags(tags []types.Tag) string {
	for _, tag := range tags {
		if awssdk.ToString(tag.Key) == "Name" {
			return awssdk.ToString(tag.Value)
		}
	}
	return ""
}

func (c *VPCClient) ListVPCs(ctx context.Context) ([]VPCInfo, error) {
	out, err := c.api.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, fmt.Errorf("DescribeVpcs: %w", err)
	}

	vpcs := make([]VPCInfo, len(out.Vpcs))
	for i, v := range out.Vpcs {
		vpcs[i] = VPCInfo{
			VPCID:     awssdk.ToString(v.VpcId),
			Name:      nameFromTags(v.Tags),
			CIDR:      awssdk.ToString(v.CidrBlock),
			IsDefault: awssdk.ToBool(v.IsDefault),
			State:     string(v.State),
		}
	}
	return vpcs, nil
}

func (c *VPCClient) ListSubnets(ctx context.Context, vpcID string) ([]SubnetInfo, error) {
	out, err := c.api.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeSubnets: %w", err)
	}

	subnets := make([]SubnetInfo, len(out.Subnets))
	for i, s := range out.Subnets {
		subnets[i] = SubnetInfo{
			SubnetID:     awssdk.ToString(s.SubnetId),
			Name:         nameFromTags(s.Tags),
			CIDR:         awssdk.ToString(s.CidrBlock),
			AZ:           awssdk.ToString(s.AvailabilityZone),
			AvailableIPs: int(awssdk.ToInt32(s.AvailableIpAddressCount)),
		}
	}
	return subnets, nil
}

func (c *VPCClient) ListSecurityGroups(ctx context.Context, vpcID string) ([]SecurityGroupInfo, error) {
	out, err := c.api.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeSecurityGroups: %w", err)
	}

	sgs := make([]SecurityGroupInfo, len(out.SecurityGroups))
	for i, sg := range out.SecurityGroups {
		sgs[i] = SecurityGroupInfo{
			GroupID:       awssdk.ToString(sg.GroupId),
			Name:          awssdk.ToString(sg.GroupName),
			Description:   awssdk.ToString(sg.Description),
			InboundRules:  len(sg.IpPermissions),
			OutboundRules: len(sg.IpPermissionsEgress),
		}
	}
	return sgs, nil
}

func (c *VPCClient) ListInternetGateways(ctx context.Context, vpcID string) ([]InternetGatewayInfo, error) {
	out, err := c.api.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{Name: awssdk.String("attachment.vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeInternetGateways: %w", err)
	}

	igws := make([]InternetGatewayInfo, len(out.InternetGateways))
	for i, igw := range out.InternetGateways {
		state := "detached"
		for _, att := range igw.Attachments {
			if awssdk.ToString(att.VpcId) == vpcID {
				state = string(att.State)
				break
			}
		}
		igws[i] = InternetGatewayInfo{
			GatewayID: awssdk.ToString(igw.InternetGatewayId),
			Name:      nameFromTags(igw.Tags),
			State:     state,
		}
	}
	return igws, nil
}
```

**Step 2: Write test for ListVPCs**

Create `internal/aws/vpc_test.go`:

```go
package aws

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockVPCAPI struct {
	describeVpcsFunc             func(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	describeSubnetsFunc          func(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	describeSecurityGroupsFunc   func(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	describeInternetGatewaysFunc func(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
}

func (m *mockVPCAPI) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return m.describeVpcsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return m.describeSubnetsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return m.describeSecurityGroupsFunc(ctx, params, optFns...)
}
func (m *mockVPCAPI) DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
	return m.describeInternetGatewaysFunc(ctx, params, optFns...)
}

func TestListVPCs(t *testing.T) {
	mock := &mockVPCAPI{
		describeVpcsFunc: func(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
			return &ec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{
					{
						VpcId:     awssdk.String("vpc-abc123"),
						CidrBlock: awssdk.String("10.0.0.0/16"),
						IsDefault: awssdk.Bool(false),
						State:     types.VpcStateAvailable,
						Tags: []types.Tag{
							{Key: awssdk.String("Name"), Value: awssdk.String("prod-vpc")},
						},
					},
				},
			}, nil
		},
	}

	client := NewVPCClient(mock)
	vpcs, err := client.ListVPCs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vpcs) != 1 {
		t.Fatalf("expected 1 VPC, got %d", len(vpcs))
	}
	if vpcs[0].Name != "prod-vpc" {
		t.Errorf("Name = %s, want prod-vpc", vpcs[0].Name)
	}
	if vpcs[0].CIDR != "10.0.0.0/16" {
		t.Errorf("CIDR = %s, want 10.0.0.0/16", vpcs[0].CIDR)
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/aws/ -v -run TestListVPCs`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/aws/vpc.go internal/aws/vpc_test.go
git commit -m "feat: add VPC client with ListVPCs, ListSubnets, ListSecurityGroups, ListInternetGateways"
```

---

### Task 6: ECR Client

**Files:**
- Create: `internal/aws/ecr.go`
- Create: `internal/aws/ecr_test.go`

**Step 1: Create ECR client**

Create `internal/aws/ecr.go`:

```go
package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

type ECRAPI interface {
	DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	DescribeImages(ctx context.Context, params *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeImagesOutput, error)
}

type ECRClient struct {
	api ECRAPI
}

func NewECRClient(api ECRAPI) *ECRClient {
	return &ECRClient{api: api}
}

func (c *ECRClient) ListRepositories(ctx context.Context) ([]ECRRepo, error) {
	out, err := c.api.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
	if err != nil {
		return nil, fmt.Errorf("DescribeRepositories: %w", err)
	}

	repos := make([]ECRRepo, len(out.Repositories))
	for i, r := range out.Repositories {
		var createdAt time.Time
		if r.CreatedAt != nil {
			createdAt = *r.CreatedAt
		}
		repos[i] = ECRRepo{
			Name:      awssdk.ToString(r.RepositoryName),
			URI:       awssdk.ToString(r.RepositoryUri),
			CreatedAt: createdAt,
		}
	}

	// Get image counts per repo
	for i, repo := range repos {
		imgOut, err := c.api.DescribeImages(ctx, &ecr.DescribeImagesInput{
			RepositoryName: awssdk.String(repo.Name),
		})
		if err == nil {
			repos[i].ImageCount = len(imgOut.ImageDetails)
		}
	}

	return repos, nil
}

func (c *ECRClient) ListImages(ctx context.Context, repoName string) ([]ECRImage, error) {
	out, err := c.api.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RepositoryName: awssdk.String(repoName),
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeImages: %w", err)
	}

	images := make([]ECRImage, len(out.ImageDetails))
	for i, img := range out.ImageDetails {
		digest := awssdk.ToString(img.ImageDigest)
		// Shorten digest: sha256:abcdef... → sha256:abcdef
		if parts := strings.SplitN(digest, ":", 2); len(parts) == 2 && len(parts[1]) > 12 {
			digest = parts[0] + ":" + parts[1][:12]
		}

		var pushedAt time.Time
		if img.ImagePushedAt != nil {
			pushedAt = *img.ImagePushedAt
		}

		var sizeMB float64
		if img.ImageSizeInBytes != nil {
			sizeMB = float64(*img.ImageSizeInBytes) / (1024 * 1024)
		}

		images[i] = ECRImage{
			Tags:     img.ImageTags,
			Digest:   digest,
			SizeMB:   sizeMB,
			PushedAt: pushedAt,
		}
	}

	// Sort by pushed date descending (newest first)
	sort.Slice(images, func(i, j int) bool {
		return images[i].PushedAt.After(images[j].PushedAt)
	})

	return images, nil
}
```

**Step 2: Write test**

Create `internal/aws/ecr_test.go`:

```go
package aws

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type mockECRAPI struct {
	describeRepositoriesFunc func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	describeImagesFunc       func(ctx context.Context, params *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeImagesOutput, error)
}

func (m *mockECRAPI) DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	return m.describeRepositoriesFunc(ctx, params, optFns...)
}
func (m *mockECRAPI) DescribeImages(ctx context.Context, params *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeImagesOutput, error) {
	return m.describeImagesFunc(ctx, params, optFns...)
}

func TestListRepositories(t *testing.T) {
	created := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	mock := &mockECRAPI{
		describeRepositoriesFunc: func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
			return &ecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{
					{
						RepositoryName: awssdk.String("my-app"),
						RepositoryUri:  awssdk.String("123456.dkr.ecr.us-east-1.amazonaws.com/my-app"),
						CreatedAt:      &created,
					},
				},
			}, nil
		},
		describeImagesFunc: func(ctx context.Context, params *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeImagesOutput, error) {
			return &ecr.DescribeImagesOutput{
				ImageDetails: []ecrtypes.ImageDetail{
					{ImageTags: []string{"latest"}},
					{ImageTags: []string{"v1.0"}},
				},
			}, nil
		},
	}

	client := NewECRClient(mock)
	repos, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Name != "my-app" {
		t.Errorf("Name = %s, want my-app", repos[0].Name)
	}
	if repos[0].ImageCount != 2 {
		t.Errorf("ImageCount = %d, want 2", repos[0].ImageCount)
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/aws/ -v -run TestListRepositories`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/aws/ecr.go internal/aws/ecr_test.go
git commit -m "feat: add ECR client with ListRepositories, ListImages"
```

---

### Task 7: ServiceClient Factory + Region Support

**Files:**
- Create: `internal/aws/services.go`

**Step 1: Create ServiceClient factory**

Create `internal/aws/services.go`:

```go
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ServiceClient struct {
	EC2 *EC2Client
	ECS *ECSClient
	VPC *VPCClient
	ECR *ECRClient
}

func NewServiceClient(ctx context.Context, profile, region string) (*ServiceClient, error) {
	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	return &ServiceClient{
		EC2: NewEC2Client(ec2Client),
		ECS: NewECSClient(ecs.NewFromConfig(cfg)),
		VPC: NewVPCClient(ec2Client),
		ECR: NewECRClient(ecr.NewFromConfig(cfg)),
	}, nil
}
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/aws/services.go
git commit -m "feat: add ServiceClient factory with profile and region support"
```

---

### Task 8: View Interface + ViewStack + Breadcrumb

**Files:**
- Create: `internal/tui/services/view.go`
- Create: `internal/tui/services/model.go`

**Step 1: Create View interface and ViewStack**

Create `internal/tui/services/view.go`:

```go
package services

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents a navigable screen in the services browser.
type View interface {
	Title() string
	View() string
	Update(msg tea.Msg) (View, tea.Cmd)
	Init() tea.Cmd
}

// PushViewMsg signals the model to push a new view onto the stack.
type PushViewMsg struct{ View View }

// PopViewMsg signals the model to pop the current view.
type PopViewMsg struct{}

var (
	breadcrumbStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	breadcrumbSepStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	svcHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("#6B7280")).
		Padding(0, 1)

	svcDashboardStyle = lipgloss.NewStyle().
		Padding(1, 2)

	svcHelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Padding(1, 0, 0, 0)

	svcProfileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA"))

	svcErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true)

	svcMutedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	svcSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)
)

func renderBreadcrumb(titles []string) string {
	parts := make([]string, len(titles))
	for i, t := range titles {
		parts[i] = breadcrumbStyle.Render(t)
	}
	return strings.Join(parts, breadcrumbSepStyle.Render(" › "))
}
```

**Step 2: Create services model with ViewStack**

Create `internal/tui/services/model.go`:

```go
package services

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

// Model is the root Bubble Tea model for the services browser.
type Model struct {
	client  *awsclient.ServiceClient
	profile string
	region  string
	stack   []View
}

// NewModel creates a new services browser model.
func NewModel(client *awsclient.ServiceClient, profile, region string) Model {
	root := NewRootView(client)
	return Model{
		client:  client,
		profile: profile,
		region:  region,
		stack:   []View{root},
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.stack) > 0 {
		return m.stack[len(m.stack)-1].Init()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
				return m, nil
			}
			return m, tea.Quit
		}

	case PushViewMsg:
		m.stack = append(m.stack, msg.View)
		return m, msg.View.Init()

	case PopViewMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, nil
	}

	// Delegate to current view
	if len(m.stack) > 0 {
		current := m.stack[len(m.stack)-1]
		updated, cmd := current.Update(msg)
		m.stack[len(m.stack)-1] = updated
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	// Build breadcrumb
	titles := make([]string, len(m.stack))
	for i, v := range m.stack {
		titles[i] = v.Title()
	}
	breadcrumb := renderBreadcrumb(titles)

	// Profile and region info
	profileText := "default"
	if m.profile != "" {
		profileText = m.profile
	}
	regionText := "default"
	if m.region != "" {
		regionText = m.region
	}
	info := svcProfileStyle.Render(fmt.Sprintf("profile: %s  region: %s", profileText, regionText))

	header := lipgloss.JoinHorizontal(lipgloss.Top, breadcrumb, "   ", info)

	// Current view content
	content := ""
	if len(m.stack) > 0 {
		content = m.stack[len(m.stack)-1].View()
	}

	// Help
	helpText := "Esc to go back • r to refresh • q to quit"
	if len(m.stack) <= 1 {
		helpText = "Enter to select • q to quit"
	}
	help := svcHelpStyle.Render(helpText)

	return svcDashboardStyle.Render(
		svcHeaderStyle.Render(header) + "\n\n" +
			content + "\n" +
			help,
	)
}
```

**Step 3: Verify build**

Run: `go build ./...`
Expected: No errors (will fail because `NewRootView` doesn't exist yet — create a stub in the next task)

Note: This task and Task 9 should be committed together if needed to compile.

---

### Task 9: Root Service List View

**Files:**
- Create: `internal/tui/services/root.go`

**Step 1: Create root service list view**

Create `internal/tui/services/root.go`:

```go
package services

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

type serviceItem struct {
	name string
	desc string
}

func (i serviceItem) Title() string       { return i.name }
func (i serviceItem) Description() string { return i.desc }
func (i serviceItem) FilterValue() string { return i.name }

type RootView struct {
	client *awsclient.ServiceClient
	list   list.Model
}

func NewRootView(client *awsclient.ServiceClient) *RootView {
	items := []list.Item{
		serviceItem{name: "EC2", desc: "Elastic Compute Cloud — Instances"},
		serviceItem{name: "ECS", desc: "Elastic Container Service — Clusters, Services, Tasks"},
		serviceItem{name: "VPC", desc: "Virtual Private Cloud — VPCs, Subnets, Security Groups"},
		serviceItem{name: "ECR", desc: "Elastic Container Registry — Repositories, Images"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 60, 14)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &RootView{
		client: client,
		list:   l,
	}
}

func (v *RootView) Title() string { return "Services" }

func (v *RootView) Init() tea.Cmd { return nil }

func (v *RootView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected, ok := v.list.SelectedItem().(serviceItem)
			if !ok {
				return v, nil
			}
			return v, v.handleSelection(selected.name)
		}
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *RootView) handleSelection(name string) tea.Cmd {
	switch name {
	case "EC2":
		return func() tea.Msg {
			return PushViewMsg{View: NewEC2View(v.client)}
		}
	case "ECS":
		return func() tea.Msg {
			return PushViewMsg{View: NewECSClustersView(v.client)}
		}
	case "VPC":
		return func() tea.Msg {
			return PushViewMsg{View: NewVPCListView(v.client)}
		}
	case "ECR":
		return func() tea.Msg {
			return PushViewMsg{View: NewECRReposView(v.client)}
		}
	}
	return nil
}

func (v *RootView) View() string {
	return v.list.View()
}
```

Note: `NewEC2View`, `NewECSClustersView`, `NewVPCListView`, `NewECRReposView` don't exist yet — they'll be created in Tasks 10–13. For compilation, add stubs or implement Tasks 8–13 together, then verify build after all view files exist.

**Step 2: Commit Tasks 8+9 together**

```bash
git add internal/tui/services/view.go internal/tui/services/model.go internal/tui/services/root.go
git commit -m "feat: add ViewStack navigation and root service list"
```

---

### Task 10: EC2 Instances View

**Files:**
- Create: `internal/tui/services/ec2.go`

**Step 1: Create EC2 view with summary + table**

Create `internal/tui/services/ec2.go`:

```go
package services

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

type ec2DataMsg struct {
	instances []awsclient.EC2Instance
	summary   awsclient.EC2Summary
}

type EC2View struct {
	client    *awsclient.ServiceClient
	instances []awsclient.EC2Instance
	summary   awsclient.EC2Summary
	table     table.Model
	spinner   spinner.Model
	loading   bool
	err       error
}

func NewEC2View(client *awsclient.ServiceClient) *EC2View {
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Instance ID", Width: 22},
		{Title: "Type", Width: 12},
		{Title: "State", Width: 10},
		{Title: "Private IP", Width: 16},
		{Title: "Public IP", Width: 16},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)

	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))

	return &EC2View{client: client, table: t, spinner: sp, loading: true}
}

func (v *EC2View) Title() string { return "EC2" }

func (v *EC2View) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}

func (v *EC2View) fetchData() tea.Cmd {
	return func() tea.Msg {
		instances, summary, err := v.client.EC2.ListInstances(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return ec2DataMsg{instances: instances, summary: summary}
	}
}

type errViewMsg struct{ err error }

func (v *EC2View) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ec2DataMsg:
		v.instances = msg.instances
		v.summary = msg.summary
		v.loading = false
		rows := make([]table.Row, len(v.instances))
		for i, inst := range v.instances {
			rows[i] = table.Row{inst.Name, inst.InstanceID, inst.Type, inst.State, inst.PrivateIP, inst.PublicIP}
		}
		v.table.SetRows(rows)
		return v, nil

	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *EC2View) View() string {
	if v.loading {
		return v.spinner.View() + " Loading EC2 instances..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	summary := fmt.Sprintf(
		"%s %s   %s %s   %s %s",
		svcMutedStyle.Render("Running:"), svcSuccessStyle.Render(fmt.Sprintf("%d", v.summary.Running)),
		svcMutedStyle.Render("Stopped:"), svcMutedStyle.Render(fmt.Sprintf("%d", v.summary.Stopped)),
		svcMutedStyle.Render("Total:"), svcMutedStyle.Render(fmt.Sprintf("%d", v.summary.Total)),
	)

	return summary + "\n\n" + v.table.View()
}
```

**Step 2: Commit**

```bash
git add internal/tui/services/ec2.go
git commit -m "feat: add EC2 instances view with summary and table"
```

---

### Task 11: ECS Views (Clusters → Services → Tasks)

**Files:**
- Create: `internal/tui/services/ecs.go`

**Step 1: Create ECS views (3 levels)**

Create `internal/tui/services/ecs.go`:

```go
package services

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

// --- Messages ---

type ecsClustersMsg struct{ clusters []awsclient.ECSCluster }
type ecsServicesMsg struct{ services []awsclient.ECSService }
type ecsTasksMsg struct{ tasks []awsclient.ECSTask }

// --- Clusters View ---

type ECSClustersView struct {
	client   *awsclient.ServiceClient
	clusters []awsclient.ECSCluster
	table    table.Model
	spinner  spinner.Model
	loading  bool
	err      error
}

func NewECSClustersView(client *awsclient.ServiceClient) *ECSClustersView {
	columns := []table.Column{
		{Title: "Cluster", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Services", Width: 10},
		{Title: "Tasks", Width: 10},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &ECSClustersView{client: client, table: t, spinner: sp, loading: true}
}

func (v *ECSClustersView) Title() string { return "ECS" }
func (v *ECSClustersView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *ECSClustersView) fetchData() tea.Cmd {
	return func() tea.Msg {
		clusters, err := v.client.ECS.ListClusters(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecsClustersMsg{clusters: clusters}
	}
}
func (v *ECSClustersView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecsClustersMsg:
		v.clusters = msg.clusters
		v.loading = false
		rows := make([]table.Row, len(v.clusters))
		for i, cl := range v.clusters {
			rows[i] = table.Row{cl.Name, cl.Status, fmt.Sprintf("%d", cl.ServiceCount), fmt.Sprintf("%d", cl.RunningTaskCount)}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "enter":
			idx := v.table.Cursor()
			if idx >= 0 && idx < len(v.clusters) {
				cluster := v.clusters[idx]
				return v, func() tea.Msg {
					return PushViewMsg{View: NewECSServicesView(v.client, cluster.Name)}
				}
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *ECSClustersView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading ECS clusters..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}

// --- Services View ---

type ECSServicesView struct {
	client      *awsclient.ServiceClient
	clusterName string
	services    []awsclient.ECSService
	table       table.Model
	spinner     spinner.Model
	loading     bool
	err         error
}

func NewECSServicesView(client *awsclient.ServiceClient, clusterName string) *ECSServicesView {
	columns := []table.Column{
		{Title: "Service", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Desired", Width: 8},
		{Title: "Running", Width: 8},
		{Title: "Task Def", Width: 30},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &ECSServicesView{client: client, clusterName: clusterName, table: t, spinner: sp, loading: true}
}

func (v *ECSServicesView) Title() string { return v.clusterName }
func (v *ECSServicesView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *ECSServicesView) fetchData() tea.Cmd {
	return func() tea.Msg {
		services, err := v.client.ECS.ListServices(context.Background(), v.clusterName)
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecsServicesMsg{services: services}
	}
}
func (v *ECSServicesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecsServicesMsg:
		v.services = msg.services
		v.loading = false
		rows := make([]table.Row, len(v.services))
		for i, svc := range v.services {
			rows[i] = table.Row{svc.Name, svc.Status, fmt.Sprintf("%d", svc.DesiredCount), fmt.Sprintf("%d", svc.RunningCount), svc.TaskDef}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "enter":
			idx := v.table.Cursor()
			if idx >= 0 && idx < len(v.services) {
				svc := v.services[idx]
				return v, func() tea.Msg {
					return PushViewMsg{View: NewECSTasksView(v.client, v.clusterName, svc.Name)}
				}
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *ECSServicesView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading services..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}

// --- Tasks View ---

type ECSTasksView struct {
	client      *awsclient.ServiceClient
	clusterName string
	serviceName string
	tasks       []awsclient.ECSTask
	table       table.Model
	spinner     spinner.Model
	loading     bool
	err         error
}

func NewECSTasksView(client *awsclient.ServiceClient, clusterName, serviceName string) *ECSTasksView {
	columns := []table.Column{
		{Title: "Task ID", Width: 38},
		{Title: "Status", Width: 10},
		{Title: "Task Def", Width: 25},
		{Title: "Started", Width: 20},
		{Title: "Health", Width: 10},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &ECSTasksView{client: client, clusterName: clusterName, serviceName: serviceName, table: t, spinner: sp, loading: true}
}

func (v *ECSTasksView) Title() string { return v.serviceName }
func (v *ECSTasksView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *ECSTasksView) fetchData() tea.Cmd {
	return func() tea.Msg {
		tasks, err := v.client.ECS.ListTasks(context.Background(), v.clusterName, v.serviceName)
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecsTasksMsg{tasks: tasks}
	}
}
func (v *ECSTasksView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecsTasksMsg:
		v.tasks = msg.tasks
		v.loading = false
		rows := make([]table.Row, len(v.tasks))
		for i, t := range v.tasks {
			started := "—"
			if !t.StartedAt.IsZero() {
				started = t.StartedAt.Format("2006-01-02 15:04")
			}
			rows[i] = table.Row{t.TaskID, t.Status, t.TaskDef, started, t.HealthStatus}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *ECSTasksView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading tasks..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}
```

**Step 2: Commit**

```bash
git add internal/tui/services/ecs.go
git commit -m "feat: add ECS views (clusters, services, tasks)"
```

---

### Task 12: VPC Views (VPCs → Sub-resource List → Subnets/SGs/IGWs)

**Files:**
- Create: `internal/tui/services/vpc.go`

**Step 1: Create VPC views**

Create `internal/tui/services/vpc.go`. This is the most complex view because drilling into a VPC shows a sub-list (Subnets, Security Groups, Internet Gateways), then each of those shows a table.

```go
package services

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

// --- Messages ---

type vpcListMsg struct{ vpcs []awsclient.VPCInfo }
type subnetListMsg struct{ subnets []awsclient.SubnetInfo }
type sgListMsg struct{ sgs []awsclient.SecurityGroupInfo }
type igwListMsg struct{ igws []awsclient.InternetGatewayInfo }

// --- VPC List View ---

type VPCListView struct {
	client  *awsclient.ServiceClient
	vpcs    []awsclient.VPCInfo
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
}

func NewVPCListView(client *awsclient.ServiceClient) *VPCListView {
	columns := []table.Column{
		{Title: "VPC ID", Width: 24},
		{Title: "Name", Width: 20},
		{Title: "CIDR", Width: 18},
		{Title: "Default", Width: 8},
		{Title: "State", Width: 12},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &VPCListView{client: client, table: t, spinner: sp, loading: true}
}

func (v *VPCListView) Title() string { return "VPC" }
func (v *VPCListView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *VPCListView) fetchData() tea.Cmd {
	return func() tea.Msg {
		vpcs, err := v.client.VPC.ListVPCs(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return vpcListMsg{vpcs: vpcs}
	}
}
func (v *VPCListView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case vpcListMsg:
		v.vpcs = msg.vpcs
		v.loading = false
		rows := make([]table.Row, len(v.vpcs))
		for i, vpc := range v.vpcs {
			def := "No"
			if vpc.IsDefault {
				def = "Yes"
			}
			rows[i] = table.Row{vpc.VPCID, vpc.Name, vpc.CIDR, def, vpc.State}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "enter":
			idx := v.table.Cursor()
			if idx >= 0 && idx < len(v.vpcs) {
				vpc := v.vpcs[idx]
				return v, func() tea.Msg {
					return PushViewMsg{View: NewVPCSubMenuView(v.client, vpc.VPCID, vpc.Name)}
				}
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *VPCListView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading VPCs..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}

// --- VPC Sub-Menu View (Subnets / SGs / IGWs) ---

type vpcSubMenuItem struct {
	name string
	desc string
}

func (i vpcSubMenuItem) Title() string       { return i.name }
func (i vpcSubMenuItem) Description() string { return i.desc }
func (i vpcSubMenuItem) FilterValue() string { return i.name }

type VPCSubMenuView struct {
	client  *awsclient.ServiceClient
	vpcID   string
	vpcName string
	list    list.Model
}

func NewVPCSubMenuView(client *awsclient.ServiceClient, vpcID, vpcName string) *VPCSubMenuView {
	title := vpcName
	if title == "" {
		title = vpcID
	}
	items := []list.Item{
		vpcSubMenuItem{name: "Subnets", desc: "View subnets in this VPC"},
		vpcSubMenuItem{name: "Security Groups", desc: "View security groups in this VPC"},
		vpcSubMenuItem{name: "Internet Gateways", desc: "View internet gateways attached to this VPC"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	return &VPCSubMenuView{client: client, vpcID: vpcID, vpcName: title, list: l}
}

func (v *VPCSubMenuView) Title() string { return v.vpcName }
func (v *VPCSubMenuView) Init() tea.Cmd { return nil }
func (v *VPCSubMenuView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			selected, ok := v.list.SelectedItem().(vpcSubMenuItem)
			if !ok {
				return v, nil
			}
			switch selected.name {
			case "Subnets":
				return v, func() tea.Msg {
					return PushViewMsg{View: NewSubnetsView(v.client, v.vpcID)}
				}
			case "Security Groups":
				return v, func() tea.Msg {
					return PushViewMsg{View: NewSecurityGroupsView(v.client, v.vpcID)}
				}
			case "Internet Gateways":
				return v, func() tea.Msg {
					return PushViewMsg{View: NewIGWView(v.client, v.vpcID)}
				}
			}
		}
	}
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}
func (v *VPCSubMenuView) View() string { return v.list.View() }

// --- Subnets View ---

type SubnetsView struct {
	client  *awsclient.ServiceClient
	vpcID   string
	subnets []awsclient.SubnetInfo
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
}

func NewSubnetsView(client *awsclient.ServiceClient, vpcID string) *SubnetsView {
	columns := []table.Column{
		{Title: "Subnet ID", Width: 26},
		{Title: "Name", Width: 20},
		{Title: "CIDR", Width: 18},
		{Title: "AZ", Width: 14},
		{Title: "Available IPs", Width: 14},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &SubnetsView{client: client, vpcID: vpcID, table: t, spinner: sp, loading: true}
}

func (v *SubnetsView) Title() string { return "Subnets" }
func (v *SubnetsView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *SubnetsView) fetchData() tea.Cmd {
	return func() tea.Msg {
		subnets, err := v.client.VPC.ListSubnets(context.Background(), v.vpcID)
		if err != nil {
			return errViewMsg{err: err}
		}
		return subnetListMsg{subnets: subnets}
	}
}
func (v *SubnetsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case subnetListMsg:
		v.subnets = msg.subnets
		v.loading = false
		rows := make([]table.Row, len(v.subnets))
		for i, s := range v.subnets {
			rows[i] = table.Row{s.SubnetID, s.Name, s.CIDR, s.AZ, fmt.Sprintf("%d", s.AvailableIPs)}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *SubnetsView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading subnets..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}

// --- Security Groups View ---

type SecurityGroupsView struct {
	client  *awsclient.ServiceClient
	vpcID   string
	sgs     []awsclient.SecurityGroupInfo
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
}

func NewSecurityGroupsView(client *awsclient.ServiceClient, vpcID string) *SecurityGroupsView {
	columns := []table.Column{
		{Title: "Group ID", Width: 24},
		{Title: "Name", Width: 22},
		{Title: "Description", Width: 30},
		{Title: "Inbound", Width: 8},
		{Title: "Outbound", Width: 9},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &SecurityGroupsView{client: client, vpcID: vpcID, table: t, spinner: sp, loading: true}
}

func (v *SecurityGroupsView) Title() string { return "Security Groups" }
func (v *SecurityGroupsView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *SecurityGroupsView) fetchData() tea.Cmd {
	return func() tea.Msg {
		sgs, err := v.client.VPC.ListSecurityGroups(context.Background(), v.vpcID)
		if err != nil {
			return errViewMsg{err: err}
		}
		return sgListMsg{sgs: sgs}
	}
}
func (v *SecurityGroupsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case sgListMsg:
		v.sgs = msg.sgs
		v.loading = false
		rows := make([]table.Row, len(v.sgs))
		for i, sg := range v.sgs {
			rows[i] = table.Row{sg.GroupID, sg.Name, sg.Description, fmt.Sprintf("%d", sg.InboundRules), fmt.Sprintf("%d", sg.OutboundRules)}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *SecurityGroupsView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading security groups..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}

// --- Internet Gateways View ---

type IGWView struct {
	client  *awsclient.ServiceClient
	vpcID   string
	igws    []awsclient.InternetGatewayInfo
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
}

func NewIGWView(client *awsclient.ServiceClient, vpcID string) *IGWView {
	columns := []table.Column{
		{Title: "Gateway ID", Width: 26},
		{Title: "Name", Width: 25},
		{Title: "State", Width: 12},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &IGWView{client: client, vpcID: vpcID, table: t, spinner: sp, loading: true}
}

func (v *IGWView) Title() string { return "Internet Gateways" }
func (v *IGWView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *IGWView) fetchData() tea.Cmd {
	return func() tea.Msg {
		igws, err := v.client.VPC.ListInternetGateways(context.Background(), v.vpcID)
		if err != nil {
			return errViewMsg{err: err}
		}
		return igwListMsg{igws: igws}
	}
}
func (v *IGWView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case igwListMsg:
		v.igws = msg.igws
		v.loading = false
		rows := make([]table.Row, len(v.igws))
		for i, igw := range v.igws {
			rows[i] = table.Row{igw.GatewayID, igw.Name, igw.State}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *IGWView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading internet gateways..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}
```

**Step 2: Commit**

```bash
git add internal/tui/services/vpc.go
git commit -m "feat: add VPC views (VPCs, sub-menu, subnets, SGs, IGWs)"
```

---

### Task 13: ECR Views (Repos → Images)

**Files:**
- Create: `internal/tui/services/ecr.go`

**Step 1: Create ECR views**

Create `internal/tui/services/ecr.go`:

```go
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "tasnim.dev/aws-utils/internal/aws"
)

// --- Messages ---

type ecrReposMsg struct{ repos []awsclient.ECRRepo }
type ecrImagesMsg struct{ images []awsclient.ECRImage }

// --- Repos View ---

type ECRReposView struct {
	client  *awsclient.ServiceClient
	repos   []awsclient.ECRRepo
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
}

func NewECRReposView(client *awsclient.ServiceClient) *ECRReposView {
	columns := []table.Column{
		{Title: "Repository", Width: 35},
		{Title: "Images", Width: 8},
		{Title: "Created", Width: 20},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &ECRReposView{client: client, table: t, spinner: sp, loading: true}
}

func (v *ECRReposView) Title() string { return "ECR" }
func (v *ECRReposView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *ECRReposView) fetchData() tea.Cmd {
	return func() tea.Msg {
		repos, err := v.client.ECR.ListRepositories(context.Background())
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecrReposMsg{repos: repos}
	}
}
func (v *ECRReposView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecrReposMsg:
		v.repos = msg.repos
		v.loading = false
		rows := make([]table.Row, len(v.repos))
		for i, r := range v.repos {
			created := "—"
			if !r.CreatedAt.IsZero() {
				created = r.CreatedAt.Format("2006-01-02")
			}
			rows[i] = table.Row{r.Name, fmt.Sprintf("%d", r.ImageCount), created}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		case "enter":
			idx := v.table.Cursor()
			if idx >= 0 && idx < len(v.repos) {
				repo := v.repos[idx]
				return v, func() tea.Msg {
					return PushViewMsg{View: NewECRImagesView(v.client, repo.Name)}
				}
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *ECRReposView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading repositories..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}

// --- Images View ---

type ECRImagesView struct {
	client   *awsclient.ServiceClient
	repoName string
	images   []awsclient.ECRImage
	table    table.Model
	spinner  spinner.Model
	loading  bool
	err      error
}

func NewECRImagesView(client *awsclient.ServiceClient, repoName string) *ECRImagesView {
	columns := []table.Column{
		{Title: "Tag", Width: 25},
		{Title: "Digest", Width: 22},
		{Title: "Size (MB)", Width: 10},
		{Title: "Pushed", Width: 20},
	}
	t := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#6B7280")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))))
	return &ECRImagesView{client: client, repoName: repoName, table: t, spinner: sp, loading: true}
}

func (v *ECRImagesView) Title() string { return v.repoName }
func (v *ECRImagesView) Init() tea.Cmd {
	return tea.Batch(v.spinner.Tick, v.fetchData())
}
func (v *ECRImagesView) fetchData() tea.Cmd {
	return func() tea.Msg {
		images, err := v.client.ECR.ListImages(context.Background(), v.repoName)
		if err != nil {
			return errViewMsg{err: err}
		}
		return ecrImagesMsg{images: images}
	}
}
func (v *ECRImagesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ecrImagesMsg:
		v.images = msg.images
		v.loading = false
		rows := make([]table.Row, len(v.images))
		for i, img := range v.images {
			tag := "—"
			if len(img.Tags) > 0 {
				tag = strings.Join(img.Tags, ", ")
			}
			pushed := "—"
			if !img.PushedAt.IsZero() {
				pushed = img.PushedAt.Format("2006-01-02 15:04")
			}
			rows[i] = table.Row{tag, img.Digest, fmt.Sprintf("%.1f", img.SizeMB), pushed}
		}
		v.table.SetRows(rows)
		return v, nil
	case errViewMsg:
		v.err = msg.err
		v.loading = false
		return v, nil
	case tea.KeyMsg:
		if msg.String() == "r" {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.spinner.Tick, v.fetchData())
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}
func (v *ECRImagesView) View() string {
	if v.loading {
		return v.spinner.View() + " Loading images..."
	}
	if v.err != nil {
		return svcErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	return v.table.View()
}
```

**Step 2: Commit**

```bash
git add internal/tui/services/ecr.go
git commit -m "feat: add ECR views (repos, images)"
```

---

### Task 14: Wire Services Command + Build + Test

**Files:**
- Create: `cmd/services.go`
- Modify: `main.go`

**Step 1: Create `cmd/services.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	awsclient "tasnim.dev/aws-utils/internal/aws"
	"tasnim.dev/aws-utils/internal/tui/services"
)

func NewServicesCmd() *cobra.Command {
	var profile string
	var region string

	cmd := &cobra.Command{
		Use:   "services",
		Short: "Browse AWS services (EC2, ECS, VPC, ECR)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := awsclient.NewServiceClient(context.Background(), profile, region)
			if err != nil {
				return fmt.Errorf("initializing AWS client: %w", err)
			}

			model := services.NewModel(client, profile, region)
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region to use")

	return cmd
}
```

**Step 2: Add services command to `main.go`**

Add after the existing `rootCmd.AddCommand(cmd.NewCostCmd())` line:

```go
rootCmd.AddCommand(cmd.NewServicesCmd())
```

**Step 3: Verify build**

Run: `go build -o aws-utils .`
Expected: Binary compiles successfully

**Step 4: Verify help**

Run: `./aws-utils services --help`
Expected: Shows services command help with `--profile` and `--region` flags

**Step 5: Run all tests**

Run: `go test ./... -v`
Expected: All existing + new tests pass

**Step 6: Commit**

```bash
git add cmd/services.go main.go
git commit -m "feat: wire services command into CLI"
```

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Install SDK deps | `go.mod` |
| 2 | Service data types | `internal/aws/service_types.go` |
| 3 | EC2 client + tests | `internal/aws/ec2.go`, `ec2_test.go` |
| 4 | ECS client + tests | `internal/aws/ecs.go`, `ecs_test.go` |
| 5 | VPC client + tests | `internal/aws/vpc.go`, `vpc_test.go` |
| 6 | ECR client + tests | `internal/aws/ecr.go`, `ecr_test.go` |
| 7 | ServiceClient factory | `internal/aws/services.go` |
| 8 | View interface + ViewStack | `internal/tui/services/view.go`, `model.go` |
| 9 | Root service list | `internal/tui/services/root.go` |
| 10 | EC2 instances view | `internal/tui/services/ec2.go` |
| 11 | ECS views (3 levels) | `internal/tui/services/ecs.go` |
| 12 | VPC views (5 views) | `internal/tui/services/vpc.go` |
| 13 | ECR views (2 levels) | `internal/tui/services/ecr.go` |
| 14 | Wire command + build | `cmd/services.go`, `main.go` |

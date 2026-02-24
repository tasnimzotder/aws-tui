package ecs

import "time"

type ECSCluster struct {
	Name             string
	ARN              string
	Status           string
	RunningTaskCount int
	ServiceCount     int
}

type ECSService struct {
	Name         string
	ARN          string
	Status       string
	DesiredCount int
	RunningCount int
	PendingCount int
	TaskDef      string
}

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

type ECSServiceEvent struct {
	ID        string
	CreatedAt time.Time
	Message   string
}

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

type ECSLoadBalancerRef struct {
	TargetGroupARN string
	ContainerName  string
	ContainerPort  int
}

type ECSPlacementConstraint struct {
	Type       string
	Expression string
}

type ECSPlacementStrategy struct {
	Type  string
	Field string
}

type ECSTask struct {
	TaskID       string
	ARN          string
	Status       string
	TaskDef      string
	StartedAt    time.Time
	HealthStatus string
}

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

type EnvVar struct {
	Name  string
	Value string
}

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

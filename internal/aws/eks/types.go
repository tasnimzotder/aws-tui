package eks

import "time"

type EKSCluster struct {
	Name            string
	ARN             string
	Status          string
	Version         string
	PlatformVersion string
	Endpoint        string
	EndpointPublic  bool
	EndpointPrivate bool
	VPCID           string
	RoleARN         string
	CertAuthority   string // base64-encoded CA for K8s API
	CreatedAt       time.Time
}

type EKSNodeGroup struct {
	Name           string
	ARN            string
	Status         string
	InstanceTypes  []string
	AMIType        string
	MinSize        int
	MaxSize        int
	DesiredSize    int
	Labels         map[string]string
	Taints         []NodeGroupTaint
	Subnets        []string
	LaunchTemplate string
}

type NodeGroupTaint struct {
	Key    string
	Value  string
	Effect string
}

type EKSAddon struct {
	Name                string
	ARN                 string
	Version             string
	Status              string
	Health              string
	ServiceAccountRole  string
	ConfigurationValues string
}

type EKSFargateProfile struct {
	Name             string
	ARN              string
	Status           string
	PodExecutionRole string
	Selectors        []FargateSelector
	Subnets          []string
}

type FargateSelector struct {
	Namespace string
	Labels    map[string]string
}

type EKSAccessEntry struct {
	PrincipalARN string
	Type         string
	Username     string
	Groups       []string
	CreatedAt    time.Time
}

package elb

import "time"

type ELBLoadBalancer struct {
	Name      string
	ARN       string
	Type      string // "application" / "network" / "gateway"
	State     string
	Scheme    string // "internet-facing" / "internal"
	DNSName   string
	VPCID     string
	CreatedAt time.Time
}

type ELBListener struct {
	ARN           string
	Port          int
	Protocol      string
	DefaultAction string // "forward → tg-name" or "redirect → url" etc.
	SSLPolicy     string
	Certificates  []string // certificate ARNs
}

type ELBListenerRule struct {
	ARN        string
	Priority   string // "default" or numeric
	Conditions []string
	Actions    []string
	IsDefault  bool
}

type ELBTarget struct {
	ID           string
	Port         int
	AZ           string
	HealthState  string
	HealthReason string
	HealthDesc   string
}

type ELBAttribute struct {
	Key   string
	Value string
}

type ELBTargetGroup struct {
	Name           string
	ARN            string
	Protocol       string
	Port           int
	TargetType     string // "instance" / "ip" / "lambda"
	HealthyCount   int
	UnhealthyCount int
}

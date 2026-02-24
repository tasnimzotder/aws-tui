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

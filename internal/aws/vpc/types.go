package vpc

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

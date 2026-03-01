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

type RouteTableInfo struct {
	RouteTableID string
	Name         string
	IsMain       bool
	Routes       []RouteEntry
	Associations []RouteTableAssociation
}

type RouteEntry struct {
	Destination string // CIDR or prefix list
	Target      string // igw-xxx, nat-xxx, local, etc.
	Status      string // active, blackhole
	Origin      string // CreateRouteTable, CreateRoute, EnableVgwRoutePropagation
}

type RouteTableAssociation struct {
	SubnetID string
	IsMain   bool
}

type NATGatewayInfo struct {
	GatewayID string
	Name      string
	State     string // available, pending, failed, deleting, deleted
	Type      string // public, private
	SubnetID  string
	ElasticIP string
	PrivateIP string
}

type SecurityGroupRule struct {
	Direction   string // "inbound" or "outbound"
	Protocol    string // tcp, udp, icmp, all, or number
	PortRange   string // "80", "80-443", "All"
	Source      string // CIDR, security group ID, or prefix list (for inbound)
	Description string
}

type VPCEndpointInfo struct {
	EndpointID    string
	ServiceName   string
	Type          string // Interface, Gateway, GatewayLoadBalancer
	State         string
	SubnetIDs     []string
	RouteTableIDs []string
}

type VPCPeeringInfo struct {
	PeeringID     string
	Name          string
	Status        string
	RequesterVPC  string
	RequesterCIDR string
	AccepterVPC   string
	AccepterCIDR  string
}

type NetworkACLInfo struct {
	NACLID    string
	Name      string
	IsDefault bool
	Inbound   int
	Outbound  int
}

type NetworkACLEntry struct {
	RuleNumber int
	Direction  string // inbound / outbound
	Protocol   string
	PortRange  string
	CIDRBlock  string
	Action     string // allow / deny
}

type FlowLogInfo struct {
	FlowLogID      string
	Status         string
	TrafficType    string
	LogDestination string
	LogFormat      string
}

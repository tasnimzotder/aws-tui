package ec2

// EC2Instance represents a single EC2 instance.
type EC2Instance struct {
	Name       string
	InstanceID string
	Type       string
	State      string
	PrivateIP  string
	PublicIP   string
}

// EC2Summary holds aggregate instance counts.
type EC2Summary struct {
	Total   int
	Running int
	Stopped int
}

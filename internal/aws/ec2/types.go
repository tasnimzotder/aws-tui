package ec2

import "time"

// EC2Instance represents a single EC2 instance.
type EC2Instance struct {
	Name       string
	InstanceID string
	Type       string
	State      string
	PrivateIP  string
	PublicIP   string

	LaunchTime     time.Time
	AZ             string
	Architecture   string
	ImageID        string
	KeyName        string
	IAMProfile     string
	VpcID          string
	SubnetID       string
	SecurityGroups []EC2SecurityGroup
	Volumes        []EC2BlockDevice
	Tags           map[string]string
	Platform       string
}

// EC2SecurityGroup is a minimal SG reference attached to an instance.
type EC2SecurityGroup struct {
	GroupID   string
	GroupName string
}

// EC2BlockDevice represents a block device mapping on an instance.
type EC2BlockDevice struct {
	DeviceName          string
	VolumeID            string
	Status              string
	DeleteOnTermination bool
}

// EBSVolume holds details for a single EBS volume.
type EBSVolume struct {
	VolumeID   string
	Size       int32
	VolumeType string
	State      string
	IOPS       int32
	Encrypted  bool
	AZ         string
}

// EC2Summary holds aggregate instance counts.
type EC2Summary struct {
	Total   int
	Running int
	Stopped int
}

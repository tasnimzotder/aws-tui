package autoscaling

type AutoScalingTarget struct {
	MinCapacity int
	MaxCapacity int
	ResourceID  string
}

type AutoScalingPolicy struct {
	PolicyName  string
	PolicyType  string
	TargetValue float64
	MetricName  string
}

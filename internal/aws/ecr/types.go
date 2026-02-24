package ecr

import "time"

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

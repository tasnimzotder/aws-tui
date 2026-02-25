package iam

import "time"

type IAMUser struct {
	Name      string
	UserID    string
	ARN       string
	Path      string
	CreatedAt time.Time
}

type IAMRole struct {
	Name                     string
	RoleID                   string
	ARN                      string
	Path                     string
	Description              string
	CreatedAt                time.Time
	AssumeRolePolicyDocument string // URL-encoded JSON
}

type IAMPolicy struct {
	Name            string
	PolicyID        string
	ARN             string
	Path            string
	AttachmentCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type IAMAttachedPolicy struct {
	Name string
	ARN  string
}

type IAMGroup struct {
	Name string
	ARN  string
}

type IAMPolicyEntity struct {
	Name string
	Type string // "User", "Role", "Group"
}

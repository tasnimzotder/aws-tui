package aws

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Session holds the loaded AWS configuration along with the region and profile
// that were used to create it.
type Session struct {
	Config  aws.Config
	Region  string
	Profile string
}

// NewSession loads an AWS config with optional region and profile overrides and
// returns a Session wrapping the result.
func NewSession(ctx context.Context, region, profile string) (*Session, error) {
	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &Session{
		Config:  cfg,
		Region:  cfg.Region,
		Profile: profile,
	}, nil
}

// Identity holds the caller's AWS identity from STS.
type Identity struct {
	Account string
	ARN     string
	UserID  string
}

// CallerIdentity calls STS GetCallerIdentity and returns the caller's identity.
func (s *Session) CallerIdentity(ctx context.Context) (Identity, error) {
	out, err := sts.NewFromConfig(s.Config).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Account: aws.ToString(out.Account),
		ARN:     aws.ToString(out.Arn),
		UserID:  aws.ToString(out.UserId),
	}, nil
}

// AccountID calls STS GetCallerIdentity and returns the AWS account ID.
// Returns an empty string on error (non-fatal).
func (s *Session) AccountID(ctx context.Context) string {
	id, err := s.CallerIdentity(ctx)
	if err != nil {
		return ""
	}
	return id.Account
}

// ListProfiles parses ~/.aws/config for profile names, including "default".
func ListProfiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	f, err := os.Open(filepath.Join(home, ".aws", "config"))
	if err != nil {
		return nil
	}
	defer f.Close()

	var profiles []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[default]" {
			profiles = append(profiles, "default")
		} else if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			name := strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
			name = strings.TrimSpace(name)
			if name != "" {
				profiles = append(profiles, name)
			}
		}
	}
	return profiles
}

// ListRegions returns a hardcoded list of common AWS regions.
func ListRegions() []string {
	return []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"af-south-1",
		"ap-east-1",
		"ap-south-1",
		"ap-south-2",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-southeast-3",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-northeast-3",
		"ca-central-1",
		"eu-central-1",
		"eu-central-2",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"eu-south-1",
		"eu-south-2",
		"eu-north-1",
		"me-south-1",
		"me-central-1",
		"sa-east-1",
	}
}

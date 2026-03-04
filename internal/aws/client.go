package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// ServiceClient aggregates the AWS session and provides a place to hold all
// service-specific clients. Individual service client fields will be added as
// the corresponding service packages are ported.
type ServiceClient struct {
	Session *Session
	Cfg     aws.Config
}

// NewServiceClient creates a new Session and returns a ServiceClient wrapping
// it. Service-specific clients will be initialised here once ported.
func NewServiceClient(ctx context.Context, profile, region string) (*ServiceClient, error) {
	sess, err := NewSession(ctx, region, profile)
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return &ServiceClient{
		Session: sess,
		Cfg:     sess.Config,
	}, nil
}

package eks

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const (
	// tokenPrefix is the required prefix for EKS bearer tokens.
	tokenPrefix = "k8s-aws-v1."

	// tokenExpiry is how long the token is considered valid.
	tokenExpiry = 15 * time.Minute

	// presignURLExpiry is the number of seconds the presigned URL itself is valid.
	presignURLExpiry = "60"

	// tokenRefreshBuffer is how close to expiry we consider the token stale.
	tokenRefreshBuffer = 1 * time.Minute

	// clusterIDHeader is the header that identifies the cluster for token auth.
	clusterIDHeader = "x-k8s-aws-id"
)

// generateFunc is the signature for token generation, allowing test injection.
type generateFunc func() (token string, expiry time.Time, err error)

// TokenProvider generates and caches EKS bearer tokens for K8s API authentication.
// Thread-safe via sync.Mutex.
type TokenProvider struct {
	mu       sync.Mutex
	token    string
	expiry   time.Time
	generate generateFunc
}

// NewTokenProvider creates a TokenProvider that generates real STS-based tokens.
func NewTokenProvider(cfg aws.Config, clusterName string) *TokenProvider {
	tp := &TokenProvider{}
	tp.generate = func() (string, time.Time, error) {
		return generateToken(cfg, clusterName)
	}
	return tp
}

// GetToken returns a cached token if it is valid (more than 1 minute until expiry),
// otherwise generates a new token and caches it.
func (tp *TokenProvider) GetToken() (string, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.token != "" && time.Until(tp.expiry) > tokenRefreshBuffer {
		return tp.token, nil
	}

	token, expiry, err := tp.generate()
	if err != nil {
		return "", fmt.Errorf("generating EKS token: %w", err)
	}

	tp.token = token
	tp.expiry = expiry
	return tp.token, nil
}

// generateToken creates a presigned STS GetCallerIdentity URL and encodes it
// as an EKS bearer token. This implements the same mechanism as `aws eks get-token`.
func generateToken(cfg aws.Config, clusterName string) (string, time.Time, error) {
	stsClient := sts.NewFromConfig(cfg)

	presignClient := sts.NewPresignClient(stsClient, func(po *sts.PresignOptions) {
		po.ClientOptions = append(po.ClientOptions, func(o *sts.Options) {
			o.APIOptions = append(o.APIOptions,
				// Add the x-k8s-aws-id header (hoisted to query param during presigning).
				smithyhttp.AddHeaderValue(clusterIDHeader, clusterName),
				// Set the presigned URL expiry to 60 seconds.
				smithyhttp.SetHeaderValue("X-Amz-Expires", presignURLExpiry),
			)
		})
	})

	ctx := context.Background()
	presigned, err := presignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presigning GetCallerIdentity: %w", err)
	}

	// Encode the presigned URL as base64url without padding.
	encoded := base64.RawURLEncoding.EncodeToString([]byte(presigned.URL))

	token := tokenPrefix + encoded
	expiry := time.Now().Add(tokenExpiry)

	return token, expiry, nil
}

// WrapTransport returns a function that wraps an http.RoundTripper to inject
// the Bearer token header on each K8s API request, refreshing if needed.
func (tp *TokenProvider) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &tokenTransport{
		base:     rt,
		provider: tp,
	}
}

// tokenTransport is an http.RoundTripper that injects the EKS bearer token.
type tokenTransport struct {
	base     http.RoundTripper
	provider *TokenProvider
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.provider.GetToken()
	if err != nil {
		return nil, fmt.Errorf("getting EKS bearer token: %w", err)
	}

	// Clone the request to avoid mutating the original.
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+token)

	return t.base.RoundTrip(req)
}

// tokenHasPrefix checks if a token has the expected k8s-aws-v1. prefix.
func tokenHasPrefix(token string) bool {
	return strings.HasPrefix(token, tokenPrefix)
}

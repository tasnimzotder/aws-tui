package aws

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAPIError implements smithy.APIError for testing.
type mockAPIError struct {
	code    string
	message string
}

func (e *mockAPIError) Error() string            { return fmt.Sprintf("%s: %s", e.code, e.message) }
func (e *mockAPIError) ErrorCode() string         { return e.code }
func (e *mockAPIError) ErrorMessage() string       { return e.message }
func (e *mockAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrKind
	}{
		// nil
		{name: "nil error", err: nil, want: ErrKindUnknown},

		// Auth errors via API error codes
		{name: "ExpiredToken", err: &mockAPIError{code: "ExpiredToken", message: "token expired"}, want: ErrKindAuth},
		{name: "ExpiredTokenException", err: &mockAPIError{code: "ExpiredTokenException", message: "expired"}, want: ErrKindAuth},
		{name: "InvalidClientTokenId", err: &mockAPIError{code: "InvalidClientTokenId", message: "invalid"}, want: ErrKindAuth},
		{name: "RequestExpired", err: &mockAPIError{code: "RequestExpired", message: "expired"}, want: ErrKindAuth},

		// Access denied via API error codes
		{name: "AccessDenied code", err: &mockAPIError{code: "AccessDenied", message: "denied"}, want: ErrKindAccessDenied},
		{name: "AccessDeniedException code", err: &mockAPIError{code: "AccessDeniedException", message: "denied"}, want: ErrKindAccessDenied},
		{name: "UnauthorizedAccess code", err: &mockAPIError{code: "UnauthorizedAccess", message: "unauth"}, want: ErrKindAccessDenied},

		// Throttling via API error codes
		{name: "Throttling code", err: &mockAPIError{code: "Throttling", message: "slow down"}, want: ErrKindThrottled},
		{name: "ThrottlingException code", err: &mockAPIError{code: "ThrottlingException", message: "slow down"}, want: ErrKindThrottled},
		{name: "TooManyRequestsException code", err: &mockAPIError{code: "TooManyRequestsException", message: "slow down"}, want: ErrKindThrottled},

		// Network errors via typed errors
		{name: "net.OpError", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, want: ErrKindNetwork},
		{name: "net.DNSError", err: &net.DNSError{Err: "no such host", Name: "example.com"}, want: ErrKindNetwork},

		// String-based matching for auth
		{name: "SSO session string", err: errors.New("refreshing SSO session credentials"), want: ErrKindAuth},
		{name: "SSO token string", err: errors.New("failed to refresh SSO token"), want: ErrKindAuth},
		{name: "token expired string", err: errors.New("the token has expired"), want: ErrKindAuth},
		{name: "credentials expired string", err: errors.New("credentials have expired"), want: ErrKindAuth},
		{name: "security token expired", err: errors.New("the security token included in the request is expired"), want: ErrKindAuth},

		// String-based matching for access denied
		{name: "access denied string", err: errors.New("access denied for resource"), want: ErrKindAccessDenied},
		{name: "unauthorized string", err: errors.New("unauthorized request"), want: ErrKindAccessDenied},

		// String-based matching for throttle
		{name: "throttling string", err: errors.New("request throttled"), want: ErrKindThrottled},
		{name: "rate exceeded string", err: errors.New("rate exceeded"), want: ErrKindThrottled},
		{name: "too many requests string", err: errors.New("too many requests"), want: ErrKindThrottled},

		// String-based matching for timeout
		{name: "context deadline string", err: errors.New("context deadline exceeded"), want: ErrKindTimeout},
		{name: "request timeout string", err: errors.New("request timeout"), want: ErrKindTimeout},
		{name: "timed out string", err: errors.New("operation timed out"), want: ErrKindTimeout},

		// String-based matching for network
		{name: "connection refused string", err: errors.New("connection refused"), want: ErrKindNetwork},
		{name: "no such host string", err: errors.New("no such host"), want: ErrKindNetwork},
		{name: "network unreachable string", err: errors.New("network is unreachable"), want: ErrKindNetwork},

		// Wrapped errors
		{name: "wrapped auth error", err: fmt.Errorf("operation failed: %w", &mockAPIError{code: "ExpiredToken", message: "expired"}), want: ErrKindAuth},
		{name: "wrapped net error", err: fmt.Errorf("request: %w", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}), want: ErrKindNetwork},

		// Unknown
		{name: "unknown error", err: errors.New("something unexpected"), want: ErrKindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			assert.Equal(t, tt.want, got, "ClassifyError(%v)", tt.err)
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		kind ErrKind
		want bool
	}{
		{ErrKindAuth, false},
		{ErrKindAccessDenied, false},
		{ErrKindThrottled, true},
		{ErrKindNetwork, true},
		{ErrKindTimeout, false},
		{ErrKindUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetryable(tt.kind))
		})
	}
}

func TestFormatError(t *testing.T) {
	t.Run("nil error returns empty", func(t *testing.T) {
		assert.Equal(t, "", FormatError(nil))
	})

	t.Run("auth error message", func(t *testing.T) {
		msg := FormatError(&mockAPIError{code: "ExpiredToken", message: "expired"})
		assert.Contains(t, msg, "expired")
		assert.Contains(t, msg, "re-authenticate")
	})

	t.Run("access denied message", func(t *testing.T) {
		msg := FormatError(&mockAPIError{code: "AccessDenied", message: "denied"})
		assert.Contains(t, msg, "Access denied")
	})

	t.Run("throttled message", func(t *testing.T) {
		msg := FormatError(&mockAPIError{code: "Throttling", message: "slow"})
		assert.Contains(t, msg, "throttled")
	})

	t.Run("unknown includes original", func(t *testing.T) {
		msg := FormatError(errors.New("kaboom"))
		assert.Contains(t, msg, "kaboom")
	})
}

func TestErrKindString(t *testing.T) {
	require.Equal(t, "Auth", ErrKindAuth.String())
	require.Equal(t, "AccessDenied", ErrKindAccessDenied.String())
	require.Equal(t, "Throttled", ErrKindThrottled.String())
	require.Equal(t, "Network", ErrKindNetwork.String())
	require.Equal(t, "Timeout", ErrKindTimeout.String())
	require.Equal(t, "Unknown", ErrKindUnknown.String())
}

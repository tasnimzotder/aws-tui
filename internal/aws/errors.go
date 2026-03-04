package aws

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/smithy-go"
)

// ErrKind categorises AWS errors into actionable groups.
type ErrKind int

const (
	ErrKindUnknown      ErrKind = iota
	ErrKindAuth                 // expired credentials, SSO session expired
	ErrKindAccessDenied         // access denied, unauthorized
	ErrKindThrottled            // throttling, rate limiting
	ErrKindNetwork              // connection refused, timeout, DNS resolution
	ErrKindTimeout              // request timeout (context deadline, etc.)
)

// String returns a human-readable label for the error kind.
func (k ErrKind) String() string {
	switch k {
	case ErrKindAuth:
		return "Auth"
	case ErrKindAccessDenied:
		return "AccessDenied"
	case ErrKindThrottled:
		return "Throttled"
	case ErrKindNetwork:
		return "Network"
	case ErrKindTimeout:
		return "Timeout"
	default:
		return "Unknown"
	}
}

// authErrorCodes contains AWS API error codes that indicate authentication
// issues (expired tokens, invalid credentials, expired SSO sessions).
var authErrorCodes = []string{
	"ExpiredToken",
	"ExpiredTokenException",
	"RequestExpired",
	"InvalidClientTokenId",
	"UnrecognizedClientException",
	"InvalidIdentityToken",
}

// accessDeniedCodes contains AWS API error codes for authorisation failures.
var accessDeniedCodes = []string{
	"AccessDenied",
	"AccessDeniedException",
	"UnauthorizedAccess",
	"AuthorizationError",
}

// throttleCodes contains AWS API error codes for rate limiting.
var throttleCodes = []string{
	"Throttling",
	"ThrottlingException",
	"TooManyRequestsException",
	"RequestThrottled",
	"RequestLimitExceeded",
}

// ClassifyError inspects the error chain and returns the appropriate ErrKind.
func ClassifyError(err error) ErrKind {
	if err == nil {
		return ErrKindUnknown
	}

	// Check for smithy API errors first (most AWS SDK errors).
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()

		for _, c := range authErrorCodes {
			if code == c {
				return ErrKindAuth
			}
		}
		for _, c := range accessDeniedCodes {
			if code == c {
				return ErrKindAccessDenied
			}
		}
		for _, c := range throttleCodes {
			if code == c {
				return ErrKindThrottled
			}
		}
	}

	// Check for network-level errors.
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return ErrKindNetwork
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrKindNetwork
	}

	// Fall back to string matching for errors that don't implement typed
	// interfaces (e.g. wrapped or third-party errors).
	msg := strings.ToLower(err.Error())

	// SSO-specific strings that appear in credential provider errors.
	if strings.Contains(msg, "sso session") ||
		strings.Contains(msg, "sso token") ||
		strings.Contains(msg, "token has expired") ||
		strings.Contains(msg, "credentials have expired") ||
		strings.Contains(msg, "security token included in the request is expired") {
		return ErrKindAuth
	}

	if strings.Contains(msg, "access denied") ||
		strings.Contains(msg, "unauthorized") {
		return ErrKindAccessDenied
	}

	if strings.Contains(msg, "throttl") ||
		strings.Contains(msg, "rate exceeded") ||
		strings.Contains(msg, "too many requests") {
		return ErrKindThrottled
	}

	if strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "request timeout") ||
		strings.Contains(msg, "timed out") {
		return ErrKindTimeout
	}

	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "dns") {
		return ErrKindNetwork
	}

	return ErrKindUnknown
}

// IsRetryable returns true for error kinds where an automatic retry is
// reasonable (throttled and transient network errors).
func IsRetryable(kind ErrKind) bool {
	return kind == ErrKindThrottled || kind == ErrKindNetwork
}

// FormatError returns a concise, user-friendly message based on the error kind.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	kind := ClassifyError(err)
	switch kind {
	case ErrKindAuth:
		return "AWS credentials have expired. Please re-authenticate (e.g. aws sso login)."
	case ErrKindAccessDenied:
		return "Access denied. Check your IAM permissions for this operation."
	case ErrKindThrottled:
		return "Request was throttled by AWS. Retrying automatically..."
	case ErrKindNetwork:
		return "Network error. Check your internet connection and VPN settings."
	case ErrKindTimeout:
		return "Request timed out. The service may be slow or unreachable."
	default:
		return fmt.Sprintf("Unexpected error: %s", err.Error())
	}
}

// StartSSOLogin spawns `aws sso login --profile <profile>` as a subprocess.
// It blocks until the login process completes and returns any error encountered.
func StartSSOLogin(profile string) error {
	if profile == "" {
		profile = "default"
	}
	cmd := exec.Command("aws", "sso", "login", "--profile", profile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sso login failed for profile %q: %w", profile, err)
	}
	return nil
}

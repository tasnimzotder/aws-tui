package eks

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenProvider_CachesToken(t *testing.T) {
	callCount := 0
	tp := &TokenProvider{
		generate: func() (string, time.Time, error) {
			callCount++
			return "k8s-aws-v1.test-token", time.Now().Add(15 * time.Minute), nil
		},
	}

	tok1, err := tp.GetToken()
	if err != nil {
		t.Fatalf("first GetToken: %v", err)
	}
	tok2, err := tp.GetToken()
	if err != nil {
		t.Fatalf("second GetToken: %v", err)
	}

	if tok1 != tok2 {
		t.Errorf("tokens differ: %q vs %q", tok1, tok2)
	}
	if callCount != 1 {
		t.Errorf("generateFunc called %d times, want 1", callCount)
	}
}

func TestTokenProvider_RefreshesExpiredToken(t *testing.T) {
	callCount := 0
	tp := &TokenProvider{
		generate: func() (string, time.Time, error) {
			callCount++
			return fmt.Sprintf("k8s-aws-v1.token-%d", callCount), time.Now().Add(15 * time.Minute), nil
		},
	}

	tok1, err := tp.GetToken()
	if err != nil {
		t.Fatalf("first GetToken: %v", err)
	}

	// Simulate an expired token by setting expiry in the past.
	tp.mu.Lock()
	tp.expiry = time.Now().Add(-1 * time.Minute)
	tp.mu.Unlock()

	tok2, err := tp.GetToken()
	if err != nil {
		t.Fatalf("second GetToken: %v", err)
	}

	if tok1 == tok2 {
		t.Error("expected different token after expiry, got same")
	}
	if callCount != 2 {
		t.Errorf("generateFunc called %d times, want 2", callCount)
	}
}

func TestTokenProvider_RefreshesNearExpiry(t *testing.T) {
	callCount := 0
	tp := &TokenProvider{
		generate: func() (string, time.Time, error) {
			callCount++
			return fmt.Sprintf("k8s-aws-v1.token-%d", callCount), time.Now().Add(15 * time.Minute), nil
		},
	}

	tok1, err := tp.GetToken()
	if err != nil {
		t.Fatalf("first GetToken: %v", err)
	}

	// Set expiry to less than 1 minute from now (within the refresh buffer).
	tp.mu.Lock()
	tp.expiry = time.Now().Add(30 * time.Second)
	tp.mu.Unlock()

	tok2, err := tp.GetToken()
	if err != nil {
		t.Fatalf("second GetToken: %v", err)
	}

	if tok1 == tok2 {
		t.Error("expected different token near expiry, got same")
	}
	if callCount != 2 {
		t.Errorf("generateFunc called %d times, want 2", callCount)
	}
}

func TestTokenProvider_ThreadSafe(t *testing.T) {
	var callCount atomic.Int32
	tp := &TokenProvider{
		generate: func() (string, time.Time, error) {
			callCount.Add(1)
			return "k8s-aws-v1.concurrent-token", time.Now().Add(15 * time.Minute), nil
		},
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			tok, err := tp.GetToken()
			if err != nil {
				errs <- err
				return
			}
			if tok != "k8s-aws-v1.concurrent-token" {
				errs <- fmt.Errorf("unexpected token: %q", tok)
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("goroutine error: %v", err)
	}

	// With proper locking, generateFunc should only be called once since
	// after the first call the token is cached and valid.
	count := callCount.Load()
	if count != 1 {
		t.Errorf("generateFunc called %d times, want 1", count)
	}
}

func TestTokenProvider_GenerateError(t *testing.T) {
	tp := &TokenProvider{
		generate: func() (string, time.Time, error) {
			return "", time.Time{}, fmt.Errorf("sts failure")
		},
	}

	_, err := tp.GetToken()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sts failure") {
		t.Errorf("error = %q, want it to contain 'sts failure'", err.Error())
	}
}

func TestWrapTransport(t *testing.T) {
	tp := &TokenProvider{
		generate: func() (string, time.Time, error) {
			return "k8s-aws-v1.bearer-test", time.Now().Add(15 * time.Minute), nil
		},
	}

	// Create a mock base transport that captures the request.
	var capturedReq *http.Request
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		capturedReq = req
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})

	wrapped := tp.WrapTransport(base)

	req, _ := http.NewRequest("GET", "https://k8s-api.example.com/api/v1/pods", nil)
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	defer resp.Body.Close()

	if capturedReq == nil {
		t.Fatal("base transport was not called")
	}

	authHeader := capturedReq.Header.Get("Authorization")
	if authHeader != "Bearer k8s-aws-v1.bearer-test" {
		t.Errorf("Authorization header = %q, want %q", authHeader, "Bearer k8s-aws-v1.bearer-test")
	}

	// Verify the original request was not mutated.
	if req.Header.Get("Authorization") != "" {
		t.Error("original request should not be mutated")
	}
}

func TestTokenHasPrefix(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"k8s-aws-v1.abc123", true},
		{"k8s-aws-v1.", true},
		{"invalid-token", false},
		{"", false},
	}

	for _, tt := range tests {
		got := tokenHasPrefix(tt.token)
		if got != tt.want {
			t.Errorf("tokenHasPrefix(%q) = %v, want %v", tt.token, got, tt.want)
		}
	}
}

// roundTripperFunc adapts a function to the http.RoundTripper interface.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

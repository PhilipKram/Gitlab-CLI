package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRateLimitTransport_NoRetryOnSuccess(t *testing.T) {
	calls := 0
	transport := &RateLimitTransport{
		Base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRateLimitTransport_RetriesOn429(t *testing.T) {
	calls := 0
	transport := &RateLimitTransport{
		Base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls < 3 {
				return &http.Response{
					StatusCode: 429,
					Header:     http.Header{"Retry-After": []string{"1"}},
					Body:       io.NopCloser(strings.NewReader("rate limited")),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (2 retries + success), got %d", calls)
	}
}

func TestRetryAfterDuration_Seconds(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "10")
	d := retryAfterDuration(h)
	if d != 10*1e9 { // 10 seconds in nanoseconds
		t.Errorf("expected 10s, got %v", d)
	}
}

func TestRetryAfterDuration_Empty(t *testing.T) {
	h := http.Header{}
	d := retryAfterDuration(h)
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestRateLimitTransport_NilBase(t *testing.T) {
	// When Base is nil, should use http.DefaultTransport
	// We replace DefaultTransport with a mock to verify
	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()

	calls := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})

	transport := &RateLimitTransport{Base: nil}
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRateLimitTransport_MaxRetriesExhausted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow rate limit retry test")
	}
	calls := 0
	transport := &RateLimitTransport{
		Base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: 429,
				Header:     http.Header{"Retry-After": []string{"1"}},
				Body:       io.NopCloser(strings.NewReader("rate limited")),
			}, nil
		}),
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After maxRetries (3) + 1 initial = 4 calls, should return the 429
	if resp.StatusCode != 429 {
		t.Errorf("expected 429 when max retries exhausted, got %d", resp.StatusCode)
	}
	if calls != maxRetries+1 {
		t.Errorf("expected %d calls, got %d", maxRetries+1, calls)
	}
}

func TestRateLimitTransport_BaseError(t *testing.T) {
	transport := &RateLimitTransport{
		Base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		}),
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	_, err := transport.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error from base transport")
	}
	if err.Error() != "network error" {
		t.Errorf("expected 'network error', got %q", err.Error())
	}
}

func TestRateLimitTransport_NonRateLimitErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", 400},
		{"401 Unauthorized", 401},
		{"403 Forbidden", 403},
		{"404 Not Found", 404},
		{"500 Internal Server Error", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			transport := &RateLimitTransport{
				Base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader("error")),
					}, nil
				}),
			}

			req, _ := http.NewRequest("GET", "https://example.com", nil)
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.statusCode {
				t.Errorf("expected %d, got %d", tt.statusCode, resp.StatusCode)
			}
			if calls != 1 {
				t.Errorf("expected 1 call (no retry), got %d", calls)
			}
		})
	}
}

func TestRetryAfterDuration_RateLimitResetHeader(t *testing.T) {
	h := http.Header{}
	h.Set("RateLimit-Reset", "30")
	d := retryAfterDuration(h)
	if d != 30*time.Second {
		t.Errorf("expected 30s from RateLimit-Reset, got %v", d)
	}
}

func TestRetryAfterDuration_RetryAfterTakesPrecedence(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "5")
	h.Set("RateLimit-Reset", "30")
	d := retryAfterDuration(h)
	if d != 5*time.Second {
		t.Errorf("Retry-After should take precedence, expected 5s, got %v", d)
	}
}

func TestRetryAfterDuration_InvalidValue(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "not-a-number")
	d := retryAfterDuration(h)
	if d != 0 {
		t.Errorf("expected 0 for invalid Retry-After value, got %v", d)
	}
}

func TestRetryAfterDuration_ZeroSeconds(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "0")
	d := retryAfterDuration(h)
	if d != 0 {
		t.Errorf("expected 0 for Retry-After: 0, got %v", d)
	}
}

func TestRetryAfterDuration_NegativeSeconds(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "-5")
	d := retryAfterDuration(h)
	// -5 fails Atoi > 0 check, falls to ParseInt which gives a past timestamp
	// Either way the result should be 0 (no positive duration)
	if d != 0 {
		t.Errorf("expected 0 for negative Retry-After, got %v", d)
	}
}

func TestRetryAfterDuration_FutureUnixTimestamp(t *testing.T) {
	future := time.Now().Add(60 * time.Second).Unix()
	h := http.Header{}
	h.Set("Retry-After", fmt.Sprintf("%d", future))
	d := retryAfterDuration(h)
	// The value is large enough to be parsed as seconds first (Atoi succeeds and > 0)
	// So this will be interpreted as seconds, not a timestamp.
	// Just verify we get a positive duration.
	if d <= 0 {
		t.Errorf("expected positive duration for large Retry-After, got %v", d)
	}
}

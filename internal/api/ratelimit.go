package api

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"fmt"
)

const (
	maxRetries       = 3
	defaultRetryWait = 5 * time.Second
	maxRetryWait     = 60 * time.Second
)

// RateLimitTransport wraps an http.RoundTripper with automatic retry on HTTP 429 responses.
type RateLimitTransport struct {
	Base http.RoundTripper
}

// RoundTrip executes the request and retries on HTTP 429 with exponential backoff.
func (t *RateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := base.RoundTrip(req)
		if err != nil {
			return resp, err
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		if attempt == maxRetries {
			return resp, nil
		}

		// Determine wait time from Retry-After header or use exponential backoff
		wait := retryAfterDuration(resp.Header)
		if wait == 0 {
			wait = defaultRetryWait * time.Duration(1<<uint(attempt))
		}
		if wait > maxRetryWait {
			wait = maxRetryWait
		}

		// Close the 429 response body before retrying
		resp.Body.Close()

		fmt.Fprintf(os.Stderr, "Rate limited by GitLab API, retrying in %s...\n", wait)
		time.Sleep(wait)
	}

	// Unreachable, but satisfy the compiler
	return nil, fmt.Errorf("rate limit: max retries exceeded")
}

// retryAfterDuration parses the Retry-After header value as seconds.
func retryAfterDuration(h http.Header) time.Duration {
	val := h.Get("Retry-After")
	if val == "" {
		val = h.Get("RateLimit-Reset")
	}
	if val == "" {
		return 0
	}

	// Try parsing as seconds
	seconds, err := strconv.Atoi(val)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as Unix timestamp
	ts, err := strconv.ParseInt(val, 10, 64)
	if err == nil {
		d := time.Until(time.Unix(ts, 0))
		if d > 0 {
			return d
		}
	}

	return 0
}

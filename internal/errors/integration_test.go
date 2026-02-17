package errors_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/errors"
)

// TestE2E_401AuthError verifies 401 errors show re-login suggestion
func TestE2E_401AuthError(t *testing.T) {
	// Create a mock server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"401 Unauthorized"}`))
	}))
	defer server.Close()

	// Create an auth error
	err := errors.NewAuthError(
		"gitlab.com",
		"GET",
		server.URL+"/api/v4/user",
		401,
		"Authentication failed",
		nil,
	)

	errMsg := err.Error()

	// Verify error message contains required elements
	requiredElements := []string{
		"Authentication failed",
		"GET " + server.URL + "/api/v4/user",
		"401 Unauthorized",
		"gitlab.com",
		"glab auth login",
	}

	for _, element := range requiredElements {
		if !strings.Contains(errMsg, element) {
			t.Errorf("401 error message missing element %q\nGot: %s", element, errMsg)
		}
	}

	// Verify the suggestion is actionable
	if !strings.Contains(errMsg, "re-authenticate") {
		t.Error("401 error should suggest re-authentication")
	}
}

// TestE2E_403PermissionError verifies 403 errors show required scope
func TestE2E_403PermissionError(t *testing.T) {
	// Create a mock server that returns 403
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"403 Forbidden"}`))
	}))
	defer server.Close()

	// Create an auth error for permission denied
	err := errors.NewAuthError(
		"gitlab.com",
		"POST",
		server.URL+"/api/v4/projects",
		403,
		"Permission denied",
		nil,
	)

	errMsg := err.Error()

	// Verify error message contains required elements
	requiredElements := []string{
		"Permission denied",
		"POST " + server.URL + "/api/v4/projects",
		"403 Forbidden",
		"gitlab.com",
		"required scopes",
		"api",
		"read_user",
		"write_repository",
	}

	for _, element := range requiredElements {
		if !strings.Contains(errMsg, element) {
			t.Errorf("403 error message missing element %q\nGot: %s", element, errMsg)
		}
	}

	// Verify the suggestion is actionable
	if !strings.Contains(errMsg, "token has the required scopes") {
		t.Error("403 error should suggest verifying token scopes")
	}
}

// TestE2E_404NotFoundError verifies 404 errors show verify project path
func TestE2E_404NotFoundError(t *testing.T) {
	// Create a mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"404 Project Not Found"}`))
	}))
	defer server.Close()

	// Create an API error for not found
	err := errors.NewAPIError(
		"GET",
		server.URL+"/api/v4/projects/invalid-project",
		404,
		"Project not found",
		nil,
	)

	errMsg := err.Error()

	// Verify error message contains required elements
	requiredElements := []string{
		"Project not found",
		"GET " + server.URL + "/api/v4/projects/invalid-project",
		"404 Not Found",
		"Resource not found",
		"Verify the project path",
	}

	for _, element := range requiredElements {
		if !strings.Contains(errMsg, element) {
			t.Errorf("404 error message missing element %q\nGot: %s", element, errMsg)
		}
	}

	// Verify the suggestion is actionable
	if !strings.Contains(errMsg, "resource ID is correct") {
		t.Error("404 error should suggest verifying project path or resource ID")
	}
}

// TestE2E_429RateLimitError verifies 429 errors show retry-after time
func TestE2E_429RateLimitError(t *testing.T) {
	// Create a mock server that returns 429
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.Header().Set("RateLimit-Remaining", "0")
		w.Header().Set("RateLimit-Reset", "1234567890")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"429 Too Many Requests"}`))
	}))
	defer server.Close()

	// Create an API error for rate limit
	err := errors.NewAPIError(
		"GET",
		server.URL+"/api/v4/projects",
		429,
		"Rate limit exceeded",
		nil,
	)

	errMsg := err.Error()

	// Verify error message contains required elements
	requiredElements := []string{
		"Rate limit exceeded",
		"GET " + server.URL + "/api/v4/projects",
		"429 Too Many Requests",
		"Rate limit exceeded",
		"Wait a moment",
		"Personal Access Token",
	}

	for _, element := range requiredElements {
		if !strings.Contains(errMsg, element) {
			t.Errorf("429 error message missing element %q\nGot: %s", element, errMsg)
		}
	}

	// Verify the suggestion is actionable
	if !strings.Contains(errMsg, "higher limits") {
		t.Error("429 error should suggest using PAT for higher limits")
	}
}

// TestE2E_NetworkError verifies network errors show connectivity suggestions
func TestE2E_NetworkError(t *testing.T) {
	// Create a network error (simulating connection failure)
	err := errors.NewNetworkError(
		"gitlab.unreachable.example.com",
		"https://gitlab.unreachable.example.com/api/v4/projects",
		"Connection failed",
		fmt.Errorf("dial tcp: lookup gitlab.unreachable.example.com: no such host"),
	)

	errMsg := err.Error()

	// Verify error message contains required elements
	requiredElements := []string{
		"Connection failed",
		"https://gitlab.unreachable.example.com/api/v4/projects",
		"gitlab.unreachable.example.com",
		"internet connection",
		"proxy settings",
		"firewall rules",
	}

	for _, element := range requiredElements {
		if !strings.Contains(errMsg, element) {
			t.Errorf("Network error message missing element %q\nGot: %s", element, errMsg)
		}
	}

	// Verify the suggestion is actionable
	if !strings.Contains(errMsg, "Cannot connect to") {
		t.Error("Network error should suggest connectivity checks")
	}
}

// TestE2E_VerboseMode verifies verbose mode shows full request/response
func TestE2E_VerboseMode(t *testing.T) {
	// Save original verbose mode and GLAB_DEBUG settings
	originalVerbose := errors.IsVerboseMode()
	originalDebug := os.Getenv("GLAB_DEBUG")
	defer func() {
		errors.SetVerboseMode(originalVerbose)
		_ = os.Setenv("GLAB_DEBUG", originalDebug)
	}()

	// Test with SetVerboseMode
	t.Run("verbose mode via SetVerboseMode", func(t *testing.T) {
		errors.SetVerboseMode(true)
		defer errors.SetVerboseMode(false)

		if !errors.IsVerboseMode() {
			t.Error("IsVerboseMode() should return true when SetVerboseMode(true)")
		}

		// Capture stderr output
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Create a mock server
		requestReceived := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1,"name":"test-project"}`))
		}))
		defer server.Close()

		// Make a request using the logging transport
		client := errors.NewLoggingHTTPClient()
		resp, err := client.Get(server.URL + "/api/v4/projects/1")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Close writer and restore stderr
		_ = w.Close()
		os.Stderr = oldStderr

		// Read captured output
		output, _ := io.ReadAll(r)
		outputStr := string(output)

		// Verify request was made
		if !requestReceived {
			t.Error("Request was not received by mock server")
		}

		// Verify verbose output contains required elements
		requiredElements := []string{
			"HTTP Request",
			"GET " + server.URL + "/api/v4/projects/1",
			"Headers:",
			"HTTP Response",
			"Status: 200", // Check for status code, ignore full text
		}

		for _, element := range requiredElements {
			if !strings.Contains(outputStr, element) {
				t.Errorf("Verbose output missing element %q\nGot: %s", element, outputStr)
			}
		}
	})

	// Test with GLAB_DEBUG environment variable
	t.Run("verbose mode via GLAB_DEBUG=1", func(t *testing.T) {
		errors.SetVerboseMode(false)
		_ = os.Setenv("GLAB_DEBUG", "1")
		defer func() { _ = os.Setenv("GLAB_DEBUG", "") }()

		if !errors.IsVerboseMode() {
			t.Error("IsVerboseMode() should return true when GLAB_DEBUG=1")
		}
	})

	t.Run("verbose mode via GLAB_DEBUG=true", func(t *testing.T) {
		errors.SetVerboseMode(false)
		_ = os.Setenv("GLAB_DEBUG", "true")
		defer func() { _ = os.Setenv("GLAB_DEBUG", "") }()

		if !errors.IsVerboseMode() {
			t.Error("IsVerboseMode() should return true when GLAB_DEBUG=true")
		}
	})

	t.Run("verbose mode disabled when GLAB_DEBUG=0", func(t *testing.T) {
		errors.SetVerboseMode(false)
		_ = os.Setenv("GLAB_DEBUG", "0")
		defer func() { _ = os.Setenv("GLAB_DEBUG", "") }()

		if errors.IsVerboseMode() {
			t.Error("IsVerboseMode() should return false when GLAB_DEBUG=0")
		}
	})
}

// TestE2E_AllStatusCodes verifies suggestions for all common status codes
func TestE2E_AllStatusCodes(t *testing.T) {
	testCases := []struct {
		statusCode      int
		expectedSuggest string
	}{
		{401, "glab auth login"},
		{403, "Permission denied"},
		{404, "Resource not found"},
		{422, "Validation failed"},
		{429, "Rate limit exceeded"},
		{500, "GitLab server error"},
		{502, "GitLab server error"},
		{503, "GitLab server error"},
		{504, "GitLab server error"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("status_%d", tc.statusCode), func(t *testing.T) {
			// Create a mock server with the status code
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = fmt.Fprintf(w, `{"message":"Error %d"}`, tc.statusCode)
			}))
			defer server.Close()

			// Create an API error
			err := errors.NewAPIError(
				"GET",
				server.URL+"/api/v4/test",
				tc.statusCode,
				fmt.Sprintf("Request failed with status %d", tc.statusCode),
				nil,
			)

			errMsg := err.Error()

			// Verify the error contains the expected suggestion
			if !strings.Contains(errMsg, tc.expectedSuggest) {
				t.Errorf("Status %d error missing expected suggestion %q\nGot: %s",
					tc.statusCode, tc.expectedSuggest, errMsg)
			}

			// Verify HTTP details are present
			requiredElements := []string{
				fmt.Sprintf("GET %s/api/v4/test", server.URL),
				fmt.Sprintf("%d", tc.statusCode),
			}

			for _, element := range requiredElements {
				if !strings.Contains(errMsg, element) {
					t.Errorf("Status %d error missing element %q\nGot: %s",
						tc.statusCode, element, errMsg)
				}
			}
		})
	}
}

// TestE2E_ErrorFormatting verifies all error types format correctly
func TestE2E_ErrorFormatting(t *testing.T) {
	t.Run("APIError includes all components", func(t *testing.T) {
		err := errors.NewAPIError(
			"POST",
			"https://gitlab.com/api/v4/projects/1/issues",
			422,
			"Validation failed: Title can't be blank",
			fmt.Errorf("underlying validation error"),
		)

		errMsg := err.Error()

		// Verify all components are present and formatted correctly
		components := []string{
			"Validation failed: Title can't be blank",
			"Request: POST https://gitlab.com/api/v4/projects/1/issues",
			"Status: 422",
			"Error: underlying validation error",
			"→ ",
		}

		for _, component := range components {
			if !strings.Contains(errMsg, component) {
				t.Errorf("APIError missing component %q\nGot: %s", component, errMsg)
			}
		}
	})

	t.Run("AuthError includes host information", func(t *testing.T) {
		err := errors.NewAuthError(
			"gitlab.example.com",
			"GET",
			"https://gitlab.example.com/api/v4/user",
			401,
			"Token expired",
			nil,
		)

		errMsg := err.Error()

		// Verify host is included
		if !strings.Contains(errMsg, "gitlab.example.com") {
			t.Errorf("AuthError should include host\nGot: %s", errMsg)
		}
	})

	t.Run("NetworkError prioritizes URL over host", func(t *testing.T) {
		err := errors.NewNetworkError(
			"gitlab.com",
			"https://gitlab.com/api/v4/projects",
			"Connection timeout",
			nil,
		)

		errMsg := err.Error()

		// Should show URL, not just host
		if !strings.Contains(errMsg, "https://gitlab.com/api/v4/projects") {
			t.Errorf("NetworkError should show full URL\nGot: %s", errMsg)
		}
	})
}

// TestE2E_VerboseLoggingHTTPClient tests the logging HTTP client in verbose mode
func TestE2E_VerboseLoggingHTTPClient(t *testing.T) {
	// Save and restore verbose mode
	originalVerbose := errors.IsVerboseMode()
	defer errors.SetVerboseMode(originalVerbose)

	errors.SetVerboseMode(true)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Make request with logging client
	client := errors.NewLoggingHTTPClient()
	req, _ := http.NewRequest("POST", server.URL+"/api/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body to ensure it's logged
	_, _ = io.ReadAll(resp.Body)

	// Close writer and restore stderr
	_ = w.Close()
	os.Stderr = oldStderr

	// Read captured output
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Verify request logging
	requestChecks := []string{
		"→ HTTP Request",
		"POST " + server.URL + "/api/test",
		"Authorization: [REDACTED]", // Token should be redacted
		"Content-Type: [application/json]",
		`"key":"value"`, // Request body
	}

	for _, check := range requestChecks {
		if !strings.Contains(outputStr, check) {
			t.Errorf("Request logging missing %q\nGot: %s", check, outputStr)
		}
	}

	// Verify response logging
	responseChecks := []string{
		"← HTTP Response",
		"Status: 200", // Check for status code, ignore full text
		"X-Custom-Header: [test-value]",
		`"status":"ok"`, // Response body
	}

	for _, check := range responseChecks {
		if !strings.Contains(outputStr, check) {
			t.Errorf("Response logging missing %q\nGot: %s", check, outputStr)
		}
	}

	// Verify token is redacted (should not appear in output)
	if strings.Contains(outputStr, "secret-token") {
		t.Error("Authorization token should be redacted in verbose output")
	}
}

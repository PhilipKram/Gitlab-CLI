package errors

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// verboseMode controls whether to show detailed request/response information.
var verboseMode bool

// SetVerboseMode enables or disables verbose error output.
func SetVerboseMode(enabled bool) {
	verboseMode = enabled
}

// IsVerboseMode returns whether verbose mode is enabled.
func IsVerboseMode() bool {
	if verboseMode {
		return true
	}
	// Also check GLAB_DEBUG environment variable
	return os.Getenv("GLAB_DEBUG") == "1" || os.Getenv("GLAB_DEBUG") == "true"
}

// APIError represents an error from a GitLab API request with detailed context.
type APIError struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string
	// URL is the full request URL that failed
	URL string
	// StatusCode is the HTTP status code from the response
	StatusCode int
	// Message is the error message from the API or a descriptive error
	Message string
	// Suggestion provides actionable guidance to resolve the error
	Suggestion string
	// Err is the underlying error that caused this API error
	Err error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var b strings.Builder

	// Primary error message
	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString("API request failed")
	}

	// HTTP details
	if e.Method != "" && e.URL != "" {
		fmt.Fprintf(&b, "\n  Request: %s %s", e.Method, e.URL)
	}
	if e.StatusCode > 0 {
		fmt.Fprintf(&b, "\n  Status: %d %s", e.StatusCode, http.StatusText(e.StatusCode))
	}

	// Underlying error
	if e.Err != nil {
		fmt.Fprintf(&b, "\n  Error: %v", e.Err)
	}

	// Actionable suggestion
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n→ %s", e.Suggestion)
	}

	return b.String()
}

// Unwrap returns the underlying error.
func (e *APIError) Unwrap() error {
	return e.Err
}

// ErrorCode returns the error code for this APIError.
func (e *APIError) ErrorCode() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("API_%d", e.StatusCode)
	}
	return "API_ERROR"
}

// ErrorDetails returns structured details for JSON error formatting.
func (e *APIError) ErrorDetails() map[string]interface{} {
	details := make(map[string]interface{})

	if e.Method != "" {
		details["method"] = e.Method
	}
	if e.URL != "" {
		details["url"] = e.URL
	}
	if e.StatusCode > 0 {
		details["status_code"] = e.StatusCode
		details["status_text"] = http.StatusText(e.StatusCode)
	}
	if e.Suggestion != "" {
		details["suggestion"] = e.Suggestion
	}
	if e.Err != nil {
		details["underlying_error"] = e.Err.Error()
	}

	return details
}

// AuthError represents an authentication or authorization error.
type AuthError struct {
	// Host is the GitLab host where authentication failed
	Host string
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string
	// URL is the full request URL that failed
	URL string
	// StatusCode is the HTTP status code (typically 401 or 403)
	StatusCode int
	// Message is a descriptive error message
	Message string
	// Suggestion provides actionable guidance to resolve the auth issue
	Suggestion string
	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	var b strings.Builder

	// Primary error message
	if e.Message != "" {
		b.WriteString(e.Message)
	} else if e.StatusCode == 401 {
		b.WriteString("Authentication failed")
	} else if e.StatusCode == 403 {
		b.WriteString("Permission denied")
	} else {
		b.WriteString("Authorization error")
	}

	// HTTP details
	if e.Method != "" && e.URL != "" {
		fmt.Fprintf(&b, "\n  Request: %s %s", e.Method, e.URL)
	}
	if e.StatusCode > 0 {
		fmt.Fprintf(&b, "\n  Status: %d %s", e.StatusCode, http.StatusText(e.StatusCode))
	}
	if e.Host != "" {
		fmt.Fprintf(&b, "\n  Host: %s", e.Host)
	}

	// Underlying error
	if e.Err != nil {
		fmt.Fprintf(&b, "\n  Error: %v", e.Err)
	}

	// Actionable suggestion
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n→ %s", e.Suggestion)
	}

	return b.String()
}

// Unwrap returns the underlying error.
func (e *AuthError) Unwrap() error {
	return e.Err
}

// ErrorCode returns the error code for this AuthError.
func (e *AuthError) ErrorCode() string {
	if e.StatusCode == 401 {
		return "AUTH_UNAUTHENTICATED"
	}
	if e.StatusCode == 403 {
		return "AUTH_FORBIDDEN"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("AUTH_%d", e.StatusCode)
	}
	return "AUTH_ERROR"
}

// ErrorDetails returns structured details for JSON error formatting.
func (e *AuthError) ErrorDetails() map[string]interface{} {
	details := make(map[string]interface{})

	if e.Host != "" {
		details["host"] = e.Host
	}
	if e.Method != "" {
		details["method"] = e.Method
	}
	if e.URL != "" {
		details["url"] = e.URL
	}
	if e.StatusCode > 0 {
		details["status_code"] = e.StatusCode
		details["status_text"] = http.StatusText(e.StatusCode)
	}
	if e.Suggestion != "" {
		details["suggestion"] = e.Suggestion
	}
	if e.Err != nil {
		details["underlying_error"] = e.Err.Error()
	}

	return details
}

// NetworkError represents a network connectivity error.
type NetworkError struct {
	// Host is the remote host that could not be reached
	Host string
	// URL is the full request URL that failed
	URL string
	// Message is a descriptive error message
	Message string
	// Suggestion provides actionable guidance to resolve the network issue
	Suggestion string
	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *NetworkError) Error() string {
	var b strings.Builder

	// Primary error message
	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString("Network error")
	}

	// Connection details
	if e.URL != "" {
		fmt.Fprintf(&b, "\n  URL: %s", e.URL)
	} else if e.Host != "" {
		fmt.Fprintf(&b, "\n  Host: %s", e.Host)
	}

	// Underlying error
	if e.Err != nil {
		fmt.Fprintf(&b, "\n  Error: %v", e.Err)
	}

	// Actionable suggestion
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n→ %s", e.Suggestion)
	}

	return b.String()
}

// Unwrap returns the underlying error.
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// ErrorCode returns the error code for this NetworkError.
func (e *NetworkError) ErrorCode() string {
	return "NETWORK_ERROR"
}

// ErrorDetails returns structured details for JSON error formatting.
func (e *NetworkError) ErrorDetails() map[string]interface{} {
	details := make(map[string]interface{})

	if e.Host != "" {
		details["host"] = e.Host
	}
	if e.URL != "" {
		details["url"] = e.URL
	}
	if e.Suggestion != "" {
		details["suggestion"] = e.Suggestion
	}
	if e.Err != nil {
		details["underlying_error"] = e.Err.Error()
	}

	return details
}

// VersionError represents a GitLab API version mismatch error.
type VersionError struct {
	// RequiredVersion is the minimum GitLab version required
	RequiredVersion string
	// ActualVersion is the detected GitLab version
	ActualVersion string
	// Feature is the feature or operation that requires the minimum version
	Feature string
	// Message is a descriptive error message
	Message string
	// Suggestion provides actionable guidance to resolve the version issue
	Suggestion string
	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *VersionError) Error() string {
	var b strings.Builder

	// Primary error message
	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString("GitLab version mismatch")
	}

	// Version details
	if e.Feature != "" {
		fmt.Fprintf(&b, "\n  Feature: %s", e.Feature)
	}
	if e.RequiredVersion != "" {
		fmt.Fprintf(&b, "\n  Required: GitLab %s or higher", e.RequiredVersion)
	}
	if e.ActualVersion != "" {
		fmt.Fprintf(&b, "\n  Detected: GitLab %s", e.ActualVersion)
	}

	// Underlying error
	if e.Err != nil {
		fmt.Fprintf(&b, "\n  Error: %v", e.Err)
	}

	// Actionable suggestion
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n\n→ %s", e.Suggestion)
	}

	return b.String()
}

// Unwrap returns the underlying error.
func (e *VersionError) Unwrap() error {
	return e.Err
}

// SuggestForStatusCode returns an actionable suggestion based on HTTP status code.
func SuggestForStatusCode(statusCode int) string {
	switch statusCode {
	case 401:
		return "Authentication failed. Try running 'glab auth login' to re-authenticate or verify your access token."
	case 403:
		return "Permission denied. Verify your token has the required scopes and you have access to this resource."
	case 404:
		return "Resource not found. Verify the project path, branch name, or resource ID is correct. If the resource exists, this may indicate an API version mismatch - the endpoint might not be available in your GitLab version."
	case 422:
		return "Validation failed. Check your input parameters and try again."
	case 429:
		return "Rate limit exceeded. Wait a moment and try again, or use a Personal Access Token for higher limits."
	case 500, 502, 503, 504:
		return "GitLab server error. The service may be temporarily unavailable. Try again in a few moments."
	default:
		if statusCode >= 400 && statusCode < 500 {
			return "Request failed. Check your input and permissions."
		}
		if statusCode >= 500 {
			return "Server error. The GitLab service may be experiencing issues."
		}
		return ""
	}
}

// SuggestForAuth returns an actionable suggestion for authentication errors.
func SuggestForAuth(statusCode int, host string) string {
	if statusCode == 401 {
		return fmt.Sprintf("Authentication failed for %s. Run 'glab auth login --hostname %s' to re-authenticate.", host, host)
	}
	if statusCode == 403 {
		return fmt.Sprintf("Permission denied for %s. Verify your token has the required scopes (api, read_user, write_repository).", host)
	}
	return SuggestForStatusCode(statusCode)
}

// SuggestForNetwork returns an actionable suggestion for network errors.
func SuggestForNetwork(host string) string {
	return fmt.Sprintf("Cannot connect to %s. Check your internet connection, proxy settings, or firewall rules.", host)
}

// NewAPIError creates a new APIError with the given details.
// The suggestion is automatically generated from the status code if not provided.
func NewAPIError(method, url string, statusCode int, message string, err error) *APIError {
	suggestion := SuggestForStatusCode(statusCode)
	return &APIError{
		Method:     method,
		URL:        url,
		StatusCode: statusCode,
		Message:    message,
		Suggestion: suggestion,
		Err:        err,
	}
}

// NewAuthError creates a new AuthError with the given details.
// The suggestion is automatically generated from the status code and host if not provided.
func NewAuthError(host, method, url string, statusCode int, message string, err error) *AuthError {
	suggestion := SuggestForAuth(statusCode, host)
	return &AuthError{
		Host:       host,
		Method:     method,
		URL:        url,
		StatusCode: statusCode,
		Message:    message,
		Suggestion: suggestion,
		Err:        err,
	}
}

// NewNetworkError creates a new NetworkError with the given details.
// The suggestion is automatically generated from the host if not provided.
func NewNetworkError(host, url, message string, err error) *NetworkError {
	suggestion := SuggestForNetwork(host)
	return &NetworkError{
		Host:       host,
		URL:        url,
		Message:    message,
		Suggestion: suggestion,
		Err:        err,
	}
}

// NewVersionError creates a new VersionError with the given details.
func NewVersionError(requiredVersion, actualVersion, feature, message string, err error) *VersionError {
	suggestion := fmt.Sprintf("This feature requires GitLab %s or higher. Your instance is running GitLab %s.", requiredVersion, actualVersion)
	if message == "" {
		message = fmt.Sprintf("GitLab version %s is required for %s", requiredVersion, feature)
	}
	return &VersionError{
		RequiredVersion: requiredVersion,
		ActualVersion:   actualVersion,
		Feature:         feature,
		Message:         message,
		Suggestion:      suggestion,
		Err:             err,
	}
}

// WrapAPIError wraps an existing error with API context.
// If err is nil, returns nil. If err is already an APIError, returns it unchanged.
func WrapAPIError(err error, method, url string, statusCode int) error {
	if err == nil {
		return nil
	}

	// If already an APIError, return as-is
	var apiErr *APIError
	if As(err, &apiErr) {
		return err
	}

	// If it's an auth error (401, 403), return as-is
	var authErr *AuthError
	if As(err, &authErr) {
		return err
	}

	// Wrap generic errors
	return NewAPIError(method, url, statusCode, "", err)
}

// IsAPIError checks if an error is or wraps an APIError.
func IsAPIError(err error) bool {
	var apiErr *APIError
	return As(err, &apiErr)
}

// IsAuthError checks if an error is or wraps an AuthError.
func IsAuthError(err error) bool {
	var authErr *AuthError
	return As(err, &authErr)
}

// IsNetworkError checks if an error is or wraps a NetworkError.
func IsNetworkError(err error) bool {
	var netErr *NetworkError
	return As(err, &netErr)
}

// IsVersionError checks if an error is or wraps a VersionError.
func IsVersionError(err error) bool {
	var verErr *VersionError
	return As(err, &verErr)
}

// As is a convenience wrapper around errors.As from the standard library.
// It finds the first error in err's chain that matches target's type.
func As(err error, target interface{}) bool {
	// Import errors package functionality inline to avoid naming conflicts
	type unwrapper interface {
		Unwrap() error
	}

	if err == nil {
		return false
	}

	// Try direct assignment
	switch t := target.(type) {
	case **APIError:
		if apiErr, ok := err.(*APIError); ok {
			*t = apiErr
			return true
		}
	case **AuthError:
		if authErr, ok := err.(*AuthError); ok {
			*t = authErr
			return true
		}
	case **NetworkError:
		if netErr, ok := err.(*NetworkError); ok {
			*t = netErr
			return true
		}
	case **VersionError:
		if verErr, ok := err.(*VersionError); ok {
			*t = verErr
			return true
		}
	}

	// Walk the error chain
	for {
		unwrap, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = unwrap.Unwrap()
		if err == nil {
			return false
		}

		// Try assignment at each level
		switch t := target.(type) {
		case **APIError:
			if apiErr, ok := err.(*APIError); ok {
				*t = apiErr
				return true
			}
		case **AuthError:
			if authErr, ok := err.(*AuthError); ok {
				*t = authErr
				return true
			}
		case **NetworkError:
			if netErr, ok := err.(*NetworkError); ok {
				*t = netErr
				return true
			}
		case **VersionError:
			if verErr, ok := err.(*VersionError); ok {
				*t = verErr
				return true
			}
		}
	}
}

// loggingTransport wraps an http.RoundTripper to log requests and responses in verbose mode.
type loggingTransport struct {
	transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper interface.
func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !IsVerboseMode() {
		return t.transport.RoundTrip(req)
	}

	// Log request
	fmt.Fprintf(os.Stderr, "\n→ HTTP Request\n")
	fmt.Fprintf(os.Stderr, "  %s %s\n", req.Method, req.URL.String())
	fmt.Fprintf(os.Stderr, "  Headers:\n")
	for k, v := range req.Header {
		// Redact authorization tokens
		if k == "Authorization" || k == "Private-Token" {
			fmt.Fprintf(os.Stderr, "    %s: [REDACTED]\n", k)
		} else {
			fmt.Fprintf(os.Stderr, "    %s: %v\n", k, v)
		}
	}

	// Log request body if present
	if req.Body != nil && req.ContentLength > 0 {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			fmt.Fprintf(os.Stderr, "  Body: %s\n", string(bodyBytes))
		}
	}

	// Perform request
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n\n", err)
		return resp, err
	}

	// Log response
	fmt.Fprintf(os.Stderr, "\n← HTTP Response\n")
	fmt.Fprintf(os.Stderr, "  Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Fprintf(os.Stderr, "  Headers:\n")
	for k, v := range resp.Header {
		fmt.Fprintf(os.Stderr, "    %s: %v\n", k, v)
	}

	// Log response body if present
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			// Truncate very long responses
			bodyStr := string(bodyBytes)
			if len(bodyStr) > 1000 {
				bodyStr = bodyStr[:1000] + "... (truncated)"
			}
			fmt.Fprintf(os.Stderr, "  Body: %s\n", bodyStr)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
	return resp, nil
}

// NewLoggingHTTPClient creates an HTTP client that logs requests/responses in verbose mode.
func NewLoggingHTTPClient() *http.Client {
	return &http.Client{
		Transport: &loggingTransport{
			transport: http.DefaultTransport,
		},
	}
}

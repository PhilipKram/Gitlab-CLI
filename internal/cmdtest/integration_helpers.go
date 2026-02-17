package cmdtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/git"
)

// IntegrationTestFactory extends TestFactory for integration testing scenarios.
type IntegrationTestFactory struct {
	*TestFactory
	APIServer   *http.Server
	APIBaseURL  string
	MockHandler http.Handler
}

// SetupAuthContext configures authentication context for integration tests.
// It sets up the test environment with a valid token and hostname.
func SetupAuthContext(t *testing.T, tf *TestFactory, hostname, token string) {
	t.Helper()

	// Set the token environment variable (this is the primary auth mechanism for tests)
	t.Setenv("GITLAB_TOKEN", token)

	// Configure the default host in the config
	tf.Config.DefaultHost = hostname

	// Update the Remote to use the specified hostname
	if tf.Remote == nil {
		tf.Remote = &git.Remote{}
	}
	tf.Remote.Host = hostname

	// Update the Client factory to use the hostname
	originalClient := tf.Factory.Client
	tf.Factory.Client = func() (*api.Client, error) {
		return api.NewClient(hostname)
	}

	// Store original client factory for cleanup if needed
	t.Cleanup(func() {
		tf.Factory.Client = originalClient
	})
}

// SetupProjectContext configures project context for integration tests.
// It sets up a git remote with the specified owner and repository.
func SetupProjectContext(t *testing.T, tf *TestFactory, hostname, owner, repo string) {
	t.Helper()

	// Configure the git remote
	tf.Remote = &git.Remote{
		Name:  "origin",
		Host:  hostname,
		Owner: owner,
		Repo:  repo,
	}

	// Update the Remote factory to return this remote
	tf.Factory.Remote = func() (*git.Remote, error) {
		return tf.Remote, nil
	}

	// Update the Config to use this remote
	tf.Config.GitRemote = "origin"
}

// MockAPIRouter provides a multiplexer for routing mock API requests.
// Use this to create complex mock API servers that handle multiple endpoints.
type MockAPIRouter struct {
	routes map[string]http.HandlerFunc
	t      *testing.T
}

// NewMockAPIRouter creates a new mock API router for integration tests.
func NewMockAPIRouter(t *testing.T) *MockAPIRouter {
	t.Helper()
	return &MockAPIRouter{
		routes: make(map[string]http.HandlerFunc),
		t:      t,
	}
}

// Register adds a handler for a specific path and method.
// The key format is "METHOD /path", e.g., "GET /api/v4/user" or "POST /api/v4/projects".
func (r *MockAPIRouter) Register(method, path string, handler http.HandlerFunc) {
	r.t.Helper()
	key := method + " " + path
	r.routes[key] = handler
}

// RegisterWithPattern adds a handler that matches path patterns.
// Use this for dynamic paths like "/api/v4/projects/:id/merge_requests".
func (r *MockAPIRouter) RegisterWithPattern(method, pathPattern string, handler http.HandlerFunc) {
	r.t.Helper()
	// Store the pattern with a special marker
	key := method + " " + pathPattern
	r.routes[key] = handler
}

// ServeHTTP implements http.Handler to route requests to registered handlers.
func (r *MockAPIRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.t.Helper()

	// Try exact match first
	key := req.Method + " " + req.URL.Path
	if handler, ok := r.routes[key]; ok {
		handler(w, req)
		return
	}

	// Try pattern matching
	for pattern, handler := range r.routes {
		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) != 2 {
			continue
		}
		method, pathPattern := parts[0], parts[1]

		if req.Method == method && matchPath(pathPattern, req.URL.Path) {
			handler(w, req)
			return
		}
	}

	// No match found - return 404
	r.t.Logf("Mock API: No handler found for %s %s", req.Method, req.URL.Path)
	ErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Not found: %s %s", req.Method, req.URL.Path))
}

// matchPath checks if a URL path matches a pattern.
// Patterns can include :id, :iid, etc. for dynamic segments.
func matchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, part := range patternParts {
		// Skip dynamic segments (starting with :)
		if strings.HasPrefix(part, ":") {
			continue
		}
		// Exact match required for non-dynamic segments
		if part != pathParts[i] {
			return false
		}
	}

	return true
}

// AssertJSONOutput parses JSON output and verifies it matches expected structure.
// The expected parameter can be a map[string]interface{} for objects or []interface{} for arrays.
func AssertJSONOutput(t *testing.T, jsonString string, expected interface{}) {
	t.Helper()

	// Parse the JSON string
	var actual interface{}
	if err := json.Unmarshal([]byte(jsonString), &actual); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nJSON: %s", err, jsonString)
	}

	// Compare based on type
	switch exp := expected.(type) {
	case map[string]interface{}:
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			t.Fatalf("expected JSON object, got %T", actual)
		}
		compareJSONObjects(t, actualMap, exp)

	case []interface{}:
		actualArray, ok := actual.([]interface{})
		if !ok {
			t.Fatalf("expected JSON array, got %T", actual)
		}
		compareJSONArrays(t, actualArray, exp)

	default:
		// For primitive values, use direct comparison
		if actual != expected {
			t.Errorf("JSON mismatch:\nExpected: %v\nActual:   %v", expected, actual)
		}
	}
}

// AssertJSONContains verifies that JSON output contains specific key-value pairs.
// This is useful for partial matching when you don't want to verify the entire structure.
func AssertJSONContains(t *testing.T, jsonString string, expectedFields map[string]interface{}) {
	t.Helper()

	var actual map[string]interface{}
	if err := json.Unmarshal([]byte(jsonString), &actual); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nJSON: %s", err, jsonString)
	}

	for key, expectedValue := range expectedFields {
		actualValue, exists := actual[key]
		if !exists {
			t.Errorf("expected field %q not found in JSON output", key)
			continue
		}

		if !compareValues(actualValue, expectedValue) {
			t.Errorf("field %q mismatch:\nExpected: %v (%T)\nActual:   %v (%T)",
				key, expectedValue, expectedValue, actualValue, actualValue)
		}
	}
}

// AssertJSONArray verifies that JSON output is an array with expected length.
func AssertJSONArray(t *testing.T, jsonString string, expectedLength int) []interface{} {
	t.Helper()

	var actual []interface{}
	if err := json.Unmarshal([]byte(jsonString), &actual); err != nil {
		t.Fatalf("failed to parse JSON array: %v\nJSON: %s", err, jsonString)
	}

	if len(actual) != expectedLength {
		t.Errorf("expected JSON array length %d, got %d", expectedLength, len(actual))
	}

	return actual
}

// compareJSONObjects compares two JSON objects field by field.
func compareJSONObjects(t *testing.T, actual, expected map[string]interface{}) {
	t.Helper()

	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			t.Errorf("expected field %q not found in JSON output", key)
			continue
		}

		if !compareValues(actualValue, expectedValue) {
			t.Errorf("field %q mismatch:\nExpected: %v (%T)\nActual:   %v (%T)",
				key, expectedValue, expectedValue, actualValue, actualValue)
		}
	}
}

// compareJSONArrays compares two JSON arrays element by element.
func compareJSONArrays(t *testing.T, actual, expected []interface{}) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Errorf("array length mismatch:\nExpected: %d\nActual:   %d", len(expected), len(actual))
		return
	}

	for i := range expected {
		if !compareValues(actual[i], expected[i]) {
			t.Errorf("array element %d mismatch:\nExpected: %v (%T)\nActual:   %v (%T)",
				i, expected[i], expected[i], actual[i], actual[i])
		}
	}
}

// compareValues compares two values, handling different JSON types appropriately.
func compareValues(actual, expected interface{}) bool {
	// Handle nil values
	if expected == nil {
		return actual == nil
	}

	// For nested objects, recursively compare
	if expMap, ok := expected.(map[string]interface{}); ok {
		actMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		for key, expValue := range expMap {
			actValue, exists := actMap[key]
			if !exists || !compareValues(actValue, expValue) {
				return false
			}
		}
		return true
	}

	// For nested arrays, recursively compare
	if expArray, ok := expected.([]interface{}); ok {
		actArray, ok := actual.([]interface{})
		if !ok || len(actArray) != len(expArray) {
			return false
		}
		for i := range expArray {
			if !compareValues(actArray[i], expArray[i]) {
				return false
			}
		}
		return true
	}

	// For numbers, handle JSON's float64 representation
	if expNum, ok := expected.(float64); ok {
		actNum, ok := actual.(float64)
		return ok && actNum == expNum
	}
	if expNum, ok := expected.(int); ok {
		actNum, ok := actual.(float64)
		return ok && actNum == float64(expNum)
	}

	// For other types, use direct comparison
	return actual == expected
}

// CreateIntegrationClient creates a GitLab API client configured for integration testing.
// It uses the provided test server URL and sets up proper authentication.
func CreateIntegrationClient(t *testing.T, baseURL, token string) *api.Client {
	t.Helper()

	// Set the token in environment for the client to pick up
	t.Setenv("GITLAB_TOKEN", token)

	// Parse the hostname from baseURL
	// In real integration tests, this would be the test server's address
	client, err := api.NewClient("gitlab.com")
	if err != nil {
		t.Fatalf("failed to create integration test client: %v", err)
	}

	return client
}

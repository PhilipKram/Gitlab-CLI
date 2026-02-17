package cmdtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// roundTripFunc is a function type that implements http.RoundTripper.
// This allows us to use a simple function as an HTTP transport.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip executes a single HTTP transaction, implementing http.RoundTripper.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// FailTransport replaces http.DefaultTransport so that all HTTPS requests
// to the given host return an immediate error instead of hitting real DNS.
// The cleanup is handled automatically via t.Cleanup.
func FailTransport(t *testing.T, targetHost string) {
	t.Helper()
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == targetHost {
			return nil, fmt.Errorf("mock: connection refused")
		}
		return orig.RoundTrip(req)
	})
}

// InterceptTransport replaces http.DefaultTransport so that HTTPS requests
// to targetHost are rewritten to hit the test server instead.
// The cleanup is handled automatically via t.Cleanup.
func InterceptTransport(t *testing.T, targetHost string, srv *httptest.Server) {
	t.Helper()
	srvURL, _ := url.Parse(srv.URL)
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == targetHost {
			req.URL.Scheme = srvURL.Scheme
			req.URL.Host = srvURL.Host
		}
		return orig.RoundTrip(req)
	})
}

// MockServer creates an httptest.Server with a custom handler.
// The server is automatically closed via t.Cleanup.
func MockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// MockGitLabServer creates a mock GitLab API server that intercepts requests
// to the given hostname and routes them to the provided handler.
// The server and transport cleanup are handled automatically via t.Cleanup.
func MockGitLabServer(t *testing.T, hostname string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := MockServer(t, handler)
	InterceptTransport(t, hostname, srv)
	return srv
}

// JSONResponse is a helper to write JSON responses in mock handlers.
func JSONResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// ErrorResponse is a helper to write GitLab API error responses.
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	JSONResponse(w, statusCode, map[string]interface{}{
		"error":   http.StatusText(statusCode),
		"message": message,
	})
}

// MockMergeRequest returns a sample merge request for testing.
func MockMergeRequest(id int, title, state string) map[string]interface{} {
	return map[string]interface{}{
		"id":          id,
		"iid":         id,
		"title":       title,
		"state":       state,
		"description": "Test merge request description",
		"web_url":     fmt.Sprintf("https://gitlab.com/test-owner/test-repo/-/merge_requests/%d", id),
		"author": map[string]interface{}{
			"id":       1,
			"username": "test-user",
			"name":     "Test User",
		},
		"source_branch": "feature-branch",
		"target_branch": "main",
		"merged_at":     nil,
		"closed_at":     nil,
		"created_at":    "2024-01-01T00:00:00.000Z",
		"updated_at":    "2024-01-01T00:00:00.000Z",
	}
}

// MockIssue returns a sample issue for testing.
func MockIssue(id int, title, state string) map[string]interface{} {
	return map[string]interface{}{
		"id":          id,
		"iid":         id,
		"title":       title,
		"state":       state,
		"description": "Test issue description",
		"web_url":     fmt.Sprintf("https://gitlab.com/test-owner/test-repo/-/issues/%d", id),
		"author": map[string]interface{}{
			"id":       1,
			"username": "test-user",
			"name":     "Test User",
		},
		"assignees":  []interface{}{},
		"labels":     []string{"bug"},
		"created_at": "2024-01-01T00:00:00.000Z",
		"updated_at": "2024-01-01T00:00:00.000Z",
		"closed_at":  nil,
	}
}

// MockPipeline returns a sample pipeline for testing.
func MockPipeline(id int, ref, status string) map[string]interface{} {
	return map[string]interface{}{
		"id":      id,
		"ref":     ref,
		"sha":     "abc123def456",
		"status":  status,
		"web_url": fmt.Sprintf("https://gitlab.com/test-owner/test-repo/-/pipelines/%d", id),
		"user": map[string]interface{}{
			"id":       1,
			"username": "test-user",
			"name":     "Test User",
		},
		"created_at":  "2024-01-01T00:00:00.000Z",
		"updated_at":  "2024-01-01T00:00:00.000Z",
		"started_at":  "2024-01-01T00:00:00.000Z",
		"finished_at": nil,
	}
}

// MockProject returns a sample project for testing.
func MockProject(id int, name, path string) map[string]interface{} {
	return map[string]interface{}{
		"id":                  id,
		"name":                name,
		"path":                path,
		"description":         "Test project description",
		"web_url":             fmt.Sprintf("https://gitlab.com/test-owner/%s", path),
		"ssh_url_to_repo":     fmt.Sprintf("git@gitlab.com:test-owner/%s.git", path),
		"http_url_to_repo":    fmt.Sprintf("https://gitlab.com/test-owner/%s.git", path),
		"path_with_namespace": fmt.Sprintf("test-owner/%s", path),
		"visibility":          "public",
		"default_branch":      "main",
		"created_at":          "2024-01-01T00:00:00.000Z",
		"last_activity_at":    "2024-01-01T00:00:00.000Z",
	}
}

// MockUser returns a sample user for testing.
func MockUser(id int, username, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":         id,
		"username":   username,
		"name":       name,
		"email":      fmt.Sprintf("%s@example.com", username),
		"state":      "active",
		"web_url":    fmt.Sprintf("https://gitlab.com/%s", username),
		"avatar_url": "",
		"created_at": "2024-01-01T00:00:00.000Z",
	}
}

// MockRelease returns a sample release for testing.
func MockRelease(tagName, name, description string) map[string]interface{} {
	return map[string]interface{}{
		"tag_name":    tagName,
		"name":        name,
		"description": description,
		"created_at":  "2024-01-01T00:00:00.000Z",
		"released_at": "2024-01-01T00:00:00.000Z",
		"author": map[string]interface{}{
			"id":       1,
			"username": "test-user",
			"name":     "Test User",
		},
		"commit": map[string]interface{}{
			"id":       "abc123def456",
			"short_id": "abc123",
			"title":    "Release commit",
		},
		"assets": map[string]interface{}{
			"count": 0,
			"sources": []interface{}{
				map[string]string{
					"format": "zip",
					"url":    "https://gitlab.com/test-owner/test-repo/-/archive/v1.0.0/test-repo-v1.0.0.zip",
				},
				map[string]string{
					"format": "tar.gz",
					"url":    "https://gitlab.com/test-owner/test-repo/-/archive/v1.0.0/test-repo-v1.0.0.tar.gz",
				},
			},
		},
	}
}

// MockLabel returns a sample label for testing.
func MockLabel(id int, name, color string) map[string]interface{} {
	return map[string]interface{}{
		"id":                        id,
		"name":                      name,
		"color":                     color,
		"description":               fmt.Sprintf("Test label: %s", name),
		"text_color":                "#FFFFFF",
		"open_issues_count":         0,
		"closed_issues_count":       0,
		"open_merge_requests_count": 0,
	}
}

// MockSnippet returns a sample snippet for testing.
func MockSnippet(id int, title, fileName string) map[string]interface{} {
	return map[string]interface{}{
		"id":          id,
		"title":       title,
		"file_name":   fileName,
		"description": "Test snippet description",
		"visibility":  "public",
		"web_url":     fmt.Sprintf("https://gitlab.com/-/snippets/%d", id),
		"author": map[string]interface{}{
			"id":       1,
			"username": "test-user",
			"name":     "Test User",
		},
		"created_at": "2024-01-01T00:00:00.000Z",
		"updated_at": "2024-01-01T00:00:00.000Z",
	}
}

// RouterMux creates a simple request router for mock servers.
// It allows you to register handlers for different paths.
type RouterMux struct {
	routes map[string]http.HandlerFunc
}

// NewRouterMux creates a new RouterMux.
func NewRouterMux() *RouterMux {
	return &RouterMux{
		routes: make(map[string]http.HandlerFunc),
	}
}

// HandleFunc registers a handler for the given path.
func (rm *RouterMux) HandleFunc(path string, handler http.HandlerFunc) {
	rm.routes[path] = handler
}

// ServeHTTP implements http.Handler to route requests.
func (rm *RouterMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := rm.routes[r.URL.Path]; ok {
		handler(w, r)
		return
	}
	http.NotFound(w, r)
}

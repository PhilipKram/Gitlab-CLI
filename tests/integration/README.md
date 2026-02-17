# Integration Tests

This directory contains integration tests that verify `glab` commands against a **mock GitLab API server**. Unlike unit tests, integration tests exercise complete command execution flows including CLI parsing, API communication, and output formatting.

## Overview

Integration tests use `net/http/httptest` to create mock GitLab API servers that respond to API requests. This allows testing realistic command flows without requiring a real GitLab instance.

**Key characteristics:**
- Tests run against mock HTTP servers (not real GitLab)
- Test complete command execution paths
- Verify API request/response handling
- No external dependencies or network calls
- Fast execution (subsecond per test)

## Writing Integration Tests

### Basic Pattern

```go
package integration_test

import (
    "net/http"
    "testing"

    "github.com/PhilipKram/gitlab-cli/cmd"
    "github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestAuthStatus(t *testing.T) {
    // 1. Create mock GitLab API server
    mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v4/user" {
            cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
                "id":       123,
                "username": "testuser",
                "name":     "Test User",
            })
            return
        }
        http.NotFound(w, r)
    })

    // 2. Set up test factory with auth context
    tf := cmdtest.NewTestFactory(t)
    cmdtest.SetupAuthContext(t, tf, "gitlab.com", "glpat-test-token-12345")

    // 3. Execute command
    cmd := cmd.NewAuthStatusCmd(tf)
    cmd.SetArgs([]string{})

    err := cmd.Execute()
    if err != nil {
        t.Fatalf("command failed: %v", err)
    }

    // 4. Assert output
    output := tf.IO.String()
    cmdtest.AssertContains(t, output, "testuser")
    cmdtest.AssertContains(t, output, "gitlab.com")
}
```

## Test Infrastructure

### MockGitLabServer

Creates an `httptest.Server` that intercepts HTTPS requests to a GitLab hostname:

```go
mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/api/v4/user":
        cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
            "id": 123,
            "username": "testuser",
        })
    case "/api/v4/projects/1/merge_requests":
        cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
            {"iid": 42, "title": "Add feature"},
        })
    default:
        http.NotFound(w, r)
    }
})
```

**Important:** The server and transport cleanup happen automatically via `t.Cleanup()`.

### MockAPIRouter

For complex tests with multiple endpoints, use `MockAPIRouter`:

```go
router := cmdtest.NewRouterMux()
router.HandleFunc("GET /api/v4/user", func(w http.ResponseWriter, r *http.Request) {
    cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
        "id": 123,
        "username": "testuser",
    })
})
router.HandleFunc("GET /api/v4/projects/1/merge_requests", func(w http.ResponseWriter, r *http.Request) {
    // Check query parameters
    state := r.URL.Query().Get("state")
    if state == "opened" {
        cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
            {"iid": 42, "title": "Add feature", "state": "opened"},
        })
        return
    }
    cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{})
})

mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
```

### TestFactory

`cmdtest.NewTestFactory(t)` creates a test factory with captured I/O:

```go
tf := cmdtest.NewTestFactory(t)

// Factory provides:
// - tf.IO.Out: captured stdout
// - tf.IO.ErrOut: captured stderr
// - tf.IO.In: stdin for input
// - tf.Config: test configuration
// - tf.Client: API client (points to mock server)
// - tf.Remote: git remote information

// Execute command
cmd := cmd.NewMRListCmd(tf)
cmd.SetArgs([]string{"--state", "opened"})
err := cmd.Execute()

// Check output
output := tf.IO.String()         // stdout as string
errOutput := tf.IO.ErrString()   // stderr as string
```

### Authentication Setup

Use `SetupAuthContext` to configure authentication for tests:

```go
// Token-based auth (most common)
cmdtest.SetupAuthContext(t, tf, "gitlab.com", "glpat-test-token-12345")

// OAuth auth with expiry
cmdtest.SetupAuthContext(t, tf, "gitlab.com", "oauth-token-abc", cmdtest.WithExpiry(time.Now().Add(2*time.Hour)))
```

### Project Context

Use `SetupProjectContext` to configure git remote information:

```go
cmdtest.SetupProjectContext(t, tf, "gitlab.com", "owner/repo")

// Now commands that need project context will work
// e.g., `glab mr list` (uses project from git remote)
```

## Assertion Helpers

### String Assertions

```go
output := tf.IO.String()

// Check output contains expected strings
cmdtest.AssertContains(t, output, "testuser")
cmdtest.AssertContains(t, output, "Logged in")

// Check output does NOT contain strings
cmdtest.AssertNotContains(t, output, "Error")
```

### JSON Assertions

```go
output := tf.IO.String()

// Parse and check JSON structure
var result map[string]interface{}
cmdtest.AssertJSONContains(t, output, "username", "testuser")
cmdtest.AssertJSONField(t, output, "id", float64(123)) // JSON numbers are float64
```

## Test Fixtures

Use pre-defined fixtures from `internal/cmdtest/fixtures.go` for common API responses:

```go
import "github.com/PhilipKram/gitlab-cli/internal/cmdtest"

// Merge request fixtures
cmdtest.FixtureMROpen        // Open MR with full details
cmdtest.FixtureMRClosed      // Closed MR
cmdtest.FixtureMRMerged      // Merged MR

// Issue fixtures
cmdtest.FixtureIssueOpen     // Open issue
cmdtest.FixtureIssueClosed   // Closed issue

// Pipeline fixtures
cmdtest.FixturePipelineSuccess   // Successful pipeline
cmdtest.FixturePipelineFailed    // Failed pipeline
cmdtest.FixturePipelineRunning   // Running pipeline

// Example usage:
router.HandleFunc("GET /api/v4/projects/1/merge_requests/42", func(w http.ResponseWriter, r *http.Request) {
    cmdtest.JSONResponse(w, http.StatusOK, cmdtest.FixtureMROpen)
})
```

## Table-Driven Tests

Use table-driven tests for testing multiple scenarios:

```go
func TestMRList(t *testing.T) {
    tests := []struct {
        name           string
        args           []string
        mockResponse   interface{}
        expectContains []string
    }{
        {
            name: "list opened MRs",
            args: []string{"--state", "opened"},
            mockResponse: []map[string]interface{}{
                {"iid": 42, "title": "Add feature", "state": "opened"},
            },
            expectContains: []string{"42", "Add feature"},
        },
        {
            name: "list merged MRs",
            args: []string{"--state", "merged"},
            mockResponse: []map[string]interface{}{
                {"iid": 99, "title": "Fix bug", "state": "merged"},
            },
            expectContains: []string{"99", "Fix bug"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
                if r.URL.Path == "/api/v4/projects/1/merge_requests" {
                    cmdtest.JSONResponse(w, http.StatusOK, tt.mockResponse)
                    return
                }
                http.NotFound(w, r)
            })

            tf := cmdtest.NewTestFactory(t)
            cmdtest.SetupAuthContext(t, tf, "gitlab.com", "test-token")
            cmdtest.SetupProjectContext(t, tf, "gitlab.com", "owner/repo")

            cmd := cmd.NewMRListCmd(tf)
            cmd.SetArgs(tt.args)

            if err := cmd.Execute(); err != nil {
                t.Fatalf("command failed: %v", err)
            }

            output := tf.IO.String()
            for _, expected := range tt.expectContains {
                cmdtest.AssertContains(t, output, expected)
            }
        })
    }
}
```

## Error Testing

Test error scenarios by returning error responses from the mock server:

```go
func TestMRView_NotFound(t *testing.T) {
    mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v4/projects/1/merge_requests/999" {
            cmdtest.JSONResponse(w, http.StatusNotFound, map[string]interface{}{
                "message": "404 Not Found",
            })
            return
        }
        http.NotFound(w, r)
    })

    tf := cmdtest.NewTestFactory(t)
    cmdtest.SetupAuthContext(t, tf, "gitlab.com", "test-token")
    cmdtest.SetupProjectContext(t, tf, "gitlab.com", "owner/repo")

    cmd := cmd.NewMRViewCmd(tf)
    cmd.SetArgs([]string{"999"})

    err := cmd.Execute()
    if err == nil {
        t.Fatal("expected error for non-existent MR")
    }

    // Check error message
    if !strings.Contains(err.Error(), "404") {
        t.Errorf("expected 404 error, got: %v", err)
    }
}
```

## Authentication Errors

Test authentication failures (401, 403):

```go
func TestAuthStatus_Unauthorized(t *testing.T) {
    mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v4/user" {
            cmdtest.JSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
                "message": "401 Unauthorized",
            })
            return
        }
        http.NotFound(w, r)
    })

    tf := cmdtest.NewTestFactory(t)
    cmdtest.SetupAuthContext(t, tf, "gitlab.com", "invalid-token")

    cmd := cmd.NewAuthStatusCmd(tf)
    cmd.SetArgs([]string{})

    err := cmd.Execute()
    if err == nil {
        t.Fatal("expected error for invalid token")
    }

    // Verify error suggests re-authentication
    errMsg := err.Error()
    if !strings.Contains(errMsg, "glab auth login") {
        t.Errorf("expected error to suggest 'glab auth login', got: %v", errMsg)
    }
}
```

## Running Tests

### Run all integration tests

```bash
go test -v ./tests/integration/...
```

### Run specific test

```bash
go test -v ./tests/integration -run TestAuthStatus
```

### Run with coverage

```bash
go test -v -coverprofile=coverage.out ./tests/integration/...
go tool cover -html=coverage.out
```

### Run integration tests via Makefile

```bash
make test-integration
```

## Best Practices

### 1. Use `t.Helper()` in helper functions

```go
func assertMRInOutput(t *testing.T, output string, iid int) {
    t.Helper() // Makes test failure point to caller, not this function
    if !strings.Contains(output, fmt.Sprintf("%d", iid)) {
        t.Errorf("output missing MR !%d", iid)
    }
}
```

### 2. Clean test data with `t.Cleanup()`

```go
func TestWithTempConfig(t *testing.T) {
    // Cleanup is handled automatically by MockGitLabServer and NewTestFactory
    // But if you create external resources, use t.Cleanup():
    tmpDir := t.TempDir() // Automatically cleaned up

    // Or manually:
    file, _ := os.Create("/tmp/test-file")
    t.Cleanup(func() {
        os.Remove("/tmp/test-file")
    })
}
```

### 3. Test both success and error paths

Always test:
- Happy path (command succeeds)
- Error responses (404, 401, 403, 500)
- Edge cases (empty lists, missing fields)
- Flag variations (--state, --json, etc.)

### 4. Verify API request details

Check request method, path, headers, and body:

```go
mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
    // Verify request method
    if r.Method != "POST" {
        t.Errorf("expected POST, got %s", r.Method)
    }

    // Verify authorization header
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        t.Error("missing or invalid Authorization header")
    }

    // Verify request body
    var body map[string]interface{}
    json.NewDecoder(r.Body).Decode(&body)
    if body["title"] != "Test MR" {
        t.Errorf("unexpected title: %v", body["title"])
    }

    cmdtest.JSONResponse(w, http.StatusCreated, map[string]interface{}{
        "iid": 42,
        "web_url": "https://gitlab.com/owner/repo/-/merge_requests/42",
    })
})
```

### 5. Use subtests for better organization

```go
func TestAuthCommands(t *testing.T) {
    t.Run("status shows logged in user", func(t *testing.T) {
        // Test auth status
    })

    t.Run("token returns valid token", func(t *testing.T) {
        // Test auth token
    })

    t.Run("logout removes credentials", func(t *testing.T) {
        // Test auth logout
    })
}
```

## Comparing with Unit Tests

| Aspect | Unit Tests | Integration Tests |
|--------|------------|-------------------|
| **Location** | `cmd/*_test.go` | `tests/integration/*_test.go` |
| **Scope** | Single function/component | Complete command flow |
| **Dependencies** | Mocked via interfaces | Real HTTP client + mock server |
| **Speed** | Very fast (microseconds) | Fast (milliseconds) |
| **Purpose** | Verify logic correctness | Verify command behavior |
| **Example** | Test flag parsing, output formatting | Test `glab mr list` end-to-end |

**When to write integration tests:**
- Testing complete CLI command execution
- Verifying API request/response handling
- Testing authentication flows
- Testing error handling across layers
- Validating output formatting with real data

**When to write unit tests:**
- Testing individual functions
- Testing edge cases in business logic
- Testing utility functions
- Testing internal packages

## Example: Complete Integration Test

```go
package integration_test

import (
    "encoding/json"
    "net/http"
    "testing"

    "github.com/PhilipKram/gitlab-cli/cmd"
    "github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestMRList_Integration(t *testing.T) {
    // Create mock API server with realistic responses
    router := cmdtest.NewRouterMux()

    // Handle /api/v4/user (for auth verification)
    router.HandleFunc("GET /api/v4/user", func(w http.ResponseWriter, r *http.Request) {
        cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
            "id":       123,
            "username": "testuser",
        })
    })

    // Handle /api/v4/projects/:id (for project info)
    router.HandleFunc("GET /api/v4/projects/owner%2Frepo", func(w http.ResponseWriter, r *http.Request) {
        cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
            "id":                1,
            "path_with_namespace": "owner/repo",
        })
    })

    // Handle /api/v4/projects/:id/merge_requests (the main endpoint)
    router.HandleFunc("GET /api/v4/projects/1/merge_requests", func(w http.ResponseWriter, r *http.Request) {
        // Verify query parameters
        state := r.URL.Query().Get("state")

        // Verify authorization
        if r.Header.Get("Authorization") != "Bearer glpat-test-token-12345" {
            cmdtest.JSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
                "message": "401 Unauthorized",
            })
            return
        }

        // Return filtered results
        mrs := []map[string]interface{}{}
        if state == "" || state == "opened" {
            mrs = append(mrs, map[string]interface{}{
                "iid":    42,
                "title":  "Add feature",
                "state":  "opened",
                "author": map[string]interface{}{"username": "alice"},
            })
        }

        cmdtest.JSONResponse(w, http.StatusOK, mrs)
    })

    mockAPI := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)

    // Set up test factory with auth and project context
    tf := cmdtest.NewTestFactory(t)
    cmdtest.SetupAuthContext(t, tf, "gitlab.com", "glpat-test-token-12345")
    cmdtest.SetupProjectContext(t, tf, "gitlab.com", "owner/repo")

    // Execute command
    cmd := cmd.NewMRListCmd(tf)
    cmd.SetArgs([]string{"--state", "opened"})

    err := cmd.Execute()
    if err != nil {
        t.Fatalf("command failed: %v", err)
    }

    // Verify output
    output := tf.IO.String()
    cmdtest.AssertContains(t, output, "42")
    cmdtest.AssertContains(t, output, "Add feature")
    cmdtest.AssertContains(t, output, "alice")

    // Verify no errors in stderr
    errOutput := tf.IO.ErrString()
    if errOutput != "" {
        t.Errorf("unexpected stderr output: %s", errOutput)
    }
}
```

## See Also

- [E2E Tests README](../e2e/README.md) - End-to-end tests against real GitLab
- [Test Helpers](../../internal/cmdtest/) - Test infrastructure and fixtures
- [Error Integration Tests](../../internal/errors/integration_test.go) - Example integration tests for error handling

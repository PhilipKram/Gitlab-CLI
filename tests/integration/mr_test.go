package integration_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/cmd"
	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

// TestMRList verifies the mr list command integration
func TestMRList(t *testing.T) {
	// Create test factory with mock server
	f := cmdtest.NewTestFactory(t)

	// Set up project context
	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	// Create a mock GitLab API server
	router := cmdtest.NewMockAPIRouter(t)

	// Mock the list merge requests endpoint
	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		// Return a list of merge requests
		mrs := []interface{}{
			cmdtest.FixtureMROpen,
			cmdtest.FixtureMRMerged,
		}
		cmdtest.JSONResponse(w, http.StatusOK, mrs)
	})

	// Start the mock server and intercept requests
	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv // Server is automatically cleaned up via t.Cleanup

	// Create the mr command
	mrCmd := cmd.NewMRCmd(f.Factory)

	// Set up the command to execute the list subcommand
	mrCmd.SetArgs([]string{"list"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	// Execute the mr list command
	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected merge requests
	// The default output format is table format
	if output == "" {
		t.Error("expected output for mr list")
	}

	// Verify output contains merge request titles from fixtures
	expectedElements := []string{
		"Add new feature",  // From FixtureMROpen
		"Fix critical bug", // From FixtureMRMerged
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("mr list output missing element %q\nGot: %s", element, output)
		}
	}
}

// TestMRList_WithState verifies mr list filters by state
func TestMRList_WithState(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		// Check that state parameter is passed
		state := r.URL.Query().Get("state")
		if state != "merged" {
			t.Errorf("expected state parameter to be 'merged', got %q", state)
		}

		// Return only merged merge requests
		mrs := []interface{}{
			cmdtest.FixtureMRMerged,
		}
		cmdtest.JSONResponse(w, http.StatusOK, mrs)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"list", "--state", "merged"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Should contain merged MR
	if !strings.Contains(output, "Fix critical bug") {
		t.Errorf("output should contain merged MR\nGot: %s", output)
	}
}

// TestMRList_EmptyResult verifies mr list handles no results gracefully
func TestMRList_EmptyResult(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		// Return empty list
		mrs := []interface{}{}
		cmdtest.JSONResponse(w, http.StatusOK, mrs)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"list"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle empty results gracefully
	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No merge requests match your search") {
		t.Errorf("expected message about no results\nGot: %s", errOutput)
	}
}

// TestMRList_JSONFormat verifies mr list JSON output format
func TestMRList_JSONFormat(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		mrs := []interface{}{
			cmdtest.FixtureMROpen,
		}
		cmdtest.JSONResponse(w, http.StatusOK, mrs)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"list", "--format", "json"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify JSON output is valid and contains expected data
	if output == "" {
		t.Fatal("expected JSON output")
	}

	// Should contain JSON array markers
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Errorf("expected JSON format output\nGot: %s", output)
	}

	// Should contain merge request data
	if !strings.Contains(output, "Add new feature") {
		t.Errorf("JSON output should contain MR title\nGot: %s", output)
	}
}

// TestMRList_WithLimit verifies mr list respects limit parameter
func TestMRList_WithLimit(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		// Check that per_page parameter is passed
		perPage := r.URL.Query().Get("per_page")
		if perPage != "10" {
			t.Errorf("expected per_page parameter to be '10', got %q", perPage)
		}

		mrs := []interface{}{
			cmdtest.FixtureMROpen,
		}
		cmdtest.JSONResponse(w, http.StatusOK, mrs)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"list", "--limit", "10"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just verify it doesn't error - the mock handler verifies the limit parameter
}

// TestMRList_APIError verifies mr list handles API errors gracefully
func TestMRList_APIError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		// Return an API error
		cmdtest.ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"list"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()

	// Should return an error
	if err == nil {
		t.Fatal("expected error when API returns 401, got nil")
	}

	// Error should be about authentication or authorization
	errMsg := err.Error()
	if !strings.Contains(errMsg, "401") && !strings.Contains(errMsg, "Unauthorized") {
		t.Errorf("expected error about authorization, got: %v", err)
	}
}

// TestMRView verifies the mr view command integration
func TestMRView(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	// Mock the get merge request endpoint
	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		// Return a specific merge request
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.FixtureMROpen)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"view", "1"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected merge request details
	if output == "" {
		t.Error("expected output for mr view")
	}

	// Verify output contains merge request details from fixture
	expectedElements := []string{
		"Add new feature",                   // Title from FixtureMROpen
		"feature/new-feature",               // Source branch
		"main",                              // Target branch
		"test-user",                         // Author username
		"This MR adds a new feature to improve user experience", // Description
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("mr view output missing element %q\nGot: %s", element, output)
		}
	}
}

// TestMRView_JSONFormat verifies mr view JSON output format
func TestMRView_JSONFormat(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.FixtureMROpen)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"view", "1", "--format", "json"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify JSON output is valid and contains expected data
	if output == "" {
		t.Fatal("expected JSON output")
	}

	// Should contain JSON object markers
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Errorf("expected JSON format output\nGot: %s", output)
	}

	// Should contain merge request data
	if !strings.Contains(output, "Add new feature") {
		t.Errorf("JSON output should contain MR title\nGot: %s", output)
	}
}

// TestMRView_NotFound verifies mr view handles non-existent MR
func TestMRView_NotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/merge_requests/999", func(w http.ResponseWriter, r *http.Request) {
		// Return a 404 error
		cmdtest.ErrorResponse(w, http.StatusNotFound, "404 Not Found")
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	mrCmd := cmd.NewMRCmd(f.Factory)
	mrCmd.SetArgs([]string{"view", "999"})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	err := mrCmd.Execute()

	// Should return an error
	if err == nil {
		t.Fatal("expected error when MR not found, got nil")
	}

	// Error should be about not found
	errMsg := err.Error()
	if !strings.Contains(errMsg, "404") && !strings.Contains(errMsg, "Not Found") && !strings.Contains(errMsg, "not found") {
		t.Errorf("expected error about not found, got: %v", err)
	}
}

// TestMRCreate verifies the mr create command integration
func TestMRCreate(t *testing.T) {
	// Create test factory with mock server
	f := cmdtest.NewTestFactory(t)

	// Set up project context
	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	// Create a mock GitLab API server
	router := cmdtest.NewMockAPIRouter(t)

	// Mock the create merge request endpoint
	router.Register("POST", "/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		// Return a newly created merge request
		newMR := map[string]interface{}{
			"id":            103,
			"iid":           4,
			"title":         "New feature branch",
			"state":         "opened",
			"description":   "This is a new feature implementation",
			"web_url":       "https://gitlab.com/test-owner/test-repo/-/merge_requests/4",
			"source_branch": "feature/new-feature",
			"target_branch": "main",
			"author": map[string]interface{}{
				"id":       1,
				"username": "test-user",
				"name":     "Test User",
			},
			"created_at": "2024-01-10T10:00:00.000Z",
		}
		cmdtest.JSONResponse(w, http.StatusCreated, newMR)
	})

	// Start the mock server and intercept requests
	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv // Server is automatically cleaned up via t.Cleanup

	// Create the mr command
	mrCmd := cmd.NewMRCmd(f.Factory)

	// Set up the command to execute the create subcommand
	mrCmd.SetArgs([]string{
		"create",
		"--title", "New feature branch",
		"--description", "This is a new feature implementation",
		"--source-branch", "feature/new-feature",
		"--target-branch", "main",
	})
	mrCmd.SetOut(f.IO.Out)
	mrCmd.SetErr(f.IO.ErrOut)

	// Execute the mr create command
	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains success message or URL
	if output == "" {
		t.Error("expected output for mr create")
	}

	// Verify output contains merge request URL or confirmation
	expectedElements := []string{
		"gitlab.com/test-owner/test-repo/-/merge_requests/4",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("mr create output missing element %q\nGot: %s", element, output)
		}
	}
}

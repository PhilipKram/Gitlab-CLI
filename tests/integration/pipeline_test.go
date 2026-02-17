package integration_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/cmd"
	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

// TestPipelineList verifies the pipeline list command integration
func TestPipelineList(t *testing.T) {
	// Create test factory with mock server
	f := cmdtest.NewTestFactory(t)

	// Set up project context
	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	// Create a mock GitLab API server
	router := cmdtest.NewMockAPIRouter(t)

	// Mock the list pipelines endpoint
	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		// Return a list of pipelines
		pipelines := []interface{}{
			cmdtest.FixturePipelineSuccess,
			cmdtest.FixturePipelineRunning,
		}
		cmdtest.JSONResponse(w, http.StatusOK, pipelines)
	})

	// Start the mock server and intercept requests
	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv // Server is automatically cleaned up via t.Cleanup

	// Create the pipeline command
	pipelineCmd := cmd.NewPipelineCmd(f.Factory)

	// Set up the command to execute the list subcommand
	pipelineCmd.SetArgs([]string{"list"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	// Execute the pipeline list command
	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected pipelines
	// The default output format is table format
	if output == "" {
		t.Error("expected output for pipeline list")
	}

	// Verify output contains pipeline refs from fixtures
	expectedElements := []string{
		"main",     // From FixturePipelineSuccess
		"success",  // Status from FixturePipelineSuccess
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("pipeline list output missing element %q\nGot: %s", element, output)
		}
	}
}

// TestPipelineList_WithStatus verifies pipeline list filters by status
func TestPipelineList_WithStatus(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		// Check that status parameter is passed
		status := r.URL.Query().Get("status")
		if status != "success" {
			t.Errorf("expected status parameter to be 'success', got %q", status)
		}

		// Return only successful pipelines
		pipelines := []interface{}{
			cmdtest.FixturePipelineSuccess,
		}
		cmdtest.JSONResponse(w, http.StatusOK, pipelines)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"list", "--status", "success"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Should contain successful pipeline
	if !strings.Contains(output, "success") {
		t.Errorf("output should contain successful pipeline\nGot: %s", output)
	}
}

// TestPipelineList_WithRef verifies pipeline list filters by ref
func TestPipelineList_WithRef(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		// Check that ref parameter is passed
		ref := r.URL.Query().Get("ref")
		if ref != "main" {
			t.Errorf("expected ref parameter to be 'main', got %q", ref)
		}

		// Return only pipelines for main branch
		pipelines := []interface{}{
			cmdtest.FixturePipelineSuccess,
		}
		cmdtest.JSONResponse(w, http.StatusOK, pipelines)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"list", "--ref", "main"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Should contain main branch
	if !strings.Contains(output, "main") {
		t.Errorf("output should contain main branch\nGot: %s", output)
	}
}

// TestPipelineList_EmptyResult verifies pipeline list handles no results gracefully
func TestPipelineList_EmptyResult(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		// Return empty list
		pipelines := []interface{}{}
		cmdtest.JSONResponse(w, http.StatusOK, pipelines)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"list"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle empty results gracefully
	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No pipelines found") {
		t.Errorf("expected message about no results\nGot: %s", errOutput)
	}
}

// TestPipelineList_JSONFormat verifies pipeline list JSON output format
func TestPipelineList_JSONFormat(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		pipelines := []interface{}{
			cmdtest.FixturePipelineSuccess,
		}
		cmdtest.JSONResponse(w, http.StatusOK, pipelines)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"list", "--format", "json"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
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

	// Should contain pipeline data
	if !strings.Contains(output, "success") {
		t.Errorf("JSON output should contain pipeline status\nGot: %s", output)
	}
}

// TestPipelineList_WithLimit verifies pipeline list respects limit parameter
func TestPipelineList_WithLimit(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		// Check that per_page parameter is passed
		perPage := r.URL.Query().Get("per_page")
		if perPage != "10" {
			t.Errorf("expected per_page parameter to be '10', got %q", perPage)
		}

		pipelines := []interface{}{
			cmdtest.FixturePipelineSuccess,
		}
		cmdtest.JSONResponse(w, http.StatusOK, pipelines)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"list", "--limit", "10"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just verify it doesn't error - the mock handler verifies the limit parameter
}

// TestPipelineList_APIError verifies pipeline list handles API errors gracefully
func TestPipelineList_APIError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		// Return an API error
		cmdtest.ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"list"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()

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

// TestPipelineView verifies the pipeline view command integration
func TestPipelineView(t *testing.T) {
	// Create test factory with mock server
	f := cmdtest.NewTestFactory(t)

	// Set up project context
	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	// Create a mock GitLab API server
	router := cmdtest.NewMockAPIRouter(t)

	// Mock the get pipeline endpoint
	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines/300", func(w http.ResponseWriter, r *http.Request) {
		// Return a successful pipeline
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.FixturePipelineSuccess)
	})

	// Start the mock server and intercept requests
	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv // Server is automatically cleaned up via t.Cleanup

	// Create the pipeline command
	pipelineCmd := cmd.NewPipelineCmd(f.Factory)

	// Set up the command to execute the view subcommand with a pipeline ID
	pipelineCmd.SetArgs([]string{"view", "300"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	// Execute the pipeline view command
	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected pipeline details
	if output == "" {
		t.Error("expected output for pipeline view")
	}

	// Verify output contains pipeline details from fixture
	expectedElements := []string{
		"Pipeline #300", // Pipeline ID
		"main",          // Ref from FixturePipelineSuccess
		"success",       // Status from FixturePipelineSuccess
		"abc123def",     // Part of SHA from FixturePipelineSuccess
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("pipeline view output missing element %q\nGot: %s", element, output)
		}
	}
}

// TestPipelineView_NotFound verifies pipeline view handles missing pipeline
func TestPipelineView_NotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines/99999", func(w http.ResponseWriter, r *http.Request) {
		// Return 404 not found
		cmdtest.ErrorResponse(w, http.StatusNotFound, "404 Pipeline Not Found")
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"view", "99999"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing pipeline, got nil")
	}

	// Verify error message mentions the failure
	if !strings.Contains(err.Error(), "Failed to get pipeline") {
		t.Errorf("expected error about pipeline not found, got: %v", err)
	}
}

// TestPipelineView_JSONFormat verifies pipeline view JSON output format
func TestPipelineView_JSONFormat(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("GET", "/api/v4/projects/test-owner/test-repo/pipelines/300", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.FixturePipelineSuccess)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"view", "300", "--json"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Should output JSON object format
	if !strings.HasPrefix(output, "{") {
		t.Errorf("expected JSON object format\nGot: %s", output)
	}

	// Verify JSON contains expected fields
	expectedFields := []string{
		`"id"`,
		`"ref"`,
		`"status"`,
		`"sha"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("JSON output missing field %s\nGot: %s", field, output)
		}
	}
}

// TestPipelineRun verifies the pipeline run command integration
func TestPipelineRun(t *testing.T) {
	// Create test factory with mock server
	f := cmdtest.NewTestFactory(t)

	// Set up project context
	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	// Create a mock GitLab API server
	router := cmdtest.NewMockAPIRouter(t)

	// Mock the create pipeline endpoint (POST)
	router.Register("POST", "/api/v4/projects/test-owner/test-repo/pipeline", func(w http.ResponseWriter, r *http.Request) {
		// Return a newly created pipeline (pending status)
		cmdtest.JSONResponse(w, http.StatusCreated, cmdtest.FixturePipelinePending)
	})

	// Start the mock server and intercept requests
	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv // Server is automatically cleaned up via t.Cleanup

	// Create the pipeline command
	pipelineCmd := cmd.NewPipelineCmd(f.Factory)

	// Set up the command to execute the run subcommand with a ref
	pipelineCmd.SetArgs([]string{"run", "--ref", "main"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	// Execute the pipeline run command
	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected elements
	if output == "" {
		t.Error("expected output for pipeline run")
	}

	// Verify output contains pipeline creation confirmation
	expectedElements := []string{
		"Created pipeline",
		"Status:",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("pipeline run output missing element %q\nGot: %s", element, output)
		}
	}
}

// TestPipelineRun_WithVariables verifies pipeline run with variables
func TestPipelineRun_WithVariables(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("POST", "/api/v4/projects/test-owner/test-repo/pipeline", func(w http.ResponseWriter, r *http.Request) {
		// Return a newly created pipeline
		cmdtest.JSONResponse(w, http.StatusCreated, cmdtest.FixturePipelinePending)
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"run", "--ref", "main", "--variables", "ENV=production", "--variables", "DEBUG=false"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Should contain pipeline creation confirmation
	if !strings.Contains(output, "Created pipeline") {
		t.Errorf("expected pipeline creation confirmation\nGot: %s", output)
	}
}

// TestPipelineRun_MissingRef verifies pipeline run requires ref flag
func TestPipelineRun_MissingRef(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)
	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"run"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()
	if err == nil {
		t.Fatal("expected error when ref is missing, got nil")
	}

	// Should error about missing ref
	if !strings.Contains(err.Error(), "ref") && !strings.Contains(err.Error(), "branch") {
		t.Errorf("expected error about missing ref/branch, got: %v", err)
	}
}

// TestPipelineRun_APIError verifies pipeline run handles API errors gracefully
func TestPipelineRun_APIError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	cmdtest.SetupProjectContext(t, f, "gitlab.com", "test-owner", "test-repo")
	cmdtest.SetupAuthContext(t, f, "gitlab.com", "glpat-test-token-12345")

	router := cmdtest.NewMockAPIRouter(t)

	router.Register("POST", "/api/v4/projects/test-owner/test-repo/pipeline", func(w http.ResponseWriter, r *http.Request) {
		// Return an API error
		cmdtest.ErrorResponse(w, http.StatusBadRequest, "Reference not found")
	})

	srv := cmdtest.MockGitLabServer(t, "gitlab.com", router.ServeHTTP)
	_ = srv

	pipelineCmd := cmd.NewPipelineCmd(f.Factory)
	pipelineCmd.SetArgs([]string{"run", "--ref", "nonexistent-branch"})
	pipelineCmd.SetOut(f.IO.Out)
	pipelineCmd.SetErr(f.IO.ErrOut)

	err := pipelineCmd.Execute()

	// Should return an error
	if err == nil {
		t.Fatal("expected error when API returns 400, got nil")
	}

	// Error should mention the failure
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Failed to create pipeline") {
		t.Errorf("expected error about pipeline creation failure, got: %v", err)
	}
}

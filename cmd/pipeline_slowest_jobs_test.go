package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestPipelineSlowestJobsCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineSlowestJobsCmd(f)

	if cmd.Use != "slowest-jobs" {
		t.Errorf("expected Use to be 'slowest-jobs', got %q", cmd.Use)
	}

	if cmd.Short != "Identify slowest CI jobs" {
		t.Errorf("expected Short to be 'Identify slowest CI jobs', got %q", cmd.Short)
	}
}

func TestPipelineSlowestJobsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineSlowestJobsCmd(f)

	expectedFlags := []string{
		"branch",
		"days",
		"limit",
		"format",
		"json",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default days is 30
	daysFlag := cmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Fatal("days flag not found")
	}
	if daysFlag.DefValue != "30" {
		t.Errorf("expected default days to be 30, got %q", daysFlag.DefValue)
	}

	// Verify default limit is 10
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.DefValue != "10" {
		t.Errorf("expected default limit to be 10, got %q", limitFlag.DefValue)
	}

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be table, got %q", formatFlag.DefValue)
	}
}

func TestPipelineSlowestJobsCmd_FlagAliases(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineSlowestJobsCmd(f)

	// Verify branch flag has alias 'b'
	branchFlag := cmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("branch flag not found")
	}
	if branchFlag.Shorthand != "b" {
		t.Errorf("expected branch flag to have shorthand 'b', got %q", branchFlag.Shorthand)
	}

	// Verify days flag has alias 'd'
	daysFlag := cmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Fatal("days flag not found")
	}
	if daysFlag.Shorthand != "d" {
		t.Errorf("expected days flag to have shorthand 'd', got %q", daysFlag.Shorthand)
	}

	// Verify limit flag has alias 'L'
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.Shorthand != "L" {
		t.Errorf("expected limit flag to have shorthand 'L', got %q", limitFlag.Shorthand)
	}

	// Verify format flag has alias 'F'
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.Shorthand != "F" {
		t.Errorf("expected format flag to have shorthand 'F', got %q", formatFlag.Shorthand)
	}
}

// ============================================================================
// EXECUTION TESTS
// ============================================================================

func TestPipelineSlowestJobs_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			// Return pipeline list
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"ref":    "main",
					"status": "success",
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			// Return jobs for pipeline
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":       101,
					"name":     "test",
					"stage":    "test",
					"status":   "success",
					"duration": 120.5,
				},
				map[string]interface{}{
					"id":       102,
					"name":     "build",
					"stage":    "build",
					"status":   "success",
					"duration": 300.2,
				},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineSlowestJobs_WithBranch(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			// Verify branch parameter is present
			ref := r.URL.Query().Get("ref")
			if ref != "develop" {
				t.Errorf("expected ref to be 'develop', got %q", ref)
			}
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"ref":    "develop",
					"status": "success",
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":       101,
					"name":     "test",
					"stage":    "test",
					"status":   "success",
					"duration": 120.5,
				},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	cmd.SetArgs([]string{"--branch", "develop"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineSlowestJobs_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"ref":    "main",
					"status": "success",
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":       101,
					"name":     "test",
					"stage":    "test",
					"status":   "success",
					"duration": 120.5,
				},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	cmd.SetArgs([]string{"--format", "json"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "jobs") {
		t.Errorf("expected JSON output to contain 'jobs' field, got: %s", output)
	}
}

func TestPipelineSlowestJobs_DeprecatedJSONFlag(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"ref":    "main",
					"status": "success",
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":       101,
					"name":     "test",
					"stage":    "test",
					"status":   "success",
					"duration": 120.5,
				},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	cmd.SetArgs([]string{"--json"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "jobs") {
		t.Errorf("expected JSON output to contain 'jobs' field, got: %s", output)
	}
}

// ============================================================================
// ERROR TESTS
// ============================================================================

func TestPipelineSlowestJobs_NoPipelines(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed but output message about no pipelines to stderr
	output := f.IO.ErrString()
	if !strings.Contains(output, "No pipelines found") {
		t.Errorf("expected output to mention no pipelines, got: %s", output)
	}
}

func TestPipelineSlowestJobs_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestPipelineSlowestJobs_InvalidFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"ref":    "main",
					"status": "success",
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":       101,
					"name":     "test",
					"stage":    "test",
					"status":   "success",
					"duration": 120.5,
				},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineSlowestJobsCmd(f.Factory)
	cmd.SetArgs([]string{"--format", "invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("expected error message to mention invalid format, got: %v", err)
	}
}

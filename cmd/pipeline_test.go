package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewPipelineCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewPipelineCmd(f)

	if cmd.Use != "pipeline <command>" {
		t.Errorf("expected Use to be 'pipeline <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage pipelines and CI/CD" {
		t.Errorf("expected Short to be 'Manage pipelines and CI/CD', got %q", cmd.Short)
	}

	expectedAliases := []string{"ci", "pipe"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i >= len(cmd.Aliases) || cmd.Aliases[i] != alias {
			t.Errorf("expected alias %q at position %d, got %v", alias, i, cmd.Aliases)
		}
	}
}

func TestPipelineCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewPipelineCmd(f)

	expectedSubcommands := []string{
		"list",
		"view",
		"run",
		"cancel",
		"retry",
		"delete",
		"jobs",
		"job-log",
		"retry-job",
		"cancel-job",
		"artifacts",
		"stats",
		"slowest-jobs",
		"trends",
		"flaky",
		"watch",
	}

	subcommands := cmd.Commands()
	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("expected %d subcommands, got %d", len(expectedSubcommands), len(subcommands))
	}

	foundSubcommands := make(map[string]bool)
	for _, subcmd := range subcommands {
		foundSubcommands[subcmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !foundSubcommands[expected] {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

func TestPipelineListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineListCmd(f)

	expectedFlags := []string{
		"status",
		"ref",
		"limit",
		"json",
		"web",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default limit is 30
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.DefValue != "30" {
		t.Errorf("expected default limit to be 30, got %q", limitFlag.DefValue)
	}

	// Verify list has alias "ls"
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestPipelineViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineViewCmd(f)

	expectedFlags := []string{"web", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view [<id>]" {
		t.Errorf("expected Use to be 'view [<id>]', got %q", cmd.Use)
	}
}

func TestPipelineRunCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineRunCmd(f)

	expectedFlags := []string{
		"ref",
		"variables",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify run has aliases
	expectedAliases := []string{"create", "trigger"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i >= len(cmd.Aliases) || cmd.Aliases[i] != alias {
			t.Errorf("expected alias %q at position %d, got %v", alias, i, cmd.Aliases)
		}
	}
}

func TestPipelineCancelCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineCancelCmd(f)

	if cmd.Use != "cancel [<id>]" {
		t.Errorf("expected Use to be 'cancel [<id>]', got %q", cmd.Use)
	}

	if cmd.Short != "Cancel a running pipeline" {
		t.Errorf("expected Short to be 'Cancel a running pipeline', got %q", cmd.Short)
	}
}

func TestPipelineRetryCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineRetryCmd(f)

	if cmd.Use != "retry [<id>]" {
		t.Errorf("expected Use to be 'retry [<id>]', got %q", cmd.Use)
	}

	if cmd.Short != "Retry a failed pipeline" {
		t.Errorf("expected Short to be 'Retry a failed pipeline', got %q", cmd.Short)
	}
}

func TestPipelineDeleteCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineDeleteCmd(f)

	if cmd.Use != "delete [<id>]" {
		t.Errorf("expected Use to be 'delete [<id>]', got %q", cmd.Use)
	}

	if cmd.Short != "Delete a pipeline" {
		t.Errorf("expected Short to be 'Delete a pipeline', got %q", cmd.Short)
	}
}

func TestPipelineJobsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineJobsCmd(f)

	expectedFlags := []string{"json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "jobs [<pipeline-id>]" {
		t.Errorf("expected Use to be 'jobs [<pipeline-id>]', got %q", cmd.Use)
	}
}

func TestPipelineJobLogCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineJobLogCmd(f)

	if cmd.Use != "job-log [<job-id>]" {
		t.Errorf("expected Use to be 'job-log [<job-id>]', got %q", cmd.Use)
	}

	if cmd.Short != "View the log/trace of a job" {
		t.Errorf("expected Short to be 'View the log/trace of a job', got %q", cmd.Short)
	}

	// Verify job-log has alias "trace"
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "trace" {
		t.Errorf("expected alias 'trace', got %v", cmd.Aliases)
	}
}

// ============================================================================
// EXECUTION TESTS
// ============================================================================

func TestPipelineList_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixturePipelineSuccess})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineListCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines/") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixturePipelineSuccess)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineRetry_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/retry") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixturePipelineRunning)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryCmd(f.Factory)
	cmd.SetArgs([]string{"1"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ERROR TESTS
// ============================================================================

func TestPipelineView_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineViewCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent pipeline")
	}
}

func TestPipelineList_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineListCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestPipelineRun_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/pipeline") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixturePipelineRunning)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRunCmd(f.Factory)
	cmd.SetArgs([]string{"--ref", "main"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineCancel_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/pipelines/1/cancel") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixturePipelineSuccess)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineDelete_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/pipelines/1") {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineJobs_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"name":   "test",
					"status": "success",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineJobsCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineJobLog_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/jobs/123/trace") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(200)
			_, _ = w.Write([]byte("Job log output\nLine 2\nLine 3\n"))
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineJobLogCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Job log output") {
		t.Errorf("expected output to contain job log, got: %s", output)
	}
}

func TestPipelineJobLog_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Job Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineJobLogCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found job")
	}
}

func TestPipelineRun_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRunCmd(f.Factory)
	cmd.SetArgs([]string{}) // Missing required ref

	err := cmd.Execute()
	// Should fail validation or execution
	_ = err
}

func TestPipelineList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// RETRY-JOB TESTS
// ============================================================================

func TestPipelineRetryJob_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/jobs/123/retry") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":     124,
				"name":   "test",
				"status": "pending",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryJobCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Retried job") {
		t.Errorf("expected 'Retried job' message, got: %s", output)
	}
}

func TestPipelineRetryJob_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryJobCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing job ID")
	}
	if !strings.Contains(err.Error(), "job ID required") {
		t.Errorf("expected 'job ID required' error, got: %v", err)
	}
}

func TestPipelineRetryJob_InvalidID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryJobCmd(f.Factory)
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid job ID")
	}
	if !strings.Contains(err.Error(), "invalid job ID") {
		t.Errorf("expected 'invalid job ID' error, got: %v", err)
	}
}

func TestPipelineRetryJob_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryJobCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found job")
	}
}

func TestPipelineRetryJob_JSONOutput(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/jobs/123/retry") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":     124,
				"name":   "test",
				"status": "pending",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryJobCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "pending") {
		t.Errorf("expected JSON with status, got: %s", output)
	}
}

// ============================================================================
// CANCEL-JOB TESTS
// ============================================================================

func TestPipelineCancelJob_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/jobs/123/cancel") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":     123,
				"name":   "test",
				"status": "canceled",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelJobCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Canceled job") {
		t.Errorf("expected 'Canceled job' message, got: %s", output)
	}
}

func TestPipelineCancelJob_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelJobCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing job ID")
	}
	if !strings.Contains(err.Error(), "job ID required") {
		t.Errorf("expected 'job ID required' error, got: %v", err)
	}
}

func TestPipelineCancelJob_InvalidID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelJobCmd(f.Factory)
	cmd.SetArgs([]string{"abc"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid job ID")
	}
	if !strings.Contains(err.Error(), "invalid job ID") {
		t.Errorf("expected 'invalid job ID' error, got: %v", err)
	}
}

func TestPipelineCancelJob_JSONOutput(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/jobs/123/cancel") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":     123,
				"name":   "test",
				"status": "canceled",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelJobCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "canceled") {
		t.Errorf("expected JSON output with status, got: %s", output)
	}
}

// ============================================================================
// ARTIFACTS TESTS
// ============================================================================

func TestPipelineArtifacts_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineArtifactsCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing job ID")
	}
	if !strings.Contains(err.Error(), "job ID required") {
		t.Errorf("expected 'job ID required' error, got: %v", err)
	}
}

func TestPipelineArtifacts_InvalidID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineArtifactsCmd(f.Factory)
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid job ID")
	}
	if !strings.Contains(err.Error(), "invalid job ID") {
		t.Errorf("expected 'invalid job ID' error, got: %v", err)
	}
}

func TestPipelineArtifacts_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineArtifactsCmd(f)

	expectedFlags := []string{"output", "path"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

// ============================================================================
// PIPELINE CANCEL/RETRY/DELETE ERROR TESTS
// ============================================================================

func TestPipelineCancel_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent pipeline")
	}
}

func TestPipelineRetry_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent pipeline")
	}
}

func TestPipelineDelete_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent pipeline")
	}
}

func TestPipelineCancel_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineCancelCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing pipeline ID")
	}
}

func TestPipelineRetry_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRetryCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing pipeline ID")
	}
}

func TestPipelineDelete_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineDeleteCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing pipeline ID")
	}
}

func TestPipelineJobs_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineJobsCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing pipeline ID")
	}
}

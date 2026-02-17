package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestPipelineStatsCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineStatsCmd(f)

	if cmd.Use != "stats" {
		t.Errorf("expected Use to be 'stats', got %q", cmd.Use)
	}

	if cmd.Short != "Show pipeline statistics" {
		t.Errorf("expected Short to be 'Show pipeline statistics', got %q", cmd.Short)
	}

	expectedLong := "Display pipeline success/failure rates and statistics for a configurable time period."
	if cmd.Long != expectedLong {
		t.Errorf("expected Long to be %q, got %q", expectedLong, cmd.Long)
	}

	if cmd.Example == "" {
		t.Error("expected Example to be non-empty")
	}
}

func TestPipelineStatsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineStatsCmd(f)

	expectedFlags := []string{
		"branch",
		"days",
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

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be 'table', got %q", formatFlag.DefValue)
	}

	// Verify json flag default is false
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("json flag not found")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("expected default json to be false, got %q", jsonFlag.DefValue)
	}

	// Verify branch flag has shorthand -b
	branchFlag := cmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("branch flag not found")
	}
	if branchFlag.Shorthand != "b" {
		t.Errorf("expected branch shorthand to be 'b', got %q", branchFlag.Shorthand)
	}

	// Verify days flag has shorthand -d
	if daysFlag.Shorthand != "d" {
		t.Errorf("expected days shorthand to be 'd', got %q", daysFlag.Shorthand)
	}

	// Verify format flag has shorthand -F
	if formatFlag.Shorthand != "F" {
		t.Errorf("expected format shorthand to be 'F', got %q", formatFlag.Shorthand)
	}
}

func TestPipelineStatsCmd_NoAliases(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineStatsCmd(f)

	if len(cmd.Aliases) != 0 {
		t.Errorf("expected no aliases, got %v", cmd.Aliases)
	}
}

// ============================================================================
// EXECUTION TESTS
// ============================================================================

func TestPipelineStats_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixturePipelineSuccess,
				cmdtest.FixturePipelineFailed,
				cmdtest.FixturePipelineRunning,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineStatsCmd(f.Factory)
	cmd.SetArgs([]string{"--days", "7"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineStats_NoPipelines(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineStatsCmd(f.Factory)
	cmd.SetArgs([]string{"--days", "7"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No pipelines found") {
		t.Errorf("expected 'No pipelines found' message, got: %s", errOutput)
	}
}

func TestPipelineStats_WithBranch(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") {
			// Verify branch filter is passed
			if r.URL.Query().Get("ref") != "main" {
				t.Errorf("expected ref=main, got %s", r.URL.Query().Get("ref"))
			}
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixturePipelineSuccess,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineStatsCmd(f.Factory)
	cmd.SetArgs([]string{"--branch", "main", "--days", "7"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineStats_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixturePipelineSuccess,
				cmdtest.FixturePipelineFailed,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineStatsCmd(f.Factory)
	cmd.SetArgs([]string{"--format", "json", "--days", "7"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	// JSON output should contain total_pipelines key
	if !strings.Contains(output, "total_pipelines") {
		t.Errorf("expected JSON output with total_pipelines, got: %s", output)
	}
}

func TestPipelineStats_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineStatsCmd(f.Factory)
	cmd.SetArgs([]string{"--days", "7"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

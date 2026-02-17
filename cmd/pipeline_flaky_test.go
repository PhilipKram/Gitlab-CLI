package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestPipelineFlakyCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineFlakyCmd(f)

	if cmd.Use != "flaky" {
		t.Errorf("expected Use to be 'flaky', got %q", cmd.Use)
	}

	if cmd.Short != "Detect flaky jobs with inconsistent results" {
		t.Errorf("expected Short to be 'Detect flaky jobs with inconsistent results', got %q", cmd.Short)
	}
}

func TestPipelineFlakyCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineFlakyCmd(f)

	expectedFlags := []string{
		"branch",
		"days",
		"threshold",
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

	// Verify default threshold is 0.0
	thresholdFlag := cmd.Flags().Lookup("threshold")
	if thresholdFlag == nil {
		t.Fatal("threshold flag not found")
	}
	if thresholdFlag.DefValue != "0" {
		t.Errorf("expected default threshold to be 0, got %q", thresholdFlag.DefValue)
	}

	// Verify default limit is 20
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.DefValue != "20" {
		t.Errorf("expected default limit to be 20, got %q", limitFlag.DefValue)
	}

	// Verify default format is "table"
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be 'table', got %q", formatFlag.DefValue)
	}

	// Verify default json is false
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("json flag not found")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("expected default json to be false, got %q", jsonFlag.DefValue)
	}
}

// ============================================================================
// abs() TESTS
// ============================================================================

func TestAbs(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{5.0, 5.0},
		{-5.0, 5.0},
		{0.0, 0.0},
		{-0.1, 0.1},
		{100.5, 100.5},
		{-100.5, 100.5},
	}

	for _, tt := range tests {
		got := abs(tt.input)
		if got != tt.want {
			t.Errorf("abs(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ============================================================================
// EXECUTION TESTS
// ============================================================================

func TestPipelineFlaky_NoPipelines(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineFlakyCmd(f.Factory)
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

func TestPipelineFlaky_WithFlakyJobs(t *testing.T) {
	callCount := 0
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		// List pipelines
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":         1,
					"ref":        "main",
					"sha":        "abc123",
					"status":     "success",
					"created_at": "2024-01-01T00:00:00.000Z",
					"updated_at": "2024-01-01T00:00:00.000Z",
				},
				map[string]interface{}{
					"id":         2,
					"ref":        "main",
					"sha":        "def456",
					"status":     "failed",
					"created_at": "2024-01-02T00:00:00.000Z",
					"updated_at": "2024-01-02T00:00:00.000Z",
				},
			})
			return
		}
		// List jobs per pipeline
		if strings.Contains(r.URL.Path, "/jobs") {
			callCount++
			if callCount == 1 {
				// First pipeline: test job succeeds
				cmdtest.JSONResponse(w, 200, []interface{}{
					map[string]interface{}{
						"id":          10,
						"name":        "unit-tests",
						"stage":       "test",
						"status":      "success",
						"duration":    60.0,
						"finished_at": "2024-01-01T01:00:00.000Z",
					},
				})
			} else {
				// Second pipeline: test job fails
				cmdtest.JSONResponse(w, 200, []interface{}{
					map[string]interface{}{
						"id":          11,
						"name":        "unit-tests",
						"stage":       "test",
						"status":      "failed",
						"duration":    45.0,
						"finished_at": "2024-01-02T01:00:00.000Z",
					},
				})
			}
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineFlakyCmd(f.Factory)
	cmd.SetArgs([]string{"--days", "7"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineFlaky_NoFlakyJobs(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") && !strings.Contains(r.URL.Path, "/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":         1,
					"ref":        "main",
					"sha":        "abc123",
					"status":     "success",
					"created_at": "2024-01-01T00:00:00.000Z",
					"updated_at": "2024-01-01T00:00:00.000Z",
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/jobs") {
			// All jobs succeed - no flaky jobs
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":          10,
					"name":        "unit-tests",
					"stage":       "test",
					"status":      "success",
					"duration":    60.0,
					"finished_at": "2024-01-01T01:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineFlakyCmd(f.Factory)
	cmd.SetArgs([]string{"--days", "7"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No flaky jobs detected") {
		t.Errorf("expected 'No flaky jobs detected' message, got: %s", errOutput)
	}
}

func TestPipelineFlaky_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineFlakyCmd(f.Factory)
	cmd.SetArgs([]string{"--days", "7"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

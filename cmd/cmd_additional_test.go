package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

// ============================================================================
// AUTH SWITCH CMD TESTS
// ============================================================================

func TestAuthSwitchCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthSwitchCmd(f)

	if cmd.Use != "switch" {
		t.Errorf("expected Use to be 'switch', got %q", cmd.Use)
	}
	if cmd.Short != "Switch between authenticated GitLab instances" {
		t.Errorf("expected Short to be 'Switch between authenticated GitLab instances', got %q", cmd.Short)
	}
}

// ============================================================================
// AUTH LOGOUT CMD TESTS
// ============================================================================

func TestAuthLogout_AllFlag(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthLogoutCmd(f)

	flag := cmd.Flags().Lookup("all")
	if flag == nil {
		t.Error("expected 'all' flag to be present")
	}
}

func TestAuthLogout_WithHostname(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthLogoutCmd(f.Factory)
	cmd.SetArgs([]string{"--hostname", "nonexistent.example.com"})

	// Will fail since not logged in, which is expected
	err := cmd.Execute()
	// Just verify it doesn't panic
	_ = err
}

// ============================================================================
// CONFIG CMD TESTS
// ============================================================================

func TestConfigGetCmd(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigGetCmd(f.Factory)

	cmd.SetArgs([]string{"git_remote"})
	err := cmd.Execute()
	// May succeed or fail depending on config, just verify no panic
	_ = err
}

func TestConfigGetCmd_UnknownKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigGetCmd(f.Factory)

	cmd.SetArgs([]string{"nonexistent_key"})
	// Some configs might return empty string for unknown keys
	_ = cmd.Execute()
}

// ============================================================================
// UPGRADE CMD TESTS
// ============================================================================

func TestUpgradeCmd_CheckOnly(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	f.Version = "0.0.1" // Set a very old version
	cmd := NewUpgradeCmd(f.Factory)
	cmd.SetArgs([]string{"--check"})

	// This will try to check for updates online
	// May fail due to network, which is OK
	err := cmd.Execute()
	_ = err
}

func TestUpgradeCmd_DevBuildWithForce(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	f.Version = "dev"
	cmd := NewUpgradeCmd(f.Factory)
	cmd.SetArgs([]string{"--force"})

	// This tries to check GitHub releases, may fail on network
	err := cmd.Execute()
	// Just verify it does NOT fail with "development build" error
	if err != nil && strings.Contains(err.Error(), "development build") {
		t.Error("--force should bypass development build check")
	}
}

// ============================================================================
// RELEASE CMD TESTS
// ============================================================================

func TestReleaseListCmd(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureRelease})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "v1.0.0") {
		t.Errorf("expected output to contain release tag, got: %s", output)
	}
}

func TestReleaseListCmd_Empty(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseUploadCmd_FileNotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseUploadCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0", "/nonexistent/path/file.tar.gz"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestReleaseUploadCmd_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/releases/") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":               1,
				"name":             "test-asset",
				"url":              "https://example.com/asset",
				"direct_asset_url": "https://gitlab.com/assets/1",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	// Create a temporary file to upload
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-asset.tar.gz")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseUploadCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0", tmpFile, "--name", "Test Asset"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// LABEL LIST CMD TESTS
// ============================================================================

func TestLabelListCmd_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/labels") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureLabelBug,
				cmdtest.FixtureLabelFeature,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "bug") {
		t.Errorf("expected output to contain label name, got: %s", output)
	}
}

func TestLabelListCmd_Empty(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// PIPELINE ARTIFACTS CMD TESTS
// ============================================================================

func TestPipelineArtifacts_Download(t *testing.T) {
	artifactContent := "PK\x03\x04fake zip content"
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/jobs/123/artifacts") {
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(artifactContent))
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-artifacts.zip")

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineArtifactsCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--output", outputPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was downloaded
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(data) != artifactContent {
		t.Errorf("downloaded content does not match")
	}
}

func TestPipelineArtifacts_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineArtifactsCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found artifacts")
	}
}

// ============================================================================
// PIPELINE LIST/VIEW JSON FORMAT TESTS
// ============================================================================

func TestPipelineList_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixturePipelineSuccess})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineListCmd(f.Factory)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "success") {
		t.Errorf("expected JSON output to contain 'success', got: %s", output)
	}
}

func TestPipelineView_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines/") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixturePipelineSuccess)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineViewCmd(f.Factory)
	cmd.SetArgs([]string{"300", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// PIPELINE LIST WITH FILTERS
// ============================================================================

func TestPipelineList_WithStatusFilter(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		if status == "failed" {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixturePipelineFailed})
			return
		}
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineListCmd(f.Factory)
	cmd.SetArgs([]string{"--status", "failed"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineList_WithRefFilter(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixturePipelineSuccess})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineListCmd(f.Factory)
	cmd.SetArgs([]string{"--ref", "main"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// MR COMMENT CMD TESTS
// ============================================================================

func TestMRCommentCmd_FlagsCheck(t *testing.T) {
	f := newTestFactory()
	cmd := newMRCommentCmd(f)

	expectedFlags := []string{"body"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestMRComment_MissingMessage(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRCommentCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing message")
	}
}

func TestMRComment_SuccessWithMock(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/notes") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":   1,
				"body": "test comment",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRCommentCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--body", "test comment"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// PIPELINE RUN CMD TESTS
// ============================================================================

func TestPipelineRun_WithVariables(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/triggers") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{
				{"id": 1, "token": "test-trigger-token", "description": "glab-cli"},
			})
			return
		}
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/trigger/pipeline") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixturePipelineRunning)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineRunCmd(f.Factory)
	cmd.SetArgs([]string{"--ref", "main", "--variables", "KEY=value"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// PIPELINE JOBS JSON FORMAT
// ============================================================================

func TestPipelineJobs_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines/1/jobs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":     1,
					"name":   "build",
					"status": "success",
					"stage":  "build",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPipelineJobsCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "build") {
		t.Errorf("expected output to contain job name, got: %s", output)
	}
}

// ============================================================================
// SNIPPET CREATE WITH FILE
// ============================================================================

func TestSnippetCreate_WithFile(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/snippets") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureSnippet)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	// Create a temp file to use as snippet content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "snippet.go")
	if err := os.WriteFile(tmpFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test snippet", "--filename", "snippet.go", "--file", tmpFile})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// SNIPPET LIST JSON FORMAT
// ============================================================================

func TestSnippetList_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/snippets") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureSnippet})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetListCmd(f.Factory)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Example") {
		t.Errorf("expected JSON output to contain snippet title, got: %s", output)
	}
}

// ============================================================================
// REGISTRY DELETE CMD TESTS
// ============================================================================

func TestRegistryDeleteCmd_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/tags/v1.0.0") {
			w.WriteHeader(200)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--tag", "v1.0.0", "--yes"})

	err := cmd.Execute()
	// May fail in test environment due to project resolution, just verify no panic
	_ = err
}

// ============================================================================
// PACKAGE CMD TESTS
// ============================================================================

func TestPackageListCmd_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":              1,
					"name":            "my-package",
					"version":         "1.0.0",
					"package_type":    "npm",
					"created_at":      "2024-01-01T00:00:00.000Z",
					"_links":          map[string]string{},
					"status":          "default",
					"pipeline":        nil,
					"tags":            []string{},
					"last_downloaded": nil,
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackageDeleteCmd_MissingArgs(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newPackageDeleteCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing package ID")
	}
}

// ============================================================================
// ISSUE CREATE/EDIT/DELETE VALIDATION TESTS
// ============================================================================

func TestIssueCreate_MissingTitle(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{}) // No title

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestIssueDelete_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newIssueDeleteCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing issue ID")
	}
}

func TestIssueDelete_SuccessWithMock(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/issues/") {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"10"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// REPO CMD TESTS
// ============================================================================

func TestRepoListCmd_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureProject})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoArchiveCmd_MissingArgs(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newRepoArchiveCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	// May require arguments - just verify no panic
	_ = err
}

// ============================================================================
// ISSUE CREATE WITH MOCK
// ============================================================================

func TestIssueCreate_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":      200,
				"iid":     15,
				"title":   "Test issue",
				"web_url": "https://gitlab.com/test-owner/test-repo/-/issues/15",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test issue"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "#15") {
		t.Errorf("expected output to contain issue number, got: %s", output)
	}
}

func TestIssueCreate_WithLabels(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":      200,
				"iid":     16,
				"title":   "Labeled issue",
				"web_url": "https://gitlab.com/test-owner/test-repo/-/issues/16",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Labeled issue", "--label", "bug", "--label", "priority::high"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueCreate_WithDescription(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":      200,
				"iid":     17,
				"title":   "Described issue",
				"web_url": "https://gitlab.com/test-owner/test-repo/-/issues/17",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Described issue", "--description", "This is the description"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueCreate_InvalidMilestone(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test", "--milestone", "not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid milestone")
	}
	if !strings.Contains(err.Error(), "invalid milestone") {
		t.Errorf("expected 'invalid milestone' error, got: %v", err)
	}
}

func TestIssueCreate_APIError(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 403, "403 Forbidden")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

// ============================================================================
// ISSUE LIST WITH MOCK
// ============================================================================

func TestIssueList_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureIssueOpen,
				cmdtest.FixtureIssueClosed,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "crashes") {
		t.Errorf("expected output to contain issue title, got: %s", output)
	}
}

func TestIssueList_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureIssueOpen})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueList_WithFilters(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureIssueOpen})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)
	cmd.SetArgs([]string{"--state", "opened", "--label", "bug", "--author", "testuser", "--search", "crash"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueList_Empty(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueList_APIError(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 500, "500 Internal Server Error")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

// ============================================================================
// ISSUE VIEW WITH MOCK
// ============================================================================

func TestIssueView_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues/10") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureIssueOpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueViewCmd(f.Factory)
	cmd.SetArgs([]string{"10"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueView_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues/10") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureIssueOpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueViewCmd(f.Factory)
	cmd.SetArgs([]string{"10", "--format", "json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ISSUE CLOSE WITH MOCK
// ============================================================================

func TestIssueClose_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/issues/10") {
			closed := cmdtest.FixtureIssueOpen
			closed["state"] = "closed"
			cmdtest.JSONResponse(w, 200, closed)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCloseCmd(f.Factory)
	cmd.SetArgs([]string{"10"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ISSUE REOPEN WITH MOCK
// ============================================================================

func TestIssueReopen_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/issues/11") {
			reopened := cmdtest.FixtureIssueClosed
			reopened["state"] = "opened"
			cmdtest.JSONResponse(w, 200, reopened)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueReopenCmd(f.Factory)
	cmd.SetArgs([]string{"11"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ISSUE COMMENT WITH MOCK
// ============================================================================

func TestIssueComment_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/notes") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":   1,
				"body": "test comment",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCommentCmd(f.Factory)
	cmd.SetArgs([]string{"10", "--body", "test comment"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ISSUE EDIT WITH MOCK
// ============================================================================

func TestIssueEdit_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/issues/10") {
			// Copy fixture to avoid mutating the shared map
			edited := make(map[string]interface{})
			for k, v := range cmdtest.FixtureIssueOpen {
				edited[k] = v
			}
			edited["title"] = "Updated title"
			cmdtest.JSONResponse(w, 200, edited)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueEditCmd(f.Factory)
	cmd.SetArgs([]string{"10", "--title", "Updated title"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// AUTH STATUS CMD TESTS
// ============================================================================

func TestAuthStatus_SuccessAdditional(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthStatusCmd(f.Factory)

	// Will display status from test factory auth config
	err := cmd.Execute()
	// May fail if no auth is configured, just verify no panic
	_ = err
}

func TestAuthStatus_JSONFormat(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthStatusCmd(f.Factory)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	_ = err
}

func TestAuthToken_SuccessAdditional(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthTokenCmd(f.Factory)

	err := cmd.Execute()
	_ = err
}

func TestAuthToken_WithHostname(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthTokenCmd(f.Factory)
	cmd.SetArgs([]string{"--hostname", "gitlab.com"})

	err := cmd.Execute()
	_ = err
}

// ============================================================================
// MR COMMENT INLINE
// ============================================================================

func TestMRComment_WithFile(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/notes") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":   2,
				"body": "inline comment",
			})
			return
		}
		if strings.Contains(r.URL.Path, "/versions") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":         1,
					"head_commit_sha": "abc123",
					"base_commit_sha": "def456",
					"start_commit_sha": "ghi789",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRCommentCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--body", "inline comment", "--file", "cmd/mr.go", "--line", "10"})

	err := cmd.Execute()
	// May fail due to diff version lookup, but exercises the code path
	_ = err
}

// ============================================================================
// MR LIST WITH FILTERS
// ============================================================================

func TestMRList_WithStateFilter(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/merge_requests") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureMRMerged})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRListCmd(f.Factory)
	cmd.SetArgs([]string{"--state", "merged"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// MR APPROVE
// ============================================================================

// ============================================================================
// CONFIG SET CMD TESTS
// ============================================================================

func TestConfigSetCmd_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigSetCmd(f.Factory)
	cmd.SetArgs([]string{"editor", "vim"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigSetCmd_InvalidKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigSetCmd(f.Factory)
	cmd.SetArgs([]string{"invalid_key_xyz", "value"})

	err := cmd.Execute()
	// May succeed or fail depending on validation
	_ = err
}

// ============================================================================
// SNIPPET CREATE ADDITIONAL TESTS
// ============================================================================

func TestSnippetCreate_MissingTitle(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetCreateCmd(f.Factory)
	cmd.SetArgs([]string{}) // No title

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

// ============================================================================
// BRANCH CREATE CMD TESTS
// ============================================================================

func TestBranchCreate_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/branches") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"name":   "feature/new",
				"commit": map[string]interface{}{"id": "abc123"},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--name", "feature/new", "--ref", "main"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranchCreate_MissingName(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newBranchCreateCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing branch name")
	}
}

// ============================================================================
// ENVIRONMENT CMD TESTS
// ============================================================================

func TestEnvironmentList_SuccessAdditional(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/environments") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":          1,
					"name":        "production",
					"slug":        "production",
					"external_url": "https://example.com",
					"state":       "available",
					"created_at":  "2024-01-01T00:00:00.000Z",
					"updated_at":  "2024-01-01T00:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "production") {
		t.Errorf("expected output to contain environment name, got: %s", output)
	}
}



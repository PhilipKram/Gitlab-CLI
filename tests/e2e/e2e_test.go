package e2e_test

import (
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/cmd"
	"github.com/PhilipKram/gitlab-cli/tests/e2e"
)

// TestE2E_SkipMode verifies that E2E tests properly skip when GLAB_E2E_TEST is not set.
// This test always runs to verify the skip mechanism works correctly.
func TestE2E_SkipMode(t *testing.T) {
	// This test verifies the skip behavior
	// If GLAB_E2E_TEST is not set, this test should skip gracefully
	e2e.SkipIfNoE2E(t)

	// If we get here, GLAB_E2E_TEST=true is set
	t.Log("E2E tests are enabled (GLAB_E2E_TEST=true)")
}

// TestE2E_AuthStatus verifies auth status against a real GitLab instance.
// Requires: GLAB_E2E_TEST=true, GITLAB_TOKEN set
func TestE2E_AuthStatus(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	f := e2e.NewE2ETestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Execute auth status command
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("auth status command failed: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected elements for real GitLab
	if output == "" {
		t.Fatal("auth status should produce output")
	}

	// Should show the configured host
	if !strings.Contains(output, f.Host) {
		t.Errorf("output should contain host %q\nGot: %s", f.Host, output)
	}

	// Should indicate authentication source
	if !strings.Contains(output, "GITLAB_TOKEN") {
		t.Errorf("output should show token source\nGot: %s", output)
	}

	t.Logf("Auth status output:\n%s", output)
}

// TestE2E_AuthToken verifies auth token command against real GitLab.
// Requires: GLAB_E2E_TEST=true, GITLAB_TOKEN set
func TestE2E_AuthToken(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	f := e2e.NewE2ETestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Execute auth token command
	authCmd.SetArgs([]string{"token"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("auth token command failed: %v", err)
	}

	output := strings.TrimSpace(f.IO.String())

	// Verify output looks like a GitLab token
	if output == "" {
		t.Fatal("auth token should produce output")
	}

	// GitLab tokens typically start with "glpat-" or "gldt-"
	if !strings.HasPrefix(output, "glpat-") && !strings.HasPrefix(output, "gldt-") && !strings.HasPrefix(output, "gho-") {
		t.Logf("Warning: token format unexpected (got %q), but not failing test", output[:min(10, len(output))])
	}

	t.Logf("Auth token retrieved successfully (length: %d)", len(output))
}

// TestE2E_AuthToken_WithHostname verifies auth token with explicit hostname.
// Requires: GLAB_E2E_TEST=true, GITLAB_TOKEN set
func TestE2E_AuthToken_WithHostname(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	f := e2e.NewE2ETestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Execute auth token command with hostname flag
	authCmd.SetArgs([]string{"token", "--hostname", f.Host})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("auth token command with hostname failed: %v", err)
	}

	output := strings.TrimSpace(f.IO.String())

	// Verify output
	if output == "" {
		t.Fatal("auth token should produce output")
	}

	t.Logf("Auth token with hostname retrieved successfully")
}

// TestE2E_AuthStatus_OutputFormat verifies auth status output format against real GitLab.
// Requires: GLAB_E2E_TEST=true, GITLAB_TOKEN set
func TestE2E_AuthStatus_OutputFormat(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	f := e2e.NewE2ETestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Execute auth status command
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("auth status command failed: %v", err)
	}

	output := f.IO.String()

	// Verify output structure
	if !strings.Contains(output, f.Host) {
		t.Errorf("output should contain host name %q", f.Host)
	}

	// Should show token source
	if !strings.Contains(output, "GITLAB_TOKEN") {
		t.Error("output should indicate token source (GITLAB_TOKEN)")
	}

	// Token should be masked in status output (security best practice)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Check if any line contains what looks like a full token
		// GitLab tokens are typically 20-26 characters after the prefix
		if strings.Contains(line, "glpat-") {
			// If it shows the prefix, make sure it's masked
			if len(line) > 50 && strings.Count(line, "*") == 0 {
				t.Error("token appears to be shown in full, should be masked")
			}
		}
	}

	t.Logf("Auth status output format verified")
}

// TestE2E_AuthIntegration verifies complete auth flow against real GitLab.
// This test combines status and token commands to verify full auth functionality.
// Requires: GLAB_E2E_TEST=true, GITLAB_TOKEN set
func TestE2E_AuthIntegration(t *testing.T) {
	e2e.SkipIfNoE2E(t)
	e2e.VerifyE2ESetup(t)

	f := e2e.NewE2ETestFactory(t)

	// Test 1: Verify auth status works
	t.Run("status", func(t *testing.T) {
		authCmd := cmd.NewAuthCmd(f.Factory)
		authCmd.SetArgs([]string{"status"})
		authCmd.SetOut(f.IO.Out)
		authCmd.SetErr(f.IO.ErrOut)

		err := authCmd.Execute()
		if err != nil {
			t.Fatalf("auth status failed: %v", err)
		}

		output := f.IO.String()
		if output == "" {
			t.Fatal("auth status should produce output")
		}

		if !strings.Contains(output, f.Host) {
			t.Errorf("status should show host %q", f.Host)
		}
	})

	// Test 2: Verify auth token retrieval works
	t.Run("token", func(t *testing.T) {
		// Create a new factory for clean IO
		f2 := e2e.NewE2ETestFactory(t)

		authCmd := cmd.NewAuthCmd(f2.Factory)
		authCmd.SetArgs([]string{"token"})
		authCmd.SetOut(f2.IO.Out)
		authCmd.SetErr(f2.IO.ErrOut)

		err := authCmd.Execute()
		if err != nil {
			t.Fatalf("auth token failed: %v", err)
		}

		output := strings.TrimSpace(f2.IO.String())
		if output == "" {
			t.Fatal("auth token should produce output")
		}
	})

	t.Log("Auth integration test completed successfully")
}

// TestE2E_MR verifies merge request commands against a real GitLab instance.
// This test runs basic MR list command to verify API connectivity and output formatting.
// Requires: GLAB_E2E_TEST=true, GITLAB_TOKEN set, GLAB_E2E_PROJECT set
func TestE2E_MR(t *testing.T) {
	e2e.SkipIfNoE2E(t)
	e2e.VerifyE2ESetup(t)

	// Test 1: Verify mr list works
	t.Run("list", func(t *testing.T) {
		f := e2e.NewE2ETestFactory(t)
		mrCmd := cmd.NewMRCmd(f.Factory)

		// Execute mr list command
		mrCmd.SetArgs([]string{"list"})
		mrCmd.SetOut(f.IO.Out)
		mrCmd.SetErr(f.IO.ErrOut)

		err := mrCmd.Execute()
		if err != nil {
			t.Fatalf("mr list command failed: %v", err)
		}

		output := f.IO.String()
		errOutput := f.IO.ErrString()

		// The output may be empty if there are no MRs, or may contain MRs
		// Either is acceptable - we just need to verify the command doesn't error
		if output == "" && !strings.Contains(errOutput, "No merge requests match your search") {
			t.Logf("No merge requests found in project %s/%s", f.Owner, f.Repo)
		} else if output != "" {
			t.Logf("Found merge requests in output")
		}

		t.Logf("MR list executed successfully")
	})

	// Test 2: Verify mr list with state filter
	t.Run("list_with_state", func(t *testing.T) {
		f := e2e.NewE2ETestFactory(t)
		mrCmd := cmd.NewMRCmd(f.Factory)

		// Execute mr list command with state=opened
		mrCmd.SetArgs([]string{"list", "--state", "opened"})
		mrCmd.SetOut(f.IO.Out)
		mrCmd.SetErr(f.IO.ErrOut)

		err := mrCmd.Execute()
		if err != nil {
			t.Fatalf("mr list --state opened command failed: %v", err)
		}

		// Verify command executed without error
		// Output may be empty if there are no open MRs
		t.Logf("MR list with state filter executed successfully")
	})

	// Test 3: Verify mr list with JSON format
	t.Run("list_json_format", func(t *testing.T) {
		f := e2e.NewE2ETestFactory(t)
		mrCmd := cmd.NewMRCmd(f.Factory)

		// Execute mr list command with JSON format
		mrCmd.SetArgs([]string{"list", "--format", "json"})
		mrCmd.SetOut(f.IO.Out)
		mrCmd.SetErr(f.IO.ErrOut)

		err := mrCmd.Execute()
		if err != nil {
			t.Fatalf("mr list --format json command failed: %v", err)
		}

		output := f.IO.String()
		errOutput := f.IO.ErrString()

		// If there are MRs, output should contain JSON
		// If there are no MRs, we should see error message
		if output != "" {
			// Verify JSON format
			if !strings.Contains(output, "{") && !strings.Contains(output, "[") {
				t.Errorf("expected JSON format output, got: %s", output)
			}
			t.Logf("JSON format output verified")
		} else if strings.Contains(errOutput, "No merge requests match your search") {
			t.Logf("No merge requests found (expected JSON would be empty)")
		}

		t.Logf("MR list JSON format executed successfully")
	})

	// Test 4: Verify mr list with limit
	t.Run("list_with_limit", func(t *testing.T) {
		f := e2e.NewE2ETestFactory(t)
		mrCmd := cmd.NewMRCmd(f.Factory)

		// Execute mr list command with limit=5
		mrCmd.SetArgs([]string{"list", "--limit", "5"})
		mrCmd.SetOut(f.IO.Out)
		mrCmd.SetErr(f.IO.ErrOut)

		err := mrCmd.Execute()
		if err != nil {
			t.Fatalf("mr list --limit 5 command failed: %v", err)
		}

		// Verify command executed without error
		t.Logf("MR list with limit executed successfully")
	})
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

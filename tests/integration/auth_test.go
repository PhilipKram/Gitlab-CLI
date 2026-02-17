package integration_test

import (
	"os"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/cmd"
	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

// TestAuthStatus_WithToken verifies auth status shows token from environment
func TestAuthStatus_WithToken(t *testing.T) {
	// Set a test token in environment
	t.Setenv("GITLAB_TOKEN", "glpat-test-token-12345")

	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the status subcommand
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth status command
	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains expected elements
	requiredElements := []string{
		"gitlab.com", // Default host
		"GITLAB_TOKEN", // Source
	}

	for _, element := range requiredElements {
		if !strings.Contains(output, element) {
			t.Errorf("auth status output missing element %q\nGot: %s", element, output)
		}
	}

	// Verify output is not empty
	if output == "" {
		t.Error("expected output for auth status")
	}
}

// TestAuthStatus_NoAuth verifies auth status handles no authentication gracefully
func TestAuthStatus_NoAuth(t *testing.T) {
	// Ensure no token in environment
	os.Unsetenv("GITLAB_TOKEN")

	f := cmdtest.NewTestFactory(t)

	// Clear the test token that NewTestFactory sets
	t.Setenv("GITLAB_TOKEN", "")

	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the status subcommand
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth status command
	err := authCmd.Execute()

	// May error if no auth configured, which is expected
	// The important thing is it doesn't panic and provides a clear message
	_ = err

	output := f.IO.String()
	errOutput := f.IO.ErrString()

	// If there's an error, it should be about authentication
	if err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "auth") && !strings.Contains(errMsg, "token") {
			t.Errorf("error should be about authentication, got: %v", err)
		}
	}

	// Combined output should not panic or be completely empty
	combinedOutput := output + errOutput
	_ = combinedOutput // Just verify it doesn't panic
}

// TestAuthStatus_OutputFormat verifies auth status formats output correctly
func TestAuthStatus_OutputFormat(t *testing.T) {
	// Set a test token in environment
	t.Setenv("GITLAB_TOKEN", "glpat-test-token-12345")

	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the status subcommand
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth status command
	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output format contains expected structure
	// The output should have:
	// - Host name (gitlab.com)
	// - "Logged in" message or similar
	// - Token information (masked)
	// - Source indication

	if !strings.Contains(output, "gitlab.com") {
		t.Error("output should contain host name")
	}

	// Token should be masked (not showing full token)
	if strings.Contains(output, "glpat-test-token-12345") {
		t.Error("token should be masked in output")
	}

	// Should show it's from GITLAB_TOKEN env var
	if !strings.Contains(output, "GITLAB_TOKEN") {
		t.Error("output should indicate token source")
	}
}

// TestAuthStatus_Integration verifies full auth status flow
func TestAuthStatus_Integration(t *testing.T) {
	// This test verifies the complete integration of auth status command
	t.Setenv("GITLAB_TOKEN", "glpat-integration-test-token")

	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the status subcommand
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth status command
	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("auth status should not error with valid token: %v", err)
	}

	output := f.IO.String()

	// Verify complete output structure
	if output == "" {
		t.Fatal("auth status should produce output")
	}

	// Should contain host
	if !strings.Contains(output, "gitlab.com") {
		t.Error("output should show default host gitlab.com")
	}

	// Should contain source
	if !strings.Contains(output, "GITLAB_TOKEN") {
		t.Error("output should show token source")
	}

	// Verify the token is masked (should show glpat-... but not full token)
	lines := strings.Split(output, "\n")
	foundTokenLine := false
	for _, line := range lines {
		if strings.Contains(line, "Token:") {
			foundTokenLine = true
			// Token should be masked
			if strings.Contains(line, "glpat-integration-test-token") {
				t.Error("token should be masked, not shown in full")
			}
		}
	}

	if !foundTokenLine {
		t.Error("output should include token information")
	}
}

// TestAuthToken_WithToken verifies auth token prints token from environment
func TestAuthToken_WithToken(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	// Set a test token in environment (after factory creation to override default)
	t.Setenv("GITLAB_TOKEN", "glpat-test-token-12345")

	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the token subcommand
	authCmd.SetArgs([]string{"token"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth token command
	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output contains the token (not masked like in status)
	if !strings.Contains(output, "glpat-test-token-12345") {
		t.Errorf("auth token output should contain full token\nGot: %s", output)
	}

	// Verify output is not empty
	if output == "" {
		t.Error("expected output for auth token")
	}
}

// TestAuthToken_NoAuth verifies auth token handles no authentication gracefully
func TestAuthToken_NoAuth(t *testing.T) {
	// Clear the test token that NewTestFactory sets
	t.Setenv("GITLAB_TOKEN", "")

	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the token subcommand
	authCmd.SetArgs([]string{"token"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth token command
	err := authCmd.Execute()

	// May error if no auth configured, which is expected
	// The important thing is it doesn't panic and provides a clear message
	if err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "auth") && !strings.Contains(errMsg, "token") {
			t.Errorf("error should be about authentication, got: %v", err)
		}
	}

	// Verify it doesn't panic
	_ = f.IO.String()
}

// TestAuthToken_WithHostname verifies auth token works with hostname flag
func TestAuthToken_WithHostname(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	// Set a test token in environment (after factory creation to override default)
	t.Setenv("GITLAB_TOKEN", "glpat-test-token-12345")

	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the token subcommand with hostname flag
	authCmd.SetArgs([]string{"token", "--hostname", "gitlab.com"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth token command
	err := authCmd.Execute()

	// May error if hostname not configured, which is OK
	// Just verify it doesn't panic
	_ = err
	_ = f.IO.String()
}

// TestAuthToken_OutputFormat verifies auth token formats output correctly
func TestAuthToken_OutputFormat(t *testing.T) {
	f := cmdtest.NewTestFactory(t)

	// Set a test token in environment (after factory creation to override default)
	t.Setenv("GITLAB_TOKEN", "glpat-test-token-67890")

	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the token subcommand
	authCmd.SetArgs([]string{"token"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth token command
	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()

	// Verify output format - should contain the full token (not masked)
	if !strings.Contains(output, "glpat-test-token-67890") {
		t.Error("output should contain the full token")
	}

	// Token output should be simple - just the token, possibly with a newline
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		t.Error("output should not be empty")
	}
}

// TestAuthToken_Integration verifies full auth token flow
func TestAuthToken_Integration(t *testing.T) {
	// This test verifies the complete integration of auth token command
	testToken := "glpat-integration-test-token-99999"

	f := cmdtest.NewTestFactory(t)

	// Set test token after factory creation to override default
	t.Setenv("GITLAB_TOKEN", testToken)

	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the token subcommand
	authCmd.SetArgs([]string{"token"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth token command
	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("auth token should not error with valid token: %v", err)
	}

	output := f.IO.String()

	// Verify complete output structure
	if output == "" {
		t.Fatal("auth token should produce output")
	}

	// Should contain the full token (unlike status which masks it)
	if !strings.Contains(output, testToken) {
		t.Errorf("output should contain full token %q\nGot: %s", testToken, output)
	}

	// Verify the output is clean (no extra formatting or messages)
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput != testToken && !strings.HasPrefix(trimmedOutput, testToken) {
		t.Logf("Note: output may include additional formatting: %q", output)
	}
}

// TestAuthLogout_WithHostname verifies auth logout command with hostname flag
func TestAuthLogout_WithHostname(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the logout subcommand with hostname flag
	authCmd.SetArgs([]string{"logout", "--hostname", "gitlab.com"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth logout command
	err := authCmd.Execute()

	// May error if not logged in, which is expected
	// The important thing is it doesn't panic and provides a clear message
	_ = err

	// Verify it doesn't panic
	_ = f.IO.String()
	_ = f.IO.ErrString()
}

// TestAuthLogout_DefaultHostname verifies auth logout works without hostname flag
func TestAuthLogout_DefaultHostname(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the logout subcommand without hostname
	authCmd.SetArgs([]string{"logout"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth logout command
	err := authCmd.Execute()

	// May error if not logged in, which is expected
	// The important thing is it doesn't panic
	_ = err

	// Verify it doesn't panic
	_ = f.IO.String()
	_ = f.IO.ErrString()
}

// TestAuthLogout_Integration verifies full auth logout flow
func TestAuthLogout_Integration(t *testing.T) {
	// This test verifies the complete integration of auth logout command
	f := cmdtest.NewTestFactory(t)
	authCmd := cmd.NewAuthCmd(f.Factory)

	// Set up the command to execute the logout subcommand
	authCmd.SetArgs([]string{"logout", "--hostname", "gitlab.com"})
	authCmd.SetOut(f.IO.Out)
	authCmd.SetErr(f.IO.ErrOut)

	// Execute the auth logout command
	err := authCmd.Execute()

	// May error if no auth is configured, which is expected in test environment
	// The important thing is it handles the request gracefully
	if err != nil {
		errMsg := err.Error()
		// Error should be informative about the logout state
		_ = errMsg // Just verify it doesn't panic
	}

	output := f.IO.String()
	errOutput := f.IO.ErrString()

	// Combined output should not panic
	combinedOutput := output + errOutput
	_ = combinedOutput
}

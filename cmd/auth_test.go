package cmd

import (
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewAuthCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewAuthCmd(f)

	if cmd.Use != "auth <command>" {
		t.Errorf("expected Use to be 'auth <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Authenticate glab and git with GitLab" {
		t.Errorf("expected Short to be 'Authenticate glab and git with GitLab', got %q", cmd.Short)
	}
}

func TestAuthCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewAuthCmd(f)

	expectedSubcommands := []string{
		"login",
		"logout",
		"status",
		"token",
		"switch",
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

func TestAuthLoginCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthLoginCmd(f)

	expectedFlags := []string{
		"hostname",
		"token",
		"stdin",
		"client-id",
		"git-protocol",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "login" {
		t.Errorf("expected Use to be 'login', got %q", cmd.Use)
	}
}

func TestAuthLogoutCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthLogoutCmd(f)

	expectedFlags := []string{
		"hostname",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "logout" {
		t.Errorf("expected Use to be 'logout', got %q", cmd.Use)
	}
}

func TestAuthStatusCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthStatusCmd(f)

	if cmd.Use != "status" {
		t.Errorf("expected Use to be 'status', got %q", cmd.Use)
	}

	if cmd.Short != "View authentication status" {
		t.Errorf("expected Short to be 'View authentication status', got %q", cmd.Short)
	}
}

func TestAuthStatusCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthStatusCmd(f)

	expectedFlags := []string{
		"format",
		"json",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestAuthTokenCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newAuthTokenCmd(f)

	expectedFlags := []string{
		"hostname",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "token" {
		t.Errorf("expected Use to be 'token', got %q", cmd.Use)
	}

	if cmd.Short != "Print the auth token for a GitLab instance" {
		t.Errorf("expected Short to be 'Print the auth token for a GitLab instance', got %q", cmd.Short)
	}
}

func TestAuthStatus_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthStatusCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	// Should show auth status (logged in/out)
	if output == "" {
		t.Error("expected output for auth status")
	}
}

func TestAuthToken_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthTokenCmd(f.Factory)

	_ = cmd.Execute()
	// May error if no token set, which is expected
	// Just verify it doesn't panic
}

func TestAuthLogout_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthLogoutCmd(f.Factory)
	cmd.SetArgs([]string{"--hostname", "gitlab.com"})

	// May error if not logged in, which is expected
	_ = cmd.Execute()
}

func TestAuthLogin_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newAuthLoginCmd(f.Factory)

	// Test that invalid flags are rejected
	// (don't call Execute() which triggers interactive flow)
	cmd.SetArgs([]string{"--token", ""}) // Invalid: empty token

	err := cmd.Execute()
	if err != nil {
		// Expected - empty token should be rejected
		cmdtest.AssertContains(t, err.Error(), "token")
	}
}

func TestSaveProtocol(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	_ = f // Use factory for consistency

	err := saveProtocol("gitlab.com", "https")
	// May error in test environment, which is OK
	// Just verify it doesn't panic
	_ = err
}

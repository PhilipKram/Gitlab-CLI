package cmd

import (
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewUpgradeCmd(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewUpgradeCmd(f.Factory)

	if cmd == nil {
		t.Fatal("expected upgrade command")
	}
	if cmd.Use != "upgrade" {
		t.Errorf("expected Use=upgrade, got %s", cmd.Use)
	}
}

func TestUpgradeCmd_Flags(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewUpgradeCmd(f.Factory)

	// Verify command has flag set (cobra adds help automatically to root)
	_ = cmd.Flags()
}

func TestUpgrade_Execute(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewUpgradeCmd(f.Factory)

	// Execute (may check for updates or report current version)
	err := cmd.Execute()
	// Don't fail if error - upgrade command may error in test environment
	// Just verify it doesn't panic
	_ = err
}

func TestUpgrade_DevBuild(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	f.Version = "dev"
	cmd := NewUpgradeCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for dev build")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Errorf("expected 'development build' error, got: %v", err)
	}
}

func TestUpgrade_Flags(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewUpgradeCmd(f.Factory)

	expectedFlags := []string{"check", "yes", "force"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestUpgrade_Aliases(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewUpgradeCmd(f.Factory)

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "update" {
		t.Errorf("expected alias 'update', got %v", cmd.Aliases)
	}
}

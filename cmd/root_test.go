package cmd

import (
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd("test-version")

	if cmd == nil {
		t.Fatal("expected root command")
	}

	// Verify basic configuration
	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage=true")
	}
	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors=true")
	}

	// Verify main subcommands are registered
	expectedCommands := []string{"mr", "issue", "pipeline", "repo", "project", "auth", "config"}
	for _, expected := range expectedCommands {
		found := false
		for _, child := range cmd.Commands() {
			if child.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %s to be registered", expected)
		}
	}
}

func TestRootCmd_Version(t *testing.T) {
	cmd := NewRootCmd("1.2.3")
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

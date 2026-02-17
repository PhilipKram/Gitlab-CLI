package cmd

import (
	"testing"
)

func TestNewCompletionCmd(t *testing.T) {
	cmd := NewCompletionCmd()

	if cmd == nil {
		t.Fatal("expected completion command")
	}
	if cmd.Use != "completion <shell>" {
		t.Errorf("expected Use='completion <shell>', got %s", cmd.Use)
	}

	// Verify valid args exist (not subcommands)
	expectedArgs := []string{"bash", "zsh", "fish", "powershell"}
	if len(cmd.ValidArgs) != len(expectedArgs) {
		t.Errorf("expected %d valid args, got %d", len(expectedArgs), len(cmd.ValidArgs))
	}
	for i, arg := range expectedArgs {
		if i < len(cmd.ValidArgs) && cmd.ValidArgs[i] != arg {
			t.Errorf("expected valid arg %s, got %s", arg, cmd.ValidArgs[i])
		}
	}
}

func TestCompletion_Bash(t *testing.T) {
	cmd := NewCompletionCmd()
	cmd.SetArgs([]string{"bash"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletion_Zsh(t *testing.T) {
	cmd := NewCompletionCmd()
	cmd.SetArgs([]string{"zsh"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletion_Fish(t *testing.T) {
	cmd := NewCompletionCmd()
	cmd.SetArgs([]string{"fish"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletion_PowerShell(t *testing.T) {
	cmd := NewCompletionCmd()
	cmd.SetArgs([]string{"powershell"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

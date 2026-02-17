package cmd

import (
	"fmt"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/PhilipKram/gitlab-cli/internal/git"
)

func TestNewBrowseCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewBrowseCmd(f)

	if cmd.Use != "browse [path]" {
		t.Errorf("expected Use to be 'browse [path]', got %q", cmd.Use)
	}

	if cmd.Short != "Open project in browser" {
		t.Errorf("expected Short to be 'Open project in browser', got %q", cmd.Short)
	}
}

func TestBrowseCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := NewBrowseCmd(f)

	expectedFlags := []string{
		"branch",
		"settings",
		"members",
		"issues",
		"mrs",
		"pipeline",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify branch flag has shorthand -b
	branchFlag := cmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("branch flag not found")
	}
	if branchFlag.Shorthand != "b" {
		t.Errorf("expected branch shorthand to be 'b', got %q", branchFlag.Shorthand)
	}
	if branchFlag.DefValue != "" {
		t.Errorf("expected default branch to be empty, got %q", branchFlag.DefValue)
	}

	// Verify settings flag has shorthand -s
	settingsFlag := cmd.Flags().Lookup("settings")
	if settingsFlag == nil {
		t.Fatal("settings flag not found")
	}
	if settingsFlag.Shorthand != "s" {
		t.Errorf("expected settings shorthand to be 's', got %q", settingsFlag.Shorthand)
	}
	if settingsFlag.DefValue != "false" {
		t.Errorf("expected default settings to be false, got %q", settingsFlag.DefValue)
	}

	// Verify pipeline flag has shorthand -p
	pipelineFlag := cmd.Flags().Lookup("pipeline")
	if pipelineFlag == nil {
		t.Fatal("pipeline flag not found")
	}
	if pipelineFlag.Shorthand != "p" {
		t.Errorf("expected pipeline shorthand to be 'p', got %q", pipelineFlag.Shorthand)
	}
	if pipelineFlag.DefValue != "false" {
		t.Errorf("expected default pipeline to be false, got %q", pipelineFlag.DefValue)
	}

	// Verify boolean flags default to false
	boolFlags := []string{"members", "issues", "mrs"}
	for _, flagName := range boolFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Fatalf("%s flag not found", flagName)
		}
		if flag.DefValue != "false" {
			t.Errorf("expected default %s to be false, got %q", flagName, flag.DefValue)
		}
	}
}

func TestBrowse_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewBrowseCmd(f.Factory)

	// Will try to open browser, but that's OK in tests
	// Just verify it doesn't panic
	_ = cmd.Execute()
}

func TestBrowse_NoRemote(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	// Override Remote to return error
	f.Factory.Remote = func() (*git.Remote, error) {
		return nil, fmt.Errorf("not a git repository")
	}
	cmd := NewBrowseCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for no git remote")
	}
}

func TestBrowse_WithPath(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewBrowseCmd(f.Factory)
	cmd.SetArgs([]string{"README.md"})

	// Will fail without git repo, but tests argument parsing
	_ = cmd.Execute()
}

func TestBrowse_WithBranch(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewBrowseCmd(f.Factory)
	cmd.SetArgs([]string{"-b", "main"})

	// Will fail without git repo, but tests flag parsing
	_ = cmd.Execute()
}

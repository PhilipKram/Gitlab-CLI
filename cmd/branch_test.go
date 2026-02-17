package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewBranchCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewBranchCmd(f)

	if cmd.Use != "branch <command>" {
		t.Errorf("expected Use to be 'branch <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage branches" {
		t.Errorf("expected Short to be 'Manage branches', got %q", cmd.Short)
	}
}

func TestBranchCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewBranchCmd(f)

	expectedSubcommands := []string{
		"list",
		"create",
		"delete",
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

func TestBranchListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newBranchListCmd(f)

	expectedFlags := []string{
		"limit",
		"format",
		"json",
		"search",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default limit is 30
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.DefValue != "30" {
		t.Errorf("expected default limit to be 30, got %q", limitFlag.DefValue)
	}

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", cmd.Use)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestBranchCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newBranchCreateCmd(f)

	expectedFlags := []string{
		"name",
		"ref",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default ref is "main"
	refFlag := cmd.Flags().Lookup("ref")
	if refFlag == nil {
		t.Fatal("ref flag not found")
	}
	if refFlag.DefValue != "main" {
		t.Errorf("expected default ref to be 'main', got %q", refFlag.DefValue)
	}

	if cmd.Use != "create" {
		t.Errorf("expected Use to be 'create', got %q", cmd.Use)
	}
}

func TestBranchDeleteCmd_Args(t *testing.T) {
	f := newTestFactory()
	cmd := newBranchDeleteCmd(f)

	if cmd.Use != "delete <branch>" {
		t.Errorf("expected Use to be 'delete <branch>', got %q", cmd.Use)
	}
}

func TestBranchList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repository/branches") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"name":               "main",
					"default":            true,
					"merged":             false,
					"protected":          true,
					"developers_can_push": false,
					"developers_can_merge": false,
					"can_push":           true,
					"web_url":            "https://gitlab.com/owner/repo/-/tree/main",
					"commit": map[string]interface{}{
						"id":      "abc123",
						"message": "Initial commit",
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "main") {
		t.Errorf("expected output to contain branch name, got: %s", output)
	}
}

func TestBranchCreate_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/repository/branches") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"name":    "feature-branch",
				"default": false,
				"commit": map[string]interface{}{
					"id":      "abc123",
					"message": "Initial commit",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--name", "feature-branch"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranchDelete_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"feature-branch"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranchCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newBranchCreateCmd(f.Factory)
	// Missing required --name flag
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestBranchList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBranchList_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestBranchDelete_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Branch Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newBranchDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found branch")
	}
}

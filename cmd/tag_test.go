package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewTagCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewTagCmd(f)

	if cmd.Use != "tag <command>" {
		t.Errorf("expected Use to be 'tag <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage tags" {
		t.Errorf("expected Short to be 'Manage tags', got %q", cmd.Short)
	}
}

func TestTagCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewTagCmd(f)

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

func TestTagListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newTagListCmd(f)

	expectedFlags := []string{
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

func TestTagCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newTagCreateCmd(f)

	expectedFlags := []string{
		"name",
		"ref",
		"message",
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

func TestTagDeleteCmd_Args(t *testing.T) {
	f := newTestFactory()
	cmd := newTagDeleteCmd(f)

	if cmd.Use != "delete <tag>" {
		t.Errorf("expected Use to be 'delete <tag>', got %q", cmd.Use)
	}
}

func TestTagList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repository/tags") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"name":    "v1.0.0",
					"message": "Release v1.0.0",
					"target":  "abc123",
					"commit": map[string]interface{}{
						"id":      "abc123",
						"message": "Release commit",
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newTagListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "v1.0.0") {
		t.Errorf("expected output to contain tag name, got: %s", output)
	}
}

func TestTagCreate_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/repository/tags") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"name":    "v1.0.0",
				"message": "Release v1.0.0",
				"target":  "abc123",
				"commit": map[string]interface{}{
					"id":      "abc123",
					"message": "Release commit",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newTagCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--name", "v1.0.0", "--message", "Release v1.0.0"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagDelete_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newTagDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newTagCreateCmd(f.Factory)
	// Missing required --name flag
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestTagList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newTagListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagList_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newTagListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestTagDelete_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Tag Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newTagDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found tag")
	}
}

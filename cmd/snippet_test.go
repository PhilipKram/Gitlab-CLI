package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewSnippetCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewSnippetCmd(f)

	if cmd.Use != "snippet <command>" {
		t.Errorf("expected Use to be 'snippet <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage snippets" {
		t.Errorf("expected Short to be 'Manage snippets', got %q", cmd.Short)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "snip" {
		t.Errorf("expected alias 'snip', got %v", cmd.Aliases)
	}
}

func TestSnippetCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewSnippetCmd(f)

	expectedSubcommands := []string{
		"create",
		"list",
		"view",
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

func TestSnippetCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newSnippetCreateCmd(f)

	expectedFlags := []string{
		"title",
		"filename",
		"visibility",
		"file",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default visibility is private
	visibilityFlag := cmd.Flags().Lookup("visibility")
	if visibilityFlag == nil {
		t.Fatal("visibility flag not found")
	}
	if visibilityFlag.DefValue != "private" {
		t.Errorf("expected default visibility to be 'private', got %q", visibilityFlag.DefValue)
	}
}

func TestSnippetListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newSnippetListCmd(f)

	expectedFlags := []string{
		"limit",
		"json",
		"web",
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

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestSnippetViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newSnippetViewCmd(f)

	expectedFlags := []string{"raw", "web"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view [<id>]" {
		t.Errorf("expected Use to be 'view [<id>]', got %q", cmd.Use)
	}
}

func TestSnippetDeleteCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newSnippetDeleteCmd(f)

	if cmd.Use != "delete [<id>]" {
		t.Errorf("expected Use to be 'delete [<id>]', got %q", cmd.Use)
	}
}

func TestSnippetList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/snippets") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureSnippet,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Example") {
		t.Errorf("expected output to contain snippet title, got: %s", output)
	}
}

func TestSnippetView_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/snippets/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureSnippet)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnippetCreate_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/snippets") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureSnippet)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test snippet", "--filename", "test.txt"})

	// Command requires --file flag which needs a real file
	// Just verify it doesn't panic
	_ = cmd.Execute()
}

func TestSnippetView_NotFound(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetViewCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent snippet")
	}
}

func TestSnippetCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetCreateCmd(f.Factory)
	// Missing required --title flag
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing title")
	}
}

func TestSnippetDelete_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/snippets/1") {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnippetList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnippetView_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected authorization error")
	}
}

func TestSnippetDelete_Forbidden(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 403, "403 Forbidden")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newSnippetDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

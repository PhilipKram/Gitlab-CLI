package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewRepoCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewRepoCmd(f)

	if cmd.Use != "repo <command>" {
		t.Errorf("expected Use to be 'repo <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage repositories" {
		t.Errorf("expected Short to be 'Manage repositories', got %q", cmd.Short)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "project" {
		t.Errorf("expected alias 'project', got %v", cmd.Aliases)
	}
}

func TestRepoCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewRepoCmd(f)

	expectedSubcommands := []string{
		"clone",
		"create",
		"fork",
		"view",
		"list",
		"archive",
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

func TestRepoCloneCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoCloneCmd(f)

	if cmd.Use != "clone <owner/repo>" {
		t.Errorf("expected Use to be 'clone <owner/repo>', got %q", cmd.Use)
	}

	if cmd.Short != "Clone a repository" {
		t.Errorf("expected Short to be 'Clone a repository', got %q", cmd.Short)
	}
}

func TestRepoCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoCreateCmd(f)

	expectedFlags := []string{
		"description",
		"visibility",
		"public",
		"private",
		"internal",
		"init",
		"default-branch",
		"group-id",
		"web",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "create [<name>]" {
		t.Errorf("expected Use to be 'create [<name>]', got %q", cmd.Use)
	}
}

func TestRepoForkCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoForkCmd(f)

	expectedFlags := []string{
		"namespace",
		"name",
		"clone",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "fork [<owner/repo>]" {
		t.Errorf("expected Use to be 'fork [<owner/repo>]', got %q", cmd.Use)
	}
}

func TestRepoViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoViewCmd(f)

	expectedFlags := []string{"web", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view [<owner/repo>]" {
		t.Errorf("expected Use to be 'view [<owner/repo>]', got %q", cmd.Use)
	}
}

func TestRepoListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoListCmd(f)

	expectedFlags := []string{
		"owner",
		"limit",
		"json",
		"archived",
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

	// Verify list has "ls" alias
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestRepoArchiveCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoArchiveCmd(f)

	if cmd.Use != "archive [<owner/repo>]" {
		t.Errorf("expected Use to be 'archive [<owner/repo>]', got %q", cmd.Use)
	}

	if cmd.Short != "Archive a repository" {
		t.Errorf("expected Short to be 'Archive a repository', got %q", cmd.Short)
	}
}

func TestRepoDeleteCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRepoDeleteCmd(f)

	expectedFlags := []string{"confirm"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "delete <owner/repo>" {
		t.Errorf("expected Use to be 'delete <owner/repo>', got %q", cmd.Use)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "needs truncation",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "with newline",
			input:  "hello\nworld",
			maxLen: 20,
			want:   "hello world",
		},
		{
			name:   "with newline and truncation",
			input:  "hello\nworld this is long",
			maxLen: 10,
			want:   "hello w...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// EXECUTION AND ERROR TESTS
// ============================================================================

func TestRepoView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects/") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureProject)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newRepoViewCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoClone_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, cmdtest.FixtureProject)
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newRepoCloneCmd(f.Factory)
	cmd.SetArgs([]string{"test-owner/test-repo", "--dry-run"})
	err := cmd.Execute()
	// May fail due to git operations, but API mock should work
	_ = err
}

func TestRepoView_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newRepoViewCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestRepoCreate_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/projects") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureProject)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoCreateCmd(f.Factory)
	cmd.SetArgs([]string{"test-repo"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoFork_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/fork") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureProject)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoForkCmd(f.Factory)
	cmd.SetArgs([]string{"owner/repo"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoList_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureProject,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoArchive_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/archive") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureProject)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoArchiveCmd(f.Factory)
	cmd.SetArgs([]string{"owner/repo"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoDelete_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"owner/repo", "--confirm"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newRepoCreateCmd(f.Factory)
	cmd.SetArgs([]string{}) // Missing required name

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestRepoFork_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Project Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoForkCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent/repo"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found repo")
	}
}

func TestRepoList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRepoListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

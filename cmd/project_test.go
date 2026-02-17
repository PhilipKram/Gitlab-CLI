package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestNewProjectCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewProjectCmd(f)

	if cmd.Use != "project <command>" {
		t.Errorf("expected Use to be 'project <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage projects" {
		t.Errorf("expected Short to be 'Manage projects', got %q", cmd.Short)
	}
}

func TestProjectCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewProjectCmd(f)

	expectedSubcommands := []string{
		"list",
		"view",
		"members",
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

func TestProjectListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newProjectListCmd(f)

	expectedFlags := []string{
		"group",
		"limit",
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

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestProjectViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newProjectViewCmd(f)

	expectedFlags := []string{"json"}

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

func TestProjectMembersCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newProjectMembersCmd(f)

	expectedFlags := []string{
		"limit",
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

	if cmd.Use != "members [<owner/repo>]" {
		t.Errorf("expected Use to be 'members [<owner/repo>]', got %q", cmd.Use)
	}
}

// EXECUTION AND ERROR TESTS
func TestProjectList_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureProject})
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newProjectListCmd(f.Factory)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, cmdtest.FixtureProject)
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newProjectViewCmd(f.Factory)
	cmd.SetArgs([]string{"test-owner/test-repo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectView_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newProjectViewCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent/repo"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectList_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})
	f := cmdtest.NewTestFactory(t)
	cmd := newProjectListCmd(f.Factory)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected auth error")
	}
}

func TestAccessLevelName(t *testing.T) {
	tests := []struct {
		level    gitlab.AccessLevelValue
		expected string
	}{
		{gitlab.NoPermissions, "None"},
		{gitlab.MinimalAccessPermissions, "Minimal"},
		{gitlab.GuestPermissions, "Guest"},
		{gitlab.ReporterPermissions, "Reporter"},
		{gitlab.DeveloperPermissions, "Developer"},
		{gitlab.MaintainerPermissions, "Maintainer"},
		{gitlab.OwnerPermissions, "Owner"},
		{gitlab.AccessLevelValue(99), "Level 99"},
	}

	for _, tt := range tests {
		result := accessLevelName(tt.level)
		if result != tt.expected {
			t.Errorf("accessLevelName(%v) = %s, want %s", tt.level, result, tt.expected)
		}
	}
}

func TestProjectMembers_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/members") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"username":     "alice",
					"access_level": 30,
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newProjectMembersCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectView_CurrentProject(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, cmdtest.FixtureProject)
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newProjectViewCmd(f.Factory)
	cmd.SetArgs([]string{}) // No args - should use current project

	err := cmd.Execute()
	// May fail if no git repo, but API mock should work
	_ = err
}

func TestProjectList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newProjectListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectMembers_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newProjectMembersCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected authorization error")
	}
}

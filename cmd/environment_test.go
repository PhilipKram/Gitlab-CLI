package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewEnvironmentCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewEnvironmentCmd(f)

	if cmd.Use != "environment <command>" {
		t.Errorf("expected Use to be 'environment <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage environments" {
		t.Errorf("expected Short to be 'Manage environments', got %q", cmd.Short)
	}

	expectedAliases := []string{"env"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i >= len(cmd.Aliases) || cmd.Aliases[i] != alias {
			t.Errorf("expected alias %q at position %d, got %v", alias, i, cmd.Aliases)
		}
	}
}

func TestEnvironmentCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewEnvironmentCmd(f)

	expectedSubcommands := []string{
		"list",
		"view",
		"stop",
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

func TestEnvironmentListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newEnvironmentListCmd(f)

	expectedFlags := []string{
		"state",
		"limit",
		"format",
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

	// Verify list has alias "ls"
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestEnvironmentViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newEnvironmentViewCmd(f)

	expectedFlags := []string{"web", "format", "json"}

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

func TestEnvironmentStopCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newEnvironmentStopCmd(f)

	if cmd.Use != "stop [<id>]" {
		t.Errorf("expected Use to be 'stop [<id>]', got %q", cmd.Use)
	}

	if cmd.Short != "Stop an environment" {
		t.Errorf("expected Short to be 'Stop an environment', got %q", cmd.Short)
	}
}

func TestEnvironmentDeleteCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newEnvironmentDeleteCmd(f)

	if cmd.Use != "delete [<id>]" {
		t.Errorf("expected Use to be 'delete [<id>]', got %q", cmd.Use)
	}

	if cmd.Short != "Delete an environment" {
		t.Errorf("expected Short to be 'Delete an environment', got %q", cmd.Short)
	}
}

// ============================================================================
// EXECUTION TESTS
// ============================================================================

func TestEnvironmentList_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/environments") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":    1,
					"name":  "production",
					"state": "available",
				},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentListCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvironmentView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/environments/") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":    1,
				"name":  "production",
				"state": "available",
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvironmentStop_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/stop") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":    1,
				"name":  "production",
				"state": "stopped",
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentStopCmd(f.Factory)
	cmd.SetArgs([]string{"1"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvironmentDelete_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/environments/1") {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ERROR TESTS
// ============================================================================

func TestEnvironmentView_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentViewCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent environment")
	}
}

func TestEnvironmentList_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentListCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestEnvironmentList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvironmentStop_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Environment Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentStopCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found environment")
	}
}

func TestEnvironmentDelete_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Environment Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newEnvironmentDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found environment")
	}
}

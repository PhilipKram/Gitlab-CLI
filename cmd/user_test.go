package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewUserCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewUserCmd(f)

	if cmd.Use != "user <command>" {
		t.Errorf("expected Use to be 'user <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage users and user information" {
		t.Errorf("expected Short to be 'Manage users and user information', got %q", cmd.Short)
	}
}

func TestUserCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewUserCmd(f)

	expectedSubcommands := []string{
		"whoami",
		"view",
		"ssh-keys",
		"emails",
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

func TestUserWhoamiCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newUserWhoamiCmd(f)

	expectedFlags := []string{"format", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestUserViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newUserViewCmd(f)

	expectedFlags := []string{"format", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view <username>" {
		t.Errorf("expected Use to be 'view <username>', got %q", cmd.Use)
	}
}

func TestUserSSHKeysCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newUserSSHKeysCmd(f)

	expectedFlags := []string{"format", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "keys" {
		t.Errorf("expected alias 'keys', got %v", cmd.Aliases)
	}
}

func TestUserEmailsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newUserEmailsCmd(f)

	expectedFlags := []string{"format", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestUserViewCmd_RequiresArgs(t *testing.T) {
	f := newTestFactory()
	cmd := newUserViewCmd(f)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing username argument")
	}
}

func TestUserWhoami_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/user" {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureUser)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserWhoamiCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "test-user") {
		t.Errorf("expected output to contain username, got: %s", output)
	}
}

func TestUserView_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users") {
			cmdtest.JSONResponse(w, 200, []interface{}{cmdtest.FixtureUser})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserViewCmd(f.Factory)
	cmd.SetArgs([]string{"test-user"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "test-user") {
		t.Errorf("expected output to contain username, got: %s", output)
	}
}

func TestUserView_NotFound(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users") {
			cmdtest.JSONResponse(w, 200, []interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserViewCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestUserSSHKeys_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/keys") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":         1,
					"title":      "My SSH Key",
					"created_at": "2024-01-01T00:00:00.000Z",
					"key":        "ssh-rsa AAAAB3...",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserSSHKeysCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "My SSH Key") {
		t.Errorf("expected output to contain SSH key title, got: %s", output)
	}
}

func TestUserSSHKeys_Empty(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/keys") {
			cmdtest.JSONResponse(w, 200, []interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserSSHKeysCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No SSH keys found") {
		t.Errorf("expected stderr to contain 'No SSH keys found', got: %s", errOutput)
	}
}

func TestUserEmails_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/emails") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":    1,
					"email": "test@example.com",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserEmailsCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "test@example.com") {
		t.Errorf("expected output to contain email, got: %s", output)
	}
}

func TestUserEmails_Empty(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/emails") {
			cmdtest.JSONResponse(w, 200, []interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserEmailsCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No emails found") {
		t.Errorf("expected stderr to contain 'No emails found', got: %s", errOutput)
	}
}

func TestUserWhoami_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newUserWhoamiCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected authorization error")
	}
}

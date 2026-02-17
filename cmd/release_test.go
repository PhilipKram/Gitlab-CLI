package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewReleaseCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewReleaseCmd(f)

	if cmd.Use != "release <command>" {
		t.Errorf("expected Use to be 'release <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage releases" {
		t.Errorf("expected Short to be 'Manage releases', got %q", cmd.Short)
	}
}

func TestReleaseCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewReleaseCmd(f)

	expectedSubcommands := []string{
		"create",
		"list",
		"view",
		"delete",
		"download",
		"upload",
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

func TestReleaseCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newReleaseCreateCmd(f)

	expectedFlags := []string{
		"name",
		"description",
		"ref",
		"milestone",
		"asset",
		"web",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "create <tag>" {
		t.Errorf("expected Use to be 'create <tag>', got %q", cmd.Use)
	}
}

func TestReleaseListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newReleaseListCmd(f)

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

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestReleaseViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newReleaseViewCmd(f)

	expectedFlags := []string{"web", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view <tag>" {
		t.Errorf("expected Use to be 'view <tag>', got %q", cmd.Use)
	}
}

func TestReleaseDeleteCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newReleaseDeleteCmd(f)

	if cmd.Use != "delete <tag>" {
		t.Errorf("expected Use to be 'delete <tag>', got %q", cmd.Use)
	}
}

func TestReleaseDownloadCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newReleaseDownloadCmd(f)

	if cmd.Use != "download <tag>" {
		t.Errorf("expected Use to be 'download <tag>', got %q", cmd.Use)
	}
}

func TestReleaseUploadCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newReleaseUploadCmd(f)

	expectedFlags := []string{
		"name",
		"type",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "upload <tag> <file>" {
		t.Errorf("expected Use to be 'upload <tag> <file>', got %q", cmd.Use)
	}
}

func TestReleaseList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureRelease,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "v1.0.0") {
		t.Errorf("expected output to contain release tag, got: %s", output)
	}
}

func TestReleaseView_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/v1.0.0") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureRelease)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseViewCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseCreate_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/releases") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureRelease)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseCreateCmd(f.Factory)
	cmd.SetArgs([]string{"v2.0.0", "--name", "Release 2.0", "--ref", "main"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseView_NotFound(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseViewCmd(f.Factory)
	cmd.SetArgs([]string{"v9.9.9"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent release")
	}
}

func TestReleaseCreate_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseCreateCmd(f.Factory)
	cmd.SetArgs([]string{"v3.0.0", "--name", "Release 3.0", "--ref", "main"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestReleaseDelete_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/releases/v1.0.0") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureRelease)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseUpload_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseUploadCmd(f.Factory)
	cmd.SetArgs([]string{}) // Missing required args

	err := cmd.Execute()
	if err == nil {
		t.Error("expected validation error for missing arguments")
	}
}

func TestReleaseDownload_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseDownloadCmd(f.Factory)
	cmd.SetArgs([]string{}) // Missing required tag

	err := cmd.Execute()
	if err == nil {
		t.Error("expected validation error for missing tag")
	}
}

func TestReleaseList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseView_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseViewCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected authorization error")
	}
}

func TestReleaseDelete_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 403, "403 Forbidden")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newReleaseDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"v1.0.0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewLabelCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewLabelCmd(f)

	if cmd.Use != "label <command>" {
		t.Errorf("expected Use to be 'label <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage labels" {
		t.Errorf("expected Short to be 'Manage labels', got %q", cmd.Short)
	}
}

func TestLabelCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewLabelCmd(f)

	expectedSubcommands := []string{
		"create",
		"list",
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

func TestLabelCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newLabelCreateCmd(f)

	expectedFlags := []string{
		"name",
		"color",
		"description",
		"priority",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "create" {
		t.Errorf("expected Use to be 'create', got %q", cmd.Use)
	}
}

func TestLabelListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newLabelListCmd(f)

	expectedFlags := []string{
		"limit",
		"json",
		"search",
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

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", cmd.Use)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestLabelDeleteCmd_Args(t *testing.T) {
	f := newTestFactory()
	cmd := newLabelDeleteCmd(f)

	if cmd.Use != "delete <name>" {
		t.Errorf("expected Use to be 'delete <name>', got %q", cmd.Use)
	}
}

func TestLabelList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/labels") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureLabelBug,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "bug") {
		t.Errorf("expected output to contain label, got: %s", output)
	}
}

func TestLabelCreate_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/labels") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureLabelBug)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--name", "test-label", "--color", "#ff0000"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLabelDelete_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"bug"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLabelCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newLabelCreateCmd(f.Factory)
	// Missing required --name flag
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestLabelList_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestLabelDelete_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Label Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found label")
	}
}

func TestLabelList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newLabelListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

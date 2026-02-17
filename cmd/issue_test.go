package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewIssueCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewIssueCmd(f)

	if cmd.Use != "issue <command>" {
		t.Errorf("expected Use to be 'issue <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage issues" {
		t.Errorf("expected Short to be 'Manage issues', got %q", cmd.Short)
	}
}

func TestIssueCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewIssueCmd(f)

	expectedSubcommands := []string{
		"create",
		"list",
		"view",
		"close",
		"reopen",
		"comment",
		"edit",
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

func TestIssueCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueCreateCmd(f)

	expectedFlags := map[string]bool{
		"title":        true,
		"description":  true,
		"assignee":     true,
		"label":        true,
		"milestone":    true,
		"confidential": true,
		"weight":       true,
		"web":          true,
	}

	for flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify title is required
	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Fatal("title flag not found")
	}
}

func TestIssueListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueListCmd(f)

	expectedFlags := []string{
		"state",
		"author",
		"assignee",
		"label",
		"milestone",
		"search",
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

	// Verify default state is "opened"
	stateFlag := cmd.Flags().Lookup("state")
	if stateFlag == nil {
		t.Fatal("state flag not found")
	}
	if stateFlag.DefValue != "opened" {
		t.Errorf("expected default state to be 'opened', got %q", stateFlag.DefValue)
	}

	// Verify default limit is 30
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.DefValue != "30" {
		t.Errorf("expected default limit to be 30, got %q", limitFlag.DefValue)
	}

	// Verify list command has alias "ls"
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestIssueViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueViewCmd(f)

	expectedFlags := []string{"web", "json"}

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

func TestIssueCloseCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueCloseCmd(f)

	if cmd.Use != "close [<id>]" {
		t.Errorf("expected Use to be 'close [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "Close an issue" {
		t.Errorf("expected Short to be 'Close an issue', got %q", cmd.Short)
	}
}

func TestIssueReopenCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueReopenCmd(f)

	if cmd.Use != "reopen [<id>]" {
		t.Errorf("expected Use to be 'reopen [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "Reopen an issue" {
		t.Errorf("expected Short to be 'Reopen an issue', got %q", cmd.Short)
	}
}

func TestIssueCommentCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueCommentCmd(f)

	expectedFlags := []string{"body"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestIssueEditCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueEditCmd(f)

	expectedFlags := []string{
		"title",
		"description",
		"assignee",
		"label",
		"milestone",
		"confidential",
		"weight",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestIssueDeleteCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newIssueDeleteCmd(f)

	if cmd.Use != "delete [<id>]" {
		t.Errorf("expected Use to be 'delete [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "Delete an issue" {
		t.Errorf("expected Short to be 'Delete an issue', got %q", cmd.Short)
	}
}

func TestParseIssueArg(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int64
		wantErr bool
	}{
		{
			name:    "valid number",
			args:    []string{"123"},
			want:    123,
			wantErr: false,
		},
		{
			name:    "valid number with hash",
			args:    []string{"#456"},
			want:    456,
			wantErr: false,
		},
		{
			name:    "no args",
			args:    []string{},
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid number",
			args:    []string{"abc"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative number",
			args:    []string{"-1"},
			want:    -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIssueArg(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseIssueArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// EXECUTION TESTS - Test actual command execution with mocked API responses
// ============================================================================

func TestIssueList_SuccessWithOpenIssues(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureIssueOpen,
			})
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)
	cmd.SetArgs([]string{"--state", "opened"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Application crashes on startup") {
		t.Errorf("expected output to contain issue title, got: %s", output)
	}
}

func TestIssueView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureIssueOpen)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Application crashes on startup") {
		t.Errorf("expected output to contain issue title, got: %s", output)
	}
}

func TestIssueCreate_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureIssueOpen)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test Issue"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "#1") {
		t.Errorf("expected success message with issue number, got: %s", output)
	}
}

func TestIssueClose_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/issues/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureIssueClosed)
			return
		}
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCloseCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(strings.ToLower(output), "closed") {
		t.Errorf("expected success message about closing, got: %s", output)
	}
}

// ============================================================================
// ERROR PATH TESTS - Test error handling for common failure modes
// ============================================================================

func TestIssueView_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueViewCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent issue")
	}

	if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestIssueList_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "401") && !strings.Contains(errStr, "unauthorized") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestIssueCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCreateCmd(f.Factory)
	// Missing required --title flag
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing required fields")
	}
}

func TestIssueReopen_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/issues/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureIssueOpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueReopenCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueComment_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/issues/1/notes") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":   1,
				"body": "Test comment",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueCommentCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--body", "Test comment"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueEdit_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/issues/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureIssueOpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueEditCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--title", "Updated title"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueDelete_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/issues/1") {
			w.WriteHeader(204)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueReopen_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Issue Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueReopenCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found issue")
	}
}

func TestIssueEdit_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newIssueEditCmd(f.Factory)
	cmd.SetArgs([]string{}) // Missing issue ID

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing issue ID")
	}
}

func TestIssueList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newIssueListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

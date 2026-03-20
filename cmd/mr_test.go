package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/git"
	"github.com/PhilipKram/gitlab-cli/pkg/iostreams"
)

func newTestFactory() *cmdutil.Factory {
	return &cmdutil.Factory{
		IOStreams: &iostreams.IOStreams{
			In:     &bytes.Buffer{},
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		},
		Config: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Client: func() (*api.Client, error) {
			return nil, nil
		},
		Remote: func() (*git.Remote, error) {
			return &git.Remote{
				Name:  "origin",
				Host:  "gitlab.com",
				Owner: "test-owner",
				Repo:  "test-repo",
			}, nil
		},
	}
}

func TestNewMRCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewMRCmd(f)

	if cmd.Use != "mr <command>" {
		t.Errorf("expected Use to be 'mr <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage merge requests" {
		t.Errorf("expected Short to be 'Manage merge requests', got %q", cmd.Short)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "merge-request" {
		t.Errorf("expected alias 'merge-request', got %v", cmd.Aliases)
	}
}

func TestMRCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewMRCmd(f)

	expectedSubcommands := []string{
		"create",
		"list",
		"view",
		"merge",
		"close",
		"reopen",
		"approve",
		"checkout",
		"diff",
		"comment",
		"edit",
		"discussions",
		"reply",
		"suggest",
		"resolve",
		"unresolve",
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

func TestMRCreateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRCreateCmd(f)

	expectedFlags := map[string]bool{
		"title":                true,
		"description":          true,
		"source-branch":        true,
		"target-branch":        true,
		"assignee":             true,
		"reviewer":             true,
		"label":                true,
		"milestone":            true,
		"draft":                true,
		"squash":               true,
		"remove-source-branch": true,
		"web":                  true,
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

func TestMRListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRListCmd(f)

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
}

func TestMRViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRViewCmd(f)

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

func TestMRMergeCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRMergeCmd(f)

	expectedFlags := []string{
		"squash",
		"remove-source-branch",
		"message",
		"when-pipeline-succeeds",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestMRCloseCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newMRCloseCmd(f)

	if cmd.Use != "close [<id>]" {
		t.Errorf("expected Use to be 'close [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "Close a merge request" {
		t.Errorf("expected Short to be 'Close a merge request', got %q", cmd.Short)
	}
}

func TestMRReopenCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newMRReopenCmd(f)

	if cmd.Use != "reopen [<id>]" {
		t.Errorf("expected Use to be 'reopen [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "Reopen a merge request" {
		t.Errorf("expected Short to be 'Reopen a merge request', got %q", cmd.Short)
	}
}

func TestMRApproveCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newMRApproveCmd(f)

	if cmd.Use != "approve [<id>]" {
		t.Errorf("expected Use to be 'approve [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "Approve a merge request" {
		t.Errorf("expected Short to be 'Approve a merge request', got %q", cmd.Short)
	}
}

func TestMRCheckoutCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newMRCheckoutCmd(f)

	if cmd.Use != "checkout [<id>]" {
		t.Errorf("expected Use to be 'checkout [<id>]', got %q", cmd.Use)
	}
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "co" {
		t.Errorf("expected alias 'co', got %v", cmd.Aliases)
	}
}

func TestMRDiffCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newMRDiffCmd(f)

	if cmd.Use != "diff [<id>]" {
		t.Errorf("expected Use to be 'diff [<id>]', got %q", cmd.Use)
	}
	if cmd.Short != "View changes in a merge request" {
		t.Errorf("expected Short to be 'View changes in a merge request', got %q", cmd.Short)
	}
}

func TestMRCommentCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRCommentCmd(f)

	expectedFlags := []string{
		"body",
		"file",
		"line",
		"old-line",
		"commit",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestMREditCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMREditCmd(f)

	expectedFlags := []string{
		"title",
		"description",
		"assignee",
		"reviewer",
		"label",
		"milestone",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}
}

func TestMRDiscussionsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRDiscussionsCmd(f)

	expectedFlags := []string{
		"format",
		"json",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default format is "table"
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be 'table', got %q", formatFlag.DefValue)
	}

	if cmd.Use != "discussions [<id>]" {
		t.Errorf("expected Use to be 'discussions [<id>]', got %q", cmd.Use)
	}
}

func TestMRReplyCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRReplyCmd(f)

	expectedFlags := []string{
		"body",
		"discussion",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "reply [<id>]" {
		t.Errorf("expected Use to be 'reply [<id>]', got %q", cmd.Use)
	}
}

func TestMRSuggestCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRSuggestCmd(f)

	expectedFlags := []string{
		"body",
		"file",
		"line",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "suggest [<id>]" {
		t.Errorf("expected Use to be 'suggest [<id>]', got %q", cmd.Use)
	}
}

func TestMRResolveCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRResolveCmd(f)

	expectedFlags := []string{
		"discussion",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "resolve [<id>]" {
		t.Errorf("expected Use to be 'resolve [<id>]', got %q", cmd.Use)
	}
}

func TestMRUnresolveCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newMRUnresolveCmd(f)

	expectedFlags := []string{
		"discussion",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "unresolve [<id>]" {
		t.Errorf("expected Use to be 'unresolve [<id>]', got %q", cmd.Use)
	}
}

func TestParseMRArg(t *testing.T) {
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
			name:    "valid number with exclamation",
			args:    []string{"!456"},
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
			got, err := parseMRArg(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMRArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMRArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		t    *time.Time
		want string
	}{
		{
			name: "nil time",
			t:    nil,
			want: "",
		},
		{
			name: "just now",
			t:    ptrTime(now.Add(-30 * time.Second)),
			want: "just now",
		},
		{
			name: "1 minute ago",
			t:    ptrTime(now.Add(-1 * time.Minute)),
			want: "1 minute ago",
		},
		{
			name: "5 minutes ago",
			t:    ptrTime(now.Add(-5 * time.Minute)),
			want: "5 minutes ago",
		},
		{
			name: "1 hour ago",
			t:    ptrTime(now.Add(-1 * time.Hour)),
			want: "1 hour ago",
		},
		{
			name: "3 hours ago",
			t:    ptrTime(now.Add(-3 * time.Hour)),
			want: "3 hours ago",
		},
		{
			name: "1 day ago",
			t:    ptrTime(now.Add(-24 * time.Hour)),
			want: "1 day ago",
		},
		{
			name: "5 days ago",
			t:    ptrTime(now.Add(-5 * 24 * time.Hour)),
			want: "5 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeAgo(tt.t)
			if got != tt.want {
				t.Errorf("timeAgo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeAgo_OldDate(t *testing.T) {
	// For dates older than 30 days, should return formatted date
	oldTime := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
	result := timeAgo(&oldTime)
	if !strings.Contains(result, "Jan") || !strings.Contains(result, "2020") {
		t.Errorf("expected formatted date for old time, got %q", result)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// ============================================================================
// EXECUTION TESTS - Test actual command execution with mocked API responses
// ============================================================================

func TestMRList_SuccessWithOpenMRs(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		// Match any GET request to merge_requests endpoint
		if strings.Contains(r.URL.Path, "/merge_requests") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureMROpen,
			})
			return
		}
		// Return 200 for any other requests (like auth checks)
		cmdtest.JSONResponse(w, 200, map[string]interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRListCmd(f.Factory)
	cmd.SetArgs([]string{"--state", "opened"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Add new feature") {
		t.Errorf("expected output to contain MR title, got: %s", output)
	}
}

func TestMRList_SuccessWithMergedMRs(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/merge_requests") && strings.Contains(r.URL.Query().Get("state"), "merged") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				cmdtest.FixtureMRMerged,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRListCmd(f.Factory)
	cmd.SetArgs([]string{"--state", "merged"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Fix critical bug") {
		t.Errorf("expected output to contain MR title, got: %s", output)
	}
}

func TestMRView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/merge_requests/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureMROpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Add new feature") {
		t.Errorf("expected output to contain MR title, got: %s", output)
	}
}

func TestMRCreate_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureMROpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", "Test MR", "--source-branch", "feature", "--target-branch", "main"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "!1") {
		t.Errorf("expected success message with MR number, got: %s", output)
	}
}

func TestMRMerge_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/1/merge") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureMRMerged)
			return
		}
		if strings.Contains(r.URL.Path, "/merge_requests/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureMROpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRMergeCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(strings.ToLower(output), "merged") {
		t.Errorf("expected success message about merging, got: %s", output)
	}
}

func TestMRClose_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureMRClosed)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRCloseCmd(f.Factory)
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

func TestMRApprove_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests/1/approve") {
			cmdtest.JSONResponse(w, 201, cmdtest.FixtureMROpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRApproveCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(strings.ToLower(output), "approved") {
		t.Errorf("expected success message about approval, got: %s", output)
	}
}

// ============================================================================
// ERROR PATH TESTS - Test error handling for common failure modes
// ============================================================================

func TestMRView_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRViewCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent MR")
	}

	if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestMRList_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "401") && !strings.Contains(errStr, "unauthorized") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestMRCreate_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRCreateCmd(f.Factory)
	// Missing required --title flag
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing required fields")
	}
}

func TestMRMerge_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "merge request not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRMergeCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent MR")
	}

	if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestMRClose_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 403, "403 Forbidden")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRCloseCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected permission error")
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "403") && !strings.Contains(errStr, "forbidden") {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestResolveUserIDs(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users") && strings.Contains(r.URL.Query().Get("username"), "alice") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{"id": 123, "username": "alice"},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	client, err := f.Factory.Client()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ids, err := resolveUserIDs(client, []string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID, got %d", len(ids))
	}
	if ids[0] != 123 {
		t.Errorf("expected ID 123, got %d", ids[0])
	}
}

func TestMRReopen_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureMROpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRReopenCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMRComment_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests/1/notes") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":   1,
				"body": "Test comment",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRCommentCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--body", "Test comment"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMREdit_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge_requests/1") {
			cmdtest.JSONResponse(w, 200, cmdtest.FixtureMROpen)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMREditCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--title", "Updated title"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMRDiff_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/merge_requests/1/diffs") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"old_path": "file.go",
					"new_path": "file.go",
					"diff":     "@@ -1,1 +1,1 @@\n-old\n+new",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiffCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMRReopen_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 MR Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRReopenCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found MR")
	}
}

func TestMREdit_ValidationError(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMREditCmd(f.Factory)
	cmd.SetArgs([]string{}) // Missing MR ID

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for missing MR ID")
	}
}

func TestMRList_EmptyResult(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRListCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMRCreate_EmptyTitle(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRCreateCmd(f.Factory)
	cmd.SetArgs([]string{"--title", ""}) // Empty title

	err := cmd.Execute()
	// May fail validation
	_ = err
}

func TestMRComment_EmptyBody(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRCommentCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--body", ""}) // Empty comment body

	err := cmd.Execute()
	// Should fail validation
	_ = err
}

// ============================================================================
// MR RESOLVE / UNRESOLVE / DISCUSSIONS / REPLY / SUGGEST EXECUTION TESTS
// ============================================================================

func TestMRResolve_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/discussions/disc123") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":            "disc123",
				"individual_note": false,
				"notes": []interface{}{
					map[string]interface{}{
						"id":   1,
						"body": "test note",
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRResolveCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--discussion", "disc123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Resolved discussion") {
		t.Errorf("expected 'Resolved discussion' message, got: %s", output)
	}
}

func TestMRResolve_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRResolveCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--discussion", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent discussion")
	}
}

func TestMRResolve_MissingDiscussion(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRResolveCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --discussion flag")
	}
}

func TestMRResolve_MissingMRID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRResolveCmd(f.Factory)
	cmd.SetArgs([]string{"--discussion", "disc123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing MR ID")
	}
}

func TestMRUnresolve_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/discussions/disc456") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":            "disc456",
				"individual_note": false,
				"notes": []interface{}{
					map[string]interface{}{
						"id":   2,
						"body": "test note",
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRUnresolveCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--discussion", "disc456"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Unresolved discussion") {
		t.Errorf("expected 'Unresolved discussion' message, got: %s", output)
	}
}

func TestMRUnresolve_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRUnresolveCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--discussion", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent discussion")
	}
}

func TestMRDiscussions_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/merge_requests/1/discussions") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":              "disc001",
					"individual_note": false,
					"notes": []interface{}{
						map[string]interface{}{
							"id":         1,
							"body":       "This is a discussion note",
							"author":     map[string]interface{}{"id": 1, "username": "test-user"},
							"created_at": "2024-01-01T00:00:00.000Z",
							"resolvable": true,
							"resolved":   false,
						},
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiscussionsCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "#1 (unresolved) test-user") {
		t.Errorf("expected header with author and status, got: %s", output)
	}
	if !strings.Contains(output, "This is a discussion note") {
		t.Errorf("expected note body in output, got: %s", output)
	}
}

func TestMRDiscussions_InlineComment(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/merge_requests/1/discussions") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":              "disc001",
					"individual_note": false,
					"notes": []interface{}{
						map[string]interface{}{
							"id":         1,
							"type":       "DiffNote",
							"body":       "Is this correct?",
							"author":     map[string]interface{}{"id": 1, "username": "pkramer"},
							"created_at": "2024-01-01T00:00:00.000Z",
							"resolvable": true,
							"resolved":   false,
							"position": map[string]interface{}{
								"position_type": "text",
								"new_path":      "src/devices/DevicesController.kt",
								"new_line":      47,
								"old_path":      "src/devices/DevicesController.kt",
								"old_line":      45,
							},
						},
						map[string]interface{}{
							"id":         2,
							"body":       "Yes, the return type changed",
							"author":     map[string]interface{}{"id": 2, "username": "reviewer"},
							"created_at": "2024-01-01T01:00:00.000Z",
							"system":     false,
						},
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiscussionsCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "pkramer on src/devices/DevicesController.kt:47") {
		t.Errorf("expected file:line context in output, got: %s", output)
	}
	if !strings.Contains(output, "Is this correct?") {
		t.Errorf("expected note body, got: %s", output)
	}
	if !strings.Contains(output, "reviewer: Yes, the return type changed") {
		t.Errorf("expected threaded reply, got: %s", output)
	}
}

func TestMRDiscussions_Empty(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/discussions") {
			cmdtest.JSONResponse(w, 200, []interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiscussionsCmd(f.Factory)
	cmd.SetArgs([]string{"1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No discussions found") {
		t.Errorf("expected 'No discussions found' message, got: %s", errOutput)
	}
}

func TestMRDiscussions_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiscussionsCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent MR")
	}
}

func TestMRDiscussions_MissingMRID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiscussionsCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing MR ID")
	}
}

func TestMRDiscussions_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/discussions") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":            "disc001",
					"individual_note": false,
					"notes": []interface{}{
						map[string]interface{}{
							"id":         1,
							"body":       "Discussion note",
							"created_at": "2024-01-01T00:00:00.000Z",
						},
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRDiscussionsCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--format", "json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMRReply_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/discussions/disc123/notes") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":   10,
				"body": "Thanks for the review!",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRReplyCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--discussion", "disc123", "--body", "Thanks for the review!"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Replied to discussion") {
		t.Errorf("expected 'Replied to discussion' message, got: %s", output)
	}
}

func TestMRReply_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRReplyCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--discussion", "nonexistent", "--body", "test"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMRSuggest_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		// GET MR to get diff refs
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/merge_requests/1") &&
			!strings.Contains(r.URL.Path, "/discussions") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":    100,
				"iid":   1,
				"title": "Test MR",
				"state": "opened",
				"diff_refs": map[string]interface{}{
					"base_sha":  "aaa111",
					"head_sha":  "bbb222",
					"start_sha": "ccc333",
				},
				"web_url":       "https://gitlab.com/test-owner/test-repo/-/merge_requests/1",
				"source_branch": "feature",
				"target_branch": "main",
			})
			return
		}
		// POST discussion
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/discussions") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id": "disc_suggest",
				"notes": []interface{}{
					map[string]interface{}{
						"id":   20,
						"body": "```suggestion\nnewCode\n```",
					},
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newMRSuggestCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--file", "main.go", "--line", "10", "--body", "newCode"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Added suggestion") {
		t.Errorf("expected 'Added suggestion' message, got: %s", output)
	}
}

func TestMRSuggest_MissingRequiredFlags(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMRSuggestCmd(f.Factory)
	cmd.SetArgs([]string{"1"}) // Missing --body, --file, --line

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestResolveUserIDs_UserNotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users") {
			cmdtest.JSONResponse(w, 200, []interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	client, err := f.Factory.Client()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = resolveUserIDs(client, []string{"nonexistent-user"})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("expected 'user not found' error, got: %v", err)
	}
}

func TestResolveUserIDs_WithAtPrefix(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users") && strings.Contains(r.URL.Query().Get("username"), "bob") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{"id": 456, "username": "bob"},
			})
			return
		}
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	client, err := f.Factory.Client()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ids, err := resolveUserIDs(client, []string{"@bob"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 456 {
		t.Errorf("expected [456], got %v", ids)
	}
}

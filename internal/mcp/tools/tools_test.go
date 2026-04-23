package tools

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupServer creates an MCP server with all tools registered, a mock GitLab API,
// and returns a connected client session. The mock API routes requests through the
// provided RouterMux.
func setupServer(t *testing.T, mux *cmdtest.RouterMux) *mcp.ClientSession {
	t.Helper()

	tf := cmdtest.NewTestFactory(t)

	// Set up mock GitLab API server
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", mux.ServeHTTP)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-mcp",
		Version: "0.0.1",
	}, nil)

	RegisterMRTools(server, tf.Factory)
	RegisterIssueTools(server, tf.Factory)
	RegisterPipelineTools(server, tf.Factory)
	RegisterRepoTools(server, tf.Factory)
	RegisterReleaseTools(server, tf.Factory)
	RegisterLabelTools(server, tf.Factory)
	RegisterSnippetTools(server, tf.Factory)
	RegisterBranchTools(server, tf.Factory)
	RegisterUserTools(server, tf.Factory)
	RegisterVariableTools(server, tf.Factory)
	RegisterEnvironmentTools(server, tf.Factory)
	RegisterDeploymentTools(server, tf.Factory)

	st, ct := mcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	return cs
}

// callTool is a test helper that calls a tool and returns the text content.
// If the tool returns an error (IsError=true), it returns it as a Go error.
func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, error) {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in result")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("expected TextContent, got %T", result.Content[0])
	}
	if result.IsError {
		return "", fmt.Errorf("tool error: %s", tc.Text)
	}
	return tc.Text, nil
}

// --- Helper function tests ---

func TestClampPerPage(t *testing.T) {
	tests := []struct {
		input int64
		want  int64
	}{
		{0, 30},
		{-1, 30},
		{1, 1},
		{50, 50},
		{100, 100},
		{101, 100},
		{999, 100},
	}
	for _, tt := range tests {
		got := clampPerPage(tt.input)
		if got != tt.want {
			t.Errorf("clampPerPage(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestRequireID(t *testing.T) {
	if err := requireID(0, "test"); err == nil {
		t.Error("requireID(0) should return error")
	}
	if err := requireID(-1, "test"); err == nil {
		t.Error("requireID(-1) should return error")
	}
	if err := requireID(1, "test"); err != nil {
		t.Errorf("requireID(1) should not return error, got: %v", err)
	}
}

func TestRequireString(t *testing.T) {
	if err := requireString("", "test"); err == nil {
		t.Error("requireString('') should return error")
	}
	if err := requireString("  ", "test"); err == nil {
		t.Error("requireString('  ') should return error")
	}
	if err := requireString("hello", "test"); err != nil {
		t.Errorf("requireString('hello') should not return error, got: %v", err)
	}
}

func TestReadLog(t *testing.T) {
	t.Run("normal log", func(t *testing.T) {
		r := strings.NewReader("hello world")
		got, err := readLog(r)
		if err != nil {
			t.Fatal(err)
		}
		if got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("truncated log", func(t *testing.T) {
		// Create a reader larger than 1 MiB
		data := strings.Repeat("x", 1024*1024+100)
		r := strings.NewReader(data)
		got, err := readLog(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(got, "\n[log truncated at 1 MiB]") {
			t.Error("expected truncation notice")
		}
	})
}

func TestTextResult(t *testing.T) {
	result, _, err := textResult(map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(tc.Text, `"key": "value"`) {
		t.Errorf("expected JSON output, got: %s", tc.Text)
	}
}

func TestPlainResult(t *testing.T) {
	result := plainResult("hello")
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if tc.Text != "hello" {
		t.Errorf("got %q, want %q", tc.Text, "hello")
	}
}

// --- Branch tool tests ---

func TestBranchList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/repository/branches", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"name": "main", "default": true, "merged": false},
			{"name": "feature", "default": false, "merged": false},
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "branch_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "main") {
		t.Errorf("expected 'main' in output, got: %s", text)
	}
	if !strings.Contains(text, "feature") {
		t.Errorf("expected 'feature' in output, got: %s", text)
	}
}

func TestBranchCreate(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/repository/branches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"name":    "new-branch",
			"default": false,
			"merged":  false,
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "branch_create", map[string]any{
		"repo": "test-owner/test-repo",
		"name": "new-branch",
		"ref":  "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "new-branch") {
		t.Errorf("expected 'new-branch' in output, got: %s", text)
	}
}

func TestBranchDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/repository/branches/old-branch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "branch_delete", map[string]any{
		"repo": "test-owner/test-repo",
		"name": "old-branch",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "deleted successfully") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- User tool tests ---

func TestUserWhoami(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/user", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockUser(1, "testuser", "Test User"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "user_whoami", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "testuser") {
		t.Errorf("expected 'testuser' in output, got: %s", text)
	}
}

// --- Issue tool tests ---

func TestIssueList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockIssue(1, "Bug report", "opened"),
			cmdtest.MockIssue(2, "Feature request", "opened"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Bug report") {
		t.Errorf("expected 'Bug report' in output, got: %s", text)
	}
}

func TestIssueListWithFilters(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters are passed
		if r.URL.Query().Get("state") != "closed" {
			cmdtest.ErrorResponse(w, http.StatusBadRequest, "expected state=closed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockIssue(3, "Fixed bug", "closed"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_list", map[string]any{
		"repo":  "test-owner/test-repo",
		"state": "closed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Fixed bug") {
		t.Errorf("expected 'Fixed bug' in output, got: %s", text)
	}
}

func TestIssueView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockIssue(1, "Bug report", "opened"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_view", map[string]any{
		"repo":  "test-owner/test-repo",
		"issue": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Bug report") {
		t.Errorf("expected 'Bug report' in output, got: %s", text)
	}
}

func TestIssueViewRequiresID(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "issue_view", map[string]any{
		"repo":  "test-owner/test-repo",
		"issue": 0,
	})
	if err == nil {
		t.Error("expected error for missing issue ID")
	}
}

func TestIssueCreate(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusCreated, cmdtest.MockIssue(10, "New Issue", "opened"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_create", map[string]any{
		"repo":  "test-owner/test-repo",
		"title": "New Issue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created issue #10") {
		t.Errorf("expected creation confirmation, got: %s", text)
	}
}

func TestIssueCreateRequiresTitle(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "issue_create", map[string]any{
		"repo":  "test-owner/test-repo",
		"title": "",
	})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestIssueClose(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues/5", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockIssue(5, "Bug", "closed"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_close", map[string]any{
		"repo":  "test-owner/test-repo",
		"issue": 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Closed issue #5") {
		t.Errorf("expected close confirmation, got: %s", text)
	}
}

func TestIssueReopen(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues/5", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockIssue(5, "Bug", "opened"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_reopen", map[string]any{
		"repo":  "test-owner/test-repo",
		"issue": 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Reopened issue #5") {
		t.Errorf("expected reopen confirmation, got: %s", text)
	}
}

func TestIssueComment(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues/3/notes", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"id":   100,
			"body": "test comment",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_comment", map[string]any{
		"repo":    "test-owner/test-repo",
		"issue":   3,
		"message": "test comment",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Added comment to #3") {
		t.Errorf("expected comment confirmation, got: %s", text)
	}
}

func TestIssueCommentRequiresMessage(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "issue_comment", map[string]any{
		"repo":    "test-owner/test-repo",
		"issue":   3,
		"message": "",
	})
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestIssueEdit(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues/7", func(w http.ResponseWriter, r *http.Request) {
		issue := cmdtest.MockIssue(7, "Updated Title", "opened")
		issue["web_url"] = "https://gitlab.com/test-owner/test-repo/-/issues/7"
		cmdtest.JSONResponse(w, http.StatusOK, issue)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_edit", map[string]any{
		"repo":  "test-owner/test-repo",
		"issue": 7,
		"title": "Updated Title",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Updated issue #7") {
		t.Errorf("expected edit confirmation, got: %s", text)
	}
}

func TestIssueDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues/9", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "issue_delete", map[string]any{
		"repo":  "test-owner/test-repo",
		"issue": 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Deleted issue #9") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- MR tool tests ---

func TestMRList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockMergeRequest(1, "Fix bug", "opened"),
			cmdtest.MockMergeRequest(2, "Add feature", "merged"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Fix bug") {
		t.Errorf("expected 'Fix bug' in output, got: %s", text)
	}
}

func TestMRView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockMergeRequest(1, "Fix bug", "opened"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_view", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Fix bug") {
		t.Errorf("expected 'Fix bug' in output, got: %s", text)
	}
}

func TestMRViewRequiresID(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "mr_view", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   0,
	})
	if err == nil {
		t.Error("expected error for missing MR ID")
	}
}

func TestMRDiff(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1/diffs", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{
				"old_path": "file.go",
				"new_path": "file.go",
				"diff":     "@@ -1,3 +1,4 @@\n+new line\n",
			},
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_diff", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "--- a/file.go") {
		t.Errorf("expected diff header, got: %s", text)
	}
	if !strings.Contains(text, "+new line") {
		t.Errorf("expected diff content, got: %s", text)
	}
}

func TestMRNotes(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1/notes", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"id": 1, "body": "please rename this function", "system": false, "author": map[string]any{"username": "alice"}},
			{"id": 2, "body": "changed label to ready", "system": true, "author": map[string]any{"username": "bob"}},
			{"id": 3, "body": "LGTM after rebase", "system": false, "author": map[string]any{"username": "carol"}},
		})
	})

	cs := setupServer(t, mux)

	// Default: system notes excluded.
	text, err := callTool(t, cs, "mr_notes", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "please rename this function") {
		t.Errorf("expected user note body, got: %s", text)
	}
	if !strings.Contains(text, "LGTM after rebase") {
		t.Errorf("expected second user note body, got: %s", text)
	}
	if strings.Contains(text, "changed label to ready") {
		t.Errorf("system note should be filtered out by default, got: %s", text)
	}

	// With include_system: all notes returned.
	text, err = callTool(t, cs, "mr_notes", map[string]any{
		"repo":           "test-owner/test-repo",
		"mr":             1,
		"include_system": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "changed label to ready") {
		t.Errorf("include_system=true must surface system notes, got: %s", text)
	}
}

func TestMRNotesRequiresID(t *testing.T) {
	cs := setupServer(t, cmdtest.NewRouterMux())
	_, err := callTool(t, cs, "mr_notes", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err == nil {
		t.Fatal("expected error when mr is missing")
	}
}

func TestMRComment(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1/notes", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"id":   200,
			"body": "LGTM",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_comment", map[string]any{
		"repo":    "test-owner/test-repo",
		"mr":      1,
		"message": "LGTM",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Added comment to !1") {
		t.Errorf("expected comment confirmation, got: %s", text)
	}
}

func TestMRApprove(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1/approve", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"id": 1,
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_approve", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Approved merge request !1") {
		t.Errorf("expected approval confirmation, got: %s", text)
	}
}

func TestMRMerge(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1/merge", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockMergeRequest(1, "Fix bug", "merged"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_merge", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Merged merge request !1") {
		t.Errorf("expected merge confirmation, got: %s", text)
	}
}

func TestMRClose(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockMergeRequest(1, "Fix bug", "closed"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_close", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Closed merge request !1") {
		t.Errorf("expected close confirmation, got: %s", text)
	}
}

func TestMRReopen(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockMergeRequest(1, "Fix bug", "opened"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_reopen", map[string]any{
		"repo": "test-owner/test-repo",
		"mr":   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Reopened merge request !1") {
		t.Errorf("expected reopen confirmation, got: %s", text)
	}
}

func TestMRCreate(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		mr := cmdtest.MockMergeRequest(15, "New MR", "opened")
		mr["web_url"] = "https://gitlab.com/test-owner/test-repo/-/merge_requests/15"
		cmdtest.JSONResponse(w, http.StatusCreated, mr)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_create", map[string]any{
		"repo":          "test-owner/test-repo",
		"title":         "New MR",
		"source_branch": "feature",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created merge request !15") {
		t.Errorf("expected creation confirmation, got: %s", text)
	}
}

func TestMRCreateDraft(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		mr := cmdtest.MockMergeRequest(16, "Draft: WIP Feature", "opened")
		mr["web_url"] = "https://gitlab.com/test-owner/test-repo/-/merge_requests/16"
		cmdtest.JSONResponse(w, http.StatusCreated, mr)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_create", map[string]any{
		"repo":          "test-owner/test-repo",
		"title":         "WIP Feature",
		"source_branch": "feature",
		"draft":         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created merge request !16") {
		t.Errorf("expected creation confirmation, got: %s", text)
	}
}

func TestMRCreateRequiresTitle(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "mr_create", map[string]any{
		"repo":          "test-owner/test-repo",
		"title":         "",
		"source_branch": "feature",
	})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestMRCreateRequiresSourceBranch(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "mr_create", map[string]any{
		"repo":          "test-owner/test-repo",
		"title":         "My MR",
		"source_branch": "",
	})
	if err == nil {
		t.Error("expected error for missing source_branch")
	}
}

func TestMREdit(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		mr := cmdtest.MockMergeRequest(1, "Updated MR", "opened")
		mr["web_url"] = "https://gitlab.com/test-owner/test-repo/-/merge_requests/1"
		cmdtest.JSONResponse(w, http.StatusOK, mr)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "mr_edit", map[string]any{
		"repo":  "test-owner/test-repo",
		"mr":    1,
		"title": "Updated MR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Updated merge request !1") {
		t.Errorf("expected edit confirmation, got: %s", text)
	}
}

// --- Pipeline tool tests ---

func TestPipelineList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/pipelines", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockPipeline(100, "main", "success"),
			cmdtest.MockPipeline(101, "feature", "failed"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "success") {
		t.Errorf("expected 'success' in output, got: %s", text)
	}
}

func TestPipelineView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/pipelines/100", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockPipeline(100, "main", "success"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_view", map[string]any{
		"repo":     "test-owner/test-repo",
		"pipeline": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "success") {
		t.Errorf("expected 'success' in output, got: %s", text)
	}
}

func TestPipelineRun(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/triggers", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"id": 1, "token": "test-trigger-token", "description": "glab-cli"},
		})
	})
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/trigger/pipeline", func(w http.ResponseWriter, r *http.Request) {
		p := cmdtest.MockPipeline(200, "main", "created")
		p["web_url"] = "https://gitlab.com/test-owner/test-repo/-/pipelines/200"
		cmdtest.JSONResponse(w, http.StatusCreated, p)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_run", map[string]any{
		"repo": "test-owner/test-repo",
		"ref":  "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created pipeline #200") {
		t.Errorf("expected pipeline creation message, got: %s", text)
	}
}

func TestPipelineRunWithVariables(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/triggers", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"id": 1, "token": "test-trigger-token", "description": "glab-cli"},
		})
	})
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/trigger/pipeline", func(w http.ResponseWriter, r *http.Request) {
		p := cmdtest.MockPipeline(201, "main", "created")
		p["web_url"] = "https://gitlab.com/test-owner/test-repo/-/pipelines/201"
		cmdtest.JSONResponse(w, http.StatusCreated, p)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_run", map[string]any{
		"repo":      "test-owner/test-repo",
		"ref":       "main",
		"variables": "KEY1=val1;KEY2=val2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created pipeline #201") {
		t.Errorf("expected pipeline creation message, got: %s", text)
	}
}

func TestPipelineRunRequiresRef(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "pipeline_run", map[string]any{
		"repo": "test-owner/test-repo",
		"ref":  "",
	})
	if err == nil {
		t.Error("expected error for missing ref")
	}
}

func TestPipelineCancel(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/pipelines/100/cancel", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockPipeline(100, "main", "canceled"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_cancel", map[string]any{
		"repo":     "test-owner/test-repo",
		"pipeline": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Canceled pipeline #100") {
		t.Errorf("expected cancel confirmation, got: %s", text)
	}
}

func TestPipelineRetry(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/pipelines/100/retry", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockPipeline(100, "main", "running"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_retry", map[string]any{
		"repo":     "test-owner/test-repo",
		"pipeline": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Retried pipeline #100") {
		t.Errorf("expected retry confirmation, got: %s", text)
	}
}

func TestPipelineDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/pipelines/100", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_delete", map[string]any{
		"repo":     "test-owner/test-repo",
		"pipeline": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Deleted pipeline #100") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

func TestPipelineJobs(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/pipelines/100/jobs", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"id": 500, "name": "build", "status": "success"},
			{"id": 501, "name": "test", "status": "failed"},
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_jobs", map[string]any{
		"repo":     "test-owner/test-repo",
		"pipeline": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "build") {
		t.Errorf("expected 'build' in output, got: %s", text)
	}
}

func TestPipelineJobLog(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/jobs/500/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Running tests...\nAll tests passed!"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "pipeline_job_log", map[string]any{
		"repo": "test-owner/test-repo",
		"job":  500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "All tests passed!") {
		t.Errorf("expected job log content, got: %s", text)
	}
}

// --- Repo tool tests ---

func TestRepoList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockProject(1, "project1", "project1"),
			cmdtest.MockProject(2, "project2", "project2"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "repo_list", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "project1") {
		t.Errorf("expected 'project1' in output, got: %s", text)
	}
}

func TestRepoListByGroup(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/groups/my-group/projects", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockProject(1, "group-project", "group-project"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "repo_list", map[string]any{
		"group": "my-group",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "group-project") {
		t.Errorf("expected 'group-project' in output, got: %s", text)
	}
}

func TestRepoView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockProject(1, "test-repo", "test-repo"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "repo_view", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "test-repo") {
		t.Errorf("expected 'test-repo' in output, got: %s", text)
	}
}

func TestRepoViewRequiresRepo(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "repo_view", map[string]any{
		"repo": "",
	})
	if err == nil {
		t.Error("expected error for missing repo")
	}
}

// --- Release tool tests ---

func TestReleaseList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/releases", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockRelease("v1.0.0", "Release 1.0", "First release"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "release_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "v1.0.0") {
		t.Errorf("expected 'v1.0.0' in output, got: %s", text)
	}
}

func TestReleaseView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/releases/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockRelease("v1.0.0", "Release 1.0", "First release"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "release_view", map[string]any{
		"repo": "test-owner/test-repo",
		"tag":  "v1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Release 1.0") {
		t.Errorf("expected 'Release 1.0' in output, got: %s", text)
	}
}

func TestReleaseViewRequiresTag(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "release_view", map[string]any{
		"repo": "test-owner/test-repo",
		"tag":  "",
	})
	if err == nil {
		t.Error("expected error for missing tag")
	}
}

func TestReleaseCreate(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/releases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusCreated, cmdtest.MockRelease("v2.0.0", "Release 2.0", "Second release"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "release_create", map[string]any{
		"repo":        "test-owner/test-repo",
		"tag":         "v2.0.0",
		"name":        "Release 2.0",
		"description": "Second release",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created release v2.0.0") {
		t.Errorf("expected creation confirmation, got: %s", text)
	}
}

func TestReleaseDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/releases/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockRelease("v1.0.0", "Release 1.0", "First release"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "release_delete", map[string]any{
		"repo": "test-owner/test-repo",
		"tag":  "v1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Deleted release v1.0.0") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- Label tool tests ---

func TestLabelList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/labels", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockLabel(1, "bug", "#FF0000"),
			cmdtest.MockLabel(2, "feature", "#00FF00"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "label_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "bug") {
		t.Errorf("expected 'bug' in output, got: %s", text)
	}
}

func TestLabelCreate(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/labels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusCreated, cmdtest.MockLabel(3, "urgent", "#FF0000"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "label_create", map[string]any{
		"repo":  "test-owner/test-repo",
		"name":  "urgent",
		"color": "#FF0000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `Created label "urgent"`) {
		t.Errorf("expected creation confirmation, got: %s", text)
	}
}

func TestLabelCreateRequiresNameAndColor(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)

	// Missing name
	_, err := callTool(t, cs, "label_create", map[string]any{
		"repo":  "test-owner/test-repo",
		"name":  "",
		"color": "#FF0000",
	})
	if err == nil {
		t.Error("expected error for missing name")
	}

	// Missing color
	_, err = callTool(t, cs, "label_create", map[string]any{
		"repo":  "test-owner/test-repo",
		"name":  "test",
		"color": "",
	})
	if err == nil {
		t.Error("expected error for missing color")
	}
}

func TestLabelDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/labels/bug", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "label_delete", map[string]any{
		"repo": "test-owner/test-repo",
		"name": "bug",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `Deleted label "bug"`) {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- Snippet tool tests ---

func TestSnippetList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/snippets", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			cmdtest.MockSnippet(1, "My Snippet", "code.go"),
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "snippet_list", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "My Snippet") {
		t.Errorf("expected 'My Snippet' in output, got: %s", text)
	}
}

func TestSnippetView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/snippets/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, cmdtest.MockSnippet(1, "My Snippet", "code.go"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "snippet_view", map[string]any{
		"snippet": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "My Snippet") {
		t.Errorf("expected 'My Snippet' in output, got: %s", text)
	}
}

func TestSnippetViewRaw(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/snippets/1/raw", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("package main\n\nfunc main() {}"))
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "snippet_view", map[string]any{
		"snippet": 1,
		"raw":     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "package main") {
		t.Errorf("expected raw snippet content, got: %s", text)
	}
}

func TestSnippetCreate(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/snippets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s := cmdtest.MockSnippet(10, "New Snippet", "test.go")
		s["web_url"] = "https://gitlab.com/-/snippets/10"
		cmdtest.JSONResponse(w, http.StatusCreated, s)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "snippet_create", map[string]any{
		"title":    "New Snippet",
		"filename": "test.go",
		"content":  "package test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created snippet #10") {
		t.Errorf("expected creation confirmation, got: %s", text)
	}
}

func TestSnippetCreateRequiresFields(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupServer(t, mux)

	_, err := callTool(t, cs, "snippet_create", map[string]any{
		"title":    "",
		"filename": "test.go",
		"content":  "package test",
	})
	if err == nil {
		t.Error("expected error for missing title")
	}

	_, err = callTool(t, cs, "snippet_create", map[string]any{
		"title":    "Test",
		"filename": "",
		"content":  "package test",
	})
	if err == nil {
		t.Error("expected error for missing filename")
	}

	_, err = callTool(t, cs, "snippet_create", map[string]any{
		"title":    "Test",
		"filename": "test.go",
		"content":  "",
	})
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestSnippetDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/snippets/5", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "snippet_delete", map[string]any{
		"snippet": 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Deleted snippet #5") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- API error tests ---

func TestToolAPIError(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, http.StatusForbidden, "403 Forbidden")
	})

	cs := setupServer(t, mux)
	_, err := callTool(t, cs, "issue_list", map[string]any{
		"repo": "test-owner/test-repo",
	})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// --- Variable tool tests ---

func TestVariableList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/variables", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"key": "DEPLOY_TOKEN", "value": "secret", "masked": true},
			{"key": "ENV", "value": "prod", "masked": false},
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "variable_list", map[string]any{"repo": "test-owner/test-repo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "DEPLOY_TOKEN") || !strings.Contains(text, "ENV") {
		t.Errorf("expected variable keys in output, got: %s", text)
	}
}

func TestVariableGet(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/variables/DEPLOY_TOKEN", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"key": "DEPLOY_TOKEN", "value": "secret",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "variable_get", map[string]any{
		"repo": "test-owner/test-repo", "key": "DEPLOY_TOKEN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "DEPLOY_TOKEN") {
		t.Errorf("expected variable key in output, got: %s", text)
	}
}

func TestVariableGetRequiresKey(t *testing.T) {
	cs := setupServer(t, cmdtest.NewRouterMux())
	_, err := callTool(t, cs, "variable_get", map[string]any{"repo": "test-owner/test-repo"})
	if err == nil {
		t.Fatal("expected error when key is missing")
	}
}

func TestVariableSet_CreatesWhenUpdateFails(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	// PUT (update) fails -> tool falls through to POST (create).
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/variables/NEW_KEY", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, http.StatusNotFound, "not found")
	})
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/variables", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		cmdtest.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"key": "NEW_KEY", "value": "v",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "variable_set", map[string]any{
		"repo": "test-owner/test-repo", "key": "NEW_KEY", "value": "v",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Created") {
		t.Errorf("expected create confirmation, got: %s", text)
	}
}

func TestVariableSet_UpdatesWhenExists(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/variables/EXIST", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"key": "EXIST", "value": "new",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "variable_set", map[string]any{
		"repo": "test-owner/test-repo", "key": "EXIST", "value": "new",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Updated") {
		t.Errorf("expected update confirmation, got: %s", text)
	}
}

func TestVariableDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/variables/GONE", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "variable_delete", map[string]any{
		"repo": "test-owner/test-repo", "key": "GONE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- Environment tool tests ---

func TestEnvironmentList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/environments", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"id": 1, "name": "production", "state": "available"},
			{"id": 2, "name": "staging", "state": "stopped"},
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "environment_list", map[string]any{"repo": "test-owner/test-repo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "production") || !strings.Contains(text, "staging") {
		t.Errorf("expected environment names in output, got: %s", text)
	}
}

func TestEnvironmentView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/environments/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"id": 1, "name": "production", "state": "available",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "environment_view", map[string]any{
		"repo": "test-owner/test-repo", "environment": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "production") {
		t.Errorf("expected environment name in output, got: %s", text)
	}
}

func TestEnvironmentViewRequiresID(t *testing.T) {
	cs := setupServer(t, cmdtest.NewRouterMux())
	_, err := callTool(t, cs, "environment_view", map[string]any{"repo": "test-owner/test-repo"})
	if err == nil {
		t.Fatal("expected error when environment is missing")
	}
}

func TestEnvironmentStop(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/environments/1/stop", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"id": 1, "name": "staging", "state": "stopped",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "environment_stop", map[string]any{
		"repo": "test-owner/test-repo", "environment": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Stopped environment #1") {
		t.Errorf("expected stop confirmation, got: %s", text)
	}
}

func TestEnvironmentDelete(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/environments/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			cmdtest.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "environment_delete", map[string]any{
		"repo": "test-owner/test-repo", "environment": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Deleted environment #1") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}
}

// --- Deployment tool tests ---

func TestDeploymentList(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/deployments", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, []map[string]interface{}{
			{"id": 1, "status": "success", "ref": "main"},
			{"id": 2, "status": "failed", "ref": "feature"},
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "deployment_list", map[string]any{"repo": "test-owner/test-repo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "success") || !strings.Contains(text, "failed") {
		t.Errorf("expected deployment statuses in output, got: %s", text)
	}
}

func TestDeploymentView(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	mux.HandleFunc("/api/v4/projects/test-owner/test-repo/deployments/1", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"id": 1, "status": "success", "ref": "main",
		})
	})

	cs := setupServer(t, mux)
	text, err := callTool(t, cs, "deployment_view", map[string]any{
		"repo": "test-owner/test-repo", "deployment": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "success") {
		t.Errorf("expected deployment status in output, got: %s", text)
	}
}

func TestDeploymentViewRequiresID(t *testing.T) {
	cs := setupServer(t, cmdtest.NewRouterMux())
	_, err := callTool(t, cs, "deployment_view", map[string]any{"repo": "test-owner/test-repo"})
	if err == nil {
		t.Fatal("expected error when deployment is missing")
	}
}

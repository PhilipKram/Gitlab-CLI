package cmdtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- TestIO tests ---

func TestNewTestIO(t *testing.T) {
	tio := NewTestIO()
	if tio == nil {
		t.Fatal("NewTestIO returned nil")
	}
	if tio.In == nil || tio.Out == nil || tio.ErrOut == nil {
		t.Error("all buffers should be initialized")
	}
}

func TestTestIO_String(t *testing.T) {
	tio := NewTestIO()
	tio.Out.WriteString("hello stdout")
	if tio.String() != "hello stdout" {
		t.Errorf("got %q, want %q", tio.String(), "hello stdout")
	}
}

func TestTestIO_ErrString(t *testing.T) {
	tio := NewTestIO()
	tio.ErrOut.WriteString("hello stderr")
	if tio.ErrString() != "hello stderr" {
		t.Errorf("got %q, want %q", tio.ErrString(), "hello stderr")
	}
}

func TestTestIO_IOStreams(t *testing.T) {
	tio := NewTestIO()
	ios := tio.IOStreams()
	if ios == nil {
		t.Fatal("IOStreams returned nil")
	}
	if ios.In == nil || ios.Out == nil || ios.ErrOut == nil {
		t.Error("IOStreams should have non-nil streams")
	}

	// Write through IOStreams and verify through TestIO
	_, _ = fmt.Fprint(ios.Out, "via streams")
	if tio.String() != "via streams" {
		t.Errorf("got %q, want %q", tio.String(), "via streams")
	}
}

// --- TestFactory tests ---

func TestNewTestFactory(t *testing.T) {
	tf := NewTestFactory(t)
	if tf == nil {
		t.Fatal("NewTestFactory returned nil")
	}
	if tf.Factory == nil {
		t.Error("Factory should not be nil")
	}
	if tf.IO == nil {
		t.Error("IO should not be nil")
	}
	if tf.Config == nil {
		t.Error("Config should not be nil")
	}
	if tf.Remote == nil {
		t.Error("Remote should not be nil")
	}
	if tf.Version != "test-version" {
		t.Errorf("Version = %q, want %q", tf.Version, "test-version")
	}
}

func TestNewTestFactory_Config(t *testing.T) {
	tf := NewTestFactory(t)

	cfg, err := tf.Factory.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if cfg.GitRemote != "origin" {
		t.Errorf("GitRemote = %q, want %q", cfg.GitRemote, "origin")
	}
}

func TestNewTestFactory_Remote(t *testing.T) {
	tf := NewTestFactory(t)

	remote, err := tf.Factory.Remote()
	if err != nil {
		t.Fatalf("Remote() error: %v", err)
	}
	if remote.Host != "gitlab.com" {
		t.Errorf("Host = %q, want %q", remote.Host, "gitlab.com")
	}
	if remote.Owner != "test-owner" {
		t.Errorf("Owner = %q, want %q", remote.Owner, "test-owner")
	}
	if remote.Repo != "test-repo" {
		t.Errorf("Repo = %q, want %q", remote.Repo, "test-repo")
	}
}

func TestNewTestFactory_IOStreams(t *testing.T) {
	tf := NewTestFactory(t)

	if tf.IOStreams == nil {
		t.Fatal("IOStreams should not be nil")
	}

	// Write to stdout via factory and check via TestIO
	_, _ = fmt.Fprint(tf.IOStreams.Out, "test output")
	if tf.IO.String() != "test output" {
		t.Errorf("got %q, want %q", tf.IO.String(), "test output")
	}
}

// --- StubInput tests ---

func TestStubInput(t *testing.T) {
	tf := NewTestFactory(t)
	StubInput(t, tf, "user input\n")

	buf := make([]byte, 100)
	n, _ := tf.IO.In.Read(buf)
	got := string(buf[:n])
	if got != "user input\n" {
		t.Errorf("got %q, want %q", got, "user input\n")
	}
}

// --- Assert helpers tests ---

func TestAssertContains(t *testing.T) {
	// Should not fail
	mt := &testing.T{}
	AssertContains(mt, "hello world", "world")
	// We can't easily check mt didn't fail in the testing framework,
	// so just verify the function doesn't panic with valid data
}

func TestAssertNotContains(t *testing.T) {
	mt := &testing.T{}
	AssertNotContains(mt, "hello world", "goodbye")
}

func TestAssertEqual(t *testing.T) {
	mt := &testing.T{}
	AssertEqual(mt, 42, 42)
	AssertEqual(mt, "hello", "hello")
}

// --- CopyReader tests ---

func TestCopyReader(t *testing.T) {
	original := strings.NewReader("test data")
	buf, err := CopyReader(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "test data" {
		t.Errorf("got %q, want %q", buf.String(), "test data")
	}
}

func TestCopyReader_EmptyReader(t *testing.T) {
	original := strings.NewReader("")
	buf, err := CopyReader(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "" {
		t.Errorf("got %q, want empty", buf.String())
	}
}

// --- Mock server tests ---

func TestMockServer(t *testing.T) {
	srv := MockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

func TestJSONResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONResponse(rec, http.StatusOK, map[string]string{"key": "value"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var result map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
}

func TestJSONResponse_NilBody(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONResponse(rec, http.StatusNoContent, nil)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body should be empty, got %q", rec.Body.String())
	}
}

func TestErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	ErrorResponse(rec, http.StatusNotFound, "resource not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if result["message"] != "resource not found" {
		t.Errorf("message = %q, want %q", result["message"], "resource not found")
	}
	if result["error"] != "Not Found" {
		t.Errorf("error = %q, want %q", result["error"], "Not Found")
	}
}

// --- Mock fixture tests ---

func TestMockMergeRequest(t *testing.T) {
	mr := MockMergeRequest(42, "Test MR", "opened")
	if mr["id"] != 42 {
		t.Errorf("id = %v, want 42", mr["id"])
	}
	if mr["title"] != "Test MR" {
		t.Errorf("title = %v, want 'Test MR'", mr["title"])
	}
	if mr["state"] != "opened" {
		t.Errorf("state = %v, want 'opened'", mr["state"])
	}
	if mr["source_branch"] != "feature-branch" {
		t.Errorf("source_branch = %v, want 'feature-branch'", mr["source_branch"])
	}
}

func TestMockIssue(t *testing.T) {
	issue := MockIssue(10, "Bug Report", "opened")
	if issue["id"] != 10 {
		t.Errorf("id = %v, want 10", issue["id"])
	}
	if issue["title"] != "Bug Report" {
		t.Errorf("title = %v, want 'Bug Report'", issue["title"])
	}
	if issue["state"] != "opened" {
		t.Errorf("state = %v, want 'opened'", issue["state"])
	}
}

func TestMockPipeline(t *testing.T) {
	pipeline := MockPipeline(5, "main", "success")
	if pipeline["id"] != 5 {
		t.Errorf("id = %v, want 5", pipeline["id"])
	}
	if pipeline["ref"] != "main" {
		t.Errorf("ref = %v, want 'main'", pipeline["ref"])
	}
	if pipeline["status"] != "success" {
		t.Errorf("status = %v, want 'success'", pipeline["status"])
	}
}

func TestMockProject(t *testing.T) {
	project := MockProject(1, "My Project", "my-project")
	if project["id"] != 1 {
		t.Errorf("id = %v, want 1", project["id"])
	}
	if project["name"] != "My Project" {
		t.Errorf("name = %v, want 'My Project'", project["name"])
	}
	if project["path"] != "my-project" {
		t.Errorf("path = %v, want 'my-project'", project["path"])
	}
	if project["path_with_namespace"] != "test-owner/my-project" {
		t.Errorf("path_with_namespace = %v, want 'test-owner/my-project'", project["path_with_namespace"])
	}
}

func TestMockUser(t *testing.T) {
	user := MockUser(1, "johndoe", "John Doe")
	if user["id"] != 1 {
		t.Errorf("id = %v, want 1", user["id"])
	}
	if user["username"] != "johndoe" {
		t.Errorf("username = %v, want 'johndoe'", user["username"])
	}
	if user["email"] != "johndoe@example.com" {
		t.Errorf("email = %v, want 'johndoe@example.com'", user["email"])
	}
}

func TestMockRelease(t *testing.T) {
	release := MockRelease("v1.0.0", "Version 1.0.0", "First release")
	if release["tag_name"] != "v1.0.0" {
		t.Errorf("tag_name = %v, want 'v1.0.0'", release["tag_name"])
	}
	if release["name"] != "Version 1.0.0" {
		t.Errorf("name = %v, want 'Version 1.0.0'", release["name"])
	}
}

func TestMockLabel(t *testing.T) {
	label := MockLabel(1, "bug", "#ff0000")
	if label["id"] != 1 {
		t.Errorf("id = %v, want 1", label["id"])
	}
	if label["name"] != "bug" {
		t.Errorf("name = %v, want 'bug'", label["name"])
	}
	if label["color"] != "#ff0000" {
		t.Errorf("color = %v, want '#ff0000'", label["color"])
	}
}

func TestMockSnippet(t *testing.T) {
	snippet := MockSnippet(1, "My Snippet", "example.go")
	if snippet["id"] != 1 {
		t.Errorf("id = %v, want 1", snippet["id"])
	}
	if snippet["title"] != "My Snippet" {
		t.Errorf("title = %v, want 'My Snippet'", snippet["title"])
	}
	if snippet["file_name"] != "example.go" {
		t.Errorf("file_name = %v, want 'example.go'", snippet["file_name"])
	}
}

// --- RouterMux tests ---

func TestRouterMux(t *testing.T) {
	mux := NewRouterMux()

	mux.HandleFunc("/api/v4/projects", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, []map[string]string{{"name": "test"}})
	})
	mux.HandleFunc("/api/v4/user", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, map[string]string{"username": "admin"})
	})

	t.Run("matching route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v4/projects", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("no matching route returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v4/unknown", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

// --- MockGitLabServer tests ---

func TestMockGitLabServer(t *testing.T) {
	srv := MockGitLabServer(t, "test-gitlab.example.com", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if srv == nil {
		t.Fatal("MockGitLabServer returned nil")
	}
}

// --- InterceptTransport tests ---

func TestInterceptTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("intercepted"))
	}))
	defer srv.Close()

	InterceptTransport(t, "intercept-test.example.com", srv)

	resp, err := http.Get("http://intercept-test.example.com/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "intercepted" {
		t.Errorf("body = %q, want %q", string(body), "intercepted")
	}
}

// --- FailTransport tests ---

func TestFailTransport(t *testing.T) {
	FailTransport(t, "fail-test.example.com")

	_, err := http.Get("https://fail-test.example.com/test")
	if err == nil {
		t.Fatal("expected error for blocked host")
	}
	if !strings.Contains(err.Error(), "mock: connection refused") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Fixture variable tests ---

func TestFixtures(t *testing.T) {
	// Verify fixture variables are properly initialized and have expected fields
	tests := []struct {
		name    string
		fixture map[string]interface{}
		key     string
		want    interface{}
	}{
		{"FixtureMROpen state", FixtureMROpen, "state", "opened"},
		{"FixtureMRMerged state", FixtureMRMerged, "state", "merged"},
		{"FixtureMRClosed state", FixtureMRClosed, "state", "closed"},
		{"FixtureIssueOpen state", FixtureIssueOpen, "state", "opened"},
		{"FixtureIssueClosed state", FixtureIssueClosed, "state", "closed"},
		{"FixturePipelineSuccess status", FixturePipelineSuccess, "status", "success"},
		{"FixturePipelineFailed status", FixturePipelineFailed, "status", "failed"},
		{"FixturePipelineRunning status", FixturePipelineRunning, "status", "running"},
		{"FixturePipelinePending status", FixturePipelinePending, "status", "pending"},
		{"FixtureProject visibility", FixtureProject, "visibility", "public"},
		{"FixtureProjectPrivate visibility", FixtureProjectPrivate, "visibility", "private"},
		{"FixtureUser username", FixtureUser, "username", "test-user"},
		{"FixtureUserAdmin is_admin", FixtureUserAdmin, "is_admin", true},
		{"FixtureRelease tag_name", FixtureRelease, "tag_name", "v1.0.0"},
		{"FixtureLabelBug name", FixtureLabelBug, "name", "bug"},
		{"FixtureLabelFeature name", FixtureLabelFeature, "name", "feature"},
		{"FixtureLabelPriority name", FixtureLabelPriority, "name", "priority::high"},
		{"FixtureSnippet title", FixtureSnippet, "title", "Example Go Function"},
		{"FixtureSnippetPrivate visibility", FixtureSnippetPrivate, "visibility", "private"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.fixture[tt.key]
			if !ok {
				t.Fatalf("key %q not found in fixture", tt.key)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Integration helper tests ---

func TestSetupAuthContext(t *testing.T) {
	tf := NewTestFactory(t)
	SetupAuthContext(t, tf, "custom-gitlab.com", "custom-token")

	if tf.Config.DefaultHost != "custom-gitlab.com" {
		t.Errorf("DefaultHost = %q, want %q", tf.Config.DefaultHost, "custom-gitlab.com")
	}
	if tf.Remote.Host != "custom-gitlab.com" {
		t.Errorf("Remote.Host = %q, want %q", tf.Remote.Host, "custom-gitlab.com")
	}
}

func TestSetupProjectContext(t *testing.T) {
	tf := NewTestFactory(t)
	SetupProjectContext(t, tf, "gitlab.example.com", "mygroup", "myproject")

	if tf.Remote.Host != "gitlab.example.com" {
		t.Errorf("Host = %q, want %q", tf.Remote.Host, "gitlab.example.com")
	}
	if tf.Remote.Owner != "mygroup" {
		t.Errorf("Owner = %q, want %q", tf.Remote.Owner, "mygroup")
	}
	if tf.Remote.Repo != "myproject" {
		t.Errorf("Repo = %q, want %q", tf.Remote.Repo, "myproject")
	}

	// Verify Remote func returns updated remote
	remote, err := tf.Factory.Remote()
	if err != nil {
		t.Fatalf("Remote() error: %v", err)
	}
	if remote.Owner != "mygroup" {
		t.Errorf("Factory.Remote().Owner = %q, want %q", remote.Owner, "mygroup")
	}
}

func TestNewMockAPIRouter(t *testing.T) {
	router := NewMockAPIRouter(t)
	if router == nil {
		t.Fatal("NewMockAPIRouter returned nil")
	}
}

func TestMockAPIRouter_ExactMatch(t *testing.T) {
	router := NewMockAPIRouter(t)
	router.Register("GET", "/api/v4/user", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, map[string]string{"username": "admin"})
	})

	req := httptest.NewRequest("GET", "/api/v4/user", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMockAPIRouter_PatternMatch(t *testing.T) {
	router := NewMockAPIRouter(t)
	router.RegisterWithPattern("GET", "/api/v4/projects/:id/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, []interface{}{})
	})

	req := httptest.NewRequest("GET", "/api/v4/projects/123/merge_requests", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMockAPIRouter_NoMatch(t *testing.T) {
	router := NewMockAPIRouter(t)

	req := httptest.NewRequest("GET", "/api/v4/nonexistent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/api/v4/user", "/api/v4/user", true},
		{"/api/v4/projects/:id", "/api/v4/projects/123", true},
		{"/api/v4/projects/:id/merge_requests/:iid", "/api/v4/projects/1/merge_requests/42", true},
		{"/api/v4/projects/:id", "/api/v4/projects/123/extra", false},
		{"/api/v4/projects/:id/issues", "/api/v4/projects/123/mrs", false},
		{"/api/v4", "/api/v4/user", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.path, func(t *testing.T) {
			got := matchPath(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

// --- JSON assertion tests ---

func TestAssertJSONOutput_Object(t *testing.T) {
	jsonStr := `{"name": "test", "count": 5}`
	expected := map[string]interface{}{
		"name":  "test",
		"count": 5,
	}
	// Should not panic
	AssertJSONOutput(t, jsonStr, expected)
}

func TestAssertJSONOutput_Array(t *testing.T) {
	jsonStr := `["a", "b", "c"]`
	expected := []interface{}{"a", "b", "c"}
	AssertJSONOutput(t, jsonStr, expected)
}

func TestAssertJSONContains(t *testing.T) {
	jsonStr := `{"name": "test", "count": 5, "active": true}`
	AssertJSONContains(t, jsonStr, map[string]interface{}{
		"name":   "test",
		"active": true,
	})
}

func TestAssertJSONArray(t *testing.T) {
	jsonStr := `[1, 2, 3]`
	result := AssertJSONArray(t, jsonStr, 3)
	if len(result) != 3 {
		t.Errorf("got %d elements, want 3", len(result))
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		want     bool
	}{
		{"nil nil", nil, nil, true},
		{"nil non-nil", nil, "x", false},
		{"string match", "hello", "hello", true},
		{"string mismatch", "hello", "world", false},
		{"float64 match", float64(42), float64(42), true},
		{"int to float64", float64(42), 42, true},
		{"nested map", map[string]interface{}{"a": "b"}, map[string]interface{}{"a": "b"}, true},
		{"nested array", []interface{}{"a", "b"}, []interface{}{"a", "b"}, true},
		{"array length mismatch", []interface{}{"a"}, []interface{}{"a", "b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValues(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("compareValues(%v, %v) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

// Ensure unused imports are handled
var _ = bytes.NewBuffer

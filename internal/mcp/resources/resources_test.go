package resources

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Unit tests for helper functions ---

func TestExtractRepoFromURI(t *testing.T) {
	tests := []struct {
		uri     string
		want    string
		wantErr bool
	}{
		{"gitlab:///owner/project/README.md", "owner/project", false},
		{"gitlab:///my-org/my-repo/.gitlab-ci.yml", "my-org/my-repo", false},
		{"http:///owner/project/file", "", true},  // wrong scheme
		{"gitlab:///onlyone", "", true},            // too few path parts
		{"://invalid", "", true},                   // invalid URI
	}

	for _, tt := range tests {
		got, err := extractRepoFromURI(tt.uri)
		if (err != nil) != tt.wantErr {
			t.Errorf("extractRepoFromURI(%q): err=%v, wantErr=%v", tt.uri, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("extractRepoFromURI(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}

func TestExtractRepoAndMRFromURI(t *testing.T) {
	tests := []struct {
		uri     string
		repo    string
		mr      int64
		wantErr bool
	}{
		{"gitlab:///owner/project/mr/42/diff", "owner/project", 42, false},
		{"gitlab:///org/repo/mr/1/diff", "org/repo", 1, false},
		{"gitlab:///owner/project/mr/abc/diff", "", 0, true},  // non-numeric MR
		{"gitlab:///owner/project/issue/42/diff", "", 0, true}, // wrong path segment
		{"gitlab:///owner/project/mr", "", 0, true},            // too short
		{"http:///owner/project/mr/42/diff", "", 0, true},      // wrong scheme
	}

	for _, tt := range tests {
		repo, mr, err := extractRepoAndMRFromURI(tt.uri)
		if (err != nil) != tt.wantErr {
			t.Errorf("extractRepoAndMRFromURI(%q): err=%v, wantErr=%v", tt.uri, err, tt.wantErr)
			continue
		}
		if repo != tt.repo {
			t.Errorf("extractRepoAndMRFromURI(%q) repo=%q, want %q", tt.uri, repo, tt.repo)
		}
		if mr != tt.mr {
			t.Errorf("extractRepoAndMRFromURI(%q) mr=%d, want %d", tt.uri, mr, tt.mr)
		}
	}
}

func TestExtractRepoAndJobFromURI(t *testing.T) {
	tests := []struct {
		uri     string
		repo    string
		job     int64
		wantErr bool
	}{
		{"gitlab:///owner/project/pipeline/100/job/500/log", "owner/project", 500, false},
		{"gitlab:///org/repo/pipeline/1/job/2/log", "org/repo", 2, false},
		{"gitlab:///owner/project/pipeline/100/job/abc/log", "", 0, true},          // non-numeric job
		{"gitlab:///owner/project/build/100/job/500/log", "", 0, true},             // wrong path segment
		{"gitlab:///owner/project/pipeline/100/task/500/log", "", 0, true},         // wrong path segment
		{"gitlab:///owner/project/pipeline/100/job", "", 0, true},                  // too short
		{"http:///owner/project/pipeline/100/job/500/log", "", 0, true},            // wrong scheme
	}

	for _, tt := range tests {
		repo, job, err := extractRepoAndJobFromURI(tt.uri)
		if (err != nil) != tt.wantErr {
			t.Errorf("extractRepoAndJobFromURI(%q): err=%v, wantErr=%v", tt.uri, err, tt.wantErr)
			continue
		}
		if repo != tt.repo {
			t.Errorf("extractRepoAndJobFromURI(%q) repo=%q, want %q", tt.uri, repo, tt.repo)
		}
		if job != tt.job {
			t.Errorf("extractRepoAndJobFromURI(%q) job=%d, want %d", tt.uri, job, tt.job)
		}
	}
}

func TestReadLog(t *testing.T) {
	t.Run("normal content", func(t *testing.T) {
		r := strings.NewReader("hello world")
		got, err := readLog(r)
		if err != nil {
			t.Fatal(err)
		}
		if got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		r := strings.NewReader("")
		got, err := readLog(r)
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("truncated at 1 MiB", func(t *testing.T) {
		data := strings.Repeat("x", 1024*1024+100)
		r := strings.NewReader(data)
		got, err := readLog(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(got, "\n[log truncated at 1 MiB]") {
			t.Error("expected truncation notice at the end")
		}
		// The content before the notice should be exactly 1 MiB
		withoutNotice := strings.TrimSuffix(got, "\n[log truncated at 1 MiB]")
		if len(withoutNotice) != 1024*1024 {
			t.Errorf("expected 1 MiB of content, got %d bytes", len(withoutNotice))
		}
	})

	t.Run("exactly 1 MiB", func(t *testing.T) {
		data := strings.Repeat("x", 1024*1024)
		r := strings.NewReader(data)
		got, err := readLog(r)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "[log truncated") {
			t.Error("should not truncate at exactly 1 MiB")
		}
		if len(got) != 1024*1024 {
			t.Errorf("expected 1 MiB, got %d bytes", len(got))
		}
	})

	t.Run("reader error", func(t *testing.T) {
		r := &errReader{err: io.ErrUnexpectedEOF}
		_, err := readLog(r)
		if err == nil {
			t.Error("expected error from reader")
		}
	})
}

// errReader is a reader that always returns an error.
type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestBase64Decoding(t *testing.T) {
	// Test that the base64 decoding logic used in resources works correctly
	original := "# Hello World\n\nThis is a test README."
	encoded := base64.StdEncoding.EncodeToString([]byte(original))
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != original {
		t.Errorf("got %q, want %q", string(decoded), original)
	}
}

// --- Integration tests for resource registration and handlers ---

func setupResourceServer(t *testing.T, mux *cmdtest.RouterMux) *mcp.ClientSession {
	t.Helper()

	tf := cmdtest.NewTestFactory(t)
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", mux.ServeHTTP)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-mcp",
		Version: "0.0.1",
	}, nil)

	RegisterResources(server, tf.Factory)

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

func TestListResourceTemplates(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupResourceServer(t, mux)

	result, err := cs.ListResourceTemplates(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make(map[string]bool)
	for _, rt := range result.ResourceTemplates {
		names[rt.Name] = true
	}

	expected := []string{"readme", "ci-config", "mr-diff", "pipeline-job-log"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected resource template %q to be registered", name)
		}
	}
}

func TestResourceTemplateProperties(t *testing.T) {
	mux := cmdtest.NewRouterMux()
	cs := setupResourceServer(t, mux)

	result, err := cs.ListResourceTemplates(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Build a map for easy lookup
	templates := make(map[string]*mcp.ResourceTemplate)
	for _, rt := range result.ResourceTemplates {
		templates[rt.Name] = rt
	}

	tests := []struct {
		name        string
		uriTemplate string
		mimeType    string
	}{
		{"readme", "gitlab:///{repo}/README.md", "text/markdown"},
		{"ci-config", "gitlab:///{repo}/.gitlab-ci.yml", "text/yaml"},
		{"mr-diff", "gitlab:///{repo}/mr/{mr}/diff", "text/plain"},
		{"pipeline-job-log", "gitlab:///{repo}/pipeline/{pipeline}/job/{job}/log", "text/plain"},
	}

	for _, tt := range tests {
		tmpl, ok := templates[tt.name]
		if !ok {
			t.Errorf("template %q not found", tt.name)
			continue
		}
		if tmpl.URITemplate != tt.uriTemplate {
			t.Errorf("%s: URITemplate=%q, want %q", tt.name, tmpl.URITemplate, tt.uriTemplate)
		}
		if tmpl.MIMEType != tt.mimeType {
			t.Errorf("%s: MIMEType=%q, want %q", tt.name, tmpl.MIMEType, tt.mimeType)
		}
		if tmpl.Description == "" {
			t.Errorf("%s: expected non-empty description", tt.name)
		}
	}
}

// Note: ReadResource integration tests for the actual resource handlers are not
// included because the URI templates use {repo} (RFC 6570 single-segment match)
// but the handlers expect OWNER/PROJECT (two segments) in the URI path. This is
// a design limitation that would need to be addressed in the production code
// (e.g., by using {+repo} for reserved expansion or separate {owner}/{project}).

func TestResolveClientAndProject(t *testing.T) {
	t.Run("empty repo uses factory defaults", func(t *testing.T) {
		tf := cmdtest.NewTestFactory(t)
		_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		client, project, err := resolveClientAndProject(tf.Factory, "")
		if err != nil {
			t.Fatal(err)
		}
		if client == nil {
			t.Error("expected non-nil client")
		}
		if project != "test-owner/test-repo" {
			t.Errorf("got project %q, want 'test-owner/test-repo'", project)
		}
	})

	t.Run("explicit repo", func(t *testing.T) {
		tf := cmdtest.NewTestFactory(t)
		_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		client, project, err := resolveClientAndProject(tf.Factory, "other-owner/other-repo")
		if err != nil {
			t.Fatal(err)
		}
		if client == nil {
			t.Error("expected non-nil client")
		}
		if project != "other-owner/other-repo" {
			t.Errorf("got project %q, want 'other-owner/other-repo'", project)
		}
	})
}

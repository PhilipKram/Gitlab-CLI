package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewCILintCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newCILintCmd(f)

	if !strings.HasPrefix(cmd.Use, "lint") {
		t.Errorf("expected Use to start with 'lint', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}

	// Flags we advertise must exist.
	for _, name := range []string{"ref", "dry-run", "include-jobs", "format", "json"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag", name)
		}
	}
}

// TestCILint_ProjectMode exercises the no-arg path that lints the committed
// .gitlab-ci.yml via GET /projects/:id/ci/lint.
func TestCILint_ProjectMode(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ci/lint") && r.Method == http.MethodGet {
			cmdtest.JSONResponse(w, 200, map[string]any{
				"valid":    true,
				"errors":   []string{},
				"warnings": []string{},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newCILintCmd(f.Factory)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(f.IO.Out.String(), "CI configuration is valid") {
		t.Errorf("expected valid confirmation in output, got: %s", f.IO.Out.String())
	}
}

// TestCILint_InvalidReturnsError proves an invalid config produces a non-zero
// exit (error return) so CI pipelines using `glab pipeline lint` fail as
// intended.
func TestCILint_InvalidReturnsError(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ci/lint") {
			cmdtest.JSONResponse(w, 200, map[string]any{
				"valid":  false,
				"errors": []string{"jobs config should contain at least one visible job"},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newCILintCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when config is invalid")
	}
	if !strings.Contains(f.IO.Out.String(), "CI configuration is invalid") {
		t.Errorf("expected invalid notice in output, got: %s", f.IO.Out.String())
	}
}

// TestCILint_FileMode exercises the POST /projects/:id/ci/lint path that
// validates an arbitrary YAML file instead of the committed one.
func TestCILint_FileMode(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), ".gitlab-ci.yml")
	if err := os.WriteFile(tmpFile, []byte("stages:\n  - test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ci/lint") && r.Method == http.MethodPost {
			cmdtest.JSONResponse(w, 200, map[string]any{
				"valid":    true,
				"errors":   []string{},
				"warnings": []string{"you should add a test job"},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newCILintCmd(f.Factory)
	cmd.SetArgs([]string{tmpFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := f.IO.Out.String()
	if !strings.Contains(out, "CI configuration is valid") {
		t.Errorf("expected valid confirmation, got: %s", out)
	}
	if !strings.Contains(out, "you should add a test job") {
		t.Errorf("expected warnings in output, got: %s", out)
	}
}

// TestCILint_FileNotFound covers the file-read error path.
func TestCILint_FileNotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newCILintCmd(f.Factory)
	cmd.SetArgs([]string{"/nonexistent/path/gitlab-ci.yml"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading file") {
		t.Errorf("expected file-read error, got: %v", err)
	}
}

// TestCILint_JSONOutput covers the --format json path.
func TestCILint_JSONOutput(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ci/lint") {
			cmdtest.JSONResponse(w, 200, map[string]any{
				"valid":  true,
				"errors": []string{},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newCILintCmd(f.Factory)
	cmd.SetArgs([]string{"--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(f.IO.Out.String(), `"valid": true`) {
		t.Errorf("expected JSON output, got: %s", f.IO.Out.String())
	}
}

// TestCILint_APIError covers the API-error path (4xx / 5xx from GitLab).
func TestCILint_APIError(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newCILintCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error on API failure")
	}
}

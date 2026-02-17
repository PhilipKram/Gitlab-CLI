package cmdutil

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/formatter"
	"github.com/PhilipKram/gitlab-cli/internal/git"
	"github.com/PhilipKram/gitlab-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()

	if f == nil {
		t.Fatal("NewFactory returned nil")
	}
	if f.IOStreams == nil {
		t.Error("IOStreams should not be nil")
	}
	if f.Config == nil {
		t.Error("Config func should not be nil")
	}
	if f.Client == nil {
		t.Error("Client func should not be nil")
	}
	if f.Remote == nil {
		t.Error("Remote func should not be nil")
	}
}

func TestSetRepoOverride(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		wantHost     string
		wantPath     string
		wantOverride string
	}{
		{
			name:         "valid host/path",
			repo:         "gitlab.com/owner/repo",
			wantHost:     "gitlab.com",
			wantPath:     "owner/repo",
			wantOverride: "gitlab.com/owner/repo",
		},
		{
			name:         "custom host",
			repo:         "gitlab.example.com/group/subgroup/project",
			wantHost:     "gitlab.example.com",
			wantPath:     "group/subgroup/project",
			wantOverride: "gitlab.example.com/group/subgroup/project",
		},
		{
			name:         "no slash - single segment",
			repo:         "noslash",
			wantHost:     "",
			wantPath:     "",
			wantOverride: "noslash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{}
			f.SetRepoOverride(tt.repo)

			if f.repoOverride != tt.wantOverride {
				t.Errorf("repoOverride = %q, want %q", f.repoOverride, tt.wantOverride)
			}
			if f.overrideHost != tt.wantHost {
				t.Errorf("overrideHost = %q, want %q", f.overrideHost, tt.wantHost)
			}
			if f.overridePath != tt.wantPath {
				t.Errorf("overridePath = %q, want %q", f.overridePath, tt.wantPath)
			}
		})
	}
}

func TestFullProjectPath_WithOverride(t *testing.T) {
	f := &Factory{}
	f.SetRepoOverride("gitlab.com/myowner/myrepo")

	path, err := f.FullProjectPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "myowner/myrepo" {
		t.Errorf("got %q, want %q", path, "myowner/myrepo")
	}
}

func TestFullProjectPath_FromRemote(t *testing.T) {
	f := &Factory{}
	f.Remote = func() (*git.Remote, error) {
		return &git.Remote{
			Name:  "origin",
			Host:  "gitlab.com",
			Owner: "remoteowner",
			Repo:  "remoterepo",
		}, nil
	}

	path, err := f.FullProjectPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "remoteowner/remoterepo" {
		t.Errorf("got %q, want %q", path, "remoteowner/remoterepo")
	}
}

func TestFullProjectPath_EmptyOwnerOrRepo(t *testing.T) {
	f := &Factory{}
	f.Remote = func() (*git.Remote, error) {
		return &git.Remote{
			Name:  "origin",
			Host:  "gitlab.com",
			Owner: "",
			Repo:  "",
		}, nil
	}

	_, err := f.FullProjectPath()
	if err == nil {
		t.Fatal("expected error for empty owner/repo")
	}
}

func TestFullProjectPath_RemoteError(t *testing.T) {
	f := &Factory{}
	f.Remote = func() (*git.Remote, error) {
		return nil, fmt.Errorf("no git remote")
	}

	_, err := f.FullProjectPath()
	if err == nil {
		t.Fatal("expected error when remote fails")
	}
}

func TestSetOutputFormat(t *testing.T) {
	f := &Factory{}

	f.SetOutputFormat("json")
	if f.GetOutputFormat() != "json" {
		t.Errorf("got %q, want %q", f.GetOutputFormat(), "json")
	}

	f.SetOutputFormat("table")
	if f.GetOutputFormat() != "table" {
		t.Errorf("got %q, want %q", f.GetOutputFormat(), "table")
	}
}

func TestIsJSONFormat(t *testing.T) {
	f := &Factory{}

	f.SetOutputFormat("json")
	if !f.IsJSONFormat() {
		t.Error("expected IsJSONFormat to be true for json format")
	}

	f.SetOutputFormat("table")
	if f.IsJSONFormat() {
		t.Error("expected IsJSONFormat to be false for table format")
	}

	f.SetOutputFormat("")
	if f.IsJSONFormat() {
		t.Error("expected IsJSONFormat to be false for empty format")
	}
}

func TestAddFormatFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var format string
	var jsonFlag bool

	AddFormatFlag(cmd, &format, &jsonFlag)

	ff := cmd.Flags().Lookup("format")
	if ff == nil {
		t.Fatal("format flag not registered")
	}
	if ff.Shorthand != "f" {
		t.Errorf("format shorthand = %q, want %q", ff.Shorthand, "f")
	}

	jf := cmd.Flags().Lookup("json")
	if jf == nil {
		t.Fatal("json flag not registered")
	}
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		jsonFlag  bool
		want      formatter.OutputFormat
		wantErr   bool
		wantStderr string
	}{
		{
			name:   "json format",
			format: "json",
			want:   formatter.JSONFormat,
		},
		{
			name:   "table format",
			format: "table",
			want:   formatter.TableFormat,
		},
		{
			name:   "plain format",
			format: "plain",
			want:   formatter.PlainFormat,
		},
		{
			name:    "invalid format",
			format:  "xml",
			wantErr: true,
		},
		{
			name:       "json flag overrides format",
			format:     "",
			jsonFlag:   true,
			want:       formatter.JSONFormat,
			wantStderr: "deprecated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errBuf bytes.Buffer
			f := &Factory{
				IOStreams: &iostreams.IOStreams{
					ErrOut: &errBuf,
				},
			}

			got, err := f.ResolveFormat(tt.format, tt.jsonFlag)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if tt.wantStderr != "" {
				stderr := errBuf.String()
				if !bytes.Contains([]byte(stderr), []byte(tt.wantStderr)) {
					t.Errorf("stderr %q does not contain %q", stderr, tt.wantStderr)
				}
			}
		})
	}
}

func TestFormatAndPrint(t *testing.T) {
	var outBuf bytes.Buffer
	f := &Factory{
		IOStreams: &iostreams.IOStreams{
			Out:    &outBuf,
			ErrOut: &bytes.Buffer{},
		},
	}

	data := map[string]string{"key": "value"}
	err := f.FormatAndPrint(data, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := outBuf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatAndPrint_InvalidFormat(t *testing.T) {
	f := &Factory{
		IOStreams: &iostreams.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		},
	}

	err := f.FormatAndPrint("data", "invalid", false)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestFormatAndStream(t *testing.T) {
	var outBuf bytes.Buffer
	f := &Factory{
		IOStreams: &iostreams.IOStreams{
			Out:    &outBuf,
			ErrOut: &bytes.Buffer{},
		},
	}

	results := make(chan api.Result[string], 3)
	results <- api.Result[string]{Item: "item1"}
	results <- api.Result[string]{Item: "item2"}
	close(results)

	err := FormatAndStream(f, results, formatter.JSONFormat, 0, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := outBuf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatAndStream_WithLimit(t *testing.T) {
	var outBuf bytes.Buffer
	f := &Factory{
		IOStreams: &iostreams.IOStreams{
			Out:    &outBuf,
			ErrOut: &bytes.Buffer{},
		},
	}

	results := make(chan api.Result[string], 5)
	results <- api.Result[string]{Item: "item1"}
	results <- api.Result[string]{Item: "item2"}
	results <- api.Result[string]{Item: "item3"}
	close(results)

	err := FormatAndStream(f, results, formatter.JSONFormat, 1, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatAndStream_WithError(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	f := &Factory{
		IOStreams: &iostreams.IOStreams{
			Out:    &outBuf,
			ErrOut: &errBuf,
		},
	}

	results := make(chan api.Result[string], 2)
	results <- api.Result[string]{Error: fmt.Errorf("fetch failed")}
	close(results)

	_ = FormatAndStream(f, results, formatter.JSONFormat, 0, "items")

	if !bytes.Contains(errBuf.Bytes(), []byte("fetch failed")) {
		t.Errorf("expected error message in stderr, got %q", errBuf.String())
	}
}

func TestFormatAndStream_InvalidFormat(t *testing.T) {
	f := &Factory{
		IOStreams: &iostreams.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		},
	}

	results := make(chan api.Result[string])
	close(results)

	err := FormatAndStream(f, results, formatter.OutputFormat("invalid"), 0, "items")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestGetHostVersion_NoConfig(t *testing.T) {
	f := &Factory{}
	// GetHostVersion should return empty string gracefully when no config exists
	version := f.GetHostVersion("nonexistent.host")
	if version != "" {
		t.Errorf("expected empty string, got %q", version)
	}
}

func TestNewFactory_ClientFallback(t *testing.T) {
	// Test that NewFactory creates a factory with Client that handles fallback logic.
	// We just verify the factory is created; actual client creation depends on auth state.
	f := NewFactory()
	if f.Client == nil {
		t.Error("Client func should not be nil")
	}
}


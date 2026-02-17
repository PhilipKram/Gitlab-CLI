package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

// COMMAND STRUCTURE TESTS

func TestNewRegistryCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewRegistryCmd(f)

	if cmd.Use != "registry <command>" {
		t.Errorf("expected Use to be 'registry <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage container registries" {
		t.Errorf("expected Short to be 'Manage container registries', got %q", cmd.Short)
	}
}

func TestRegistryCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewRegistryCmd(f)

	expectedSubcommands := []string{
		"list",
		"tags",
		"view",
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

// FLAG TESTS

func TestRegistryListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRegistryListCmd(f)

	expectedFlags := []string{
		"project",
		"limit",
		"format",
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

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be table, got %q", formatFlag.DefValue)
	}

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestRegistryTagsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRegistryTagsCmd(f)

	expectedFlags := []string{
		"project",
		"limit",
		"format",
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

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be table, got %q", formatFlag.DefValue)
	}

	if cmd.Use != "tags <repository-id>" {
		t.Errorf("expected Use to be 'tags <repository-id>', got %q", cmd.Use)
	}
}

func TestRegistryViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRegistryViewCmd(f)

	expectedFlags := []string{
		"tag",
		"project",
		"format",
		"json",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be table, got %q", formatFlag.DefValue)
	}

	if cmd.Use != "view <repository-id>" {
		t.Errorf("expected Use to be 'view <repository-id>', got %q", cmd.Use)
	}
}

func TestRegistryDeleteCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newRegistryDeleteCmd(f)

	expectedFlags := []string{
		"tag",
		"older-than",
		"yes",
		"project",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "delete <repository-id>" {
		t.Errorf("expected Use to be 'delete <repository-id>', got %q", cmd.Use)
	}
}

// EXECUTION AND ERROR TESTS

func TestRegistryList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/registry/repositories") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				fixtureRegistryRepository,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "my-app") {
		t.Errorf("expected output to contain repository name, got: %s", output)
	}
}

func TestRegistryList_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryListCmd(f.Factory)

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected auth error")
	}
}

func TestRegistryList_NoRepositories(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.ErrString()
	if !strings.Contains(output, "No container repositories found") {
		t.Errorf("expected 'No container repositories found' message, got: %s", output)
	}
}

func TestRegistryTags_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tags") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				fixtureRegistryTag,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryTagsCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "latest") {
		t.Errorf("expected output to contain tag name, got: %s", output)
	}
}

func TestRegistryTags_InvalidRepositoryID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryTagsCmd(f.Factory)
	cmd.SetArgs([]string{"invalid-id"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid repository ID")
	}

	if !strings.Contains(err.Error(), "invalid repository ID") {
		t.Errorf("expected invalid repository ID error, got: %v", err)
	}
}

func TestRegistryTags_NoTags(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryTagsCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.ErrString()
	if !strings.Contains(output, "No tags found") {
		t.Errorf("expected 'No tags found' message, got: %s", output)
	}
}

func TestRegistryView_RepositorySuccess(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/registry/repositories/123") {
			cmdtest.JSONResponse(w, 200, fixtureRegistryRepository)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryViewCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "my-app") {
		t.Errorf("expected output to contain repository name, got: %s", output)
	}
}

func TestRegistryView_TagSuccess(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tags/latest") {
			cmdtest.JSONResponse(w, 200, fixtureRegistryTagDetail)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryViewCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--tag", "latest"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "latest") {
		t.Errorf("expected output to contain tag name, got: %s", output)
	}
}

func TestRegistryView_NotFound(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryViewCmd(f.Factory)
	cmd.SetArgs([]string{"999"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryDelete_MissingTagFlag(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --tag flag")
	}

	if !strings.Contains(err.Error(), "--tag flag is required") {
		t.Errorf("expected --tag flag required error, got: %v", err)
	}
}

func TestRegistryDelete_InvalidRepositoryID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"invalid-id", "--tag", "latest"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid repository ID")
	}

	if !strings.Contains(err.Error(), "invalid repository ID") {
		t.Errorf("expected invalid repository ID error, got: %v", err)
	}
}

func TestRegistryDelete_SuccessWithTag(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/tags/v1.0.0") {
			w.WriteHeader(200)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--tag", "v1.0.0", "--yes"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Deleted tag") {
		t.Errorf("expected 'Deleted tag' message, got: %s", output)
	}
}

func TestRegistryDelete_TagNotFound(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"123", "--tag", "nonexistent", "--yes"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not found tag")
	}
}

func TestRegistryDelete_MissingArgs(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newRegistryDeleteCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing repository ID")
	}
}

// HELPER FUNCTION TESTS

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
		expectDays  int
	}{
		{"30d", false, 30},
		{"7d", false, 7},
		{"1d", false, 1},
		{"24h", false, 1},
		{"invalid", true, 0},
		{"", true, 0},
		{"d", true, 0},
		{"30x", true, 0},
	}

	for _, tt := range tests {
		duration, err := parseDuration(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("parseDuration(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseDuration(%q) unexpected error: %v", tt.input, err)
			}
			expectedDuration := tt.expectDays * 24 * 60 * 60 * 1000000000 // nanoseconds
			if int(duration) != expectedDuration {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, duration, expectedDuration)
			}
		}
	}
}

// FIXTURES FOR REGISTRY TESTS

var fixtureRegistryRepository = map[string]interface{}{
	"id":         123,
	"name":       "my-app",
	"path":       "test-owner/test-repo/my-app",
	"project_id": 400,
	"location":   "registry.gitlab.com/test-owner/test-repo/my-app",
	"created_at": "2024-01-01T00:00:00.000Z",
}

var fixtureRegistryTag = map[string]interface{}{
	"name":       "latest",
	"path":       "test-owner/test-repo/my-app:latest",
	"location":   "registry.gitlab.com/test-owner/test-repo/my-app:latest",
	"created_at": "2024-01-05T10:00:00.000Z",
	"digest":     "sha256:abc123def456",
	"total_size": 123456789,
}

var fixtureRegistryTagDetail = map[string]interface{}{
	"name":       "latest",
	"path":       "test-owner/test-repo/my-app:latest",
	"location":   "registry.gitlab.com/test-owner/test-repo/my-app:latest",
	"created_at": "2024-01-05T10:00:00.000Z",
	"digest":     "sha256:abc123def456",
	"total_size": 123456789,
}

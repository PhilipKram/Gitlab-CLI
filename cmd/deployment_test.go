package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewDeploymentCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewDeploymentCmd(f)

	if cmd.Use != "deployment <command>" {
		t.Errorf("expected Use to be 'deployment <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage deployments" {
		t.Errorf("expected Short to be 'Manage deployments', got %q", cmd.Short)
	}

	expectedAliases := []string{"deploy"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i >= len(cmd.Aliases) || cmd.Aliases[i] != alias {
			t.Errorf("expected alias %q at position %d, got %v", alias, i, cmd.Aliases)
		}
	}
}

func TestDeploymentCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewDeploymentCmd(f)

	expectedSubcommands := []string{
		"list",
		"view",
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

func TestDeploymentListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newDeploymentListCmd(f)

	expectedFlags := []string{
		"status",
		"environment",
		"order-by",
		"sort",
		"limit",
		"format",
		"json",
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

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be table, got %q", formatFlag.DefValue)
	}

	// Verify list has alias "ls"
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestDeploymentViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newDeploymentViewCmd(f)

	expectedFlags := []string{"web", "format", "json"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view [<id>]" {
		t.Errorf("expected Use to be 'view [<id>]', got %q", cmd.Use)
	}

	// Verify default format is table
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be table, got %q", formatFlag.DefValue)
	}
}

func TestDeploymentViewCmd_Use(t *testing.T) {
	f := newTestFactory()
	cmd := newDeploymentViewCmd(f)

	if cmd.Use != "view [<id>]" {
		t.Errorf("expected Use to be 'view [<id>]', got %q", cmd.Use)
	}

	if cmd.Short != "View a deployment" {
		t.Errorf("expected Short to be 'View a deployment', got %q", cmd.Short)
	}
}

func TestDeploymentListCmd_Use(t *testing.T) {
	f := newTestFactory()
	cmd := newDeploymentListCmd(f)

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", cmd.Use)
	}

	if cmd.Short != "List deployments" {
		t.Errorf("expected Short to be 'List deployments', got %q", cmd.Short)
	}
}

// ============================================================================
// parseDeploymentID TESTS
// ============================================================================

func TestParseDeploymentID(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int64
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid id",
			args:    []string{"12345"},
			want:    12345,
			wantErr: false,
		},
		{
			name:    "no args",
			args:    []string{},
			want:    0,
			wantErr: true,
			errMsg:  "deployment ID required",
		},
		{
			name:    "invalid id",
			args:    []string{"abc"},
			want:    0,
			wantErr: true,
			errMsg:  "invalid deployment ID",
		},
		{
			name:    "zero id",
			args:    []string{"0"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "large id",
			args:    []string{"999999999"},
			want:    999999999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDeploymentID(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDeploymentID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseDeploymentID() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
			if got != tt.want {
				t.Errorf("parseDeploymentID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// EXECUTION TESTS
// ============================================================================

var fixtureDeployment = map[string]interface{}{
	"id":          1,
	"iid":         1,
	"status":      "success",
	"ref":         "main",
	"sha":         "abc123",
	"created_at":  "2024-01-01T00:00:00.000Z",
	"updated_at":  "2024-01-01T00:00:00.000Z",
	"environment": map[string]interface{}{"id": 1, "name": "production"},
	"deployable":  nil,
	"user": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
}

func TestDeploymentList_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/deployments") {
			cmdtest.JSONResponse(w, 200, []interface{}{fixtureDeployment})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentListCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeploymentList_Empty(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, []interface{}{})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentListCmd(f.Factory)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No deployments found") {
		t.Errorf("expected 'No deployments found' message, got: %s", errOutput)
	}
}

func TestDeploymentList_WithFilters(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/deployments") {
			// Verify query parameters
			if r.URL.Query().Get("status") != "success" {
				t.Errorf("expected status=success, got %s", r.URL.Query().Get("status"))
			}
			if r.URL.Query().Get("environment") != "production" {
				t.Errorf("expected environment=production, got %s", r.URL.Query().Get("environment"))
			}
			cmdtest.JSONResponse(w, 200, []interface{}{fixtureDeployment})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentListCmd(f.Factory)
	cmd.SetArgs([]string{"--status", "success", "--environment", "production"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeploymentList_Unauthorized(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentListCmd(f.Factory)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestDeploymentView_Success(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/deployments/1") {
			cmdtest.JSONResponse(w, 200, fixtureDeployment)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentViewCmd(f.Factory)
	cmd.SetArgs([]string{"1"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeploymentView_NotFound(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentViewCmd(f.Factory)
	cmd.SetArgs([]string{"9999"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent deployment")
	}
}

func TestDeploymentView_MissingID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentViewCmd(f.Factory)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing deployment ID")
	}
	if !strings.Contains(err.Error(), "deployment ID required") {
		t.Errorf("expected 'deployment ID required' error, got: %v", err)
	}
}

func TestDeploymentView_InvalidID(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentViewCmd(f.Factory)
	cmd.SetArgs([]string{"not-a-number"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid deployment ID")
	}
	if !strings.Contains(err.Error(), "invalid deployment ID") {
		t.Errorf("expected 'invalid deployment ID' error, got: %v", err)
	}
}

func TestDeploymentView_JSONFormat(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/deployments/1") {
			cmdtest.JSONResponse(w, 200, fixtureDeployment)
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newDeploymentViewCmd(f.Factory)
	cmd.SetArgs([]string{"1", "--format", "json"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

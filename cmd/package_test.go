package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewPackageCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewPackageCmd(f)

	if cmd.Use != "package <command>" {
		t.Errorf("expected Use to be 'package <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage package registries" {
		t.Errorf("expected Short to be 'Manage package registries', got %q", cmd.Short)
	}
}

func TestPackageCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewPackageCmd(f)

	expectedSubcommands := []string{
		"list",
		"view",
		"delete",
		"download",
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

func TestPackageListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPackageListCmd(f)

	expectedFlags := []string{
		"limit",
		"format",
		"json",
		"type",
		"group",
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

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", cmd.Use)
	}
}

func TestPackageViewCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPackageViewCmd(f)

	expectedFlags := []string{
		"format",
		"json",
		"group",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "view <package-name>" {
		t.Errorf("expected Use to be 'view <package-name>', got %q", cmd.Use)
	}
}

func TestPackageDeleteCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPackageDeleteCmd(f)

	expectedFlags := []string{
		"version",
		"group",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "delete <package-name>" {
		t.Errorf("expected Use to be 'delete <package-name>', got %q", cmd.Use)
	}
}

func TestPackageDownloadCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPackageDownloadCmd(f)

	expectedFlags := []string{
		"version",
		"output",
		"group",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	if cmd.Use != "download <package-name>" {
		t.Errorf("expected Use to be 'download <package-name>', got %q", cmd.Use)
	}
}

func TestPackageList_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "my-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "my-package") {
		t.Errorf("expected output to contain package name, got: %s", output)
	}
}

func TestPackageView_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "my-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageViewCmd(f.Factory)
	cmd.SetArgs([]string{"my-package"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "my-package") {
		t.Errorf("expected output to contain package name, got: %s", output)
	}
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain package version, got: %s", output)
	}
}

func TestPackageDelete_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/packages/1") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "my-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"my-package"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "deleted") && !strings.Contains(output, "Deleted") {
		t.Errorf("expected output to contain deletion confirmation, got: %s", output)
	}
}

func TestPackageDownload_Success(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/packages/1/package_files") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":        100,
					"file_name": "my-package-1.0.0.tgz",
					"size":      1024,
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "my-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageDownloadCmd(f.Factory)
	cmd.SetArgs([]string{"my-package"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "my-package") {
		t.Errorf("expected output to contain package name, got: %s", output)
	}
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain version, got: %s", output)
	}
}

// Group-level operation tests

func TestPackageList_GroupLevel(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") && strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "group-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"project_id":   123,
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageListCmd(f.Factory)
	cmd.SetArgs([]string{"--group", "mygroup", "--format", "json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "group-package") {
		t.Errorf("expected output to contain group package name, got: %s", output)
	}
}

func TestPackageView_GroupLevel(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/groups/") && strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "group-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"project_id":   123,
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
				map[string]interface{}{
					"id":           2,
					"name":         "group-package",
					"version":      "2.0.0",
					"package_type": "npm",
					"project_id":   123,
					"created_at":   "2024-01-02T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageViewCmd(f.Factory)
	cmd.SetArgs([]string{"group-package", "--group", "mygroup"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "group-package") {
		t.Errorf("expected output to contain package name, got: %s", output)
	}
	if !strings.Contains(output, "Group:") {
		t.Errorf("expected output to show Group label, got: %s", output)
	}
}

func TestPackageDelete_GroupLevel(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/projects/123/packages/1") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.Contains(r.URL.Path, "/groups/") && strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "group-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"project_id":   123,
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"group-package", "--group", "mygroup"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "deleted") && !strings.Contains(output, "Deleted") {
		t.Errorf("expected output to contain deletion confirmation, got: %s", output)
	}
}

func TestPackageDownload_GroupLevel(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects/123/packages/1/package_files") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":        100,
					"file_name": "group-package-1.0.0.tgz",
					"size":      2048,
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/groups/") && strings.Contains(r.URL.Path, "/packages") {
			cmdtest.JSONResponse(w, 200, []interface{}{
				map[string]interface{}{
					"id":           1,
					"name":         "group-package",
					"version":      "1.0.0",
					"package_type": "npm",
					"project_id":   123,
					"created_at":   "2024-01-01T10:00:00.000Z",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newPackageDownloadCmd(f.Factory)
	cmd.SetArgs([]string{"group-package", "--group", "mygroup"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "group-package") {
		t.Errorf("expected output to contain package name, got: %s", output)
	}
}

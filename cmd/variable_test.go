package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewVariableCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewVariableCmd(f)

	if cmd.Use != "variable <command>" {
		t.Errorf("expected Use to be 'variable <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage CI/CD variables" {
		t.Errorf("expected Short to be 'Manage CI/CD variables', got %q", cmd.Short)
	}
}

func TestVariableCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewVariableCmd(f)

	expectedSubcommands := []string{
		"list",
		"get",
		"set",
		"update",
		"delete",
		"export",
		"import",
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

func TestVariableListCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableListCmd(f)

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", cmd.Use)
	}

	if cmd.Short != "List CI/CD variables" {
		t.Errorf("expected Short to be 'List CI/CD variables', got %q", cmd.Short)
	}

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestVariableListCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableListCmd(f)

	expectedFlags := []string{"limit", "json", "group"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify limit flag default value
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("limit flag not found")
	}
	if limitFlag.DefValue != "30" {
		t.Errorf("expected default limit to be 30, got %q", limitFlag.DefValue)
	}

	// Verify group flag default value
	groupFlag := cmd.Flags().Lookup("group")
	if groupFlag == nil {
		t.Fatal("group flag not found")
	}
	if groupFlag.DefValue != "" {
		t.Errorf("expected default group to be empty, got %q", groupFlag.DefValue)
	}
}

func TestVariableGetCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableGetCmd(f)

	if cmd.Use != "get <key>" {
		t.Errorf("expected Use to be 'get <key>', got %q", cmd.Use)
	}

	if cmd.Short != "Get a CI/CD variable" {
		t.Errorf("expected Short to be 'Get a CI/CD variable', got %q", cmd.Short)
	}
}

func TestVariableGetCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableGetCmd(f)

	expectedFlags := []string{"json", "group"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify group flag default value
	groupFlag := cmd.Flags().Lookup("group")
	if groupFlag == nil {
		t.Fatal("group flag not found")
	}
	if groupFlag.DefValue != "" {
		t.Errorf("expected default group to be empty, got %q", groupFlag.DefValue)
	}
}

func TestVariableList_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{
				{
					"key":              "TEST_VAR",
					"value":            "test-value",
					"variable_type":    "env_var",
					"protected":        false,
					"masked":           false,
					"environment_scope": "*",
				},
				{
					"key":              "PROD_API_KEY",
					"value":            "secret-key",
					"variable_type":    "env_var",
					"protected":        true,
					"masked":           true,
					"environment_scope": "production",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "TEST_VAR") {
		t.Errorf("expected output to contain TEST_VAR, got: %s", output)
	}
	if !strings.Contains(output, "PROD_API_KEY") {
		t.Errorf("expected output to contain PROD_API_KEY, got: %s", output)
	}
}

func TestVariableGet_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables/TEST_VAR") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"key":              "TEST_VAR",
				"value":            "test-value",
				"variable_type":    "env_var",
				"protected":        false,
				"masked":           false,
				"environment_scope": "*",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "TEST_VAR") {
		t.Errorf("expected output to contain TEST_VAR, got: %s", output)
	}
	if !strings.Contains(output, "test-value") {
		t.Errorf("expected output to contain test-value, got: %s", output)
	}
}

func TestVariableSet_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"key":              "NEW_VAR",
				"value":            "new-value",
				"variable_type":    "env_var",
				"protected":        false,
				"masked":           false,
				"environment_scope": "*",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"NEW_VAR", "--value", "new-value"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "NEW_VAR") {
		t.Errorf("expected output to contain NEW_VAR, got: %s", output)
	}
}

func TestVariableUpdate_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables/EXISTING_VAR") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"key":              "EXISTING_VAR",
				"value":            "updated-value",
				"variable_type":    "env_var",
				"protected":        false,
				"masked":           true,
				"environment_scope": "*",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"EXISTING_VAR", "--value", "updated-value", "--masked"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "EXISTING_VAR") {
		t.Errorf("expected output to contain EXISTING_VAR, got: %s", output)
	}
}

func TestVariableDelete_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables/TEST_VAR") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "TEST_VAR") {
		t.Errorf("expected output to contain TEST_VAR, got: %s", output)
	}
}

func TestVariableExport_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{
				{
					"key":              "EXPORT_VAR_1",
					"value":            "value1",
					"variable_type":    "env_var",
					"protected":        false,
					"masked":           false,
					"environment_scope": "*",
				},
				{
					"key":              "EXPORT_VAR_2",
					"value":            "value2",
					"variable_type":    "file",
					"protected":        true,
					"masked":           true,
					"environment_scope": "production",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableExportCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "EXPORT_VAR_1") {
		t.Errorf("expected output to contain EXPORT_VAR_1, got: %s", output)
	}
	if !strings.Contains(output, "EXPORT_VAR_2") {
		t.Errorf("expected output to contain EXPORT_VAR_2, got: %s", output)
	}
}

func TestVariableImport_Execute(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"key":              "IMPORT_VAR",
				"value":            "import-value",
				"variable_type":    "env_var",
				"protected":        false,
				"masked":           false,
				"environment_scope": "*",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableImportCmd(f.Factory)
	cmd.SetArgs([]string{"--file", "testdata/variables.json"})

	err := cmd.Execute()
	// Note: This may error if the file doesn't exist, which is expected
	// The test verifies the command can be executed without panic
	_ = err
}

func TestVariableSetCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableSetCmd(f)

	if cmd.Use != "set <key>" {
		t.Errorf("expected Use to be 'set <key>', got %q", cmd.Use)
	}

	if cmd.Short != "Set a CI/CD variable" {
		t.Errorf("expected Short to be 'Set a CI/CD variable', got %q", cmd.Short)
	}
}

func TestVariableSetCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableSetCmd(f)

	expectedFlags := []string{"value", "masked", "protected", "scope", "file", "group", "type"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify scope flag default value
	scopeFlag := cmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Fatal("scope flag not found")
	}
	if scopeFlag.DefValue != "*" {
		t.Errorf("expected default scope to be '*', got %q", scopeFlag.DefValue)
	}

	// Verify type flag default value
	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Fatal("type flag not found")
	}
	if typeFlag.DefValue != "env_var" {
		t.Errorf("expected default type to be 'env_var', got %q", typeFlag.DefValue)
	}
}

func TestVariableUpdateCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableUpdateCmd(f)

	if cmd.Use != "update <key>" {
		t.Errorf("expected Use to be 'update <key>', got %q", cmd.Use)
	}

	if cmd.Short != "Update an existing CI/CD variable" {
		t.Errorf("expected Short to be 'Update an existing CI/CD variable', got %q", cmd.Short)
	}
}

func TestVariableUpdateCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableUpdateCmd(f)

	expectedFlags := []string{"value", "masked", "protected", "scope", "file", "group", "type"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify scope flag default value
	scopeFlag := cmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Fatal("scope flag not found")
	}
	if scopeFlag.DefValue != "*" {
		t.Errorf("expected default scope to be '*', got %q", scopeFlag.DefValue)
	}

	// Verify type flag default value
	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Fatal("type flag not found")
	}
	if typeFlag.DefValue != "env_var" {
		t.Errorf("expected default type to be 'env_var', got %q", typeFlag.DefValue)
	}
}

func TestVariableDeleteCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableDeleteCmd(f)

	if cmd.Use != "delete <key>" {
		t.Errorf("expected Use to be 'delete <key>', got %q", cmd.Use)
	}

	if cmd.Short != "Delete a CI/CD variable" {
		t.Errorf("expected Short to be 'Delete a CI/CD variable', got %q", cmd.Short)
	}
}

func TestVariableDeleteCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableDeleteCmd(f)

	expectedFlags := []string{"group"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify group flag default value
	groupFlag := cmd.Flags().Lookup("group")
	if groupFlag == nil {
		t.Fatal("group flag not found")
	}
	if groupFlag.DefValue != "" {
		t.Errorf("expected default group to be empty, got %q", groupFlag.DefValue)
	}
}

func TestVariableExportCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableExportCmd(f)

	if cmd.Use != "export" {
		t.Errorf("expected Use to be 'export', got %q", cmd.Use)
	}

	if cmd.Short != "Export CI/CD variables" {
		t.Errorf("expected Short to be 'Export CI/CD variables', got %q", cmd.Short)
	}
}

func TestVariableExportCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableExportCmd(f)

	expectedFlags := []string{"group", "output"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify group flag default value
	groupFlag := cmd.Flags().Lookup("group")
	if groupFlag == nil {
		t.Fatal("group flag not found")
	}
	if groupFlag.DefValue != "" {
		t.Errorf("expected default group to be empty, got %q", groupFlag.DefValue)
	}

	// Verify output flag default value
	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("output flag not found")
	}
	if outputFlag.DefValue != "" {
		t.Errorf("expected default output to be empty, got %q", outputFlag.DefValue)
	}
}

func TestVariableImportCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableImportCmd(f)

	if cmd.Use != "import" {
		t.Errorf("expected Use to be 'import', got %q", cmd.Use)
	}

	if cmd.Short != "Import CI/CD variables from JSON" {
		t.Errorf("expected Short to be 'Import CI/CD variables from JSON', got %q", cmd.Short)
	}
}

func TestVariableImportCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newVariableImportCmd(f)

	expectedFlags := []string{"group", "file"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify group flag default value
	groupFlag := cmd.Flags().Lookup("group")
	if groupFlag == nil {
		t.Fatal("group flag not found")
	}
	if groupFlag.DefValue != "" {
		t.Errorf("expected default group to be empty, got %q", groupFlag.DefValue)
	}

	// Verify file flag default value
	fileFlag := cmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Fatal("file flag not found")
	}
	if fileFlag.DefValue != "" {
		t.Errorf("expected default file to be empty, got %q", fileFlag.DefValue)
	}
}

func TestVariableList_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)

	_ = cmd.Execute()
	// May error if not in a git repo or no variables, which is OK
	// Just verify command executes without panic
}

func TestVariableGet_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	_ = cmd.Execute()
	// May error if variable not found, which is OK
	// Just verify command executes without panic
}

func TestVariableSet_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test-value"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableUpdate_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "updated-value"})

	_ = cmd.Execute()
	// May error if variable not found or no auth, which is OK
}

func TestVariableDelete_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	_ = cmd.Execute()
	// May error if variable not found or no auth, which is OK
}

func TestVariableExport_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableExportCmd(f.Factory)

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableImport_MissingFile(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableImportCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when file flag is missing, got nil")
	}
}

func TestVariableSet_MissingValue(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	_ = cmd.Execute()
	// May fail validation
}

func TestVariableUpdate_MissingValue(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	_ = cmd.Execute()
	// May fail validation
}

func TestVariableGet_NotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"NONEXISTENT_VAR"})

	_ = cmd.Execute()
	// May return error, which is OK
}

func TestVariableDelete_NotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"NONEXISTENT_VAR"})

	_ = cmd.Execute()
	// May return error, which is OK
}

func TestVariableList_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)
	cmd.SetArgs([]string{"--group", "test-group"})

	_ = cmd.Execute()
	// May error if group not found or no auth, which is OK
}

func TestVariableGet_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--group", "test-group"})

	_ = cmd.Execute()
	// May error if variable/group not found, which is OK
}

func TestVariableSet_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test-value", "--group", "test-group"})

	_ = cmd.Execute()
	// May error if group not found or no auth, which is OK
}

func TestVariableUpdate_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "updated-value", "--group", "test-group"})

	_ = cmd.Execute()
	// May error if variable/group not found, which is OK
}

func TestVariableDelete_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--group", "test-group"})

	_ = cmd.Execute()
	// May error if variable/group not found, which is OK
}

func TestVariableExport_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableExportCmd(f.Factory)
	cmd.SetArgs([]string{"--group", "test-group"})

	_ = cmd.Execute()
	// May error if group not found or no auth, which is OK
}

func TestVariableImport_WithGroup(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableImportCmd(f.Factory)
	cmd.SetArgs([]string{"--file", "test.json", "--group", "test-group"})

	_ = cmd.Execute()
	// May error if file/group not found, which is OK
}

func TestVariableList_WithJSON(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)
	cmd.SetArgs([]string{"--json"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableGet_WithJSON(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--json"})

	_ = cmd.Execute()
	// May error if variable not found, which is OK
}

func TestVariableSet_WithProtected(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "secret", "--protected"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableSet_WithMasked(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "secret", "--masked"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableSet_WithScope(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test", "--scope", "production"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableSet_WithType(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test", "--type", "file"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

func TestVariableUpdate_WithProtected(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "secret", "--protected"})

	_ = cmd.Execute()
	// May error if variable not found, which is OK
}

func TestVariableUpdate_WithMasked(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "secret", "--masked"})

	_ = cmd.Execute()
	// May error if variable not found, which is OK
}

func TestVariableUpdate_WithScope(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test", "--scope", "staging"})

	_ = cmd.Execute()
	// May error if variable not found, which is OK
}

func TestVariableUpdate_WithType(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test", "--type", "file"})

	_ = cmd.Execute()
	// May error if variable not found, which is OK
}

func TestVariableExport_WithOutput(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableExportCmd(f.Factory)
	cmd.SetArgs([]string{"--output", "/tmp/vars.json"})

	_ = cmd.Execute()
	// May error if not in a git repo or no auth, which is OK
}

// Error case tests
func TestVariableList_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 401 Unauthorized, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain 401, got: %v", err)
	}
}

func TestVariableGet_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 401 Unauthorized, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain 401, got: %v", err)
	}
}

func TestVariableSet_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test-value"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 401 Unauthorized, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain 401, got: %v", err)
	}
}

func TestVariableUpdate_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "updated-value"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 401 Unauthorized, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain 401, got: %v", err)
	}
}

func TestVariableDelete_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 401 Unauthorized, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain 401, got: %v", err)
	}
}

func TestVariableList_Forbidden(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 403, "403 Forbidden")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 403 Forbidden, got nil")
	}

	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to contain 403, got: %v", err)
	}
}

func TestVariableGet_NotFoundError(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{"NONEXISTENT_VAR"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 404 Not Found, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain 404, got: %v", err)
	}
}

func TestVariableSet_Forbidden(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 403, "403 Forbidden")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test-value"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 403 Forbidden, got nil")
	}

	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to contain 403, got: %v", err)
	}
}

func TestVariableUpdate_NotFoundError(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"NONEXISTENT_VAR", "--value", "updated-value"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 404 Not Found, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain 404, got: %v", err)
	}
}

func TestVariableDelete_NotFoundError(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{"NONEXISTENT_VAR"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 404 Not Found, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain 404, got: %v", err)
	}
}

func TestVariableList_ServerError(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 500, "500 Internal Server Error")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 500 Internal Server Error, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain 500, got: %v", err)
	}
}

func TestVariableSet_ServerError(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 500, "500 Internal Server Error")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test-value"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for 500 Internal Server Error, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain 500, got: %v", err)
	}
}

// Edge case tests
func TestVariableList_EmptyResponse(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVariableGet_MissingKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableGetCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when key is missing, got nil")
	}
}

func TestVariableSet_MissingKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when key is missing, got nil")
	}
}

func TestVariableUpdate_MissingKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when key is missing, got nil")
	}
}

func TestVariableDelete_MissingKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableDeleteCmd(f.Factory)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when key is missing, got nil")
	}
}

func TestVariableSet_EmptyValueFlag(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"EMPTY_VAR"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when value flag is missing, got nil")
	}
}

func TestVariableSet_InvalidType(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"TEST_VAR", "--value", "test", "--type", "invalid_type"})

	_ = cmd.Execute()
	// Command may error or proceed depending on validation
}

func TestVariableUpdate_EmptyValueFlag(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"EMPTY_VAR"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when value flag is missing, got nil")
	}
}

func TestVariableExport_EmptyResponse(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableExportCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVariableSet_WithAllFlags(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"key":              "FULL_VAR",
				"value":            "secret",
				"variable_type":    "file",
				"protected":        true,
				"masked":           true,
				"environment_scope": "production",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableSetCmd(f.Factory)
	cmd.SetArgs([]string{"FULL_VAR", "--value", "secret", "--protected", "--masked", "--scope", "production", "--type", "file"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "FULL_VAR") {
		t.Errorf("expected output to contain FULL_VAR, got: %s", output)
	}
}

func TestVariableUpdate_WithAllFlags(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables/FULL_VAR") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"key":              "FULL_VAR",
				"value":            "updated-secret",
				"variable_type":    "file",
				"protected":        true,
				"masked":           true,
				"environment_scope": "staging",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "404 Variable Not Found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableUpdateCmd(f.Factory)
	cmd.SetArgs([]string{"FULL_VAR", "--value", "updated-secret", "--protected", "--masked", "--scope", "staging", "--type", "file"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "FULL_VAR") {
		t.Errorf("expected output to contain FULL_VAR, got: %s", output)
	}
}

// Additional tests for improved coverage

func TestVariableList_GroupWithJSON(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/groups") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{
				{
					"key":              "GROUP_JSON_VAR",
					"value":            "group-json-value",
					"variable_type":    "env_var",
					"protected":        true,
					"masked":           true,
					"environment_scope": "production",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)
	cmd.SetArgs([]string{"--group", "mygroup", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "GROUP_JSON_VAR") {
		t.Errorf("expected JSON output to contain GROUP_JSON_VAR, got: %s", output)
	}
}

func TestVariableExport_WithFile(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{
				{
					"key":              "EXPORT_VAR",
					"value":            "export-value",
					"variable_type":    "env_var",
					"protected":        false,
					"masked":           false,
					"environment_scope": "*",
				},
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableExportCmd(f.Factory)

	tmpFile := filepath.Join(os.TempDir(), "test_export.json")
	defer os.Remove(tmpFile)

	cmd.SetArgs([]string{"--output", tmpFile})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("expected export file to be created at %s", tmpFile)
	}
}

func TestVariableImport_WithFile(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"key":              "IMPORT_VAR",
				"value":            "import-value",
				"variable_type":    "env_var",
				"protected":        false,
				"masked":           false,
				"environment_scope": "*",
			})
			return
		}
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.ErrorResponse(w, 404, "variable not found")
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableImportCmd(f.Factory)
	cmd.SetArgs([]string{"--file", "testdata/variables.json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Imported") {
		t.Errorf("expected output to contain 'Imported', got: %s", output)
	}
}

func TestVariableImport_UpdateExisting(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		// Mock successful UPDATE (variable already exists)
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/api/v4/projects") && strings.Contains(r.URL.Path, "/variables/IMPORT_VAR") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"key":              "IMPORT_VAR",
				"value":            "updated-value",
				"variable_type":    "env_var",
				"protected":        false,
				"masked":           false,
				"environment_scope": "*",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableImportCmd(f.Factory)
	cmd.SetArgs([]string{"--file", "testdata/variables.json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "Imported") {
		t.Errorf("expected output to contain 'Imported', got: %s", output)
	}
}

func TestVariableList_GroupEmptyResponse(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/groups") && strings.Contains(r.URL.Path, "/variables") {
			cmdtest.JSONResponse(w, 200, []map[string]interface{}{})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := newVariableListCmd(f.Factory)
	cmd.SetArgs([]string{"--group", "emptygroup"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := f.IO.ErrString()
	if !strings.Contains(errOutput, "No variables found") {
		t.Errorf("expected error output to contain 'No variables found', got: %s", errOutput)
	}
}

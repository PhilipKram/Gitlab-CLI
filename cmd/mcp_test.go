package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewMCPCmd(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewMCPCmd(f.Factory)

	if cmd == nil {
		t.Fatal("expected mcp command")
	}
	if cmd.Use != "mcp <command>" {
		t.Errorf("expected Use='mcp <command>', got %s", cmd.Use)
	}

	// Verify all subcommands are registered
	subcommands := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}
	expected := []string{"serve", "install", "uninstall", "status"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
	if len(cmd.Commands()) != len(expected) {
		t.Errorf("expected %d subcommands, got %d", len(expected), len(cmd.Commands()))
	}
}

func TestMCPInstallCmd(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewMCPCmd(f.Factory)

	install, _, err := cmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("expected install subcommand: %v", err)
	}
	if install.Use != "install" {
		t.Errorf("expected Use=install, got %s", install.Use)
	}

	// Verify flags exist
	if install.Flags().Lookup("scope") == nil {
		t.Error("expected --scope flag")
	}
	if install.Flags().Lookup("client") == nil {
		t.Error("expected --client flag")
	}
}

func TestMCPInstallCmdScope(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewMCPCmd(f.Factory)

	install, _, _ := cmd.Find([]string{"install"})
	scopeFlag := install.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Fatal("expected --scope flag")
	}
	if scopeFlag.DefValue != "user" {
		t.Errorf("expected default scope=user, got %s", scopeFlag.DefValue)
	}
}

func TestMCPUninstallCmd(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewMCPCmd(f.Factory)

	uninstall, _, err := cmd.Find([]string{"uninstall"})
	if err != nil {
		t.Fatalf("expected uninstall subcommand: %v", err)
	}
	if uninstall.Use != "uninstall" {
		t.Errorf("expected Use=uninstall, got %s", uninstall.Use)
	}

	if uninstall.Flags().Lookup("scope") == nil {
		t.Error("expected --scope flag")
	}
	if uninstall.Flags().Lookup("client") == nil {
		t.Error("expected --client flag")
	}
}

func TestMCPStatusCmd(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := NewMCPCmd(f.Factory)

	status, _, err := cmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("expected status subcommand: %v", err)
	}
	if status.Use != "status" {
		t.Errorf("expected Use=status, got %s", status.Use)
	}

	if status.Flags().Lookup("client") == nil {
		t.Error("expected --client flag")
	}
}

func TestGlabBinaryPath(t *testing.T) {
	path := glabBinaryPath()
	if path == "" {
		t.Error("expected non-empty binary path")
	}
	// In test environment, should resolve to something (the test binary or "glab" fallback)
}

func TestClaudeDesktopConfigPath(t *testing.T) {
	path, err := claudeDesktopConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty config path")
	}

	// Verify platform-specific path components
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(path, "Library/Application Support/Claude/claude_desktop_config.json") {
			t.Errorf("expected path to contain Library/Application Support/Claude/claude_desktop_config.json, got %s", path)
		}
	case "linux":
		if !strings.Contains(path, ".config/Claude/claude_desktop_config.json") {
			t.Errorf("expected path to contain .config/Claude/claude_desktop_config.json, got %s", path)
		}
	case "windows":
		if !strings.Contains(path, "Claude\\claude_desktop_config.json") {
			t.Errorf("expected path to contain Claude\\claude_desktop_config.json, got %s", path)
		}
	}
}

func TestMCPConfigJSON(t *testing.T) {
	configStr, err := mcpConfigJSON("/usr/local/bin/glab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if config["command"] != "/usr/local/bin/glab" {
		t.Errorf("expected command=/usr/local/bin/glab, got %v", config["command"])
	}

	args, ok := config["args"].([]interface{})
	if !ok {
		t.Fatal("expected args to be an array")
	}
	if len(args) != 2 || args[0] != "mcp" || args[1] != "serve" {
		t.Errorf("expected args=[mcp, serve], got %v", args)
	}
}

func TestMCPInstallClaudeDesktop(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	out := &bytes.Buffer{}
	err := installClaudeDesktopToPath(out, "/usr/local/bin/glab", configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read and verify the written config
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers key")
	}

	glab, ok := servers["glab"].(map[string]interface{})
	if !ok {
		t.Fatal("expected glab entry in mcpServers")
	}

	if glab["command"] != "/usr/local/bin/glab" {
		t.Errorf("expected command=/usr/local/bin/glab, got %v", glab["command"])
	}

	args, ok := glab["args"].([]interface{})
	if !ok {
		t.Fatal("expected args array")
	}
	if len(args) != 2 || args[0] != "mcp" || args[1] != "serve" {
		t.Errorf("expected args=[mcp, serve], got %v", args)
	}
}

func TestMCPUninstallClaudeDesktop(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	// Write a config with glab and another server
	initial := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"glab": map[string]interface{}{
				"command": "/usr/local/bin/glab",
				"args":    []string{"mcp", "serve"},
			},
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
				"args":    []string{"serve"},
			},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	err := uninstallClaudeDesktopFromPath(out, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify glab removed, other-server preserved
	data, _ = os.ReadFile(configPath)
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	servers := config["mcpServers"].(map[string]interface{})
	if _, exists := servers["glab"]; exists {
		t.Error("expected glab to be removed")
	}
	if _, exists := servers["other-server"]; !exists {
		t.Error("expected other-server to be preserved")
	}
}

func TestMCPInstallNoClaude(t *testing.T) {
	// Ensure claude is not in PATH
	t.Setenv("PATH", t.TempDir())

	_, err := findClaude()
	if err == nil {
		t.Fatal("expected error when claude CLI not found")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected helpful error mentioning claude, got: %v", err)
	}
	if !strings.Contains(err.Error(), "https://") {
		t.Errorf("expected error to contain install URL, got: %v", err)
	}
}

func TestMCPUninstallNotRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	// Config exists but has no glab entry
	initial := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
			},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	out := &bytes.Buffer{}
	err := uninstallClaudeDesktopFromPath(out, configPath)
	if err != nil {
		t.Fatalf("expected clean exit, got error: %v", err)
	}
	if !strings.Contains(out.String(), "not registered") {
		t.Errorf("expected 'not registered' message, got: %s", out.String())
	}
}

func TestMCPUninstallClaudeDesktop_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent", "claude_desktop_config.json")

	out := &bytes.Buffer{}
	err := uninstallClaudeDesktopFromPath(out, configPath)
	if err != nil {
		t.Fatalf("expected clean exit, got error: %v", err)
	}
	if !strings.Contains(out.String(), "not registered") {
		t.Errorf("expected 'not registered' message, got: %s", out.String())
	}
}

func TestMCPUninstallClaudeDesktop_NoMCPServers(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	// Config exists but has no mcpServers key
	initial := map[string]interface{}{
		"someOtherKey": "value",
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	err := uninstallClaudeDesktopFromPath(out, configPath)
	if err != nil {
		t.Fatalf("expected clean exit, got error: %v", err)
	}
	if !strings.Contains(out.String(), "not registered") {
		t.Errorf("expected 'not registered' message, got: %s", out.String())
	}
}

func TestMCPInstallClaudeDesktop_MergeExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	// Pre-existing config with another server
	initial := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
				"args":    []string{"serve"},
			},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	err := installClaudeDesktopToPath(out, "/usr/local/bin/glab", configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read and verify both servers exist
	data, _ = os.ReadFile(configPath)
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	servers := config["mcpServers"].(map[string]interface{})
	if _, exists := servers["glab"]; !exists {
		t.Error("expected glab entry")
	}
	if _, exists := servers["other-server"]; !exists {
		t.Error("expected other-server to be preserved")
	}
}

func TestMCPInstallCmd_UnsupportedClient(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMCPInstallCmd(f.Factory)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--client", "unsupported-client"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported client")
	}
	if !strings.Contains(err.Error(), "unsupported client") {
		t.Errorf("expected 'unsupported client' error, got: %v", err)
	}
}

func TestMCPUninstallCmd_UnsupportedClient(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMCPUninstallCmd(f.Factory)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--client", "unsupported-client"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported client")
	}
	if !strings.Contains(err.Error(), "unsupported client") {
		t.Errorf("expected 'unsupported client' error, got: %v", err)
	}
}

func TestMCPStatusCmd_UnsupportedClient(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMCPStatusCmd(f.Factory)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--client", "unsupported-client"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported client")
	}
	if !strings.Contains(err.Error(), "unsupported client") {
		t.Errorf("expected 'unsupported client' error, got: %v", err)
	}
}

func TestInstallClaudeCode_NoClaude(t *testing.T) {
	// Ensure claude is not in PATH
	t.Setenv("PATH", t.TempDir())

	out := &bytes.Buffer{}
	err := installClaudeCode(out, "user", `{"command":"glab","args":["mcp","serve"]}`)
	if err == nil {
		t.Fatal("expected error when claude CLI not found")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected error mentioning claude, got: %v", err)
	}
}

func TestUninstallClaudeCode_NoClaude(t *testing.T) {
	// Ensure claude is not in PATH
	t.Setenv("PATH", t.TempDir())

	out := &bytes.Buffer{}
	err := uninstallClaudeCode(out, "user")
	if err == nil {
		t.Fatal("expected error when claude CLI not found")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected error mentioning claude, got: %v", err)
	}
}

func TestStatusClaudeCode_NoClaude(t *testing.T) {
	// Ensure claude is not in PATH
	t.Setenv("PATH", t.TempDir())

	out := &bytes.Buffer{}
	err := statusClaudeCode(out)
	if err == nil {
		t.Fatal("expected error when claude CLI not found")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected error mentioning claude, got: %v", err)
	}
}

func TestMCPInstallIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

	out := &bytes.Buffer{}

	// Install twice
	if err := installClaudeDesktopToPath(out, "/usr/local/bin/glab", configPath); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	out.Reset()
	if err := installClaudeDesktopToPath(out, "/usr/local/bin/glab", configPath); err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	// Verify only one glab entry exists
	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	servers := config["mcpServers"].(map[string]interface{})
	if len(servers) != 1 {
		t.Errorf("expected exactly 1 server entry, got %d", len(servers))
	}
	if _, exists := servers["glab"]; !exists {
		t.Error("expected glab entry")
	}
}

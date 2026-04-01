package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/auth"
	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	glabmcp "github.com/PhilipKram/gitlab-cli/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token1) != 64 {
		t.Errorf("expected 64-char token, got %d chars", len(token1))
	}
	// Verify it's valid hex
	for _, c := range token1 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("token contains non-hex character: %c", c)
			break
		}
	}
	// Verify non-deterministic
	token2, err := generateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token1 == token2 {
		t.Error("expected different tokens on successive calls")
	}
}

func TestBearerAuthMiddleware(t *testing.T) {
	token := "test-secret-token"
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	handler := bearerAuthMiddleware(inner, token)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid token", "Bearer test-secret-token", http.StatusOK},
		{"invalid token", "Bearer wrong-token", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
		{"no bearer prefix", "Basic test-secret-token", http.StatusUnauthorized},
		{"bearer only", "Bearer ", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestMCPRemoteConfigJSON(t *testing.T) {
	configStr, err := mcpRemoteConfigJSON("myhost.example.com", 9090, "/mcp", "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if config["url"] != "http://myhost.example.com:9090/mcp" {
		t.Errorf("expected url=http://myhost.example.com:9090/mcp, got %v", config["url"])
	}

	headers, ok := config["headers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected headers object")
	}
	if headers["Authorization"] != "Bearer my-token" {
		t.Errorf("expected Authorization=Bearer my-token, got %v", headers["Authorization"])
	}
}

func TestMCPRemoteConfigJSON_NoToken(t *testing.T) {
	configStr, err := mcpRemoteConfigJSON("localhost", 8080, "/mcp", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if config["url"] != "http://localhost:8080/mcp" {
		t.Errorf("expected url=http://localhost:8080/mcp, got %v", config["url"])
	}

	if _, ok := config["headers"]; ok {
		t.Error("expected no headers when token is empty")
	}
}

func TestServeHTTP_Integration(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	server := glabmcp.NewMCPServer(f.Factory)

	token := "integration-test-token"
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	authed := bearerAuthMiddleware(handler, token)
	mux := http.NewServeMux()
	mux.Handle("/mcp", authed)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Test 401 without token
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Test initialize with valid token
	initMsg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}`
	req, err := http.NewRequest("POST", ts.URL+"/mcp", strings.NewReader(initMsg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestMCPServeCmdFlags(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMCPServeCmd(f.Factory)

	flags := []string{"transport", "port", "host", "token", "no-auth", "stateless", "base-path"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag", name)
		}
	}
}

func TestMCPInstallCmdHTTPFlags(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMCPInstallCmd(f.Factory)

	flags := []string{"transport", "host", "port", "base-path", "token"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag on install command", name)
		}
	}
}

func TestMCPServeCmdOAuthFlags(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newMCPServeCmd(f.Factory)

	flags := []string{"client-id", "gitlab-host"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag", name)
		}
	}
}

func TestSessionStore(t *testing.T) {
	store := newSessionStore()

	// No session initially
	if store.getSession("nonexistent") != nil {
		t.Error("expected nil for nonexistent session")
	}

	// Add session
	sess := &mcpSession{
		BearerToken: "test-bearer-token",
		GitLabHost:  "gitlab.com",
		AccessToken: "gl-access-token",
		Username:    "testuser",
	}
	store.addSession(sess)

	// Retrieve session
	got := store.getSession("test-bearer-token")
	if got == nil {
		t.Fatal("expected session")
	}
	if got.GitLabHost != "gitlab.com" {
		t.Errorf("expected gitlab.com, got %s", got.GitLabHost)
	}
	if got.AccessToken != "gl-access-token" {
		t.Errorf("expected gl-access-token, got %s", got.AccessToken)
	}

	// Wrong token returns nil
	if store.getSession("wrong-token") != nil {
		t.Error("expected nil for wrong token")
	}
}

func TestSessionStorePending(t *testing.T) {
	store := newSessionStore()

	req := &oauthAuthRequest{
		RegisteredClientID: "test-client",
		ClientRedirectURI:  "http://localhost/callback",
		GitLabState:        "gitlab-state-abc",
		GitLabVerifier:     "verifier123",
		CreatedAt:          time.Now(),
	}
	store.addPending("gitlab-state-abc", req)

	// Take removes it
	got := store.takePending("gitlab-state-abc")
	if got == nil {
		t.Fatal("expected pending flow")
	}
	if got.RegisteredClientID != "test-client" {
		t.Errorf("expected test-client, got %s", got.RegisteredClientID)
	}

	// Second take returns nil (consumed)
	if store.takePending("gitlab-state-abc") != nil {
		t.Error("expected nil after consuming pending")
	}
}

func TestSessionStoreClientRegistration(t *testing.T) {
	store := newSessionStore()

	// No client initially
	if store.getClient("nonexistent") != nil {
		t.Error("expected nil for nonexistent client")
	}

	client := &oauthRegisteredClient{
		ClientID:     "client-123",
		ClientName:   "Test Client",
		RedirectURIs: []string{"http://localhost/callback"},
	}
	store.addClient(client)

	got := store.getClient("client-123")
	if got == nil {
		t.Fatal("expected client")
	}
	if got.ClientName != "Test Client" {
		t.Errorf("expected 'Test Client', got %s", got.ClientName)
	}
}

func TestSessionStoreAuthCodes(t *testing.T) {
	store := newSessionStore()

	ac := &oauthAuthCode{
		Code:              "code-xyz",
		ClientID:          "client-123",
		RedirectURI:       "http://localhost/callback",
		GitLabAccessToken: "gl-token",
		CreatedAt:         time.Now(),
	}
	store.addCode(ac)

	// Take consumes it
	got := store.takeCode("code-xyz")
	if got == nil {
		t.Fatal("expected auth code")
	}
	if got.GitLabAccessToken != "gl-token" {
		t.Errorf("expected gl-token, got %s", got.GitLabAccessToken)
	}

	// Second take returns nil
	if store.takeCode("code-xyz") != nil {
		t.Error("expected nil after consuming code")
	}
}

func TestSessionBearerAuth(t *testing.T) {
	store := newSessionStore()
	store.addSession(&mcpSession{
		BearerToken: "valid-session-token",
		GitLabHost:  "gitlab.com",
		AccessToken: "gl-token",
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := store.sessionBearerAuth(inner)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid session", "Bearer valid-session-token", http.StatusOK},
		{"invalid session", "Bearer nonexistent-token", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestOAuthMetadataHandler(t *testing.T) {
	handler := oauthMetadataHandler("http://localhost:8090")

	req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if meta["issuer"] != "http://localhost:8090" {
		t.Errorf("expected issuer=http://localhost:8090, got %v", meta["issuer"])
	}
	if meta["authorization_endpoint"] != "http://localhost:8090/oauth/authorize" {
		t.Errorf("unexpected authorization_endpoint: %v", meta["authorization_endpoint"])
	}
	if meta["token_endpoint"] != "http://localhost:8090/oauth/token" {
		t.Errorf("unexpected token_endpoint: %v", meta["token_endpoint"])
	}
	if meta["registration_endpoint"] != "http://localhost:8090/oauth/register" {
		t.Errorf("unexpected registration_endpoint: %v", meta["registration_endpoint"])
	}
}

func TestOAuthRegisterHandler(t *testing.T) {
	store := newSessionStore()
	handler := oauthRegisterHandler(store)

	body := `{"client_name":"Test Client","redirect_uris":["http://localhost/callback"]}`
	req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	clientID, ok := resp["client_id"].(string)
	if !ok || clientID == "" {
		t.Error("expected non-empty client_id")
	}

	// Verify client was stored
	if store.getClient(clientID) == nil {
		t.Error("expected client to be stored")
	}
}

func TestOAuthRegisterHandler_MissingRedirectURIs(t *testing.T) {
	store := newSessionStore()
	handler := oauthRegisterHandler(store)

	body := `{"client_name":"Test Client"}`
	req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOAuthAuthorizeHandler(t *testing.T) {
	store := newSessionStore()
	// Register a client first
	store.addClient(&oauthRegisteredClient{
		ClientID:     "client-abc",
		RedirectURIs: []string{"http://localhost:9999/callback"},
	})

	handler := oauthAuthorizeHandler(store, "gitlab.example.com", "gitlab-app-id", "http://localhost:8090/oauth/callback")

	req := httptest.NewRequest("GET", "/oauth/authorize?response_type=code&client_id=client-abc&redirect_uri=http://localhost:9999/callback&code_challenge=abc123&code_challenge_method=S256&state=client-state", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "gitlab.example.com/oauth/authorize") {
		t.Errorf("expected redirect to gitlab, got %s", location)
	}
	if !strings.Contains(location, "client_id=gitlab-app-id") {
		t.Errorf("expected gitlab client_id, got %s", location)
	}

	// Verify pending was created
	store.mu.RLock()
	pendingCount := len(store.pending)
	store.mu.RUnlock()
	if pendingCount != 1 {
		t.Errorf("expected 1 pending, got %d", pendingCount)
	}
}

func TestOAuthAuthorizeHandler_UnknownClient(t *testing.T) {
	store := newSessionStore()
	handler := oauthAuthorizeHandler(store, "gitlab.com", "app-id", "http://localhost/callback")

	req := httptest.NewRequest("GET", "/oauth/authorize?client_id=unknown&redirect_uri=http://localhost/cb", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown client, got %d", w.Code)
	}
}

func TestOAuthGitLabCallbackHandler_InvalidState(t *testing.T) {
	store := newSessionStore()
	handler := oauthGitLabCallbackHandler(store, "gitlab.com", "app-id", "http://localhost/callback", io.Discard)

	req := httptest.NewRequest("GET", "/oauth/callback?state=invalid&code=abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid state, got %d", w.Code)
	}
}

func TestOAuthGitLabCallbackHandler_OAuthError(t *testing.T) {
	store := newSessionStore()
	store.addPending("test-state", &oauthAuthRequest{
		GitLabState: "test-state",
		CreatedAt:   time.Now(),
	})
	handler := oauthGitLabCallbackHandler(store, "gitlab.com", "app-id", "http://localhost/callback", io.Discard)

	req := httptest.NewRequest("GET", "/oauth/callback?state=test-state&error=access_denied&error_description=User+denied", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for OAuth error, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Authorization Failed") {
		t.Errorf("expected failure page, got: %s", w.Body.String())
	}
}

func TestOAuthTokenHandler_InvalidGrant(t *testing.T) {
	store := newSessionStore()
	handler := oauthTokenHandler(store, "gitlab.com", "test-gitlab-app-id", "http://localhost/auth/redirect", io.Discard)

	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader("grant_type=client_credentials"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOAuthTokenHandler_InvalidCode(t *testing.T) {
	store := newSessionStore()
	handler := oauthTokenHandler(store, "gitlab.com", "test-gitlab-app-id", "http://localhost/auth/redirect", io.Discard)

	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader("grant_type=authorization_code&code=invalid&client_id=x&redirect_uri=y"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOAuthTokenHandler_Success(t *testing.T) {
	store := newSessionStore()

	// Create a valid auth code (simulating post-GitLab-callback)
	verifier, _ := auth.GenerateCodeVerifier()
	challenge := auth.GenerateCodeChallenge(verifier)

	store.addCode(&oauthAuthCode{
		Code:              "valid-code",
		ClientID:          "client-123",
		RedirectURI:       "http://localhost/callback",
		CodeChallenge:     challenge,
		CodeChallengeMethod: "S256",
		GitLabAccessToken: "gl-access-token",
		GitLabRefreshToken: "gl-refresh-token",
		GitLabExpiresAt:   time.Now().Add(time.Hour).Unix(),
		CreatedAt:         time.Now(),
	})

	handler := oauthTokenHandler(store, "gitlab.com", "test-gitlab-app-id", "http://localhost/auth/redirect", io.Discard)

	body := fmt.Sprintf("grant_type=authorization_code&code=valid-code&client_id=client-123&redirect_uri=http://localhost/callback&code_verifier=%s", verifier)
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	accessToken, ok := resp["access_token"].(string)
	if !ok || accessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp["token_type"] != "bearer" {
		t.Errorf("expected token_type=bearer, got %v", resp["token_type"])
	}

	// Verify session was created
	sess := store.getSession(accessToken)
	if sess == nil {
		t.Fatal("expected session to be created")
	}
	if sess.AccessToken != "gl-access-token" {
		t.Errorf("expected GitLab token in session, got %s", sess.AccessToken)
	}
}

func TestExportedAuthHelpers(t *testing.T) {
	// Test that exported auth helpers work
	verifier, err := auth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier: %v", err)
	}
	if len(verifier) == 0 {
		t.Error("expected non-empty verifier")
	}

	challenge := auth.GenerateCodeChallenge(verifier)
	if len(challenge) == 0 {
		t.Error("expected non-empty challenge")
	}

	state, err := auth.GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}
	if len(state) == 0 {
		t.Error("expected non-empty state")
	}

	scopes := auth.DefaultScopes()
	if !strings.Contains(scopes, "api") {
		t.Errorf("expected scopes to contain 'api', got %s", scopes)
	}

	authURL := auth.BuildAuthURL("gitlab.com", "client123", "http://localhost/cb", state, challenge, scopes)
	if !strings.Contains(authURL, "gitlab.com/oauth/authorize") {
		t.Errorf("expected gitlab.com auth URL, got %s", authURL)
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

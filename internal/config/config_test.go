package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetConfigDir sets GLAB_CONFIG_DIR so each test can use its own temp dir.
func resetConfigDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("GLAB_CONFIG_DIR", dir)
}

func TestHostConfigRefreshTokenFieldsJSON(t *testing.T) {
	hc := &HostConfig{
		Token:          "access-token-123",
		RefreshToken:   "refresh-token-456",
		TokenExpiresAt: 1700000000,
		TokenCreatedAt: 1699993000,
		User:           "testuser",
		AuthMethod:     "oauth",
		ClientID:       "client-id-789",
	}

	data, err := json.Marshal(hc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded HostConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.RefreshToken != "refresh-token-456" {
		t.Errorf("RefreshToken = %q, want %q", decoded.RefreshToken, "refresh-token-456")
	}
	if decoded.TokenExpiresAt != 1700000000 {
		t.Errorf("TokenExpiresAt = %d, want %d", decoded.TokenExpiresAt, 1700000000)
	}
	if decoded.TokenCreatedAt != 1699993000 {
		t.Errorf("TokenCreatedAt = %d, want %d", decoded.TokenCreatedAt, 1699993000)
	}
}

func TestHostConfigRefreshTokenFieldsOmitEmpty(t *testing.T) {
	hc := &HostConfig{
		Token:      "access-token-123",
		User:       "testuser",
		AuthMethod: "pat",
	}

	data, err := json.Marshal(hc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	raw := string(data)
	for _, field := range []string{"refresh_token", "token_expires_at", "token_created_at"} {
		if strings.Contains(raw, field) {
			t.Errorf("JSON should omit empty %q, got: %s", field, raw)
		}
	}
}

func TestHostsRoundTripWithRefreshTokenFields(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.example.com": &HostConfig{
			Token:          "access-123",
			RefreshToken:   "refresh-456",
			TokenExpiresAt: 1700007200,
			TokenCreatedAt: 1700000000,
			User:           "alice",
			AuthMethod:     "oauth",
			ClientID:       "my-client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	}

	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	loaded, err := LoadHosts()
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}

	hc, ok := loaded["gitlab.example.com"]
	if !ok {
		t.Fatal("expected host gitlab.example.com in loaded config")
	}

	if hc.Token != "access-123" {
		t.Errorf("Token = %q, want %q", hc.Token, "access-123")
	}
	if hc.RefreshToken != "refresh-456" {
		t.Errorf("RefreshToken = %q, want %q", hc.RefreshToken, "refresh-456")
	}
	if hc.TokenExpiresAt != 1700007200 {
		t.Errorf("TokenExpiresAt = %d, want %d", hc.TokenExpiresAt, 1700007200)
	}
	if hc.TokenCreatedAt != 1700000000 {
		t.Errorf("TokenCreatedAt = %d, want %d", hc.TokenCreatedAt, 1700000000)
	}
	if hc.AuthMethod != "oauth" {
		t.Errorf("AuthMethod = %q, want %q", hc.AuthMethod, "oauth")
	}
}

func TestHostsRoundTripWithoutRefreshTokenFields(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.com": &HostConfig{
			Token:      "pat-token",
			User:       "bob",
			AuthMethod: "pat",
		},
	}

	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	loaded, err := LoadHosts()
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}

	hc := loaded["gitlab.com"]
	if hc.RefreshToken != "" {
		t.Errorf("RefreshToken should be empty for PAT, got %q", hc.RefreshToken)
	}
	if hc.TokenExpiresAt != 0 {
		t.Errorf("TokenExpiresAt should be 0 for PAT, got %d", hc.TokenExpiresAt)
	}
	if hc.TokenCreatedAt != 0 {
		t.Errorf("TokenCreatedAt should be 0 for PAT, got %d", hc.TokenCreatedAt)
	}
}

func TestLoadHostsBackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	// Write a hosts.json without the new fields (simulating old config)
	oldJSON := `{
  "gitlab.com": {
    "token": "old-token",
    "user": "olduser",
    "auth_method": "oauth",
    "client_id": "old-client"
  }
}`
	hostsPath := filepath.Join(tmpDir, "hosts.json")
	if err := os.WriteFile(hostsPath, []byte(oldJSON), 0o644); err != nil {
		t.Fatalf("writing old hosts.json: %v", err)
	}

	loaded, err := LoadHosts()
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}

	hc := loaded["gitlab.com"]
	if hc.Token != "old-token" {
		t.Errorf("Token = %q, want %q", hc.Token, "old-token")
	}
	if hc.RefreshToken != "" {
		t.Errorf("RefreshToken should be empty, got %q", hc.RefreshToken)
	}
	if hc.TokenExpiresAt != 0 {
		t.Errorf("TokenExpiresAt should be 0, got %d", hc.TokenExpiresAt)
	}
	if hc.TokenCreatedAt != 0 {
		t.Errorf("TokenCreatedAt should be 0, got %d", hc.TokenCreatedAt)
	}
}

func TestConfigDir_FromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	got := ConfigDir()
	if got != tmpDir {
		t.Errorf("ConfigDir() = %q, want %q", got, tmpDir)
	}
}

func TestLoad_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Verify defaults
	if cfg.Protocol != "https" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "https")
	}
	if cfg.GitRemote != "origin" {
		t.Errorf("GitRemote = %q, want %q", cfg.GitRemote, "origin")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON config")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want to contain 'parsing config'", err.Error())
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	cfg := &Config{
		Editor:      "vim",
		Pager:       "less",
		Browser:     "firefox",
		Protocol:    "ssh",
		GitRemote:   "upstream",
		DefaultHost: "gitlab.example.com",
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Editor != "vim" {
		t.Errorf("Editor = %q, want %q", loaded.Editor, "vim")
	}
	if loaded.Pager != "less" {
		t.Errorf("Pager = %q, want %q", loaded.Pager, "less")
	}
	if loaded.Browser != "firefox" {
		t.Errorf("Browser = %q, want %q", loaded.Browser, "firefox")
	}
	if loaded.Protocol != "ssh" {
		t.Errorf("Protocol = %q, want %q", loaded.Protocol, "ssh")
	}
	if loaded.GitRemote != "upstream" {
		t.Errorf("GitRemote = %q, want %q", loaded.GitRemote, "upstream")
	}
	if loaded.DefaultHost != "gitlab.example.com" {
		t.Errorf("DefaultHost = %q, want %q", loaded.DefaultHost, "gitlab.example.com")
	}
}

func TestConfig_Get(t *testing.T) {
	cfg := &Config{
		Editor:      "vim",
		Pager:       "less",
		Browser:     "firefox",
		Protocol:    "ssh",
		GitRemote:   "upstream",
		DefaultHost: "gitlab.example.com",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"editor", "vim"},
		{"pager", "less"},
		{"browser", "firefox"},
		{"protocol", "ssh"},
		{"git_remote", "upstream"},
		{"default_host", "gitlab.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q): %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	// Test unknown key
	_, err := cfg.Get("unknown_key")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error = %q, want to contain 'unknown config key'", err.Error())
	}
}

func TestConfig_Set(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	cfg := &Config{Protocol: "https", GitRemote: "origin"}

	tests := []struct {
		key   string
		value string
	}{
		{"editor", "nano"},
		{"pager", "more"},
		{"browser", "chrome"},
		{"protocol", "ssh"},
		{"git_remote", "upstream"},
		{"default_host", "my.gitlab.com"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := cfg.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Set(%q, %q): %v", tt.key, tt.value, err)
			}
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q): %v", tt.key, err)
			}
			if got != tt.value {
				t.Errorf("after Set, Get(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}

	// Test unknown key
	err := cfg.Set("unknown_key", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error = %q, want to contain 'unknown config key'", err.Error())
	}
}

func TestConfig_SetPersists(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	cfg := &Config{Protocol: "https", GitRemote: "origin"}
	if err := cfg.Set("editor", "emacs"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Load again and verify persistence
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Editor != "emacs" {
		t.Errorf("Editor = %q after reload, want %q", loaded.Editor, "emacs")
	}
}

func TestKeys(t *testing.T) {
	keys := Keys()
	expected := []string{"editor", "pager", "browser", "protocol", "git_remote", "default_host"}
	if len(keys) != len(expected) {
		t.Fatalf("Keys() returned %d keys, want %d", len(keys), len(expected))
	}
	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("Keys()[%d] = %q, want %q", i, keys[i], k)
		}
	}
}

func TestHostKeys(t *testing.T) {
	keys := HostKeys()
	if len(keys) == 0 {
		t.Fatal("HostKeys() returned empty slice")
	}
	// Verify expected keys are present
	expectedKeys := map[string]bool{
		"client_id":    false,
		"redirect_uri": false,
		"oauth_scopes": false,
		"protocol":     false,
		"api_host":     false,
	}
	for _, k := range keys {
		expectedKeys[k] = true
	}
	for k, found := range expectedKeys {
		if !found {
			t.Errorf("expected key %q in HostKeys()", k)
		}
	}
}

func TestGetHostValue(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.example.com": &HostConfig{
			Token:       "my-token",
			User:        "alice",
			AuthMethod:  "oauth",
			ClientID:    "client-123",
			RedirectURI: "http://localhost:7171/auth/redirect",
			OAuthScopes: "api read_user",
			Protocol:    "ssh",
			APIHost:     "api.gitlab.example.com",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"client_id", "client-123"},
		{"redirect_uri", "http://localhost:7171/auth/redirect"},
		{"oauth_scopes", "api read_user"},
		{"protocol", "ssh"},
		{"api_host", "api.gitlab.example.com"},
		{"token", "my-token"},
		{"user", "alice"},
		{"auth_method", "oauth"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := GetHostValue("gitlab.example.com", tt.key)
			if err != nil {
				t.Fatalf("GetHostValue(%q): %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("GetHostValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	// Test unknown key
	_, err := GetHostValue("gitlab.example.com", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown host config key") {
		t.Errorf("error = %q, want to contain 'unknown host config key'", err.Error())
	}

	// Test nonexistent host
	_, err = GetHostValue("nonexistent.host", "token")
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
	if !strings.Contains(err.Error(), "no configuration for host") {
		t.Errorf("error = %q, want to contain 'no configuration for host'", err.Error())
	}
}

func TestSetHostValue(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	// Start with an empty hosts config
	hosts := HostsConfig{
		"gitlab.example.com": &HostConfig{
			Token: "my-token",
			User:  "alice",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	tests := []struct {
		key   string
		value string
	}{
		{"client_id", "new-client"},
		{"redirect_uri", "http://localhost:9999/callback"},
		{"oauth_scopes", "api"},
		{"protocol", "ssh"},
		{"api_host", "api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := SetHostValue("gitlab.example.com", tt.key, tt.value); err != nil {
				t.Fatalf("SetHostValue(%q, %q): %v", tt.key, tt.value, err)
			}
			got, err := GetHostValue("gitlab.example.com", tt.key)
			if err != nil {
				t.Fatalf("GetHostValue(%q): %v", tt.key, err)
			}
			if got != tt.value {
				t.Errorf("after SetHostValue, GetHostValue(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}

	// Test unknown key
	err := SetHostValue("gitlab.example.com", "nonexistent", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown host config key") {
		t.Errorf("error = %q, want to contain 'unknown host config key'", err.Error())
	}
}

func TestSetHostValue_NewHost(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	// Start with empty hosts
	if err := SaveHosts(HostsConfig{}); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	// Setting a value for a new host should create the host entry
	if err := SetHostValue("new.host.com", "client_id", "new-client"); err != nil {
		t.Fatalf("SetHostValue: %v", err)
	}

	got, err := GetHostValue("new.host.com", "client_id")
	if err != nil {
		t.Fatalf("GetHostValue: %v", err)
	}
	if got != "new-client" {
		t.Errorf("GetHostValue() = %q, want %q", got, "new-client")
	}
}

func TestLoadHosts_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "hosts.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("writing hosts: %v", err)
	}

	_, err := LoadHosts()
	if err == nil {
		t.Fatal("expected error for invalid JSON hosts")
	}
	if !strings.Contains(err.Error(), "parsing hosts config") {
		t.Errorf("error = %q, want to contain 'parsing hosts config'", err.Error())
	}
}

func TestLoadHosts_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts, err := LoadHosts()
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected empty hosts, got %d", len(hosts))
	}
}

func TestTokenForHost_EnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	// Set env vars and ensure GITLAB_HOST is not set so default is gitlab.com
	t.Setenv("GITLAB_HOST", "")
	t.Setenv("GITLAB_TOKEN", "env-token-123")
	t.Setenv("GLAB_TOKEN", "")

	// Clear config so DefaultHost falls back to gitlab.com
	if err := SaveHosts(HostsConfig{}); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	token, source := TokenForHost("gitlab.com")
	if token != "env-token-123" {
		t.Errorf("TokenForHost() token = %q, want %q", token, "env-token-123")
	}
	if source != "GITLAB_TOKEN" {
		t.Errorf("TokenForHost() source = %q, want %q", source, "GITLAB_TOKEN")
	}
}

func TestTokenForHost_GlabTokenEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	t.Setenv("GITLAB_HOST", "")
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "glab-token-456")

	if err := SaveHosts(HostsConfig{}); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	token, source := TokenForHost("gitlab.com")
	if token != "glab-token-456" {
		t.Errorf("TokenForHost() token = %q, want %q", token, "glab-token-456")
	}
	if source != "GLAB_TOKEN" {
		t.Errorf("TokenForHost() source = %q, want %q", source, "GLAB_TOKEN")
	}
}

func TestTokenForHost_EnvVarNotForNonDefaultHost(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	t.Setenv("GITLAB_HOST", "")
	t.Setenv("GITLAB_TOKEN", "env-token-123")
	t.Setenv("GLAB_TOKEN", "")

	if err := SaveHosts(HostsConfig{}); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	// Non-default host should NOT get the env token
	token, _ := TokenForHost("other.gitlab.com")
	if token != "" {
		t.Errorf("TokenForHost(non-default) should return empty, got %q", token)
	}
}

func TestTokenForHost_FromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	hosts := HostsConfig{
		"gitlab.example.com": &HostConfig{
			Token: "config-token",
			User:  "alice",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	token, source := TokenForHost("gitlab.example.com")
	if token != "config-token" {
		t.Errorf("TokenForHost() token = %q, want %q", token, "config-token")
	}
	if source != "gitlab.example.com" {
		t.Errorf("TokenForHost() source = %q, want %q", source, "gitlab.example.com")
	}
}

func TestTokenForHost_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	if err := SaveHosts(HostsConfig{}); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	token, source := TokenForHost("nonexistent.host")
	if token != "" {
		t.Errorf("TokenForHost() token = %q, want empty", token)
	}
	if source != "" {
		t.Errorf("TokenForHost() source = %q, want empty", source)
	}
}

func TestAuthMethodForHost(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.com": &HostConfig{
			Token:      "token",
			AuthMethod: "pat",
		},
		"gitlab.example.com": &HostConfig{
			Token:      "token",
			AuthMethod: "oauth",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	if got := AuthMethodForHost("gitlab.com"); got != "pat" {
		t.Errorf("AuthMethodForHost(gitlab.com) = %q, want %q", got, "pat")
	}
	if got := AuthMethodForHost("gitlab.example.com"); got != "oauth" {
		t.Errorf("AuthMethodForHost(gitlab.example.com) = %q, want %q", got, "oauth")
	}
	if got := AuthMethodForHost("nonexistent.host"); got != "" {
		t.Errorf("AuthMethodForHost(nonexistent) = %q, want empty", got)
	}
}

func TestOAuthScopesForHost(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.com": &HostConfig{
			Token:       "token",
			OAuthScopes: "api read_user",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	if got := OAuthScopesForHost("gitlab.com"); got != "api read_user" {
		t.Errorf("OAuthScopesForHost(gitlab.com) = %q, want %q", got, "api read_user")
	}
	if got := OAuthScopesForHost("nonexistent.host"); got != "" {
		t.Errorf("OAuthScopesForHost(nonexistent) = %q, want empty", got)
	}
}

func TestRedirectURIForHost(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.com": &HostConfig{
			Token:       "token",
			RedirectURI: "http://localhost:7171/auth/redirect",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	if got := RedirectURIForHost("gitlab.com"); got != "http://localhost:7171/auth/redirect" {
		t.Errorf("RedirectURIForHost(gitlab.com) = %q, want %q", got, "http://localhost:7171/auth/redirect")
	}
	if got := RedirectURIForHost("nonexistent.host"); got != "" {
		t.Errorf("RedirectURIForHost(nonexistent) = %q, want empty", got)
	}
}

func TestClientIDForHost(t *testing.T) {
	tmpDir := t.TempDir()
	resetConfigDir(t, tmpDir)

	hosts := HostsConfig{
		"gitlab.com": &HostConfig{
			Token:    "token",
			ClientID: "my-client-id",
		},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	if got := ClientIDForHost("gitlab.com"); got != "my-client-id" {
		t.Errorf("ClientIDForHost(gitlab.com) = %q, want %q", got, "my-client-id")
	}
	if got := ClientIDForHost("nonexistent.host"); got != "" {
		t.Errorf("ClientIDForHost(nonexistent) = %q, want empty", got)
	}
}

func TestSaveHosts_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "config")
	resetConfigDir(t, nestedDir)

	hosts := HostsConfig{
		"gitlab.com": &HostConfig{Token: "token"},
	}
	if err := SaveHosts(hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(nestedDir, "hosts.json"))
	if err != nil {
		t.Fatalf("reading hosts.json: %v", err)
	}
	if len(data) == 0 {
		t.Error("hosts.json is empty")
	}
}

func TestDefaultHost(t *testing.T) {
	tests := []struct {
		name           string
		configDefault  string
		envVar         string
		expectedResult string
	}{
		{
			name:           "stored config takes priority",
			configDefault:  "gitlab.example.com",
			envVar:         "env.gitlab.com",
			expectedResult: "gitlab.example.com",
		},
		{
			name:           "env var when no stored config",
			configDefault:  "",
			envVar:         "env.gitlab.com",
			expectedResult: "env.gitlab.com",
		},
		{
			name:           "fallback to gitlab.com",
			configDefault:  "",
			envVar:         "",
			expectedResult: "gitlab.com",
		},
		{
			name:           "stored config without env var",
			configDefault:  "custom.gitlab.com",
			envVar:         "",
			expectedResult: "custom.gitlab.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			resetConfigDir(t, tmpDir)

			// Set up config file if configDefault is specified
			if tt.configDefault != "" {
				cfg := &Config{
					DefaultHost: tt.configDefault,
				}
				if err := cfg.Save(); err != nil {
					t.Fatalf("Save config: %v", err)
				}
			}

			// Set environment variable if specified
			if tt.envVar != "" {
				t.Setenv("GITLAB_HOST", tt.envVar)
			}

			result := DefaultHost()
			if result != tt.expectedResult {
				t.Errorf("DefaultHost() = %q, want %q", result, tt.expectedResult)
			}
		})
	}
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// resetConfigDir resets the sync.Once so each test can use its own temp dir.
func resetConfigDir(t *testing.T, dir string) {
	t.Helper()
	configOnce = sync.Once{}
	configDir = ""
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

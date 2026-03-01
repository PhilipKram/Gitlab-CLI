package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

var testConfigDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "api-test-*")
	if err != nil {
		panic(err)
	}
	testConfigDir = dir
	os.Setenv("GLAB_CONFIG_DIR", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func writeTestHosts(t *testing.T, hosts config.HostsConfig) {
	t.Helper()
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		t.Fatalf("marshaling test hosts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testConfigDir, "hosts.json"), data, 0o600); err != nil {
		t.Fatalf("writing test hosts.json: %v", err)
	}
}

func clearTestHosts(t *testing.T) {
	t.Helper()
	os.Remove(filepath.Join(testConfigDir, "hosts.json"))
}

func TestRefreshOAuthTokenIfNeeded_NotExpired(t *testing.T) {
	futureExpiry := time.Now().Add(2 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:          "current-token",
			RefreshToken:   "refresh-token",
			TokenExpiresAt: futureExpiry,
			TokenCreatedAt: futureExpiry - 7200,
			AuthMethod:     "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected current token when not expired, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_NoExpirySet(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:      "current-token",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected current token when no expiry set, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_HostNotFound(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	result := RefreshOAuthTokenIfNeeded("nonexistent.host", "current-token")
	if result != "current-token" {
		t.Errorf("expected current token when host not found, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_ExpiredFallsBack(t *testing.T) {
	pastExpiry := time.Now().Add(-1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:          "current-token",
			RefreshToken:   "refresh-token",
			TokenExpiresAt: pastExpiry,
			TokenCreatedAt: pastExpiry - 7200,
			AuthMethod:     "oauth",
			ClientID:       "client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// No mock server, so the HTTP call will fail.
	// The function should fall back to the current token.
	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected fallback to current token on refresh failure, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_ExpiresWithin5Minutes(t *testing.T) {
	// Token that expires in 3 minutes should trigger a refresh attempt.
	soonExpiry := time.Now().Add(3 * time.Minute).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:          "current-token",
			RefreshToken:   "refresh-token",
			TokenExpiresAt: soonExpiry,
			TokenCreatedAt: soonExpiry - 7200,
			AuthMethod:     "oauth",
			ClientID:       "client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// No mock server — refresh will fail, should fall back to current token.
	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected fallback to current token, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_NotExpiredWithin5MinBuffer(t *testing.T) {
	// Token that expires in 10 minutes should NOT trigger refresh.
	laterExpiry := time.Now().Add(10 * time.Minute).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:          "current-token",
			RefreshToken:   "refresh-token",
			TokenExpiresAt: laterExpiry,
			TokenCreatedAt: laterExpiry - 7200,
			AuthMethod:     "oauth",
			ClientID:       "client-id",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected current token when expiry is >5min away, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_NoHostsFile(t *testing.T) {
	// Remove hosts.json entirely
	clearTestHosts(t)

	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected current token when no hosts file, got %q", result)
	}
}

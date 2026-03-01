package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// failTransport replaces http.DefaultTransport so that all HTTPS requests
// to the given host return an immediate error instead of hitting real DNS.
func failTransport(t *testing.T, targetHost string) {
	t.Helper()
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == targetHost {
			return nil, fmt.Errorf("mock: connection refused")
		}
		return orig.RoundTrip(req)
	})
}

// interceptTransport replaces http.DefaultTransport so that HTTPS requests
// to targetHost are rewritten to hit the test server instead.
func interceptTransport(t *testing.T, targetHost string, srv *httptest.Server) {
	t.Helper()
	srvURL, _ := url.Parse(srv.URL)
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == targetHost {
			req.URL.Scheme = srvURL.Scheme
			req.URL.Host = srvURL.Host
		}
		return orig.RoundTrip(req)
	})
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
	testHost := "gitlab.test.local"
	pastExpiry := time.Now().Add(-1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
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
	failTransport(t, testHost)

	result := RefreshOAuthTokenIfNeeded(testHost, "current-token")
	if result != "current-token" {
		t.Errorf("expected fallback to current token on refresh failure, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_ExpiresWithin5Minutes(t *testing.T) {
	testHost := "gitlab.test.local"
	soonExpiry := time.Now().Add(3 * time.Minute).Unix()

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
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
	failTransport(t, testHost)

	result := RefreshOAuthTokenIfNeeded(testHost, "current-token")
	if result != "current-token" {
		t.Errorf("expected fallback to current token, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_NotExpiredWithin5MinBuffer(t *testing.T) {
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
	clearTestHosts(t)

	result := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if result != "current-token" {
		t.Errorf("expected current token when no hosts file, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_EnvTokenSkipsRefresh(t *testing.T) {
	testHost := "gitlab.test.local"
	pastExpiry := time.Now().Add(-1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "stored-token",
			RefreshToken:   "refresh-token",
			TokenExpiresAt: pastExpiry,
			TokenCreatedAt: pastExpiry - 7200,
			AuthMethod:     "oauth",
			ClientID:       "client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Pass a different token (simulating env override) — should skip refresh
	result := RefreshOAuthTokenIfNeeded(testHost, "env-provided-token")
	if result != "env-provided-token" {
		t.Errorf("expected env token returned unchanged, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_ExpiredSuccessfulRefresh(t *testing.T) {
	testHost := "gitlab.test.local"
	now := time.Now().Unix()
	pastExpiry := now - 3600

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "old-token",
			RefreshToken:   "old-refresh",
			TokenExpiresAt: pastExpiry,
			TokenCreatedAt: pastExpiry - 7200,
			AuthMethod:     "oauth",
			ClientID:       "client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"access_token":  "new-token",
			"token_type":    "bearer",
			"expires_in":    7200,
			"refresh_token": "new-refresh",
			"created_at":    now,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	result := RefreshOAuthTokenIfNeeded(testHost, "old-token")
	if result != "new-token" {
		t.Errorf("expected refreshed token, got %q", result)
	}
}

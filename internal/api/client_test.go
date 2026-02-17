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
	_ = os.Setenv("GLAB_CONFIG_DIR", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
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
	_ = os.Remove(filepath.Join(testConfigDir, "hosts.json"))
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

	result, err := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	result, err := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "current-token" {
		t.Errorf("expected current token when no expiry set, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_HostNotFound(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	result, err := RefreshOAuthTokenIfNeeded("nonexistent.host", "current-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "current-token" {
		t.Errorf("expected current token when host not found, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_ExpiredRefreshFails(t *testing.T) {
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

	_, err := RefreshOAuthTokenIfNeeded(testHost, "current-token")
	if err == nil {
		t.Error("expected error when token is expired and refresh fails")
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

	// Token is within 5-min buffer but not yet expired, so should fall back gracefully
	result, err := RefreshOAuthTokenIfNeeded(testHost, "current-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	result, err := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "current-token" {
		t.Errorf("expected current token when expiry is >5min away, got %q", result)
	}
}

func TestRefreshOAuthTokenIfNeeded_NoHostsFile(t *testing.T) {
	clearTestHosts(t)

	result, err := RefreshOAuthTokenIfNeeded("gitlab.test.local", "current-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	result, err := RefreshOAuthTokenIfNeeded(testHost, "env-provided-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "env-provided-token" {
		t.Errorf("expected env token returned unchanged, got %q", result)
	}
}

func TestAPIURL(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"gitlab.com", "https://gitlab.com/api/v4"},
		{"gitlab.example.com", "https://gitlab.example.com/api/v4"},
		{"my-gitlab.internal", "https://my-gitlab.internal/api/v4"},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := APIURL(tt.host)
			if got != tt.want {
				t.Errorf("APIURL(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestWebURL(t *testing.T) {
	tests := []struct {
		host string
		path string
		want string
	}{
		{"gitlab.com", "user/repo", "https://gitlab.com/user/repo"},
		{"gitlab.example.com", "group/project/-/merge_requests/1", "https://gitlab.example.com/group/project/-/merge_requests/1"},
		{"my-host", "", "https://my-host/"},
	}
	for _, tt := range tests {
		t.Run(tt.host+"/"+tt.path, func(t *testing.T) {
			got := WebURL(tt.host, tt.path)
			if got != tt.want {
				t.Errorf("WebURL(%q, %q) = %q, want %q", tt.host, tt.path, got, tt.want)
			}
		})
	}
}

func TestClientHost(t *testing.T) {
	c := &Client{host: "gitlab.example.com"}
	if got := c.Host(); got != "gitlab.example.com" {
		t.Errorf("Host() = %q, want %q", got, "gitlab.example.com")
	}
}

func TestNewClientWithToken(t *testing.T) {
	// Create a test server that serves as a GitLab API
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	srvURL, _ := url.Parse(srv.URL)
	testHost := "gitlab.tokentest.local"
	interceptTransport(t, testHost, srv)

	client, err := NewClientWithToken(testHost, "test-token-123")
	if err != nil {
		t.Fatalf("NewClientWithToken returned error: %v", err)
	}
	if client == nil {
		t.Fatal("NewClientWithToken returned nil client")
	}
	if client.Host() != testHost {
		t.Errorf("client.Host() = %q, want %q", client.Host(), testHost)
	}
	_ = srvURL
}

func TestNewOAuthClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	testHost := "gitlab.oauthtest.local"
	interceptTransport(t, testHost, srv)

	client, err := NewOAuthClient(testHost, "oauth-token-123")
	if err != nil {
		t.Fatalf("NewOAuthClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("NewOAuthClient returned nil client")
	}
	if client.Host() != testHost {
		t.Errorf("client.Host() = %q, want %q", client.Host(), testHost)
	}
}

func TestGetVersion_NoHostsFile(t *testing.T) {
	clearTestHosts(t)
	c := &Client{host: "nonexistent.host"}
	got := c.GetVersion()
	if got != "" {
		t.Errorf("GetVersion() with no hosts file = %q, want empty string", got)
	}
}

func TestGetVersion_HostNotFound(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{
		"other.host": &config.HostConfig{
			Token:         "token",
			GitLabVersion: "16.0.0",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	c := &Client{host: "missing.host"}
	got := c.GetVersion()
	if got != "" {
		t.Errorf("GetVersion() with missing host = %q, want empty string", got)
	}
}

func TestGetVersion_WithCachedVersion(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:         "token",
			GitLabVersion: "16.5.2",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	c := &Client{host: "gitlab.test.local"}
	got := c.GetVersion()
	if got != "16.5.2" {
		t.Errorf("GetVersion() = %q, want %q", got, "16.5.2")
	}
}

func TestGetVersion_EmptyVersion(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:         "token",
			GitLabVersion: "",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	c := &Client{host: "gitlab.test.local"}
	got := c.GetVersion()
	if got != "" {
		t.Errorf("GetVersion() = %q, want empty string", got)
	}
}

func TestNewClientFromHosts_NoHostsFile(t *testing.T) {
	clearTestHosts(t)
	_, err := NewClientFromHosts()
	if err == nil {
		t.Error("NewClientFromHosts should error when no hosts file exists")
	}
}

func TestNewClientFromHosts_EmptyHosts(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := NewClientFromHosts()
	if err == nil {
		t.Error("NewClientFromHosts should error when hosts map is empty")
	}
}

func TestNewClient_InvalidHost(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"host with scheme", "https://gitlab.com"},
		{"host with path", "gitlab.com/api"},
		{"host with port", "gitlab.com:443"},
		{"host with at sign", "user@gitlab.com"},
		{"host with query", "gitlab.com?foo=bar"},
		{"host with fragment", "gitlab.com#section"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.host)
			if err == nil {
				t.Errorf("NewClient(%q) should return error for invalid host", tt.host)
			}
		})
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

	result, err := RefreshOAuthTokenIfNeeded(testHost, "old-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-token" {
		t.Errorf("expected refreshed token, got %q", result)
	}
}

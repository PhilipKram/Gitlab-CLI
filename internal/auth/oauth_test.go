package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

// interceptTransport replaces http.DefaultTransport so that requests to
// https://<targetHost>/... are rewritten to hit the test server instead.
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRefreshOAuthToken_Success(t *testing.T) {
	testHost := "gitlab.test.local"
	now := time.Now().Unix()

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "old-access-token",
			RefreshToken:   "old-refresh-token",
			TokenExpiresAt: now - 600, // expired 10 min ago
			TokenCreatedAt: now - 8000,
			User:           "testuser",
			AuthMethod:     "oauth",
			ClientID:       "test-client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Mock OAuth token endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want %q", r.FormValue("grant_type"), "refresh_token")
		}
		if r.FormValue("refresh_token") != "old-refresh-token" {
			t.Errorf("refresh_token = %q, want %q", r.FormValue("refresh_token"), "old-refresh-token")
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("client_id = %q, want %q", r.FormValue("client_id"), "test-client-id")
		}
		if r.FormValue("redirect_uri") != "http://localhost:7171/auth/redirect" {
			t.Errorf("redirect_uri = %q, want %q", r.FormValue("redirect_uri"), "http://localhost:7171/auth/redirect")
		}

		resp := OAuthTokenResponse{
			AccessToken:  "new-access-token",
			TokenType:    "bearer",
			ExpiresIn:    7200,
			RefreshToken: "new-refresh-token",
			Scope:        "api",
			CreatedAt:    now,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	newToken, err := RefreshOAuthToken(testHost)
	if err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}
	if newToken != "new-access-token" {
		t.Errorf("new token = %q, want %q", newToken, "new-access-token")
	}

	// Verify the config was updated on disk
	data, err := os.ReadFile(filepath.Join(testConfigDir, "hosts.json"))
	if err != nil {
		t.Fatalf("reading hosts.json: %v", err)
	}
	var hosts config.HostsConfig
	if err := json.Unmarshal(data, &hosts); err != nil {
		t.Fatalf("parsing hosts.json: %v", err)
	}

	hc := hosts[testHost]
	if hc.Token != "new-access-token" {
		t.Errorf("saved Token = %q, want %q", hc.Token, "new-access-token")
	}
	if hc.RefreshToken != "new-refresh-token" {
		t.Errorf("saved RefreshToken = %q, want %q", hc.RefreshToken, "new-refresh-token")
	}
	if hc.TokenCreatedAt != now {
		t.Errorf("saved TokenCreatedAt = %d, want %d", hc.TokenCreatedAt, now)
	}
	if hc.TokenExpiresAt != now+7200 {
		t.Errorf("saved TokenExpiresAt = %d, want %d", hc.TokenExpiresAt, now+7200)
	}
}

func TestRefreshOAuthToken_NoHostConfig(t *testing.T) {
	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := RefreshOAuthToken("nonexistent.host")
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
	if got := err.Error(); got != "no configuration for host: nonexistent.host" {
		t.Errorf("error = %q, want 'no configuration for host: nonexistent.host'", got)
	}
}

func TestRefreshOAuthToken_NoRefreshToken(t *testing.T) {
	testHost := "gitlab.test.local"
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:      "access-token",
			AuthMethod: "oauth",
			ClientID:   "client-id",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := RefreshOAuthToken(testHost)
	if err == nil {
		t.Fatal("expected error when no refresh token stored")
	}
	expected := fmt.Sprintf("no refresh token stored for %s; run 'glab auth login' to re-authenticate", testHost)
	if got := err.Error(); got != expected {
		t.Errorf("error = %q, want %q", got, expected)
	}
}

func TestRefreshOAuthToken_ServerError(t *testing.T) {
	testHost := "gitlab.test.local"
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:        "old-access-token",
			RefreshToken: "old-refresh-token",
			AuthMethod:   "oauth",
			ClientID:     "client-id",
			RedirectURI:  "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	_, err := RefreshOAuthToken(testHost)
	if err == nil {
		t.Fatal("expected error on server error response")
	}
	if got := err.Error(); !strings.Contains(got, "token refresh failed (HTTP 401)") {
		t.Errorf("error = %q, want it to contain 'token refresh failed (HTTP 401)'", got)
	}
}

func TestRefreshOAuthToken_InvalidJSON(t *testing.T) {
	testHost := "gitlab.test.local"
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:        "old-access-token",
			RefreshToken: "old-refresh-token",
			AuthMethod:   "oauth",
			ClientID:     "client-id",
			RedirectURI:  "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "not json")
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	_, err := RefreshOAuthToken(testHost)
	if err == nil {
		t.Fatal("expected error on invalid JSON response")
	}
	if got := err.Error(); !strings.Contains(got, "parsing token refresh response") {
		t.Errorf("error = %q, want it to contain 'parsing token refresh response'", got)
	}
}

func TestRefreshOAuthToken_PreservesExistingConfig(t *testing.T) {
	testHost := "gitlab.test.local"
	now := time.Now().Unix()

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "old-access-token",
			RefreshToken:   "old-refresh-token",
			TokenExpiresAt: now - 600,
			TokenCreatedAt: now - 8000,
			User:           "testuser",
			AuthMethod:     "oauth",
			ClientID:       "my-client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
			OAuthScopes:    "api read_user",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OAuthTokenResponse{
			AccessToken:  "refreshed-token",
			TokenType:    "bearer",
			ExpiresIn:    7200,
			RefreshToken: "refreshed-refresh",
			CreatedAt:    now,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	_, err := RefreshOAuthToken(testHost)
	if err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}

	// Verify non-token fields are preserved
	data, err := os.ReadFile(filepath.Join(testConfigDir, "hosts.json"))
	if err != nil {
		t.Fatalf("reading hosts.json: %v", err)
	}
	var hosts config.HostsConfig
	if err := json.Unmarshal(data, &hosts); err != nil {
		t.Fatalf("parsing hosts.json: %v", err)
	}

	hc := hosts[testHost]
	if hc.User != "testuser" {
		t.Errorf("User = %q, want %q (should be preserved)", hc.User, "testuser")
	}
	if hc.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q, want %q (should be preserved)", hc.ClientID, "my-client-id")
	}
	if hc.AuthMethod != "oauth" {
		t.Errorf("AuthMethod = %q, want %q (should be preserved)", hc.AuthMethod, "oauth")
	}
	if hc.OAuthScopes != "api read_user" {
		t.Errorf("OAuthScopes = %q, want %q (should be preserved)", hc.OAuthScopes, "api read_user")
	}
}

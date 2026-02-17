package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

func TestRefreshOAuthToken_ZeroCreatedAt(t *testing.T) {
	testHost := "gitlab.zero-created.local"
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "old-access-token",
			RefreshToken:   "old-refresh-token",
			TokenExpiresAt: 1000,
			TokenCreatedAt: 0,
			User:           "testuser",
			AuthMethod:     "oauth",
			ClientID:       "test-client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Mock OAuth token endpoint with no created_at in response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OAuthTokenResponse{
			AccessToken:  "new-token",
			TokenType:    "bearer",
			ExpiresIn:    7200,
			RefreshToken: "new-refresh",
			Scope:        "api",
			CreatedAt:    0, // Zero - should use time.Now()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	newToken, err := RefreshOAuthToken(testHost)
	if err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}
	if newToken != "new-token" {
		t.Errorf("token = %q, want %q", newToken, "new-token")
	}

	// Verify TokenCreatedAt was set to something non-zero
	hosts, err := config.LoadHosts()
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}
	hc := hosts[testHost]
	if hc.TokenCreatedAt == 0 {
		t.Error("TokenCreatedAt should not be 0 when response CreatedAt is 0")
	}
}

func TestRefreshOAuthToken_NoExpiresIn(t *testing.T) {
	testHost := "gitlab.no-expiry.local"
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "old-access-token",
			RefreshToken:   "old-refresh-token",
			TokenExpiresAt: 1000,
			TokenCreatedAt: 500,
			User:           "testuser",
			AuthMethod:     "oauth",
			ClientID:       "test-client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OAuthTokenResponse{
			AccessToken:  "new-token",
			TokenType:    "bearer",
			ExpiresIn:    0, // No expiry info
			RefreshToken: "new-refresh",
			Scope:        "api",
			CreatedAt:    2000,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	newToken, err := RefreshOAuthToken(testHost)
	if err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}
	if newToken != "new-token" {
		t.Errorf("token = %q, want %q", newToken, "new-token")
	}
}

func TestRefreshOAuthToken_KeepsOldRefreshToken(t *testing.T) {
	testHost := "gitlab.keep-refresh.local"
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:          "old-access",
			RefreshToken:   "old-refresh",
			TokenExpiresAt: 1000,
			User:           "testuser",
			AuthMethod:     "oauth",
			ClientID:       "client-id",
			RedirectURI:    "http://localhost:7171/auth/redirect",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OAuthTokenResponse{
			AccessToken:  "new-access",
			TokenType:    "bearer",
			ExpiresIn:    7200,
			RefreshToken: "", // Empty - should keep old one
			CreatedAt:    2000,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	_, err := RefreshOAuthToken(testHost)
	if err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}

	hosts, err := config.LoadHosts()
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}
	hc := hosts[testHost]
	if hc.RefreshToken != "old-refresh" {
		t.Errorf("RefreshToken = %q, want %q (should keep old when new is empty)", hc.RefreshToken, "old-refresh")
	}
}


func TestGetStatus_OAuthScopes(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.scoped.com": &config.HostConfig{
			Token:       "oauth-token-12345678",
			User:        "scopeuser",
			AuthMethod:  "oauth",
			OAuthScopes: "api read_user write_repository",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Host == "gitlab.scoped.com" {
			found = true
			if s.Scopes != "api read_user write_repository" {
				t.Errorf("Scopes = %q, want %q", s.Scopes, "api read_user write_repository")
			}
		}
	}
	if !found {
		t.Error("expected to find gitlab.scoped.com in statuses")
	}
}

func TestGetStatus_GitLabVersion(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.versioned.com": &config.HostConfig{
			Token:         "pat-token-12345678",
			User:          "versionuser",
			AuthMethod:    "pat",
			GitLabVersion: "16.5.2",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Host == "gitlab.versioned.com" {
			found = true
			if s.GitLabVersion != "16.5.2" {
				t.Errorf("GitLabVersion = %q, want %q", s.GitLabVersion, "16.5.2")
			}
		}
	}
	if !found {
		t.Error("expected to find gitlab.versioned.com in statuses")
	}
}

func TestGetStatus_MixedExpiredAndValid(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	futureAt := time.Now().Add(1 * time.Hour).Unix()
	pastAt := time.Now().Add(-1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"valid.host": &config.HostConfig{
			Token:          "valid-token-12345678",
			User:           "validuser",
			AuthMethod:     "oauth",
			TokenExpiresAt: futureAt,
		},
		"expired.host": &config.HostConfig{
			Token:          "expired-token-12345678",
			User:           "expireduser",
			AuthMethod:     "pat",
			TokenExpiresAt: pastAt,
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}

	for _, s := range statuses {
		switch s.Host {
		case "valid.host":
			if !s.Active {
				t.Error("valid.host should be Active")
			}
			if s.HasError {
				t.Error("valid.host should not HasError")
			}
		case "expired.host":
			if s.Active {
				t.Error("expired.host should not be Active")
			}
			if !s.HasError {
				t.Error("expired.host should HasError")
			}
			if !strings.Contains(s.Error, "token expired") {
				t.Errorf("expired.host Error = %q, want to contain 'token expired'", s.Error)
			}
		}
	}
}

func TestGetToken_FromEnv(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "env-token-value-12345")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	// GetToken for the default host should return env token
	token, err := GetToken("gitlab.com")
	if err != nil {
		t.Fatalf("GetToken from env: %v", err)
	}
	if token != "env-token-value-12345" {
		t.Errorf("GetToken() = %q, want %q", token, "env-token-value-12345")
	}
}

func TestGetToken_GlabTokenEnv(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "glab-env-token-12345")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	token, err := GetToken("gitlab.com")
	if err != nil {
		t.Fatalf("GetToken from GLAB_TOKEN env: %v", err)
	}
	if token != "glab-env-token-12345" {
		t.Errorf("GetToken() = %q, want %q", token, "glab-env-token-12345")
	}
}

func TestLogin_NoTokenNoStdin(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	_, err := Login("gitlab.com", "", nil)
	if err == nil {
		t.Fatal("expected Login to fail with no token and no stdin")
	}
	if !strings.Contains(err.Error(), "No token provided") {
		t.Errorf("error = %q, want to contain 'No token provided'", err.Error())
	}
}

func TestLogin_EmptyStdinToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	stdin := strings.NewReader("\n")
	_, err := Login("gitlab.com", "", stdin)
	if err == nil {
		t.Fatal("expected Login to fail with empty stdin line")
	}
	if !strings.Contains(err.Error(), "No token provided") {
		t.Errorf("error = %q, want to contain 'No token provided'", err.Error())
	}
}

func TestSwitch_NoHosts(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := Switch(nil, nil)
	if err == nil {
		t.Fatal("expected error when no hosts")
	}
	if !strings.Contains(err.Error(), "no authenticated hosts") {
		t.Errorf("error = %q, want to contain 'no authenticated hosts'", err.Error())
	}
}

func TestSwitch_SingleHost(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "token-12345",
			User:       "user1",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := Switch(nil, nil)
	if err == nil {
		t.Fatal("expected error when only one host")
	}
	if !strings.Contains(err.Error(), "only one authenticated host") {
		t.Errorf("error = %q, want to contain 'only one authenticated host'", err.Error())
	}
}


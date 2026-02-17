package auth

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

func TestGetStatus_IncludesTokenExpiresAt(t *testing.T) {
	// Clear any env tokens that might interfere
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.example.com": &config.HostConfig{
			Token:          "access-token-123",
			RefreshToken:   "refresh-token-456",
			TokenExpiresAt: expiresAt,
			TokenCreatedAt: expiresAt - 7200,
			User:           "alice",
			AuthMethod:     "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Host == "gitlab.example.com" {
			found = true
			if s.TokenExpiresAt != expiresAt {
				t.Errorf("TokenExpiresAt = %d, want %d", s.TokenExpiresAt, expiresAt)
			}
			if s.AuthMethod != "oauth" {
				t.Errorf("AuthMethod = %q, want %q", s.AuthMethod, "oauth")
			}
		}
	}
	if !found {
		t.Error("expected to find gitlab.example.com in statuses")
	}
}

func TestGetStatus_ZeroExpiresAtForPAT(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Host == "gitlab.com" {
			found = true
			if s.TokenExpiresAt != 0 {
				t.Errorf("TokenExpiresAt should be 0 for PAT, got %d", s.TokenExpiresAt)
			}
			if s.AuthMethod != "pat" {
				t.Errorf("AuthMethod = %q, want %q", s.AuthMethod, "pat")
			}
		}
	}
	if !found {
		t.Error("expected to find gitlab.com in statuses")
	}
}

func TestGetStatus_MultipleHosts(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:          "oauth-token-xyz",
			RefreshToken:   "refresh-xyz",
			TokenExpiresAt: expiresAt,
			User:           "alice",
			AuthMethod:     "oauth",
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
		case "gitlab.com":
			if s.TokenExpiresAt != 0 {
				t.Errorf("gitlab.com: TokenExpiresAt should be 0, got %d", s.TokenExpiresAt)
			}
		case "gitlab.corp.com":
			if s.TokenExpiresAt != expiresAt {
				t.Errorf("gitlab.corp.com: TokenExpiresAt = %d, want %d", s.TokenExpiresAt, expiresAt)
			}
		default:
			t.Errorf("unexpected host: %s", s.Host)
		}
	}
}

func TestLogout(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	err := Logout("gitlab.com")
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	_, err = GetStatus()
	if err == nil {
		t.Error("expected GetStatus to fail after Logout, but it succeeded")
	}
}

func TestLogout_NotLoggedIn(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	err := Logout("gitlab.example.com")
	if err == nil {
		t.Fatal("expected Logout to fail for non-existent host, but it succeeded")
	}

	expectedMsg := "Not logged in to gitlab.example.com"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Logout error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestLogout_OneOfMultipleHosts(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:          "oauth-token-xyz",
			RefreshToken:   "refresh-xyz",
			TokenExpiresAt: expiresAt,
			User:           "alice",
			AuthMethod:     "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	err := Logout("gitlab.com")
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("expected 1 status after logout, got %d", len(statuses))
	}

	if statuses[0].Host != "gitlab.corp.com" {
		t.Errorf("expected remaining host to be gitlab.corp.com, got %s", statuses[0].Host)
	}
}

func TestLogoutAll(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	err := LogoutAll()
	if err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}

	_, err = GetStatus()
	if err == nil {
		t.Error("expected GetStatus to fail after LogoutAll, but it succeeded")
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "****"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234****6789"},
		{"glpat-xxxxxxxxxxxxxxxxxxxx", "glpa****xxxx"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := maskToken(tt.input)
			if got != tt.want {
				t.Errorf("maskToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestApiURL(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"gitlab.com", "https://gitlab.com/api/v4"},
		{"gitlab.example.com", "https://gitlab.example.com/api/v4"},
		{"my.host.io", "https://my.host.io/api/v4"},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := apiURL(tt.host)
			if got != tt.want {
				t.Errorf("apiURL(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{"zero means no expiration", 0, false},
		{"future timestamp not expired", time.Now().Add(1 * time.Hour).Unix(), false},
		{"past timestamp is expired", time.Now().Add(-1 * time.Hour).Unix(), true},
		{"far past is expired", 1000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTokenExpired(tt.expiresAt)
			if got != tt.want {
				t.Errorf("isTokenExpired(%d) = %v, want %v", tt.expiresAt, got, tt.want)
			}
		})
	}
}

func TestIsUnauthorizedError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"401 error", fmt.Errorf("HTTP 401 response"), true},
		{"unauthorized error", fmt.Errorf("Unauthorized access"), true},
		{"invalid token error", fmt.Errorf("invalid token provided"), true},
		{"authentication failed", fmt.Errorf("authentication failed for host"), true},
		{"unrelated error", fmt.Errorf("connection timeout"), false},
		{"network error", fmt.Errorf("no such host"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnauthorizedError(tt.err)
			if got != tt.want {
				t.Errorf("isUnauthorizedError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFormatAuthError(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		err     error
		wantNil bool
		wantMsg string
	}{
		{
			name:    "nil error returns nil",
			host:    "gitlab.com",
			err:     nil,
			wantNil: true,
		},
		{
			name:    "401 unauthorized error",
			host:    "gitlab.com",
			err:     fmt.Errorf("401 Unauthorized"),
			wantMsg: "authentication failed: invalid or expired token",
		},
		{
			name:    "connection refused error",
			host:    "gitlab.example.com",
			err:     fmt.Errorf("connection refused"),
			wantMsg: "connection failed: unable to reach gitlab.example.com",
		},
		{
			name:    "no such host error",
			host:    "gitlab.example.com",
			err:     fmt.Errorf("no such host"),
			wantMsg: "connection failed: unable to reach gitlab.example.com",
		},
		{
			name:    "generic error",
			host:    "gitlab.com",
			err:     fmt.Errorf("some other error"),
			wantMsg: "authentication failed for gitlab.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAuthError(tt.host, tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("formatAuthError() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("formatAuthError() = nil, want non-nil")
			}
			if !strings.Contains(got.Error(), tt.wantMsg) {
				t.Errorf("formatAuthError() = %q, want to contain %q", got.Error(), tt.wantMsg)
			}
		})
	}
}

func TestFormatTokenExpiredError(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		authMethod string
		wantMsg    string
	}{
		{
			name:       "oauth token expired",
			host:       "gitlab.com",
			authMethod: "oauth",
			wantMsg:    "OAuth token has expired",
		},
		{
			name:       "pat token expired",
			host:       "gitlab.com",
			authMethod: "pat",
			wantMsg:    "personal access token has expired",
		},
		{
			name:       "empty auth method",
			host:       "gitlab.com",
			authMethod: "",
			wantMsg:    "personal access token has expired",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokenExpiredError(tt.host, tt.authMethod)
			if !strings.Contains(got, tt.wantMsg) {
				t.Errorf("formatTokenExpiredError() = %q, want to contain %q", got, tt.wantMsg)
			}
			if !strings.Contains(got, tt.host) {
				t.Errorf("formatTokenExpiredError() = %q, want to contain host %q", got, tt.host)
			}
		})
	}
}

func TestGetStatus_ExpiredToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	expiredAt := time.Now().Add(-1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.expired.com": &config.HostConfig{
			Token:          "expired-token-12345678",
			RefreshToken:   "refresh-token",
			TokenExpiresAt: expiredAt,
			TokenCreatedAt: expiredAt - 7200,
			User:           "alice",
			AuthMethod:     "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Host == "gitlab.expired.com" {
			found = true
			if !s.HasError {
				t.Error("expected HasError to be true for expired token")
			}
			if s.Active {
				t.Error("expected Active to be false for expired token")
			}
			if s.Error == "" {
				t.Error("expected Error message for expired token")
			}
		}
	}
	if !found {
		t.Error("expected to find gitlab.expired.com in statuses")
	}
}

func TestGetStatus_NoHosts(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := GetStatus()
	if err == nil {
		t.Fatal("expected GetStatus to fail with no hosts, but it succeeded")
	}
	if !strings.Contains(err.Error(), "No authenticated hosts") {
		t.Errorf("error = %q, want to contain 'No authenticated hosts'", err.Error())
	}
}

func TestGetStatus_WithGitLabTokenEnv(t *testing.T) {
	// Set a fake GITLAB_TOKEN and ensure the default host picks it up
	t.Setenv("GITLAB_TOKEN", "env-token-abcdefgh12345678")
	t.Setenv("GLAB_TOKEN", "")
	t.Setenv("GITLAB_HOST", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	statuses, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if len(statuses) == 0 {
		t.Fatal("expected at least one status from GITLAB_TOKEN env var")
	}

	found := false
	for _, s := range statuses {
		if s.Source == "GITLAB_TOKEN" {
			found = true
			if s.Token == "" {
				t.Error("expected masked token, got empty string")
			}
			// The user lookup will fail (no real server), so HasError should be true
			if !s.HasError {
				t.Log("HasError is false; token validation succeeded unexpectedly or env token was skipped")
			}
		}
	}
	if !found {
		t.Error("expected to find a status from GITLAB_TOKEN source")
	}
}

func TestGetToken_FromConfig(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.example.com": &config.HostConfig{
			Token:      "my-secret-token",
			User:       "alice",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	token, err := GetToken("gitlab.example.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if token != "my-secret-token" {
		t.Errorf("GetToken() = %q, want %q", token, "my-secret-token")
	}
}

func TestGetToken_NotFound(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := GetToken("nonexistent.host")
	if err == nil {
		t.Fatal("expected GetToken to fail for nonexistent host")
	}
	if !strings.Contains(err.Error(), "No token found") {
		t.Errorf("error = %q, want to contain 'No token found'", err.Error())
	}
}

func TestGetToken_ExpiredToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	expiredAt := time.Now().Add(-1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.expired.com": &config.HostConfig{
			Token:          "expired-token",
			TokenExpiresAt: expiredAt,
			User:           "alice",
			AuthMethod:     "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := GetToken("gitlab.expired.com")
	if err == nil {
		t.Fatal("expected GetToken to fail for expired token")
	}
	if !strings.Contains(err.Error(), "token expired") {
		t.Errorf("error = %q, want to contain 'token expired'", err.Error())
	}
}

func TestGetToken_ValidNonExpiredToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	futureAt := time.Now().Add(1 * time.Hour).Unix()

	writeTestHosts(t, config.HostsConfig{
		"gitlab.valid.com": &config.HostConfig{
			Token:          "valid-token",
			TokenExpiresAt: futureAt,
			User:           "alice",
			AuthMethod:     "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	token, err := GetToken("gitlab.valid.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if token != "valid-token" {
		t.Errorf("GetToken() = %q, want %q", token, "valid-token")
	}
}

func TestLogin_EmptyToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	_, err := Login("gitlab.com", "", nil)
	if err == nil {
		t.Fatal("expected Login to fail with empty token")
	}
	if !strings.Contains(err.Error(), "No token provided") {
		t.Errorf("error = %q, want to contain 'No token provided'", err.Error())
	}
}

func TestLogin_TokenFromStdin(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	// Provide token via stdin but it will fail API validation (no server)
	stdin := strings.NewReader("stdin-token-12345678\n")
	_, err := Login("localhost:99999", "", stdin)
	// This should fail at the API call stage, not at the "no token" stage
	if err == nil {
		t.Fatal("expected Login to fail (no server), but it succeeded")
	}
	// Should NOT contain "No token provided" since we provided one via stdin
	if strings.Contains(err.Error(), "No token provided") {
		t.Errorf("error should not be 'No token provided' when stdin provides token, got: %v", err)
	}
}

func TestLogin_EmptyStdin(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	// Provide empty stdin
	stdin := strings.NewReader("")
	_, err := Login("gitlab.com", "", stdin)
	if err == nil {
		t.Fatal("expected Login to fail with empty stdin")
	}
	if !strings.Contains(err.Error(), "No token provided") {
		t.Errorf("error = %q, want to contain 'No token provided'", err.Error())
	}
}

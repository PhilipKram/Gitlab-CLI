package auth

import (
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

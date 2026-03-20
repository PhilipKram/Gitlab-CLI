package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

func TestNewClient_SSRFProtection(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr string
	}{
		{"host with https scheme", "https://gitlab.com", "invalid host"},
		{"host with http scheme", "http://gitlab.com", "invalid host"},
		{"host with path separator", "gitlab.com/api", "invalid host"},
		{"host with port colon", "gitlab.com:443", "invalid host"},
		{"host with at sign", "user@gitlab.com", "invalid host"},
		{"host with query", "gitlab.com?foo=bar", "invalid host"},
		{"host with fragment", "gitlab.com#section", "invalid host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.host)
			if err == nil {
				t.Fatalf("NewClient(%q) should return error", tt.host)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewClient_NoToken(t *testing.T) {
	// Clear any env tokens
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	// Use empty hosts config
	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	_, err := NewClient("gitlab.com")
	if err == nil {
		t.Fatal("expected error when no token is available")
	}
	if !strings.Contains(err.Error(), "Not authenticated") {
		t.Errorf("error = %q, want to contain 'Not authenticated'", err.Error())
	}
}

func TestNewClient_WithPATToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "test-pat-token-12345")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "test-pat-token-12345",
			User:       "testuser",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	client, err := NewClient("gitlab.com")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Host() != "gitlab.com" {
		t.Errorf("Host() = %q, want %q", client.Host(), "gitlab.com")
	}
}

func TestNewClient_WithOAuthToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.test.local": &config.HostConfig{
			Token:          "oauth-token-12345",
			User:           "testuser",
			AuthMethod:     "oauth",
			TokenExpiresAt: 0, // no expiry
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Set the env token to match the stored token so NewClient works
	t.Setenv("GITLAB_TOKEN", "oauth-token-12345")

	client, err := NewClient("gitlab.test.local")
	if err != nil {
		t.Fatalf("NewClient with OAuth: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClientFromHosts_WithValidHost(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.valid.local": &config.HostConfig{
			Token:      "valid-token-12345",
			User:       "testuser",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Set env token for the host
	t.Setenv("GITLAB_TOKEN", "valid-token-12345")

	client, err := NewClientFromHosts()
	if err != nil {
		t.Fatalf("NewClientFromHosts: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestGetAndCacheVersion(t *testing.T) {
	testHost := "gitlab.version-test.local"

	// Create a mock server that responds with version info
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"version":"17.1.0","revision":"abc123"}`)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	// Set up hosts config
	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:      "test-token",
			User:       "testuser",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Create a client using the test host
	client, err := NewClientWithToken(testHost, "test-token")
	if err != nil {
		t.Fatalf("NewClientWithToken: %v", err)
	}

	version := GetAndCacheVersion(client.Client, testHost)
	if version != "17.1.0" {
		t.Errorf("GetAndCacheVersion = %q, want %q", version, "17.1.0")
	}

	// Verify it was cached in the hosts file
	data, err := os.ReadFile(filepath.Join(testConfigDir, "hosts.json"))
	if err != nil {
		t.Fatalf("reading hosts.json: %v", err)
	}
	var hosts config.HostsConfig
	if err := json.Unmarshal(data, &hosts); err != nil {
		t.Fatalf("parsing hosts.json: %v", err)
	}
	if hc, ok := hosts[testHost]; ok {
		if hc.GitLabVersion != "17.1.0" {
			t.Errorf("cached version = %q, want %q", hc.GitLabVersion, "17.1.0")
		}
	} else {
		t.Error("host not found in hosts.json after GetAndCacheVersion")
	}
}

func TestGetAndCacheVersion_APIError(t *testing.T) {
	testHost := "gitlab.version-err.local"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	writeTestHosts(t, config.HostsConfig{
		testHost: &config.HostConfig{
			Token:      "test-token",
			User:       "testuser",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	client, err := NewClientWithToken(testHost, "test-token")
	if err != nil {
		t.Fatalf("NewClientWithToken: %v", err)
	}

	version := GetAndCacheVersion(client.Client, testHost)
	if version != "" {
		t.Errorf("GetAndCacheVersion with API error = %q, want empty string", version)
	}
}

func TestGetAndCacheVersion_HostNotInConfig(t *testing.T) {
	testHost := "gitlab.notincfg.local"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/version") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"version":"16.0.0","revision":"abc"}`)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	// Write hosts without the test host
	writeTestHosts(t, config.HostsConfig{
		"other.host": &config.HostConfig{Token: "token"},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	client, err := NewClientWithToken(testHost, "test-token")
	if err != nil {
		t.Fatalf("NewClientWithToken: %v", err)
	}

	// Should return the version but not cache it
	version := GetAndCacheVersion(client.Client, testHost)
	if version != "16.0.0" {
		t.Errorf("GetAndCacheVersion = %q, want %q", version, "16.0.0")
	}
}

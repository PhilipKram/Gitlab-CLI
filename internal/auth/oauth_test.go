package auth

import (
	"encoding/json"
	"fmt"
	"net"
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
		_ = json.NewEncoder(w).Encode(resp)
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
		_, _ = fmt.Fprint(w, "not json")
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

func TestBuildAuthURL(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		clientID      string
		redirectURI   string
		state         string
		codeChallenge string
		scopes        string
		wantBase      string
		wantParams    map[string]string
	}{
		{
			name:          "standard gitlab.com",
			host:          "gitlab.com",
			clientID:      "my-client",
			redirectURI:   "http://localhost:7171/auth/redirect",
			state:         "test-state",
			codeChallenge: "test-challenge",
			scopes:        "api read_user",
			wantBase:      "https://gitlab.com/oauth/authorize",
			wantParams: map[string]string{
				"client_id":             "my-client",
				"redirect_uri":          "http://localhost:7171/auth/redirect",
				"response_type":         "code",
				"scope":                 "api read_user",
				"state":                 "test-state",
				"code_challenge":        "test-challenge",
				"code_challenge_method": "S256",
			},
		},
		{
			name:          "self-hosted gitlab",
			host:          "gitlab.example.com",
			clientID:      "other-client",
			redirectURI:   "http://localhost:8080/callback",
			state:         "abc123",
			codeChallenge: "challenge-xyz",
			scopes:        "api",
			wantBase:      "https://gitlab.example.com/oauth/authorize",
			wantParams: map[string]string{
				"client_id":             "other-client",
				"redirect_uri":          "http://localhost:8080/callback",
				"response_type":         "code",
				"scope":                 "api",
				"state":                 "abc123",
				"code_challenge":        "challenge-xyz",
				"code_challenge_method": "S256",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAuthURL(tt.host, tt.clientID, tt.redirectURI, tt.state, tt.codeChallenge, tt.scopes)
			parsed, err := url.Parse(got)
			if err != nil {
				t.Fatalf("failed to parse URL: %v", err)
			}
			base := parsed.Scheme + "://" + parsed.Host + parsed.Path
			if base != tt.wantBase {
				t.Errorf("base URL = %q, want %q", base, tt.wantBase)
			}
			for key, want := range tt.wantParams {
				got := parsed.Query().Get(key)
				if got != want {
					t.Errorf("param %q = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	v1, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}
	if len(v1) == 0 {
		t.Error("expected non-empty verifier")
	}

	v2, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}
	if v1 == v2 {
		t.Error("expected two calls to produce different verifiers")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-verifier-123"
	c1 := generateCodeChallenge(verifier)
	if len(c1) == 0 {
		t.Error("expected non-empty challenge")
	}
	// Same input should produce same output (deterministic)
	c2 := generateCodeChallenge(verifier)
	if c1 != c2 {
		t.Errorf("same verifier produced different challenges: %q vs %q", c1, c2)
	}
	// Different input should produce different output
	c3 := generateCodeChallenge("different-verifier")
	if c1 == c3 {
		t.Error("different verifiers should produce different challenges")
	}
}

func TestGenerateState(t *testing.T) {
	s1, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if len(s1) == 0 {
		t.Error("expected non-empty state")
	}

	s2, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if s1 == s2 {
		t.Error("expected two calls to produce different states")
	}
}

func TestScopesDescription(t *testing.T) {
	got := ScopesDescription()
	if !strings.Contains(got, "api") {
		t.Errorf("ScopesDescription() = %q, want to contain 'api'", got)
	}
	// Verify spaces are replaced with ", "
	if strings.Contains(got, "profile api") {
		t.Errorf("ScopesDescription() should replace spaces with ', ', got %q", got)
	}
	if !strings.Contains(got, "profile, api") {
		t.Errorf("ScopesDescription() = %q, want to contain 'profile, api'", got)
	}
}

func TestExchangeCode_ServerError(t *testing.T) {
	testHost := "gitlab.test.local"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	_, err := exchangeCode(testHost, "client-id", "code", "http://localhost:7171/auth/redirect", "verifier")
	if err == nil {
		t.Fatal("expected error on server error response")
	}
	if !strings.Contains(err.Error(), "token exchange failed (HTTP 400)") {
		t.Errorf("error = %q, want to contain 'token exchange failed (HTTP 400)'", err.Error())
	}
}

func TestExchangeCode_InvalidJSON(t *testing.T) {
	testHost := "gitlab.test.local"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	_, err := exchangeCode(testHost, "client-id", "code", "http://localhost:7171/auth/redirect", "verifier")
	if err == nil {
		t.Fatal("expected error on invalid JSON response")
	}
	if !strings.Contains(err.Error(), "parsing token response") {
		t.Errorf("error = %q, want to contain 'parsing token response'", err.Error())
	}
}

func TestExchangeCode_Success(t *testing.T) {
	testHost := "gitlab.test.local"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q, want %q", r.FormValue("grant_type"), "authorization_code")
		}
		if r.FormValue("client_id") != "my-client" {
			t.Errorf("client_id = %q, want %q", r.FormValue("client_id"), "my-client")
		}
		if r.FormValue("code") != "auth-code" {
			t.Errorf("code = %q, want %q", r.FormValue("code"), "auth-code")
		}
		resp := OAuthTokenResponse{
			AccessToken:  "new-token",
			TokenType:    "bearer",
			ExpiresIn:    7200,
			RefreshToken: "new-refresh",
			Scope:        "api",
			CreatedAt:    1700000000,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	interceptTransport(t, testHost, srv)

	tokenResp, err := exchangeCode(testHost, "my-client", "auth-code", "http://localhost:7171/auth/redirect", "my-verifier")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}
	if tokenResp.AccessToken != "new-token" {
		t.Errorf("AccessToken = %q, want %q", tokenResp.AccessToken, "new-token")
	}
	if tokenResp.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want %q", tokenResp.RefreshToken, "new-refresh")
	}
	if tokenResp.ExpiresIn != 7200 {
		t.Errorf("ExpiresIn = %d, want %d", tokenResp.ExpiresIn, 7200)
	}
}

func TestWaitForCallback_Success(t *testing.T) {
	state := "test-state-abc"
	callbackPath := "/auth/redirect"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := listener.Addr().String()

	// Run waitForCallback in a goroutine
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		code, err := waitForCallback(listener, state, callbackPath)
		if err != nil {
			errCh <- err
			return
		}
		codeCh <- code
	}()

	// Simulate the callback
	callbackURL := fmt.Sprintf("http://%s%s?code=my-auth-code&state=%s", addr, callbackPath, state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case code := <-codeCh:
		if code != "my-auth-code" {
			t.Errorf("code = %q, want %q", code, "my-auth-code")
		}
	case err := <-errCh:
		t.Fatalf("waitForCallback error: %v", err)
	}
}

func TestWaitForCallback_StateMismatch(t *testing.T) {
	state := "expected-state"
	callbackPath := "/auth/redirect"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := listener.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		_, err := waitForCallback(listener, state, callbackPath)
		errCh <- err
	}()

	// Simulate the callback with wrong state
	callbackURL := fmt.Sprintf("http://%s%s?code=my-auth-code&state=wrong-state", addr, callbackPath)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	_ = resp.Body.Close()

	err = <-errCh
	if err == nil {
		t.Fatal("expected error for state mismatch")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("error = %q, want to contain 'state mismatch'", err.Error())
	}
}

func TestWaitForCallback_ErrorFromProvider(t *testing.T) {
	state := "test-state"
	callbackPath := "/auth/redirect"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := listener.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		_, err := waitForCallback(listener, state, callbackPath)
		errCh <- err
	}()

	// Simulate callback with error from provider
	callbackURL := fmt.Sprintf("http://%s%s?error=access_denied&error_description=user+denied+access&state=%s", addr, callbackPath, state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	_ = resp.Body.Close()

	err = <-errCh
	if err == nil {
		t.Fatal("expected error for access denied")
	}
	if !strings.Contains(err.Error(), "authorization denied") {
		t.Errorf("error = %q, want to contain 'authorization denied'", err.Error())
	}
}

func TestWaitForCallback_NoCode(t *testing.T) {
	state := "test-state"
	callbackPath := "/auth/redirect"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := listener.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		_, err := waitForCallback(listener, state, callbackPath)
		errCh <- err
	}()

	// Simulate callback with no code parameter
	callbackURL := fmt.Sprintf("http://%s%s?state=%s", addr, callbackPath, state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	_ = resp.Body.Close()

	err = <-errCh
	if err == nil {
		t.Fatal("expected error for no code")
	}
	if !strings.Contains(err.Error(), "no authorization code received") {
		t.Errorf("error = %q, want to contain 'no authorization code received'", err.Error())
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
		_ = json.NewEncoder(w).Encode(resp)
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

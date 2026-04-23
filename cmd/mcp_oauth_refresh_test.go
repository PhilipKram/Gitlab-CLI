package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockGitLab is a minimal OAuth token endpoint used by the refresh tests.
// Each refresh_token grant rotates both tokens (like real GitLab) and increments
// a counter so tests can assert how many refreshes actually happened.
type mockGitLab struct {
	server       *httptest.Server
	refreshCount atomic.Int64
	accessTTL    time.Duration
	fail         atomic.Bool
}

func newMockGitLab(t *testing.T, accessTTL time.Duration) *mockGitLab {
	t.Helper()
	m := &mockGitLab{accessTTL: accessTTL}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if m.fail.Load() {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		grant := r.FormValue("grant_type")
		if grant == "refresh_token" {
			m.refreshCount.Add(1)
		}
		n := m.refreshCount.Load()
		resp := map[string]interface{}{
			"access_token":  "mock-access-" + itoa(n),
			"refresh_token": "mock-refresh-" + itoa(n),
			"token_type":    "bearer",
			"expires_in":    int(m.accessTTL.Seconds()),
			"created_at":    time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

// tokenEndpoint returns a builder that points at this mock for any host.
func (m *mockGitLab) tokenEndpoint() func(string) string {
	return func(_ string) string {
		return m.server.URL + "/oauth/token"
	}
}

func (m *mockGitLab) count() int64 { return m.refreshCount.Load() }

func itoa(n int64) string {
	return strings.TrimSpace(string(jsonNumber(n)))
}

func jsonNumber(n int64) []byte {
	b, _ := json.Marshal(n)
	return b
}

// TestRefreshSession_UsesConfiguredEndpoint proves refreshSession calls the
// injected tokenEndpoint (and thus would call real GitLab in production).
func TestRefreshSession_UsesConfiguredEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	sess := &mcpSession{
		BearerToken:       "bearer-1",
		GitLabHost:        "gitlab.example.com",
		AccessToken:       "old-access",
		RefreshToken:      "old-refresh",
		GitLabClientID:    "gitlab-app-id",
		GitLabRedirectURI: "http://localhost/auth/redirect",
		TokenExpiresAt:    time.Now().Add(2 * time.Minute).Unix(), // within 5-min window
	}
	store.addSession(sess)

	if err := store.refreshSession(sess); err != nil {
		t.Fatalf("refreshSession: %v", err)
	}

	if mock.count() != 1 {
		t.Errorf("expected 1 refresh call, got %d", mock.count())
	}
	if !strings.HasPrefix(sess.AccessToken, "mock-access-") {
		t.Errorf("access token not rotated: %q", sess.AccessToken)
	}
	if !strings.HasPrefix(sess.RefreshToken, "mock-refresh-") {
		t.Errorf("refresh token not rotated: %q", sess.RefreshToken)
	}
	if sess.TokenExpiresAt <= time.Now().Unix() {
		t.Errorf("expiry not advanced: %d", sess.TokenExpiresAt)
	}
}

// TestGetSession_AutoRefreshesNearExpiry proves the on-demand refresh path in
// getSession triggers when a request arrives close to GitLab token expiry.
func TestGetSession_AutoRefreshesNearExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	// Session with token expiring in 2 min (inside the 5-min threshold).
	store.addSession(&mcpSession{
		BearerToken:       "bearer-auto",
		GitLabHost:        "gitlab.example.com",
		AccessToken:       "about-to-expire",
		RefreshToken:      "refresh-a",
		GitLabClientID:    "gitlab-app-id",
		GitLabRedirectURI: "http://localhost/auth/redirect",
		TokenExpiresAt:    time.Now().Add(2 * time.Minute).Unix(),
	})

	got := store.getSession("bearer-auto")
	if got == nil {
		t.Fatal("expected session to be returned")
	}
	if mock.count() != 1 {
		t.Errorf("expected 1 refresh, got %d", mock.count())
	}
	if got.AccessToken == "about-to-expire" {
		t.Error("access token was not rotated")
	}
}

// TestGetSession_NoRefreshWhenHealthy proves sessions far from expiry are NOT
// refreshed on every request — avoids hammering GitLab.
func TestGetSession_NoRefreshWhenHealthy(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	store.addSession(&mcpSession{
		BearerToken:    "bearer-healthy",
		GitLabHost:     "gitlab.example.com",
		AccessToken:    "fresh",
		RefreshToken:   "r",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	for i := 0; i < 5; i++ {
		if store.getSession("bearer-healthy") == nil {
			t.Fatal("expected session")
		}
	}
	if mock.count() != 0 {
		t.Errorf("expected 0 refreshes, got %d", mock.count())
	}
}

// TestSessionStore_PersistenceRoundtrip proves sessions written to disk are
// restored by a fresh store — i.e. server restart does not lose sessions.
func TestSessionStore_PersistenceRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	store1 := newSessionStore()
	store1.addSession(&mcpSession{
		BearerToken:       "persisted-bearer",
		GitLabHost:        "gitlab.example.com",
		AccessToken:       "gl-access",
		RefreshToken:      "gl-refresh",
		MCPRefreshToken:   "mcp-refresh",
		GitLabClientID:    "gl-client",
		GitLabRedirectURI: "http://localhost/cb",
		ClientID:          "reg-client",
		TokenExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
	})
	store1.addClient(&oauthRegisteredClient{
		ClientID:     "reg-client",
		RedirectURIs: []string{"http://localhost/cb"},
	})

	// Simulate restart.
	store2 := newSessionStore()
	got := store2.getSession("persisted-bearer")
	if got == nil {
		t.Fatal("session not restored from disk")
	}
	if got.AccessToken != "gl-access" {
		t.Errorf("access token = %q, want gl-access", got.AccessToken)
	}
	if got.MCPRefreshToken != "mcp-refresh" {
		t.Errorf("MCP refresh token = %q, want mcp-refresh", got.MCPRefreshToken)
	}
	if store2.getClient("reg-client") == nil {
		t.Error("client not restored from disk")
	}
}

// TestBackgroundRefresh_RotatesIdleSessions is the core guarantee for long
// sessions: even with zero MCP traffic, the background loop keeps GitLab tokens
// (and therefore refresh tokens) rotating so users don't get kicked out.
func TestBackgroundRefresh_RotatesIdleSessions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	// Two sessions: one close to expiry, one healthy — only the first should rotate.
	store.addSession(&mcpSession{
		BearerToken:    "near-expiry",
		GitLabHost:     "gitlab.example.com",
		AccessToken:    "old",
		RefreshToken:   "r1",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(30 * time.Second).Unix(),
	})
	store.addSession(&mcpSession{
		BearerToken:    "healthy",
		GitLabHost:     "gitlab.example.com",
		AccessToken:    "fresh",
		RefreshToken:   "r2",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errOut := &bytes.Buffer{}

	done := make(chan struct{})
	go func() {
		store.startBackgroundRefresh(ctx, 50*time.Millisecond, 5*time.Minute, errOut)
		close(done)
	}()

	// Wait long enough for at least one tick.
	waitFor(t, 2*time.Second, func() bool { return mock.count() >= 1 })

	// Cancel and wait for the goroutine to actually exit before touching
	// session state — otherwise reads race with an in-flight refresh write.
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh did not exit after cancel")
	}

	if mock.count() != 1 {
		t.Errorf("expected exactly 1 refresh (near-expiry only), got %d", mock.count())
	}

	// Snapshot session state under the store lock; refreshSession writes these
	// fields under the same lock, so this pairs safely with the race detector.
	snapshot := func(k string) string {
		store.mu.RLock()
		defer store.mu.RUnlock()
		if sess := store.sessions[k]; sess != nil {
			return sess.AccessToken
		}
		return ""
	}
	if snapshot("near-expiry") == "old" {
		t.Error("near-expiry session was not rotated")
	}
	if snapshot("healthy") != "fresh" {
		t.Error("healthy session was unexpectedly rotated")
	}
}

// TestBackgroundRefresh_StopsOnContextCancel proves the goroutine exits cleanly.
func TestBackgroundRefresh_StopsOnContextCancel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		store.startBackgroundRefresh(ctx, 10*time.Millisecond, 5*time.Minute, io.Discard)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("background refresh did not exit after cancel")
	}
}

// TestHandleRefreshGrant_RotatesBearerAndRefreshToken proves the MCP-side
// refresh flow (triggered when Claude Code's bearer is stale) rotates both the
// MCP bearer and the GitLab tokens, without invalidating the session.
func TestHandleRefreshGrant_RotatesBearerAndRefreshToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	store.addSession(&mcpSession{
		BearerToken:       "old-bearer",
		MCPRefreshToken:   "mcp-refresh-old",
		GitLabHost:        "gitlab.example.com",
		AccessToken:       "gl-old",
		RefreshToken:      "gl-refresh-old",
		GitLabClientID:    "gitlab-app-id",
		GitLabRedirectURI: "http://localhost/auth/redirect",
		ClientID:          "reg-client",
		// Within the 5-min refresh window so refreshSession actually calls GitLab.
		TokenExpiresAt: time.Now().Add(2 * time.Minute).Unix(),
	})

	handler := oauthTokenHandler(store, "gitlab.example.com", "gitlab-app-id", "http://localhost/auth/redirect", io.Discard)

	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"mcp-refresh-old"}}
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	newBearer, _ := resp["access_token"].(string)
	newRefresh, _ := resp["refresh_token"].(string)
	if newBearer == "" || newBearer == "old-bearer" {
		t.Errorf("bearer not rotated: %q", newBearer)
	}
	if newRefresh == "" || newRefresh == "mcp-refresh-old" {
		t.Errorf("mcp refresh not rotated: %q", newRefresh)
	}

	// Old bearer must be invalid; new bearer must resolve to a session.
	if store.getSession("old-bearer") != nil {
		t.Error("old bearer should be invalid after refresh")
	}
	sess := store.getSession(newBearer)
	if sess == nil {
		t.Fatal("new bearer should resolve to a session")
	}
	if !strings.HasPrefix(sess.AccessToken, "mock-access-") {
		t.Errorf("GitLab access token not rotated: %q", sess.AccessToken)
	}
	if mock.count() != 1 {
		t.Errorf("expected 1 refresh call to GitLab, got %d", mock.count())
	}
}

// TestHandleRefreshGrant_RejectsMismatchedClientID ensures an attacker holding
// someone else's refresh token can't use it under their own registered client.
// RFC 6749 §6 requires the server to validate the client when present.
func TestHandleRefreshGrant_RejectsMismatchedClientID(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	store := newSessionStore()
	store.addSession(&mcpSession{
		BearerToken:     "b",
		MCPRefreshToken: "mcp-r",
		GitLabHost:      "gitlab.example.com",
		RefreshToken:    "gl-r",
		GitLabClientID:  "app",
		ClientID:        "client-owner",
		TokenExpiresAt:  time.Now().Add(1 * time.Hour).Unix(),
	})
	handler := oauthTokenHandler(store, "gitlab.example.com", "app", "http://localhost/cb", io.Discard)

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"mcp-r"},
		"client_id":     {"client-attacker"},
	}
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on client_id mismatch, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %v", resp)
	}
	// The legitimate session must survive an attacker probe.
	if store.getSession("b") == nil {
		t.Error("owner's session should not be dropped by a failed impersonation attempt")
	}
}

// TestHandleRefreshGrant_FailsWhenGitLabRefreshFails simulates GitLab revoking
// the refresh token — the MCP client must be told to re-authorize.
func TestHandleRefreshGrant_FailsWhenGitLabRefreshFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	mock.fail.Store(true)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	store.addSession(&mcpSession{
		BearerToken:     "b",
		MCPRefreshToken: "mcp-r",
		GitLabHost:      "gitlab.example.com",
		RefreshToken:    "gl-r",
		GitLabClientID:  "app",
		// Within the 5-min window so refreshSession actually calls GitLab (and fails).
		TokenExpiresAt: time.Now().Add(2 * time.Minute).Unix(),
	})
	handler := oauthTokenHandler(store, "gitlab.example.com", "app", "http://localhost/cb", io.Discard)

	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"mcp-r"}}
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when GitLab refresh fails, got %d", w.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %v", resp)
	}
	// Terminal auth failure must also drop the session so it doesn't keep
	// failing on subsequent ticks.
	if store.getSession("b") != nil {
		t.Error("session should be dropped after GitLab permanently rejects refresh token")
	}
}

// TestRefreshSession_TransientErrorKeepsSession proves a network-level failure
// (VPN drop, passthrough down, GitLab 5xx) does NOT delete the session or
// clobber the refresh token — the next tick can recover once connectivity
// returns. This is the whole point of error classification.
func TestRefreshSession_TransientErrorKeepsSession(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	// Server that's up for TLS but always returns 503 — simulates GitLab being
	// sick. Use the mock's fail mechanism wouldn't work (it returns 400), so
	// spin up a dedicated one.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer srv.Close()

	store := newSessionStore()
	store.tokenEndpoint = func(_ string) string { return srv.URL + "/oauth/token" }

	sess := &mcpSession{
		BearerToken:    "b",
		GitLabHost:     "gitlab.example.com",
		AccessToken:    "old",
		RefreshToken:   "r-original",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	}
	store.addSession(sess)

	err := store.refreshSession(sess)
	if err == nil {
		t.Fatal("expected error on 5xx")
	}
	if errors.Is(err, errRefreshAuthFailed) {
		t.Errorf("5xx must NOT be classified as auth failure: %v", err)
	}
	if store.getSession("b") == nil {
		t.Error("session was dropped on transient error — must be kept")
	}
	if sess.RefreshToken != "r-original" {
		t.Errorf("refresh token clobbered on transient error: %q", sess.RefreshToken)
	}
}

// TestRefreshSession_AuthFailureClassified proves GitLab's 4xx response
// produces errRefreshAuthFailed so callers know to drop the session.
func TestRefreshSession_AuthFailureClassified(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	mock.fail.Store(true)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	sess := &mcpSession{
		BearerToken:    "b",
		GitLabHost:     "gitlab.example.com",
		RefreshToken:   "r",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	}
	store.addSession(sess)

	err := store.refreshSession(sess)
	if !errors.Is(err, errRefreshAuthFailed) {
		t.Fatalf("expected errRefreshAuthFailed, got %v", err)
	}
}

// TestRefreshExpiringSessions_DropsDeadSessions proves the background loop
// removes sessions GitLab has rejected, so log noise and wasted refresh
// attempts stop. Healthy sessions stay.
func TestRefreshExpiringSessions_DropsDeadSessions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	mock.fail.Store(true) // every refresh returns 400
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	store.addSession(&mcpSession{
		BearerToken:    "dead",
		GitLabHost:     "gitlab.example.com",
		RefreshToken:   "r",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(30 * time.Second).Unix(),
	})

	store.refreshExpiringSessions(5*time.Minute, io.Discard)

	if store.getSession("dead") != nil {
		t.Error("dead session was not removed after GitLab rejection")
	}
}

// TestRefreshExpiringSessions_KeepsSessionOnTransient proves a transient
// network failure in the background loop doesn't drop the session, so
// connectivity coming back lets it recover.
func TestRefreshExpiringSessions_KeepsSessionOnTransient(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gw timeout", http.StatusBadGateway)
	}))
	defer srv.Close()

	store := newSessionStore()
	store.tokenEndpoint = func(_ string) string { return srv.URL + "/oauth/token" }

	store.addSession(&mcpSession{
		BearerToken:    "flaky",
		GitLabHost:     "gitlab.example.com",
		RefreshToken:   "r",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(30 * time.Second).Unix(),
	})

	store.refreshExpiringSessions(5*time.Minute, io.Discard)

	if store.getSession("flaky") == nil {
		t.Error("session dropped on transient failure — must be kept")
	}
}

// TestRefreshSession_ConcurrentRefreshesDeduped proves the per-session mutex
// and early-return prevent two concurrent triggers (e.g. request-path refresh
// + background tick) from both consuming the single-use refresh token.
func TestRefreshSession_ConcurrentRefreshesDeduped(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	mock := newMockGitLab(t, 1*time.Hour)
	store := newSessionStore()
	store.tokenEndpoint = mock.tokenEndpoint()

	sess := &mcpSession{
		BearerToken:    "b",
		GitLabHost:     "gitlab.example.com",
		RefreshToken:   "r",
		GitLabClientID: "app",
		TokenExpiresAt: time.Now().Add(30 * time.Second).Unix(),
	}
	store.addSession(sess)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.refreshSession(sess)
		}()
	}
	wg.Wait()

	if mock.count() != 1 {
		t.Errorf("expected exactly 1 GitLab refresh call, got %d", mock.count())
	}
}

// TestHandleRefreshGrant_TransientReturns503 proves a flaky upstream produces
// a 503 (retryable) instead of a 400 invalid_grant, so the MCP client keeps
// its authorization instead of being forced back through OAuth.
func TestHandleRefreshGrant_TransientReturns503(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	store := newSessionStore()
	store.tokenEndpoint = func(_ string) string { return srv.URL + "/oauth/token" }

	store.addSession(&mcpSession{
		BearerToken:     "b",
		MCPRefreshToken: "mcp-r",
		GitLabHost:      "gitlab.example.com",
		RefreshToken:    "gl-r",
		GitLabClientID:  "app",
		TokenExpiresAt:  time.Now().Add(30 * time.Second).Unix(),
	})
	handler := oauthTokenHandler(store, "gitlab.example.com", "app", "http://localhost/cb", io.Discard)

	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"mcp-r"}}
	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 on transient upstream failure, got %d: %s", w.Code, w.Body.String())
	}
	if store.getSession("b") == nil {
		t.Error("session dropped on transient failure — must be kept so client can retry")
	}
}

// TestSaveToDisk_AtomicNoPartialFile proves a write failure can't produce a
// truncated sessions file. We assert the final file is either absent or
// valid JSON — never half-written — by spamming saveToDisk from many
// goroutines and parsing the result.
func TestSaveToDisk_AtomicNoPartialFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	store := newSessionStore()
	for i := 0; i < 50; i++ {
		store.addSession(&mcpSession{
			BearerToken:    "b-" + itoa(int64(i)),
			GitLabHost:     "gitlab.example.com",
			RefreshToken:   "r",
			GitLabClientID: "app",
			TokenExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		})
	}

	data, err := os.ReadFile(mcpSessionsPath())
	if err != nil {
		t.Fatalf("sessions file unreadable: %v", err)
	}
	var file mcpSessionsFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("sessions file is not valid JSON after many writes: %v\n%s", err, data)
	}
	if len(file.Sessions) != 50 {
		t.Errorf("expected 50 sessions persisted, got %d", len(file.Sessions))
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

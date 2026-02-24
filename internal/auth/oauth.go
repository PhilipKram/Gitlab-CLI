package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"
)

const (
	defaultScopes      = "openid profile api read_user write_repository"
	defaultRedirectURI = "http://localhost:7171/auth/redirect"
)

// OAuthTokenResponse represents the response from GitLab's OAuth token endpoint.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
}

// OAuthFlow performs the OAuth2 Authorization Code flow with PKCE.
// openBrowser is called with the authorization URL; pass nil to skip auto-open.
// If redirectURI is empty, http://localhost:7171/auth/redirect is used.
// If scopes is empty, defaultScopes is used.
func OAuthFlow(host, clientID, redirectURI, scopes string, out io.Writer, openBrowser func(string) error) (*Status, error) {
	if scopes == "" {
		scopes = defaultScopes
	}
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}

	// Parse redirect URI to determine listen address and callback path
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}
	listenHost := u.Hostname()
	if listenHost == "localhost" {
		listenHost = "127.0.0.1"
	}
	listenAddr := fmt.Sprintf("%s:%s", listenHost, u.Port())
	callbackPath := u.Path
	if callbackPath == "" {
		callbackPath = "/"
	}

	// Start a local server to receive the callback
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("starting local server: %w", err)
	}

	// Generate PKCE parameters
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generating PKCE verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// Build authorization URL
	authURL := buildAuthURL(host, clientID, redirectURI, state, codeChallenge, scopes)

	// Try to open browser automatically
	browserOpened := false
	if openBrowser != nil {
		if err := openBrowser(authURL); err == nil {
			browserOpened = true
		}
	}

	if browserOpened {
		fmt.Fprintf(out, "! Opening %s in your browser...\n", host)
	} else {
		fmt.Fprintf(out, "! Open this URL in your browser to authenticate:\n  %s\n", authURL)
	}
	fmt.Fprintf(out, "- Waiting for authentication...\n")

	// Wait for the callback
	code, err := waitForCallback(listener, state, callbackPath)
	if err != nil {
		return nil, fmt.Errorf("waiting for callback: %w", err)
	}

	// Exchange the code for a token
	tokenResp, err := exchangeCode(host, clientID, code, redirectURI, codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}

	// Validate the token
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tokenResp.AccessToken})
	client, err := gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{TokenSource: ts}, gitlab.WithBaseURL(apiURL(host)))
	if err != nil {
		return nil, fmt.Errorf("creating GitLab client: %w", err)
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("authenticating with %s: %w", host, err)
	}

	// Save credentials (merge into existing host config to preserve client_id, etc.)
	hosts, err := config.LoadHosts()
	if err != nil {
		hosts = make(config.HostsConfig)
	}

	hc, ok := hosts[host]
	if !ok {
		hc = &config.HostConfig{}
		hosts[host] = hc
	}
	hc.Token = tokenResp.AccessToken
	hc.User = user.Username
	hc.AuthMethod = "oauth"

	if err := config.SaveHosts(hosts); err != nil {
		return nil, fmt.Errorf("saving credentials: %w", err)
	}

	return &Status{
		Host:   host,
		User:   user.Username,
		Token:  maskToken(tokenResp.AccessToken),
		Source: host,
		Active: true,
	}, nil
}

func buildAuthURL(host, clientID, redirectURI, state, codeChallenge, scopes string) string {
	baseURL := fmt.Sprintf("https://%s/oauth/authorize", host)
	params := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	return baseURL + "?" + params.Encode()
}

func waitForCallback(listener net.Listener, expectedState, callbackPath string) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			errCh <- fmt.Errorf("state mismatch: possible CSRF attack")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("authorization denied: %s - %s", errMsg, desc)
			fmt.Fprintf(w, "<html><body><h1>Authorization Failed</h1><p>%s</p><p>You can close this window.</p></body></html>", desc)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		fmt.Fprint(w, "<html><body><h1>Authentication Successful</h1><p>You can close this window and return to the terminal.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Wait with a timeout
	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("timed out waiting for authentication (5 minutes)")
	}
}

func exchangeCode(host, clientID, code, redirectURI, codeVerifier string) (*OAuthTokenResponse, error) {
	tokenURL := fmt.Sprintf("https://%s/oauth/token", host)

	data := url.Values{
		"client_id":     {clientID},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("requesting token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	return &tokenResp, nil
}

func generateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func generateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// ScopesDescription returns a human-readable description of the required scopes.
func ScopesDescription() string {
	return strings.ReplaceAll(defaultScopes, " ", ", ")
}

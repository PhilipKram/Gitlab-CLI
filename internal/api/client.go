package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/auth"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"
)

// Client wraps the GitLab API client.
type Client struct {
	*gitlab.Client
	host string
}

// NewClient creates a new authenticated GitLab API client.
// It automatically selects the correct client type based on the stored auth method.
func NewClient(host string) (*Client, error) {
	// Reject hosts with scheme, path, or credential characters to prevent SSRF.
	if strings.ContainsAny(host, "/:@?#") {
		return nil, fmt.Errorf("invalid host %q: must be a plain hostname (e.g. gitlab.example.com)", host)
	}
	token, tokenSource := config.TokenForHost(host)
	if token == "" {
		return nil, errors.NewAuthError(
			host,
			"",
			"",
			0,
			fmt.Sprintf("Not authenticated with %s", host),
			nil,
		)
	}

	authMethod := config.AuthMethodForHost(host)
	// Only auto-refresh tokens from hosts.json, not env-provided tokens
	if authMethod == "oauth" {
		if tokenSource != "GITLAB_TOKEN" && tokenSource != "GLAB_TOKEN" {
			refreshedToken, err := RefreshOAuthTokenIfNeeded(host, token)
			if err != nil {
				return nil, err
			}
			token = refreshedToken
		}
		return NewOAuthClient(host, token)
	}

	return NewClientWithToken(host, token)
}

// NewClientWithToken creates a new GitLab API client with the given token.
// Optional gitlab.ClientOptionFunc values are appended after the defaults.
func NewClientWithToken(host, token string, opts ...gitlab.ClientOptionFunc) (*Client, error) {
	baseURL := APIURL(host)
	// Create client with appropriate HTTP client
	// Use http.DefaultTransport as base to allow test mocking via InterceptTransport
	var client *gitlab.Client
	var err error
	if errors.IsVerboseMode() {
		httpClient := errors.NewLoggingHTTPClient()
		httpClient.Transport = &RateLimitTransport{Base: httpClient.Transport}
		baseOpts := []gitlab.ClientOptionFunc{gitlab.WithBaseURL(baseURL), gitlab.WithHTTPClient(httpClient)}
		client, err = gitlab.NewClient(token, append(baseOpts, opts...)...)
	} else {
		httpClient := &http.Client{Transport: &RateLimitTransport{Base: http.DefaultTransport}}
		baseOpts := []gitlab.ClientOptionFunc{gitlab.WithBaseURL(baseURL), gitlab.WithHTTPClient(httpClient)}
		client, err = gitlab.NewClient(token, append(baseOpts, opts...)...)
	}

	if err != nil {
		return nil, errors.NewAPIError(
			"",
			baseURL,
			0,
			"Failed to create GitLab client",
			err,
		)
	}

	return &Client{
		Client: client,
		host:   host,
	}, nil
}

// NewOAuthClient creates a new GitLab API client using an OAuth token.
// Optional gitlab.ClientOptionFunc values are appended after the defaults.
func NewOAuthClient(host, token string, opts ...gitlab.ClientOptionFunc) (*Client, error) {
	baseURL := APIURL(host)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	// Create client with logging HTTP client if verbose mode is enabled
	var client *gitlab.Client
	var err error
	if errors.IsVerboseMode() {
		httpClient := errors.NewLoggingHTTPClient()
		httpClient.Transport = &RateLimitTransport{Base: httpClient.Transport}
		baseOpts := []gitlab.ClientOptionFunc{gitlab.WithBaseURL(baseURL), gitlab.WithHTTPClient(httpClient)}
		client, err = gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{TokenSource: ts}, append(baseOpts, opts...)...)
	} else {
		httpClient := &http.Client{Transport: &RateLimitTransport{Base: http.DefaultTransport}}
		baseOpts := []gitlab.ClientOptionFunc{gitlab.WithBaseURL(baseURL), gitlab.WithHTTPClient(httpClient)}
		client, err = gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{TokenSource: ts}, append(baseOpts, opts...)...)
	}

	if err != nil {
		return nil, errors.NewAuthError(
			host,
			"",
			baseURL,
			0,
			"Failed to create GitLab OAuth client",
			err,
		)
	}

	return &Client{
		Client: client,
		host:   host,
	}, nil
}

// NewClientFromHosts creates a client using the first authenticated host found in hosts.json.
func NewClientFromHosts() (*Client, error) {
	hosts, err := config.LoadHosts()
	if err != nil || len(hosts) == 0 {
		return nil, errors.NewAuthError(
			"",
			"",
			"",
			0,
			"Not authenticated with any host",
			err,
		)
	}
	for host := range hosts {
		client, err := NewClient(host)
		if err == nil {
			return client, nil
		}
	}
	return nil, errors.NewAuthError(
		"",
		"",
		"",
		0,
		"Not authenticated with any host",
		nil,
	)
}

// Host returns the hostname of the GitLab instance.
func (c *Client) Host() string {
	return c.host
}

// GetVersion returns the cached GitLab version for this client's host.
// Returns an empty string if the version is not cached or unknown (graceful degradation).
func (c *Client) GetVersion() string {
	hosts, err := config.LoadHosts()
	if err != nil {
		return ""
	}
	hc, ok := hosts[c.host]
	if !ok {
		return ""
	}
	return hc.GitLabVersion
}

// APIURL returns the API base URL for a given host.
func APIURL(host string) string {
	if host == "gitlab.com" {
		return "https://gitlab.com/api/v4"
	}
	return fmt.Sprintf("https://%s/api/v4", host)
}

// WebURL returns the web URL for a given host and path.
func WebURL(host, path string) string {
	return fmt.Sprintf("https://%s/%s", host, path)
}

// RefreshOAuthTokenIfNeeded checks if the OAuth token is expired (or about to expire)
// and refreshes it. Returns the refreshed token on success, or the original token on failure.
// If both the refresh fails and the current token is expired, returns an error advising re-authentication.
func RefreshOAuthTokenIfNeeded(host, currentToken string) (string, error) {
	hosts, err := config.LoadHosts()
	if err != nil {
		return currentToken, nil
	}
	hc, ok := hosts[host]
	if !ok || hc.TokenExpiresAt == 0 {
		return currentToken, nil
	}

	// Skip refresh if the token was provided via env var (doesn't match stored token)
	if currentToken != hc.Token {
		return currentToken, nil
	}

	// Refresh if token expires within 5 minutes
	if time.Now().Unix() < hc.TokenExpiresAt-300 {
		return currentToken, nil
	}

	newToken, err := auth.RefreshOAuthToken(host)
	if err != nil {
		// If the current token is also expired, advise re-authentication
		if time.Now().Unix() >= hc.TokenExpiresAt {
			return "", errors.NewAuthError(
				host, "", "", 0,
				fmt.Sprintf("OAuth token for %s has expired and refresh failed: %v\nRun 'glab auth login --hostname %s' to re-authenticate", host, err, host),
				err,
			)
		}
		// Token not yet expired, fall back to it
		return currentToken, nil
	}
	return newToken, nil
}

// GetAndCacheVersion fetches the GitLab version from the API and caches it in the hosts config.
// Returns the version string on success, or an empty string on error (best-effort).
func GetAndCacheVersion(client *gitlab.Client, host string) string {
	// Fetch version from API
	version, _, err := client.Version.GetVersion()
	if err != nil || version == nil {
		return ""
	}

	gitlabVersion := version.Version

	// Load hosts config
	hosts, err := config.LoadHosts()
	if err != nil {
		return gitlabVersion
	}

	// Update version for this host
	hc, ok := hosts[host]
	if !ok {
		return gitlabVersion
	}

	hc.GitLabVersion = gitlabVersion
	hosts[host] = hc

	_ = config.SaveHosts(hosts)

	return gitlabVersion
}

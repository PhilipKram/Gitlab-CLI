package api

import (
	"fmt"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/config"
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
	token, _ := config.TokenForHost(host)
	if token == "" {
		return nil, fmt.Errorf("not authenticated with %s; run 'glab auth login --hostname %s'", host, host)
	}

	authMethod := config.AuthMethodForHost(host)
	if authMethod == "oauth" {
		return NewOAuthClient(host, token)
	}

	return NewClientWithToken(host, token)
}

// NewClientWithToken creates a new GitLab API client with the given token.
func NewClientWithToken(host, token string) (*Client, error) {
	baseURL := APIURL(host)
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("creating GitLab client: %w", err)
	}

	return &Client{
		Client: client,
		host:   host,
	}, nil
}

// NewOAuthClient creates a new GitLab API client using an OAuth token.
func NewOAuthClient(host, token string) (*Client, error) {
	baseURL := APIURL(host)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client, err := gitlab.NewAuthSourceClient(gitlab.OAuthTokenSource{TokenSource: ts}, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("creating GitLab OAuth client: %w", err)
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
		return nil, fmt.Errorf("not authenticated with any host; run 'glab auth login'")
	}
	for host := range hosts {
		client, err := NewClient(host)
		if err == nil {
			return client, nil
		}
	}
	return nil, fmt.Errorf("not authenticated with any host; run 'glab auth login'")
}

// Host returns the hostname of the GitLab instance.
func (c *Client) Host() string {
	return c.host
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

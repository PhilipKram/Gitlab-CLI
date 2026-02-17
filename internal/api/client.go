package api

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Client wraps the GitLab API client.
type Client struct {
	*gitlab.Client
	host string
}

// NewClient creates a new authenticated GitLab API client.
func NewClient(host string) (*Client, error) {
	token, _ := config.TokenForHost(host)
	if token == "" {
		return nil, fmt.Errorf("not authenticated with %s; run 'glab auth login --hostname %s'", host, host)
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

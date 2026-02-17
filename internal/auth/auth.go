package auth

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Status represents the authentication status for a host.
type Status struct {
	Host           string
	User           string
	Token          string
	Source         string
	GitLabVersion  string
	AuthMethod     string // "pat", "oauth", or ""
	TokenExpiresAt int64  // Unix timestamp; 0 if not set
	Scopes         string // OAuth scopes; empty for PAT
	Active         bool
	HasError       bool
	Error          string
}

// Login authenticates the user with a GitLab instance.
func Login(host, token string, stdin io.Reader) (*Status, error) {
	if token == "" {
		// Try to read from stdin
		if stdin != nil {
			scanner := bufio.NewScanner(stdin)
			if scanner.Scan() {
				token = strings.TrimSpace(scanner.Text())
			}
		}
	}

	if token == "" {
		return nil, errors.NewAuthError(
			host,
			"",
			"",
			0,
			"No token provided",
			fmt.Errorf("use --token flag or pipe token via stdin"),
		)
	}

	// Validate the token by making an API call
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(apiURL(host)))
	if err != nil {
		return nil, errors.NewAuthError(
			host,
			"",
			apiURL(host),
			0,
			"Failed to create GitLab client",
			err,
		)
	}

	user, resp, err := client.Users.CurrentUser()
	if err != nil {
		// Check if this is a 401 Unauthorized
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("authentication failed: invalid or expired token for %s\n\nThe token may be invalid, expired, or revoked.\nPlease generate a new token at: https://%s/-/profile/personal_access_tokens", host, host)
		}
		// Use enhanced error formatting
		return nil, formatAuthError(host, err)
	}

	// Detect GitLab version (best-effort, don't fail auth if it fails)
	var gitlabVersion string
	version, _, err := client.Version.GetVersion()
	if err == nil && version != nil {
		gitlabVersion = version.Version
	}
	// If version detection fails, continue without it (graceful degradation)

	// Save the host configuration (merge into existing to preserve client_id, etc.)
	hosts, err := config.LoadHosts()
	if err != nil {
		hosts = make(config.HostsConfig)
	}

	hc, ok := hosts[host]
	if !ok {
		hc = &config.HostConfig{}
		hosts[host] = hc
	}
	hc.Token = token
	hc.User = user.Username
	hc.AuthMethod = "pat"
	hc.GitLabVersion = gitlabVersion

	if err := config.SaveHosts(hosts); err != nil {
		return nil, fmt.Errorf("saving credentials: %w", err)
	}

	return &Status{
		Host:   host,
		User:   user.Username,
		Token:  maskToken(token),
		Source: host,
		Active: true,
	}, nil
}

// Logout removes stored credentials for a host.
func Logout(host string) error {
	hosts, err := config.LoadHosts()
	if err != nil {
		return fmt.Errorf("loading hosts config: %w", err)
	}

	if _, ok := hosts[host]; !ok {
		return errors.NewAuthError(
			host,
			"",
			"",
			0,
			fmt.Sprintf("Not logged in to %s", host),
			nil,
		)
	}

	delete(hosts, host)
	return config.SaveHosts(hosts)
}

// LogoutAll removes stored credentials for all hosts.
func LogoutAll() error {
	hosts := make(config.HostsConfig)
	return config.SaveHosts(hosts)
}

// GetStatus returns the authentication status for all configured hosts.
func GetStatus() ([]Status, error) {
	var statuses []Status

	// Check environment variable
	if t := os.Getenv("GITLAB_TOKEN"); t != "" {
		host := config.DefaultHost()
		s := Status{
			Host:   host,
			Token:  maskToken(t),
			Source: "GITLAB_TOKEN",
			Active: true,
		}
		// Try to get user info
		client, err := gitlab.NewClient(t, gitlab.WithBaseURL(apiURL(host)))
		if err == nil {
			user, resp, err := client.Users.CurrentUser()
			if err == nil {
				s.User = user.Username
			} else {
				s.HasError = true
				// Provide actionable error message
				if resp != nil && resp.StatusCode == http.StatusUnauthorized {
					s.Error = fmt.Sprintf("Invalid or expired token. Please update your GITLAB_TOKEN environment variable or run: glab auth login --hostname %s", host)
				} else if isUnauthorizedError(err) {
					s.Error = fmt.Sprintf("Invalid or expired token. Please update your GITLAB_TOKEN environment variable or run: glab auth login --hostname %s", host)
				} else {
					s.Error = fmt.Sprintf("%v. Run 'glab auth login --hostname %s' to re-authenticate", err, host)
				}
			}
		}
		statuses = append(statuses, s)
	}

	hosts, err := config.LoadHosts()
	if err != nil {
		if len(statuses) > 0 {
			return statuses, nil
		}
		return nil, errors.NewAuthError(
			"",
			"",
			"",
			0,
			"No authenticated hosts",
			err,
		)
	}

	for host, hc := range hosts {
		s := Status{
			Host:           host,
			User:           hc.User,
			Token:          maskToken(hc.Token),
			Source:         host,
			GitLabVersion:  hc.GitLabVersion,
			AuthMethod:     hc.AuthMethod,
			TokenExpiresAt: hc.TokenExpiresAt,
			Scopes:         hc.OAuthScopes,
			Active:         true,
		}

		// Check if token is expired
		if isTokenExpired(hc.TokenExpiresAt) {
			s.HasError = true
			s.Error = formatTokenExpiredError(host, hc.AuthMethod)
			s.Active = false
		}

		statuses = append(statuses, s)
	}

	if len(statuses) == 0 {
		return nil, errors.NewAuthError(
			"",
			"",
			"",
			0,
			"No authenticated hosts",
			nil,
		)
	}

	return statuses, nil
}

// GetToken retrieves the token for a specific host.
func GetToken(host string) (string, error) {
	token, source := config.TokenForHost(host)
	if token == "" {
		return "", errors.NewAuthError(
			host,
			"",
			"",
			0,
			fmt.Sprintf("No token found for %s", host),
			nil,
		)
	}

	// Check if the token is expired (only for tokens from config, not env vars)
	if source == host {
		hosts, err := config.LoadHosts()
		if err == nil {
			if hc, ok := hosts[host]; ok && isTokenExpired(hc.TokenExpiresAt) {
				return "", fmt.Errorf("%s", formatTokenExpiredError(host, hc.AuthMethod))
			}
		}
	}

	return token, nil
}

func apiURL(host string) string {
	if host == "gitlab.com" {
		return "https://gitlab.com/api/v4"
	}
	return fmt.Sprintf("https://%s/api/v4", host)
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// isTokenExpired checks if a token has expired based on the expiration timestamp.
func isTokenExpired(expiresAt int64) bool {
	if expiresAt == 0 {
		return false // No expiration set
	}
	return time.Now().Unix() > expiresAt
}

// formatAuthError returns a user-friendly error message for authentication failures.
func formatAuthError(host string, originalErr error) error {
	if originalErr == nil {
		return nil
	}

	// Check if this is a 401 Unauthorized error
	if isUnauthorizedError(originalErr) {
		return fmt.Errorf("authentication failed: invalid or expired token for %s\n\nThe token may be invalid, expired, or revoked.\nPlease re-authenticate by running: glab auth login --hostname %s", host, host)
	}

	// Check for network/connection errors
	errStr := originalErr.Error()
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return fmt.Errorf("connection failed: unable to reach %s\n\nPlease check:\n- Your internet connection\n- The hostname is correct\n- The GitLab instance is accessible\n\nOriginal error: %w", host, originalErr)
	}

	// Generic error with actionable guidance
	return fmt.Errorf("authentication failed for %s: %w\n\nIf the problem persists, try re-authenticating: glab auth login --hostname %s", host, originalErr, host)
}

// isUnauthorizedError checks if an error is an HTTP 401 Unauthorized error.
func isUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error contains 401 status code indicators
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(strings.ToLower(errStr), "unauthorized") ||
		strings.Contains(strings.ToLower(errStr), "invalid token") ||
		strings.Contains(strings.ToLower(errStr), "authentication failed")
}

// formatTokenExpiredError returns a user-friendly error message for expired tokens.
func formatTokenExpiredError(host string, authMethod string) string {
	if authMethod == "oauth" {
		return fmt.Sprintf("token expired for %s\n\nYour OAuth token has expired. Please re-authenticate:\nglab auth login --hostname %s", host, host)
	}
	return fmt.Sprintf("token expired for %s\n\nYour personal access token has expired. Please re-authenticate:\nglab auth login --hostname %s", host, host)
}

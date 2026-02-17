package auth

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Status represents the authentication status for a host.
type Status struct {
	Host     string
	User     string
	Token    string
	Source   string
	Active   bool
	HasError bool
	Error    string
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
		return nil, fmt.Errorf("no token provided; use --token flag or pipe token via stdin")
	}

	// Validate the token by making an API call
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(apiURL(host)))
	if err != nil {
		return nil, fmt.Errorf("creating GitLab client: %w", err)
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("authenticating with %s: %w", host, err)
	}

	// Save the host configuration
	hosts, err := config.LoadHosts()
	if err != nil {
		hosts = make(config.HostsConfig)
	}

	hosts[host] = &config.HostConfig{
		Token: token,
		User:  user.Username,
	}

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
		return fmt.Errorf("not logged in to %s", host)
	}

	delete(hosts, host)
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
			user, _, err := client.Users.CurrentUser()
			if err == nil {
				s.User = user.Username
			} else {
				s.HasError = true
				s.Error = err.Error()
			}
		}
		statuses = append(statuses, s)
	}

	hosts, err := config.LoadHosts()
	if err != nil {
		if len(statuses) > 0 {
			return statuses, nil
		}
		return nil, fmt.Errorf("no authenticated hosts")
	}

	for host, hc := range hosts {
		s := Status{
			Host:   host,
			User:   hc.User,
			Token:  maskToken(hc.Token),
			Source: host,
			Active: true,
		}
		statuses = append(statuses, s)
	}

	if len(statuses) == 0 {
		return nil, fmt.Errorf("no authenticated hosts; run 'glab auth login' to authenticate")
	}

	return statuses, nil
}

// GetToken retrieves the token for a specific host.
func GetToken(host string) (string, error) {
	token, source := config.TokenForHost(host)
	if token == "" {
		return "", fmt.Errorf("no token found for %s; run 'glab auth login --hostname %s' to authenticate", host, host)
	}
	_ = source
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

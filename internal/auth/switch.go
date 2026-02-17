package auth

import (
	"fmt"
	"io"

	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/prompt"
)

// Switch allows the user to switch the active GitLab instance.
// It presents a list of authenticated hosts and updates the default_host config.
func Switch(in io.Reader, out io.Writer) (string, error) {
	// Get all authenticated hosts
	hosts, err := config.LoadHosts()
	if err != nil {
		return "", fmt.Errorf("loading hosts config: %w", err)
	}

	if len(hosts) == 0 {
		return "", fmt.Errorf("no authenticated hosts; run 'glab auth login' to authenticate")
	}

	if len(hosts) == 1 {
		// Only one host, nothing to switch to
		for host := range hosts {
			return "", fmt.Errorf("only one authenticated host (%s); add another with 'glab auth login --hostname <host>'", host)
		}
	}

	// Build list of host names
	var hostNames []string
	for host := range hosts {
		hostNames = append(hostNames, host)
	}

	// Present selection prompt
	idx, err := prompt.Select(in, out, "Select a GitLab instance:", hostNames)
	if err != nil {
		return "", fmt.Errorf("selecting host: %w", err)
	}

	selectedHost := hostNames[idx]

	// Update default_host in config
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Set("default_host", selectedHost); err != nil {
		return "", fmt.Errorf("saving default host: %w", err)
	}

	return selectedHost, nil
}

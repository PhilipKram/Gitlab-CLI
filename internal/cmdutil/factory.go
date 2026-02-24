package cmdutil

import (
	"fmt"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/git"
	"github.com/PhilipKram/gitlab-cli/pkg/iostreams"
)

// Factory provides shared dependencies for commands.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Config   func() (*config.Config, error)
	Client   func() (*api.Client, error)
	Remote   func() (*git.Remote, error)
	Version  string

	// repoOverride is set via --repo flag (HOST/OWNER/REPO format)
	repoOverride string
	overrideHost string
	overridePath string
}

// SetRepoOverride parses a HOST/OWNER/REPO string and stores it.
func (f *Factory) SetRepoOverride(repo string) {
	f.repoOverride = repo
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		f.overrideHost = parts[0]
		f.overridePath = parts[1]
	}
}

// NewFactory creates a Factory with default implementations.
func NewFactory() *Factory {
	f := &Factory{
		IOStreams: iostreams.System(),
	}

	f.Config = func() (*config.Config, error) {
		return config.Load()
	}

	f.Client = func() (*api.Client, error) {
		// If --repo is set, use its host
		if f.overrideHost != "" {
			return api.NewClient(f.overrideHost)
		}

		remote, err := f.Remote()
		if err == nil {
			// Try the remote host first
			client, err := api.NewClient(remote.Host)
			if err == nil {
				return client, nil
			}
		}
		// Fall back to default host
		client, err := api.NewClient(config.DefaultHost())
		if err == nil {
			return client, nil
		}
		// Fall back to the first authenticated host
		return api.NewClientFromHosts()
	}

	f.Remote = func() (*git.Remote, error) {
		cfg, err := f.Config()
		if err != nil {
			cfg = &config.Config{GitRemote: "origin"}
		}
		remoteName := cfg.GitRemote
		if remoteName == "" {
			remoteName = "origin"
		}
		return git.FindRemote(remoteName, config.DefaultHost())
	}

	return f
}

// FullProjectPath returns the "owner/repo" path from the current git remote,
// or from the --repo override if set.
func (f *Factory) FullProjectPath() (string, error) {
	if f.overridePath != "" {
		return f.overridePath, nil
	}

	remote, err := f.Remote()
	if err != nil {
		return "", fmt.Errorf("could not determine project: %w\nUse --repo HOST/OWNER/REPO to specify the project", err)
	}
	if remote.Owner == "" || remote.Repo == "" {
		return "", fmt.Errorf("could not determine project path from remote %s\nUse --repo HOST/OWNER/REPO to specify the project", remote.Name)
	}
	return remote.Owner + "/" + remote.Repo, nil
}

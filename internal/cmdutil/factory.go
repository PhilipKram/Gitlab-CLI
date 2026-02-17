package cmdutil

import (
	"fmt"

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
		remote, err := f.Remote()
		if err != nil {
			// Fall back to default host
			return api.NewClient(config.DefaultHost())
		}
		return api.NewClient(remote.Host)
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

// FullProjectPath returns the "owner/repo" path from the current git remote.
func (f *Factory) FullProjectPath() (string, error) {
	remote, err := f.Remote()
	if err != nil {
		return "", fmt.Errorf("could not determine project: %w", err)
	}
	if remote.Owner == "" || remote.Repo == "" {
		return "", fmt.Errorf("could not determine project path from remote %s", remote.Name)
	}
	return remote.Owner + "/" + remote.Repo, nil
}

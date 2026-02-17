package cmdutil

import (
	"fmt"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/formatter"
	"github.com/PhilipKram/gitlab-cli/internal/git"
	"github.com/PhilipKram/gitlab-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

// Factory provides shared dependencies for commands.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Config    func() (*config.Config, error)
	Client    func() (*api.Client, error)
	Remote    func() (*git.Remote, error)
	Version   string

	// repoOverride is set via --repo flag (HOST/OWNER/REPO format)
	repoOverride string
	overrideHost string
	overridePath string

	// outputFormat tracks the requested output format for error formatting
	outputFormat string
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

// AddFormatFlag adds standard format and json flags to a command.
func AddFormatFlag(cmd *cobra.Command, format *string, jsonFlag *bool) {
	cmd.Flags().StringVarP(format, "format", "f", "", "Output format (json, table)")
	cmd.Flags().BoolVar(jsonFlag, "json", false, "Output as JSON (shorthand for --format json)")
}

// FormatAndPrint formats and prints data according to format flags.
// It handles backward compatibility for the --json flag.
func (f *Factory) FormatAndPrint(data interface{}, format string, jsonFlag bool) error {
	outputFormat, err := f.ResolveFormat(format, jsonFlag)
	if err != nil {
		return err
	}

	fmtr := formatter.New(outputFormat, f.IOStreams.Out)
	if fmtr == nil {
		return fmt.Errorf("invalid format: %s", format)
	}

	return fmtr.Format(data)
}

// ResolveFormat resolves the output format from the format string and deprecated --json flag.
// It returns the validated OutputFormat and an error if the format is invalid.
// If jsonFlag is true, a deprecation warning is printed to stderr.
func (f *Factory) ResolveFormat(format string, jsonFlag bool) (formatter.OutputFormat, error) {
	if jsonFlag {
		_, _ = fmt.Fprintf(f.IOStreams.ErrOut, "Warning: --json is deprecated, use --format=json instead\n")
		format = "json"
	}
	outputFormat := formatter.OutputFormat(format)
	if outputFormat != formatter.JSONFormat && outputFormat != formatter.TableFormat && outputFormat != formatter.PlainFormat {
		return "", fmt.Errorf("invalid format: %s (must be json, table, or plain)", format)
	}
	return outputFormat, nil
}

// FormatAndStream handles the streaming output pattern common to list commands.
// It converts a Result channel to the streaming formatter output.
func FormatAndStream[T any](f *Factory, results <-chan api.Result[T], outputFormat formatter.OutputFormat, limit int, entityName string) error {
	items := make(chan interface{}, 100)

	go func() {
		defer close(items)
		count := 0
		for result := range results {
			if result.Error != nil {
				_, _ = fmt.Fprintf(f.IOStreams.ErrOut, "Error fetching %s: %v\n", entityName, result.Error)
				return
			}
			items <- result.Item
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}()

	streamFmtr := formatter.NewStreaming(outputFormat, f.IOStreams.Out)
	if streamFmtr == nil {
		return fmt.Errorf("invalid format: %s", string(outputFormat))
	}

	return streamFmtr.FormatStream(items)
}

// SetOutputFormat sets the output format for the command execution.
// This is used to determine how errors should be formatted.
func (f *Factory) SetOutputFormat(format string) {
	f.outputFormat = format
}

// GetOutputFormat returns the current output format.
func (f *Factory) GetOutputFormat() string {
	return f.outputFormat
}

// IsJSONFormat returns true if the output format is JSON.
func (f *Factory) IsJSONFormat() bool {
	return f.outputFormat == "json"
}

// GetHostVersion returns the cached GitLab version for a given host.
// Returns empty string if version is unknown or not cached (graceful degradation).
func (f *Factory) GetHostVersion(host string) string {
	hosts, err := config.LoadHosts()
	if err != nil {
		return ""
	}
	if hc, ok := hosts[host]; ok {
		return hc.GitLabVersion
	}
	return ""
}

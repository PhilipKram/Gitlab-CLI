package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/auth"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewAuthCmd creates the auth command group.
func NewAuthCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Authenticate glab and git with GitLab",
		Long:  "Manage authentication state for GitLab hosts.",
	}

	cmd.AddCommand(newAuthLoginCmd(f))
	cmd.AddCommand(newAuthLogoutCmd(f))
	cmd.AddCommand(newAuthStatusCmd(f))
	cmd.AddCommand(newAuthTokenCmd(f))

	return cmd
}

func newAuthLoginCmd(f *cmdutil.Factory) *cobra.Command {
	var hostname string
	var token string
	var stdin bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a GitLab instance",
		Long: `Authenticate with a GitLab host.

The default hostname is gitlab.com. Override with --hostname.

A personal access token can be provided via --token or piped through stdin.
Required scopes: api, read_user, read_repository.`,
		Example: `  # Authenticate with a token
  $ glab auth login --token glpat-xxxxxxxxxxxxxxxxxxxx

  # Authenticate with a self-hosted instance
  $ glab auth login --hostname gitlab.example.com --token glpat-xxxx

  # Pipe a token from a file
  $ glab auth login --stdin < token.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = config.DefaultHost()
			}

			var stdinReader = f.IOStreams.In
			if !stdin {
				stdinReader = nil
			}

			status, err := auth.Login(hostname, token, stdinReader)
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IOStreams.Out, "Logged in to %s as %s\n", status.Host, status.User)
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "hostname", "h", "", "GitLab hostname (default: gitlab.com)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Personal access token")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read token from stdin")

	return cmd
}

func newAuthLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a GitLab instance",
		Example: `  $ glab auth logout
  $ glab auth logout --hostname gitlab.example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = config.DefaultHost()
			}
			if err := auth.Logout(hostname); err != nil {
				return err
			}
			fmt.Fprintf(f.IOStreams.Out, "Logged out of %s\n", hostname)
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "hostname", "h", "", "GitLab hostname")

	return cmd
}

func newAuthStatusCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		Example: `  $ glab auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			statuses, err := auth.GetStatus()
			if err != nil {
				return err
			}

			out := f.IOStreams.Out
			for _, s := range statuses {
				fmt.Fprintf(out, "%s\n", s.Host)
				fmt.Fprintf(out, "  Logged in as: %s\n", s.User)
				fmt.Fprintf(out, "  Token: %s\n", s.Token)
				fmt.Fprintf(out, "  Token source: %s\n", s.Source)
				if s.HasError {
					fmt.Fprintf(out, "  Error: %s\n", s.Error)
				}
				fmt.Fprintln(out)
			}
			return nil
		},
	}

	return cmd
}

func newAuthTokenCmd(f *cmdutil.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the auth token for a GitLab instance",
		Long:  "Print the authentication token that glab is configured to use for a given host.",
		Example: `  $ glab auth token
  $ glab auth token --hostname gitlab.example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = config.DefaultHost()
			}
			token, err := auth.GetToken(hostname)
			if err != nil {
				return err
			}
			fmt.Fprintln(f.IOStreams.Out, token)
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "hostname", "h", "", "GitLab hostname")

	return cmd
}

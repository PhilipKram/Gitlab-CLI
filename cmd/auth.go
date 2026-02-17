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
	var web bool
	var clientID string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a GitLab instance",
		Long: `Authenticate with a GitLab host.

The default hostname is gitlab.com. Override with --hostname.

Authentication methods:
  - Personal access token: provide via --token or pipe through stdin
  - OAuth (browser-based): use --web with --client-id

For OAuth, you must first create an OAuth application in your GitLab instance
under Settings > Applications. Set the redirect URI to http://127.0.0.1 (any port)
and enable the "api", "read_user", and "read_repository" scopes.

Required scopes: api, read_user, read_repository.`,
		Example: `  # Authenticate with a token
  $ glab auth login --token glpat-xxxxxxxxxxxxxxxxxxxx

  # Authenticate via OAuth in browser
  $ glab auth login --web --client-id <your-app-id>

  # Authenticate with a self-hosted instance
  $ glab auth login --hostname gitlab.example.com --token glpat-xxxx

  # OAuth with a self-hosted instance
  $ glab auth login --web --client-id <your-app-id> --hostname gitlab.example.com

  # Pipe a token from a file
  $ glab auth login --stdin < token.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = config.DefaultHost()
			}

			if web {
				if clientID == "" {
					return fmt.Errorf("--client-id is required for OAuth login; create an application at https://%s/-/user_settings/applications", hostname)
				}

				status, err := auth.OAuthFlow(hostname, clientID, f.IOStreams.ErrOut)
				if err != nil {
					return err
				}

				fmt.Fprintf(f.IOStreams.Out, "Logged in to %s as %s (via OAuth)\n", status.Host, status.User)
				return nil
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
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Authenticate via OAuth in the browser")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth application ID (required with --web)")

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
				if s.AuthMethod != "" {
					fmt.Fprintf(out, "  Auth method: %s\n", s.AuthMethod)
				}
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

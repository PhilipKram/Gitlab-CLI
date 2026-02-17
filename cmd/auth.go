package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/auth"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/prompt"
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
	var gitProtocol string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a GitLab instance",
		Long: `Authenticate with a GitLab host.

When run interactively (no flags), you will be prompted for host, auth method,
and git protocol — similar to the GitHub CLI experience.

The default hostname is gitlab.com. Override with --hostname.

Authentication methods:
  - Personal access token: provide via --token or pipe through stdin
  - OAuth (browser-based): use --web with --client-id

For OAuth, you must first create an OAuth application in your GitLab instance
under Settings > Applications. Set the redirect URI to http://127.0.0.1 (any port)
and enable the "api", "read_user", and "read_repository" scopes.

Required token scopes: api, read_user, read_repository.`,
		Example: `  # Interactive login (prompts for everything)
  $ glab auth login

  # Authenticate with a token
  $ glab auth login --token glpat-xxxxxxxxxxxxxxxxxxxx

  # Authenticate via OAuth in browser
  $ glab auth login --web --client-id <your-app-id>

  # Authenticate with a self-hosted instance
  $ glab auth login --hostname gitlab.example.com --token glpat-xxxx

  # Pipe a token from a file
  $ glab auth login --stdin < token.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams
			out := ios.Out
			errOut := ios.ErrOut

			// Determine if we should run interactively
			interactive := !cmd.Flags().Changed("token") &&
				!cmd.Flags().Changed("stdin") &&
				!cmd.Flags().Changed("web") &&
				ios.IsStdinTTY() && ios.IsTerminal()

			if interactive {
				return loginInteractive(f, hostname, gitProtocol, clientID)
			}

			// Non-interactive path (flags-based, like before)
			if hostname == "" {
				hostname = config.DefaultHost()
			}

			if web {
				if clientID == "" {
					clientID = config.ClientIDForHost(hostname)
				}
				if clientID == "" {
					return fmt.Errorf("--client-id is required for OAuth login (or set it with: glab config set client_id <app-id> --host %s)", hostname)
				}

				status, err := auth.OAuthFlow(hostname, clientID, errOut, browser.Open)
				if err != nil {
					return err
				}

				if gitProtocol != "" {
					if err := saveProtocol(hostname, gitProtocol); err != nil {
						return err
					}
					fmt.Fprintf(errOut, "- glab config set -h %s git_protocol %s\n", hostname, gitProtocol)
				}

				fmt.Fprintf(out, "✓ Logged in to %s as %s\n", status.Host, status.User)
				return nil
			}

			var stdinReader = ios.In
			if !stdin {
				stdinReader = nil
			}

			status, err := auth.Login(hostname, token, stdinReader)
			if err != nil {
				return err
			}

			if gitProtocol != "" {
				if err := saveProtocol(hostname, gitProtocol); err != nil {
					return err
				}
				fmt.Fprintf(errOut, "- glab config set -h %s git_protocol %s\n", hostname, gitProtocol)
			}

			fmt.Fprintf(out, "✓ Logged in to %s as %s\n", status.Host, status.User)
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "hostname", "h", "", "GitLab hostname (default: gitlab.com)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Personal access token")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read token from stdin")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Authenticate via OAuth in the browser")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth application ID (required with --web)")
	cmd.Flags().StringVarP(&gitProtocol, "git-protocol", "p", "", "Preferred git protocol for operations (https or ssh)")

	return cmd
}

// loginInteractive implements the full interactive login flow, similar to `gh auth login`.
func loginInteractive(f *cmdutil.Factory, presetHost, presetProto, presetClientID string) error {
	in := f.IOStreams.In
	out := f.IOStreams.Out
	errOut := f.IOStreams.ErrOut

	// ── Step 1: Choose host ──────────────────────────────────────────
	hostname := presetHost
	if hostname == "" {
		idx, err := prompt.Select(in, errOut,
			"What GitLab instance do you want to log into?",
			[]string{"gitlab.com", "GitLab Self-managed"})
		if err != nil {
			return err
		}
		if idx == 0 {
			hostname = "gitlab.com"
		} else {
			h, err := prompt.Input(in, errOut, "Hostname:")
			if err != nil {
				return err
			}
			if h == "" {
				return fmt.Errorf("hostname cannot be empty")
			}
			hostname = h
		}
	}

	// ── Step 2: Choose git protocol ──────────────────────────────────
	gitProtocol := presetProto
	if gitProtocol == "" {
		idx, err := prompt.Select(in, errOut,
			"What is your preferred protocol for Git operations on this host?",
			[]string{"HTTPS", "SSH"})
		if err != nil {
			return err
		}
		if idx == 0 {
			gitProtocol = "https"
		} else {
			gitProtocol = "ssh"
		}
	}

	// ── Step 3: Choose auth method ───────────────────────────────────
	idx, err := prompt.Select(in, errOut,
		"How would you like to authenticate glab?",
		[]string{"Login with a web browser", "Paste a token"})
	if err != nil {
		return err
	}

	var status *auth.Status
	if idx == 0 {
		// OAuth (web browser) flow
		clientID := presetClientID
		if clientID == "" {
			clientID = config.ClientIDForHost(hostname)
		}
		if clientID == "" {
			clientID, err = prompt.Input(in, errOut,
				fmt.Sprintf("OAuth Application ID (create one at https://%s/-/user_settings/applications):", hostname))
			if err != nil {
				return err
			}
			if clientID == "" {
				return fmt.Errorf("client ID cannot be empty")
			}
		}

		fmt.Fprintln(errOut)
		status, err = auth.OAuthFlow(hostname, clientID, errOut, browser.Open)
		if err != nil {
			return err
		}
	} else {
		// Token-based flow
		fmt.Fprintf(errOut, "\nTip: you can generate a Personal Access Token at https://%s/-/user_settings/personal_access_tokens\n", hostname)
		fmt.Fprintf(errOut, "The minimum required scopes are: %s\n", auth.ScopesDescription())

		tokenStr, err := prompt.Password(errOut, "Paste your authentication token:")
		if err != nil {
			return err
		}
		if tokenStr == "" {
			return fmt.Errorf("token cannot be empty")
		}

		status, err = auth.Login(hostname, tokenStr, nil)
		if err != nil {
			return err
		}
	}

	// ── Step 4: Configure git protocol ───────────────────────────────
	if err := saveProtocol(hostname, gitProtocol); err != nil {
		return err
	}
	fmt.Fprintf(errOut, "- glab config set -h %s git_protocol %s\n", hostname, gitProtocol)

	fmt.Fprintf(out, "✓ Logged in to %s as %s\n", status.Host, status.User)
	return nil
}

func saveProtocol(host, protocol string) error {
	hosts, err := config.LoadHosts()
	if err != nil {
		return nil // not critical
	}
	if hc, ok := hosts[host]; ok {
		hc.Protocol = protocol
		return config.SaveHosts(hosts)
	}
	return nil
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
			fmt.Fprintf(f.IOStreams.Out, "✓ Logged out of %s\n", hostname)
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
				fmt.Fprintf(out, "  ✓ Logged in as %s (%s)\n", s.User, s.Source)
				fmt.Fprintf(out, "  - Token: %s\n", s.Token)
				if s.AuthMethod != "" {
					fmt.Fprintf(out, "  - Auth method: %s\n", s.AuthMethod)
				}
				if s.HasError {
					fmt.Fprintf(out, "  X Error: %s\n", s.Error)
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

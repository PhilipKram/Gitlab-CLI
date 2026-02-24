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
	var clientID string
	var gitProtocol string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a GitLab instance",
		Long: `Authenticate with a GitLab host.

By default, authentication uses OAuth (browser-based). On first run you will be
prompted for host, git protocol, and OAuth application ID. These are stored so
subsequent logins go straight to the browser with no prompts.

Alternatively, authenticate with a personal access token using --token or --stdin.

For OAuth, you must first create an OAuth application in your GitLab instance
under Settings > Applications. Set the redirect URI to http://localhost:7171/auth/redirect
and enable the "api", "read_user", and "read_repository" scopes.`,
		Example: `  # Login via OAuth (default, opens browser)
  $ glab auth login

  # Login to a specific host via OAuth
  $ glab auth login --hostname gitlab.example.com

  # Re-login (skips all prompts if previously configured)
  $ glab auth login

  # Authenticate with a personal access token
  $ glab auth login --token glpat-xxxxxxxxxxxxxxxxxxxx

  # Pipe a token from a file
  $ glab auth login --stdin < token.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams
			out := ios.Out

			hasToken := cmd.Flags().Changed("token") || cmd.Flags().Changed("stdin")

			// If no explicit token provided, default to OAuth flow
			if !hasToken {
				return loginInteractive(f, hostname, gitProtocol, clientID)
			}

			// Token-based path (--token or --stdin)
			if hostname == "" {
				hostname = config.DefaultHost()
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
			}

			fmt.Fprintf(out, "✓ Logged in to %s as %s\n", status.Host, status.User)
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "GitLab hostname (default: gitlab.com)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Personal access token")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read token from stdin")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth application ID")
	cmd.Flags().StringVarP(&gitProtocol, "git-protocol", "p", "", "Preferred git protocol for operations (https or ssh)")

	return cmd
}

// loginInteractive implements the full interactive login flow.
// On first run it prompts for host, protocol, and client_id, then stores them.
// On subsequent runs it reuses stored values and goes straight to OAuth.
func loginInteractive(f *cmdutil.Factory, presetHost, presetProto, presetClientID string) error {
	in := f.IOStreams.In
	out := f.IOStreams.Out
	errOut := f.IOStreams.ErrOut

	// Load existing hosts to check for previously stored config
	existingHosts, _ := config.LoadHosts()

	// ── Step 1: Determine host ──────────────────────────────────────
	hostname := presetHost
	if hostname == "" {
		// If there's exactly one existing host, reuse it
		if len(existingHosts) == 1 {
			for h := range existingHosts {
				hostname = h
			}
		}
	}
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

	// ── Step 2: Determine git protocol ──────────────────────────────
	gitProtocol := presetProto
	if gitProtocol == "" {
		// Check if protocol is already stored for this host
		if hc, ok := existingHosts[hostname]; ok && hc.Protocol != "" {
			gitProtocol = hc.Protocol
		}
	}
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

	// ── Step 3: Authenticate via OAuth ──────────────────────────────
	clientID := presetClientID
	if clientID == "" {
		clientID = config.ClientIDForHost(hostname)
	}
	if clientID == "" {
		var err error
		clientID, err = prompt.Input(in, errOut,
			fmt.Sprintf("OAuth Application ID (create one at https://%s/-/user_settings/applications):", hostname))
		if err != nil {
			return err
		}
		if clientID == "" {
			return fmt.Errorf("client ID cannot be empty")
		}
		// Persist the client_id so it's not asked again
		if err := config.SetHostValue(hostname, "client_id", clientID); err != nil {
			return err
		}
	}

	fmt.Fprintln(errOut)
	redirectURI := config.RedirectURIForHost(hostname)
	scopes := config.OAuthScopesForHost(hostname)
	status, err := auth.OAuthFlow(hostname, clientID, redirectURI, scopes, errOut, browser.Open)
	if err != nil {
		return err
	}

	// ── Step 4: Configure git protocol ──────────────────────────────
	if err := saveProtocol(hostname, gitProtocol); err != nil {
		return err
	}

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

	cmd.Flags().StringVar(&hostname, "hostname", "", "GitLab hostname")

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

	cmd.Flags().StringVar(&hostname, "hostname", "", "GitLab hostname")

	return cmd
}

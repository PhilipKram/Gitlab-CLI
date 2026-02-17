package cmd

import (
	"fmt"
	"time"

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
	cmd.AddCommand(newAuthSwitchCmd(f))

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

			_, _ = fmt.Fprintf(out, "✓ Logged in to %s as %s\n", status.Host, status.User)
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

	_, _ = fmt.Fprintln(errOut)
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

	_, _ = fmt.Fprintf(out, "✓ Logged in to %s as %s\n", status.Host, status.User)
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
	var all bool

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a GitLab instance",
		Example: `  $ glab auth logout
  $ glab auth logout --hostname gitlab.example.com
  $ glab auth logout --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --all flag
			if all {
				// Prompt for confirmation before logging out of all hosts
				confirmed, err := prompt.Confirm(f.IOStreams.In, f.IOStreams.ErrOut,
					"Are you sure you want to log out of all GitLab instances?", false)
				if err != nil {
					return err
				}
				if !confirmed {
					return nil
				}

				if err := auth.LogoutAll(); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(f.IOStreams.Out, "✓ Logged out of all GitLab instances\n")
				return nil
			}

			// Handle single host logout
			if hostname == "" {
				hostname = config.DefaultHost()
			}

			// Prompt for confirmation before logging out
			confirmed, err := prompt.Confirm(f.IOStreams.In, f.IOStreams.ErrOut,
				fmt.Sprintf("Are you sure you want to log out of %s?", hostname), false)
			if err != nil {
				return err
			}
			if !confirmed {
				return nil
			}

			if err := auth.Logout(hostname); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(f.IOStreams.Out, "✓ Logged out of %s\n", hostname)
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "GitLab hostname")
	cmd.Flags().BoolVar(&all, "all", false, "Log out of all GitLab instances")

	return cmd
}

func newAuthStatusCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "View authentication status",
		Example: `  $ glab auth status
  $ glab auth status --format=json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			statuses, err := auth.GetStatus()
			if err != nil {
				return err
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				format = "json"
			}

			// Use formatter for JSON output
			if format == "json" {
				return f.FormatAndPrint(statuses, format, false)
			}

			// Default plain text output
			out := f.IOStreams.Out
			for _, s := range statuses {
				_, _ = fmt.Fprintf(out, "%s\n", s.Host)
				_, _ = fmt.Fprintf(out, "  ✓ Logged in as %s (%s)\n", s.User, s.Source)
				_, _ = fmt.Fprintf(out, "  - Token: %s\n", s.Token)
				if s.AuthMethod != "" {
					_, _ = fmt.Fprintf(out, "  - Auth method: %s\n", s.AuthMethod)
				}
				if s.AuthMethod == "oauth" && s.Scopes != "" {
					_, _ = fmt.Fprintf(out, "  - Scopes: %s\n", s.Scopes)
				}
				if s.GitLabVersion != "" {
					_, _ = fmt.Fprintf(out, "  - GitLab version: %s\n", s.GitLabVersion)
				}
				if s.AuthMethod == "oauth" && s.TokenExpiresAt > 0 {
					expiresAt := time.Unix(s.TokenExpiresAt, 0)
					if time.Now().Before(expiresAt) {
						_, _ = fmt.Fprintf(out, "  - Token expires: %s (in %s)\n", expiresAt.Format(time.RFC1123), time.Until(expiresAt).Round(time.Minute))
					} else {
						_, _ = fmt.Fprintf(out, "  - Token expired: %s (will auto-refresh on next API call)\n", expiresAt.Format(time.RFC1123))
					}
				}
				if s.HasError {
					_, _ = fmt.Fprintf(out, "  X Error: %s\n", s.Error)
				}
				_, _ = fmt.Fprintln(out)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "plain", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newAuthTokenCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		hostname string
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the auth token for a GitLab instance",
		Long:  "Print the authentication token that glab is configured to use for a given host.",
		Example: `  $ glab auth token
  $ glab auth token --hostname gitlab.example.com
  $ glab auth token --format=json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				hostname = config.DefaultHost()
			}
			token, err := auth.GetToken(hostname)
			if err != nil {
				return err
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				format = "json"
			}

			// Use formatter for JSON output
			if format == "json" {
				tokenData := map[string]string{
					"hostname": hostname,
					"token":    token,
				}
				return f.FormatAndPrint(tokenData, format, false)
			}

			// Default plain text output
			_, _ = fmt.Fprintln(f.IOStreams.Out, token)
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "GitLab hostname")
	cmd.Flags().StringVarP(&format, "format", "F", "plain", "Output format (json, table, plain)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output in JSON format (shorthand for --format=json)")

	return cmd
}

func newAuthSwitchCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch between authenticated GitLab instances",
		Long:  "Switch the active GitLab instance when you have authenticated with multiple hosts.",
		Example: `  # Interactively select a host to switch to
  $ glab auth switch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams
			in := ios.In
			errOut := ios.ErrOut
			out := ios.Out

			selectedHost, err := auth.Switch(in, errOut)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(out, "✓ Switched to %s\n", selectedHost)
			return nil
		},
	}

	return cmd
}

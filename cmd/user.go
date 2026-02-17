package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewUserCmd creates the user command group.
func NewUserCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user <command>",
		Short: "Manage users and user information",
		Long:  "View user profiles, list SSH keys, and manage user-related resources.",
	}

	cmd.AddCommand(newUserWhoamiCmd(f))
	cmd.AddCommand(newUserViewCmd(f))
	cmd.AddCommand(newUserSSHKeysCmd(f))
	cmd.AddCommand(newUserEmailsCmd(f))

	return cmd
}

func newUserWhoamiCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current authenticated user",
		Example: `  $ glab user whoami
  $ glab user whoami --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			user, resp, err := client.Users.CurrentUser()
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/user"
				return errors.NewAPIError("GET", url, statusCode, "Failed to get current user", err)
			}

			return f.FormatAndPrint(user, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (shorthand for --format json)")

	return cmd
}

func newUserViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "view <username>",
		Short: "View a user's profile",
		Args:  cobra.ExactArgs(1),
		Example: `  $ glab user view johndoe
  $ glab user view johndoe --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			client, err := f.Client()
			if err != nil {
				return err
			}

			users, resp, err := client.Users.ListUsers(&gitlab.ListUsersOptions{
				Username: &username,
			})
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/users"
				return errors.NewAPIError("GET", url, statusCode, "Failed to look up user", err)
			}

			if len(users) == 0 {
				return fmt.Errorf("user %q not found", username)
			}

			user := users[0]

			return f.FormatAndPrint(user, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (shorthand for --format json)")

	return cmd
}

func newUserSSHKeysCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:     "ssh-keys",
		Short:   "List SSH keys for the authenticated user",
		Aliases: []string{"keys"},
		Example: `  $ glab user ssh-keys
  $ glab user ssh-keys --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			keys, resp, err := client.Users.ListSSHKeys(nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/user/keys"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list SSH keys", err)
			}

			if len(keys) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No SSH keys found")
				return nil
			}

			return f.FormatAndPrint(keys, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (shorthand for --format json)")

	return cmd
}

func newUserEmailsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "emails",
		Short: "List emails for the authenticated user",
		Example: `  $ glab user emails
  $ glab user emails --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			emails, resp, err := client.Users.ListEmails()
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/user/emails"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list emails", err)
			}

			if len(emails) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No emails found")
				return nil
			}

			return f.FormatAndPrint(emails, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (shorthand for --format json)")

	return cmd
}

package cmd

import (
	"fmt"
	"strconv"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewEnvironmentCmd creates the environment command group.
func NewEnvironmentCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environment <command>",
		Short:   "Manage environments",
		Long:    "View and manage GitLab environments and their deployments.",
		Aliases: []string{"env"},
	}

	cmd.AddCommand(newEnvironmentListCmd(f))
	cmd.AddCommand(newEnvironmentViewCmd(f))
	cmd.AddCommand(newEnvironmentStopCmd(f))
	cmd.AddCommand(newEnvironmentDeleteCmd(f))

	return cmd
}

func newEnvironmentListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		web      bool
		state    string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List environments",
		Aliases: []string{"ls"},
		Example: `  $ glab environment list
  $ glab environment list --state available
  $ glab environment list --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(fmt.Sprintf("https://%s/%s/-/environments", host, project))
			}

			opts := &gitlab.ListEnvironmentsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if state != "" {
				opts.States = &state
			}

			environments, resp, err := client.Environments.ListEnvironments(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/environments"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list environments", err)
			}

			if len(environments) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No environments found")
				return nil
			}

			return f.FormatAndPrint(environments, format, jsonFlag)
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by state: available or stopped")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")

	return cmd
}

func newEnvironmentViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var format string
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View an environment",
		Example: `  $ glab environment view 123
  $ glab environment view 123 --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			environmentID, err := parseEnvironmentID(args)
			if err != nil {
				return err
			}

			environment, resp, err := client.Environments.GetEnvironment(project, environmentID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/environments/" + strconv.FormatInt(environmentID, 10)
				return errors.NewAPIError("GET", url, statusCode, "Failed to get environment", err)
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(fmt.Sprintf("https://%s/%s/-/environments/%d", host, project, environmentID))
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				format = "json"
			}

			if format == "" {
				format = "table"
			}

			return f.FormatAndPrint(environment, format, false)
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newEnvironmentStopCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [<id>]",
		Short: "Stop an environment",
		Example: `  $ glab environment stop 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			environmentID, err := parseEnvironmentID(args)
			if err != nil {
				return err
			}

			_, resp, err := client.Environments.StopEnvironment(project, environmentID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/environments/" + strconv.FormatInt(environmentID, 10) + "/stop"
				return errors.NewAPIError("POST", url, statusCode, "Failed to stop environment", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Environment #%d stopped\n", environmentID)
			return nil
		},
	}

	return cmd
}

func newEnvironmentDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [<id>]",
		Short: "Delete an environment",
		Example: `  $ glab environment delete 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			environmentID, err := parseEnvironmentID(args)
			if err != nil {
				return err
			}

			resp, err := client.Environments.DeleteEnvironment(project, environmentID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/environments/" + strconv.FormatInt(environmentID, 10)
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete environment", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Environment #%d deleted\n", environmentID)
			return nil
		},
	}

	return cmd
}

func parseEnvironmentID(args []string) (int64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("environment ID is required")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid environment ID: %s", args[0])
	}
	return id, nil
}

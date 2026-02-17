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

// NewDeploymentCmd creates the deployment command group.
func NewDeploymentCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment <command>",
		Short:   "Manage deployments",
		Long:    "View and manage GitLab deployments.",
		Aliases: []string{"deploy"},
	}

	cmd.AddCommand(newDeploymentListCmd(f))
	cmd.AddCommand(newDeploymentViewCmd(f))

	return cmd
}

func newDeploymentListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit       int
		format      string
		jsonFlag    bool
		web         bool
		status      string
		environment string
		orderBy     string
		sort        string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List deployments",
		Aliases: []string{"ls"},
		Example: `  $ glab deployment list
  $ glab deployment list --environment production
  $ glab deployment list --status success --limit 50`,
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
				return browser.Open(fmt.Sprintf("https://%s/%s/-/deployments", host, project))
			}

			opts := &gitlab.ListProjectDeploymentsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if status != "" {
				opts.Status = &status
			}
			if environment != "" {
				opts.Environment = &environment
			}
			if orderBy != "" {
				opts.OrderBy = &orderBy
			}
			if sort != "" {
				opts.Sort = &sort
			}

			deployments, resp, err := client.Deployments.ListProjectDeployments(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/deployments"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list deployments", err)
			}

			if len(deployments) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No deployments found")
				return nil
			}

			return f.FormatAndPrint(deployments, format, jsonFlag)
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status: created, running, success, failed, canceled")
	cmd.Flags().StringVar(&environment, "environment", "", "Filter by environment name")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "Order by: id, iid, created_at, updated_at, ref")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order: asc or desc")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")

	return cmd
}

func newDeploymentViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var format string
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View a deployment",
		Example: `  $ glab deployment view 12345
  $ glab deployment view 12345 --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			deploymentID, err := parseDeploymentID(args)
			if err != nil {
				return err
			}

			deployment, resp, err := client.Deployments.GetProjectDeployment(project, deploymentID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/deployments/" + strconv.FormatInt(deploymentID, 10)
				return errors.NewAPIError("GET", url, statusCode, "Failed to get deployment", err)
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(fmt.Sprintf("https://%s/%s/-/deployments/%d", host, project, deploymentID))
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				format = "json"
			}

			if format == "" {
				format = "table"
			}

			return f.FormatAndPrint(deployment, format, false)
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

// parseDeploymentID parses a deployment ID from command arguments.
func parseDeploymentID(args []string) (int64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("deployment ID required")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid deployment ID: %s", args[0])
	}

	return id, nil
}

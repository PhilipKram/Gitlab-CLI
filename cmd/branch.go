package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewBranchCmd creates the branch command group.
func NewBranchCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch <command>",
		Short: "Manage branches",
		Long:  "List, create, and delete repository branches.",
	}

	cmd.AddCommand(newBranchListCmd(f))
	cmd.AddCommand(newBranchCreateCmd(f))
	cmd.AddCommand(newBranchDeleteCmd(f))

	return cmd
}

func newBranchListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		search   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List branches",
		Aliases: []string{"ls"},
		Example: `  $ glab branch list
  $ glab branch list --search feature
  $ glab branch list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.ListBranchesOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if search != "" {
				opts.Search = &search
			}

			branches, resp, err := client.Branches.ListBranches(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/repository/branches"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list branches", err)
			}

			if len(branches) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No branches found. Try adjusting --search or increase --limit.")
				return nil
			}

			return f.FormatAndPrint(branches, format, jsonFlag)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().StringVar(&search, "search", "", "Search branches by name")

	return cmd
}

func newBranchCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name string
		ref  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a branch",
		Example: `  $ glab branch create --name feature-branch
  $ glab branch create --name hotfix --ref develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.CreateBranchOptions{
				Branch: &name,
				Ref:    &ref,
			}

			branch, resp, err := client.Branches.CreateBranch(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/repository/branches"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create branch", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Created branch %q from %q\n", branch.Name, ref)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Branch name (required)")
	cmd.Flags().StringVarP(&ref, "ref", "r", "main", "Source branch or commit SHA")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newBranchDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <branch>",
		Short:   "Delete a branch",
		Example: `  $ glab branch delete feature-branch`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			branchName := args[0]

			resp, err := client.Branches.DeleteBranch(project, branchName)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/repository/branches/" + branchName
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete branch", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted branch %q\n", branchName)
			return nil
		},
	}

	return cmd
}

package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewTagCmd creates the tag command group.
func NewTagCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag <command>",
		Short: "Manage tags",
		Long:  "List, create, and delete repository tags.",
	}

	cmd.AddCommand(newTagListCmd(f))
	cmd.AddCommand(newTagCreateCmd(f))
	cmd.AddCommand(newTagDeleteCmd(f))

	return cmd
}

func newTagListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List tags",
		Aliases: []string{"ls"},
		Example: `  $ glab tag list
  $ glab tag list --limit 10
  $ glab tag list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.ListTagsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			tags, resp, err := client.Tags.ListTags(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/repository/tags"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list tags", err)
			}

			if len(tags) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No tags found")
				return nil
			}

			return f.FormatAndPrint(tags, format, jsonFlag)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newTagCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name    string
		ref     string
		message string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a tag",
		Example: `  $ glab tag create --name v1.0.0
  $ glab tag create --name v1.0.0 --ref develop --message "Release v1.0.0"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.CreateTagOptions{
				TagName: &name,
				Ref:     &ref,
			}

			if message != "" {
				opts.Message = &message
			}

			tag, resp, err := client.Tags.CreateTag(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/repository/tags"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create tag", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Created tag %q from %q\n", tag.Name, ref)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Tag name (required)")
	cmd.Flags().StringVarP(&ref, "ref", "r", "main", "Source branch or commit SHA")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Tag message (creates annotated tag)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newTagDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <tag>",
		Short:   "Delete a tag",
		Example: `  $ glab tag delete v1.0.0`,
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

			tagName := args[0]

			resp, err := client.Tags.DeleteTag(project, tagName)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/repository/tags/" + tagName
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete tag", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted tag %q\n", tagName)
			return nil
		},
	}

	return cmd
}

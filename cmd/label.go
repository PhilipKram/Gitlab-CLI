package cmd

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewLabelCmd creates the label command group.
func NewLabelCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label <command>",
		Short: "Manage labels",
		Long:  "Create, list, and delete project labels.",
	}

	cmd.AddCommand(newLabelCreateCmd(f))
	cmd.AddCommand(newLabelListCmd(f))
	cmd.AddCommand(newLabelDeleteCmd(f))

	return cmd
}

func newLabelCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name        string
		color       string
		description string
		priority    int64
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a label",
		Example: `  $ glab label create --name bug --color "#FF0000"
  $ glab label create --name feature --color "#00FF00" --description "New features"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.CreateLabelOptions{
				Name:        &name,
				Color:       &color,
				Description: &description,
			}

			if cmd.Flags().Changed("priority") {
				opts.Priority = &priority
			}

			label, resp, err := client.Labels.CreateLabel(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/labels"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create label", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Created label %q (%s)\n", label.Name, label.Color)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Label name (required)")
	cmd.Flags().StringVarP(&color, "color", "c", "", "Label color in hex (required, e.g., #FF0000)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Label description")
	cmd.Flags().Int64Var(&priority, "priority", 0, "Label priority")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("color")

	return cmd
}

func newLabelListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		search   string
		stream   bool
		web      bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List labels",
		Aliases: []string{"ls"},
		Example: `  $ glab label list`,
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
				return browser.Open(api.WebURL(host, project+"/-/labels"))
			}

			opts := &gitlab.ListLabelsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if search != "" {
				opts.Search = &search
			}

			outputFormat, err := f.ResolveFormat(format, jsonFlag)
			if err != nil {
				return err
			}

			// Use streaming mode if --stream flag is set
			if stream {
				// Create context for pagination
				ctx := context.Background()

				// Create fetch function for pagination
				fetchFunc := func(page int) ([]*gitlab.Label, *gitlab.Response, error) {
					pageOpts := *opts
					pageOpts.Page = int64(page)
					if pageOpts.PerPage == 0 {
						pageOpts.PerPage = 100
					}
					return client.Labels.ListLabels(project, &pageOpts)
				}

				// Configure pagination options
				paginateOpts := api.PaginateOptions{
					PerPage:    int(opts.PerPage),
					BufferSize: 100,
				}
				if limit > 0 && limit < 100 {
					paginateOpts.PerPage = limit
					paginateOpts.BufferSize = limit
				}

				// Start pagination
				results := api.PaginateToChannel(ctx, fetchFunc, paginateOpts)

				return cmdutil.FormatAndStream(f, results, outputFormat, limit, "labels")
			}

			// Non-streaming mode: fetch all at once
			labels, resp, err := client.Labels.ListLabels(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/labels"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list labels", err)
			}

			if len(labels) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No labels found. Try increasing --limit.")
				return nil
			}

			return f.FormatAndPrint(labels, format, jsonFlag)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().StringVar(&search, "search", "", "Search labels")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming mode")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")

	return cmd
}

func newLabelDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a label",
		Example: `  $ glab label delete bug`,
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

			resp, err := client.Labels.DeleteLabel(project, args[0], nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/labels/" + args[0]
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete label", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted label %q\n", args[0])
			return nil
		},
	}

	return cmd
}

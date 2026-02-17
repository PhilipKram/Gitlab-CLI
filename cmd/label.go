package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
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

			label, _, err := client.Labels.CreateLabel(project, opts)
			if err != nil {
				return fmt.Errorf("creating label: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Created label %q (%s)\n", label.Name, label.Color)
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
		jsonFlag bool
		search   string
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

			opts := &gitlab.ListLabelsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if search != "" {
				opts.Search = &search
			}

			labels, _, err := client.Labels.ListLabels(project, opts)
			if err != nil {
				return fmt.Errorf("listing labels: %w", err)
			}

			if len(labels) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No labels found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(labels, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, l := range labels {
				tp.AddRow(l.Name, l.Color, l.Description)
			}
			return tp.Render()
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&search, "search", "", "Search labels")

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

			_, err = client.Labels.DeleteLabel(project, args[0], nil)
			if err != nil {
				return fmt.Errorf("deleting label: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted label %q\n", args[0])
			return nil
		},
	}

	return cmd
}

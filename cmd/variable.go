package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewVariableCmd creates the variable command group.
func NewVariableCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variable <command>",
		Short: "Manage CI/CD variables",
		Long:  "Create, list, update, and delete CI/CD variables at project and group levels.",
	}

	cmd.AddCommand(newVariableListCmd(f))

	return cmd
}

func newVariableListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		jsonFlag bool
		group    string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List CI/CD variables",
		Aliases: []string{"ls"},
		Example: `  $ glab variable list
  $ glab variable list --group mygroup
  $ glab variable list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var variables []*gitlab.ProjectVariable
			var groupVariables []*gitlab.GroupVariable

			if group != "" {
				// List group-level variables
				groupVariables, _, err = client.GroupVariables.ListVariables(group, nil)
				if err != nil {
					return fmt.Errorf("listing group variables: %w", err)
				}

				if len(groupVariables) == 0 {
					fmt.Fprintln(f.IOStreams.ErrOut, "No variables found")
					return nil
				}

				if jsonFlag {
					data, err := json.MarshalIndent(groupVariables, "", "  ")
					if err != nil {
						return err
					}
					fmt.Fprintln(f.IOStreams.Out, string(data))
					return nil
				}

				tp := tableprinter.New(f.IOStreams.Out)
				tp.AddRow("KEY", "SCOPE", "PROTECTED", "MASKED", "TYPE")
				for _, v := range groupVariables {
					protected := "false"
					if v.Protected {
						protected = "true"
					}
					masked := "false"
					if v.Masked {
						masked = "true"
					}
					variableType := "env_var"
					if v.VariableType == "file" {
						variableType = "file"
					}
					tp.AddRow(v.Key, v.EnvironmentScope, protected, masked, variableType)
				}
				return tp.Render()
			}

			// List project-level variables
			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			variables, _, err = client.ProjectVariables.ListVariables(project, nil)
			if err != nil {
				return fmt.Errorf("listing project variables: %w", err)
			}

			if len(variables) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No variables found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(variables, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			tp.AddRow("KEY", "SCOPE", "PROTECTED", "MASKED", "TYPE")
			for _, v := range variables {
				protected := "false"
				if v.Protected {
					protected = "true"
				}
				masked := "false"
				if v.Masked {
					masked = "true"
				}
				variableType := "env_var"
				if v.VariableType == "file" {
					variableType = "file"
				}
				tp.AddRow(v.Key, v.EnvironmentScope, protected, masked, variableType)
			}
			return tp.Render()
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().StringVarP(&group, "group", "g", "", "List group-level variables (specify group path)")

	return cmd
}

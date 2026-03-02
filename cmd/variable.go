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
	cmd.AddCommand(newVariableGetCmd(f))

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

func newVariableGetCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		jsonFlag bool
		group    string
	)

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a CI/CD variable",
		Example: `  $ glab variable get MY_VAR
  $ glab variable get MY_VAR --group mygroup
  $ glab variable get MY_VAR --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			key := args[0]

			if group != "" {
				// Get group-level variable
				variable, _, err := client.GroupVariables.GetVariable(group, key, nil)
				if err != nil {
					return fmt.Errorf("getting group variable: %w", err)
				}

				if jsonFlag {
					data, err := json.MarshalIndent(variable, "", "  ")
					if err != nil {
						return err
					}
					fmt.Fprintln(f.IOStreams.Out, string(data))
					return nil
				}

				// Display variable details
				out := f.IOStreams.Out
				fmt.Fprintf(out, "Key: %s\n", variable.Key)
				fmt.Fprintf(out, "Value: %s\n", variable.Value)
				fmt.Fprintf(out, "Environment Scope: %s\n", variable.EnvironmentScope)
				fmt.Fprintf(out, "Protected: %t\n", variable.Protected)
				fmt.Fprintf(out, "Masked: %t\n", variable.Masked)
				fmt.Fprintf(out, "Variable Type: %s\n", variable.VariableType)
				return nil
			}

			// Get project-level variable
			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			variable, _, err := client.ProjectVariables.GetVariable(project, key, nil)
			if err != nil {
				return fmt.Errorf("getting project variable: %w", err)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(variable, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			// Display variable details
			out := f.IOStreams.Out
			fmt.Fprintf(out, "Key: %s\n", variable.Key)
			fmt.Fprintf(out, "Value: %s\n", variable.Value)
			fmt.Fprintf(out, "Environment Scope: %s\n", variable.EnvironmentScope)
			fmt.Fprintf(out, "Protected: %t\n", variable.Protected)
			fmt.Fprintf(out, "Masked: %t\n", variable.Masked)
			fmt.Fprintf(out, "Variable Type: %s\n", variable.VariableType)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Get group-level variable (specify group path)")

	return cmd
}

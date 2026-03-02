package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
	cmd.AddCommand(newVariableSetCmd(f))
	cmd.AddCommand(newVariableUpdateCmd(f))
	cmd.AddCommand(newVariableDeleteCmd(f))
	cmd.AddCommand(newVariableExportCmd(f))
	cmd.AddCommand(newVariableImportCmd(f))

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

func newVariableSetCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		value      string
		masked     bool
		protected  bool
		scope      string
		filePath   string
		group      string
		varType    string
	)

	cmd := &cobra.Command{
		Use:   "set <key>",
		Short: "Set a CI/CD variable",
		Example: `  $ glab variable set MY_VAR --value "my-value"
  $ glab variable set MY_VAR --value "secret" --masked --protected
  $ glab variable set MY_VAR --file ./config.json --scope production
  $ glab variable set MY_VAR --value "group-secret" --group mygroup`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			key := args[0]

			// Get value from file or flag
			varValue := value
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("reading file: %w", err)
				}
				varValue = string(data)
			}

			if varValue == "" {
				return fmt.Errorf("either --value or --file flag is required")
			}

			// Default scope
			if scope == "" {
				scope = "*"
			}

			// Default variable type
			variableType := gitlab.EnvVariableType
			if varType == "file" {
				variableType = gitlab.FileVariableType
			}

			if group != "" {
				// Set group-level variable
				// Try to update first, if it fails (not found), create it
				updateOpts := &gitlab.UpdateGroupVariableOptions{
					Value:            &varValue,
					Protected:        &protected,
					Masked:           &masked,
					EnvironmentScope: &scope,
					VariableType:     &variableType,
				}

				variable, _, err := client.GroupVariables.UpdateVariable(group, key, updateOpts)
				if err != nil {
					// If variable doesn't exist, create it
					createOpts := &gitlab.CreateGroupVariableOptions{
						Key:              &key,
						Value:            &varValue,
						Protected:        &protected,
						Masked:           &masked,
						EnvironmentScope: &scope,
						VariableType:     &variableType,
					}

					variable, _, err = client.GroupVariables.CreateVariable(group, createOpts)
					if err != nil {
						return fmt.Errorf("setting group variable: %w", err)
					}

					fmt.Fprintf(f.IOStreams.Out, "Created group variable %q\n", variable.Key)
					return nil
				}

				fmt.Fprintf(f.IOStreams.Out, "Updated group variable %q\n", variable.Key)
				return nil
			}

			// Set project-level variable
			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			// Try to update first, if it fails (not found), create it
			updateOpts := &gitlab.UpdateProjectVariableOptions{
				Value:            &varValue,
				Protected:        &protected,
				Masked:           &masked,
				EnvironmentScope: &scope,
				VariableType:     &variableType,
			}

			variable, _, err := client.ProjectVariables.UpdateVariable(project, key, updateOpts)
			if err != nil {
				// If variable doesn't exist, create it
				createOpts := &gitlab.CreateProjectVariableOptions{
					Key:              &key,
					Value:            &varValue,
					Protected:        &protected,
					Masked:           &masked,
					EnvironmentScope: &scope,
					VariableType:     &variableType,
				}

				variable, _, err = client.ProjectVariables.CreateVariable(project, createOpts)
				if err != nil {
					return fmt.Errorf("setting project variable: %w", err)
				}

				fmt.Fprintf(f.IOStreams.Out, "Created variable %q\n", variable.Key)
				return nil
			}

			fmt.Fprintf(f.IOStreams.Out, "Updated variable %q\n", variable.Key)
			return nil
		},
	}

	cmd.Flags().StringVarP(&value, "value", "v", "", "Variable value")
	cmd.Flags().BoolVar(&masked, "masked", false, "Mask variable value in logs")
	cmd.Flags().BoolVar(&protected, "protected", false, "Protect variable (only available in protected branches/tags)")
	cmd.Flags().StringVar(&scope, "scope", "*", "Environment scope (default: *)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read variable value from file")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Set group-level variable (specify group path)")
	cmd.Flags().StringVar(&varType, "type", "env_var", "Variable type: env_var or file")

	return cmd
}

func newVariableUpdateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		value      string
		masked     bool
		protected  bool
		scope      string
		filePath   string
		group      string
		varType    string
	)

	cmd := &cobra.Command{
		Use:   "update <key>",
		Short: "Update an existing CI/CD variable",
		Example: `  $ glab variable update MY_VAR --value "new-value"
  $ glab variable update MY_VAR --masked --protected
  $ glab variable update MY_VAR --file ./config.json --scope production
  $ glab variable update MY_VAR --value "updated-secret" --group mygroup`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			key := args[0]

			// Get value from file or flag
			varValue := value
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("reading file: %w", err)
				}
				varValue = string(data)
			}

			if varValue == "" {
				return fmt.Errorf("either --value or --file flag is required")
			}

			// Default scope
			if scope == "" {
				scope = "*"
			}

			// Default variable type
			variableType := gitlab.EnvVariableType
			if varType == "file" {
				variableType = gitlab.FileVariableType
			}

			if group != "" {
				// Update group-level variable
				updateOpts := &gitlab.UpdateGroupVariableOptions{
					Value:            &varValue,
					Protected:        &protected,
					Masked:           &masked,
					EnvironmentScope: &scope,
					VariableType:     &variableType,
				}

				variable, _, err := client.GroupVariables.UpdateVariable(group, key, updateOpts)
				if err != nil {
					return fmt.Errorf("updating group variable: %w", err)
				}

				fmt.Fprintf(f.IOStreams.Out, "Updated group variable %q\n", variable.Key)
				return nil
			}

			// Update project-level variable
			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			updateOpts := &gitlab.UpdateProjectVariableOptions{
				Value:            &varValue,
				Protected:        &protected,
				Masked:           &masked,
				EnvironmentScope: &scope,
				VariableType:     &variableType,
			}

			variable, _, err := client.ProjectVariables.UpdateVariable(project, key, updateOpts)
			if err != nil {
				return fmt.Errorf("updating project variable: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Updated variable %q\n", variable.Key)
			return nil
		},
	}

	cmd.Flags().StringVarP(&value, "value", "v", "", "Variable value")
	cmd.Flags().BoolVar(&masked, "masked", false, "Mask variable value in logs")
	cmd.Flags().BoolVar(&protected, "protected", false, "Protect variable (only available in protected branches/tags)")
	cmd.Flags().StringVar(&scope, "scope", "*", "Environment scope (default: *)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read variable value from file")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Update group-level variable (specify group path)")
	cmd.Flags().StringVar(&varType, "type", "env_var", "Variable type: env_var or file")

	return cmd
}

func newVariableDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var group string

	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a CI/CD variable",
		Example: `  $ glab variable delete MY_VAR
  $ glab variable delete MY_VAR --group mygroup`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			key := args[0]

			if group != "" {
				// Delete group-level variable
				_, err = client.GroupVariables.RemoveVariable(group, key, nil)
				if err != nil {
					return fmt.Errorf("deleting group variable: %w", err)
				}

				fmt.Fprintf(f.IOStreams.Out, "Deleted group variable %q\n", key)
				return nil
			}

			// Delete project-level variable
			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			_, err = client.ProjectVariables.RemoveVariable(project, key, nil)
			if err != nil {
				return fmt.Errorf("deleting project variable: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted variable %q\n", key)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Delete group-level variable (specify group path)")

	return cmd
}

func newVariableExportCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		group  string
		output string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export CI/CD variables to JSON",
		Example: `  $ glab variable export
  $ glab variable export --group mygroup
  $ glab variable export --output variables.json
  $ glab variable export --group mygroup --output group-vars.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var data []byte

			if group != "" {
				// Export group-level variables
				groupVariables, _, err := client.GroupVariables.ListVariables(group, nil)
				if err != nil {
					return fmt.Errorf("listing group variables: %w", err)
				}

				data, err = json.MarshalIndent(groupVariables, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling variables: %w", err)
				}
			} else {
				// Export project-level variables
				project, err := f.FullProjectPath()
				if err != nil {
					return err
				}

				variables, _, err := client.ProjectVariables.ListVariables(project, nil)
				if err != nil {
					return fmt.Errorf("listing project variables: %w", err)
				}

				data, err = json.MarshalIndent(variables, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling variables: %w", err)
				}
			}

			// Write to file or stdout
			if output != "" {
				err := os.WriteFile(output, data, 0600)
				if err != nil {
					return fmt.Errorf("writing to file: %w", err)
				}
				fmt.Fprintf(f.IOStreams.Out, "Exported variables to %s\n", output)
			} else {
				fmt.Fprintln(f.IOStreams.Out, string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Export group-level variables (specify group path)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: stdout)")

	return cmd
}

func newVariableImportCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		group string
		file  string
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import CI/CD variables from JSON",
		Example: `  $ glab variable import --file variables.json
  $ glab variable import --file group-vars.json --group mygroup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			if file == "" {
				return fmt.Errorf("--file flag is required")
			}

			// Read the JSON file
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			if group != "" {
				// Import group-level variables
				var variables []*gitlab.GroupVariable
				err = json.Unmarshal(data, &variables)
				if err != nil {
					return fmt.Errorf("parsing JSON: %w", err)
				}

				imported := 0
				for _, v := range variables {
					// Try to update first, if it fails, create it
					updateOpts := &gitlab.UpdateGroupVariableOptions{
						Value:            &v.Value,
						Protected:        &v.Protected,
						Masked:           &v.Masked,
						EnvironmentScope: &v.EnvironmentScope,
						VariableType:     &v.VariableType,
					}

					_, _, err := client.GroupVariables.UpdateVariable(group, v.Key, updateOpts)
					if err != nil {
						// Variable doesn't exist, create it
						createOpts := &gitlab.CreateGroupVariableOptions{
							Key:              &v.Key,
							Value:            &v.Value,
							Protected:        &v.Protected,
							Masked:           &v.Masked,
							EnvironmentScope: &v.EnvironmentScope,
							VariableType:     &v.VariableType,
						}

						_, _, err = client.GroupVariables.CreateVariable(group, createOpts)
						if err != nil {
							fmt.Fprintf(f.IOStreams.ErrOut, "Warning: failed to import variable %q: %v\n", v.Key, err)
							continue
						}
					}
					imported++
				}

				fmt.Fprintf(f.IOStreams.Out, "Imported %d group variable(s)\n", imported)
				return nil
			}

			// Import project-level variables
			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			var variables []*gitlab.ProjectVariable
			err = json.Unmarshal(data, &variables)
			if err != nil {
				return fmt.Errorf("parsing JSON: %w", err)
			}

			imported := 0
			for _, v := range variables {
				// Try to update first, if it fails, create it
				updateOpts := &gitlab.UpdateProjectVariableOptions{
					Value:            &v.Value,
					Protected:        &v.Protected,
					Masked:           &v.Masked,
					EnvironmentScope: &v.EnvironmentScope,
					VariableType:     &v.VariableType,
				}

				_, _, err := client.ProjectVariables.UpdateVariable(project, v.Key, updateOpts)
				if err != nil {
					// Variable doesn't exist, create it
					createOpts := &gitlab.CreateProjectVariableOptions{
						Key:              &v.Key,
						Value:            &v.Value,
						Protected:        &v.Protected,
						Masked:           &v.Masked,
						EnvironmentScope: &v.EnvironmentScope,
						VariableType:     &v.VariableType,
					}

					_, _, err = client.ProjectVariables.CreateVariable(project, createOpts)
					if err != nil {
						fmt.Fprintf(f.IOStreams.ErrOut, "Warning: failed to import variable %q: %v\n", v.Key, err)
						continue
					}
				}
				imported++
			}

			fmt.Fprintf(f.IOStreams.Out, "Imported %d variable(s)\n", imported)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Import group-level variables (specify group path)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Input JSON file path (required)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

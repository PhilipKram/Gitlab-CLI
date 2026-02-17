package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command group.
func NewConfigCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <command>",
		Short: "Manage configuration",
		Long:  "Get and set glab configuration options.",
	}

	cmd.AddCommand(newConfigGetCmd(f))
	cmd.AddCommand(newConfigSetCmd(f))
	cmd.AddCommand(newConfigListCmd(f))

	return cmd
}

func newConfigGetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Example: `  $ glab config get editor
  $ glab config get protocol`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			value, err := cfg.Get(args[0])
			if err != nil {
				return err
			}

			if value == "" {
				return fmt.Errorf("key %q is not set", args[0])
			}

			fmt.Fprintln(f.IOStreams.Out, value)
			return nil
		},
	}

	return cmd
}

func newConfigSetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Available keys:
  editor       - Preferred text editor
  pager        - Preferred pager program
  browser      - Preferred web browser
  protocol     - Preferred git protocol (https or ssh)
  git_remote   - Preferred git remote name`,
		Example: `  $ glab config set editor vim
  $ glab config set protocol ssh
  $ glab config set git_remote upstream`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			if err := cfg.Set(args[0], args[1]); err != nil {
				return err
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Fprintf(f.IOStreams.Out, "Set %s = %s\n", args[0], args[1])
			return nil
		},
	}

	return cmd
}

func newConfigListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List configuration values",
		Aliases: []string{"ls"},
		Example: `  $ glab config list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			out := f.IOStreams.Out
			for _, key := range config.Keys() {
				value, _ := cfg.Get(key)
				if value == "" {
					value = "(not set)"
				}
				fmt.Fprintf(out, "%s=%s\n", key, value)
			}

			return nil
		},
	}

	return cmd
}

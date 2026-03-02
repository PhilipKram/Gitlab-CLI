package cmd

import (
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/update"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for glab.
func NewRootCmd(version string) *cobra.Command {
	f := cmdutil.NewFactory()
	f.Version = version

	var repoOverride string

	cmd := &cobra.Command{
		Use:   "glab <command> <subcommand> [flags]",
		Short: "GitLab CLI",
		Long:  "Work seamlessly with GitLab from the command line.",
		Example: `  $ glab auth login
  $ glab mr create --title "Feature" --description "Add new feature"
  $ glab issue list --state opened
  $ glab pipeline list
  $ glab repo clone owner/repo`,
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if repoOverride != "" {
				f.SetRepoOverride(repoOverride)
			}
			// Show update banner (reads cached state, instant)
			if version != "dev" {
				update.PrintUpdateNotice(f.IOStreams.ErrOut, version)
			}
			// Kick off background check to refresh cache for next run
			if version != "dev" {
				go update.CheckAndCache(version)
			}
		},
	}

	cmd.PersistentFlags().StringVarP(&repoOverride, "repo", "R", "", "Select a GitLab repository using the HOST/OWNER/REPO format")
	cmd.SetVersionTemplate("glab version {{.Version}}\n")

	// Core commands
	cmd.AddCommand(NewAuthCmd(f))
	cmd.AddCommand(NewMRCmd(f))
	cmd.AddCommand(NewIssueCmd(f))
	cmd.AddCommand(NewRepoCmd(f))

	// CI/CD commands
	cmd.AddCommand(NewPipelineCmd(f))
	cmd.AddCommand(NewReleaseCmd(f))
	cmd.AddCommand(NewVariableCmd(f))

	// Additional commands
	cmd.AddCommand(NewSnippetCmd(f))
	cmd.AddCommand(NewLabelCmd(f))
	cmd.AddCommand(NewProjectCmd(f))

	// Utility commands
	cmd.AddCommand(NewAPICmd(f))
	cmd.AddCommand(NewBrowseCmd(f))
	cmd.AddCommand(NewConfigCmd(f))
	cmd.AddCommand(NewCompletionCmd())
	cmd.AddCommand(NewUpgradeCmd(f))

	// Use grouped help only on the root command
	cobra.AddTemplateFunc("isRootCmd", func(cmd *cobra.Command) bool {
		return !cmd.HasParent()
	})

	cmd.SetUsageTemplate(usageTemplate)

	return cmd
}

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}
{{if isRootCmd .}}
Core Commands:
  auth        Authenticate glab and git with GitLab
  mr          Manage merge requests
  issue       Manage issues
  repo        Manage repositories

CI/CD Commands:
  pipeline    Manage pipelines and CI/CD
  release     Manage releases
  variable    Manage CI/CD variables

Additional Commands:
  snippet     Manage snippets
  label       Manage labels
  project     Manage projects

Utility Commands:
  api         Make authenticated API requests
  browse      Open project in browser
  config      Manage configuration
  completion  Generate shell completion scripts
  upgrade     Upgrade glab to the latest version
{{else}}
Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

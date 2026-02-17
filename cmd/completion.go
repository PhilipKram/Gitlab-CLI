package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCmd creates the completion command.
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for glab.

To load completions:

  Bash:
    $ source <(glab completion bash)
    # or add to ~/.bashrc:
    $ glab completion bash > /etc/bash_completion.d/glab

  Zsh:
    $ glab completion zsh > "${fpath[1]}/_glab"
    # then restart your shell

  Fish:
    $ glab completion fish | source
    # or persist:
    $ glab completion fish > ~/.config/fish/completions/glab.fish

  PowerShell:
    PS> glab completion powershell | Out-String | Invoke-Expression`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}

	return cmd
}

package cli

import (
	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion scripts for stamp.

To load completions:

Bash:

  $ source <(stamp completion bash)

  # To load for each session:
  # Linux:
  $ stamp completion bash > /etc/bash_completion.d/stamp
  # macOS:
  $ stamp completion bash > $(brew --prefix)/etc/bash_completion.d/stamp

Zsh:

  # If completion not enabled, run once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  $ stamp completion zsh > "${fpath[1]}/_stamp"

fish:

  $ stamp completion fish | source

  # To load for each session:
  $ stamp completion fish > ~/.config/fish/completions/stamp.fish

PowerShell:

  PS> stamp completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return nil
			}
		},
	}

	return cmd
}

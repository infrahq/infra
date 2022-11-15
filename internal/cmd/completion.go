package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCompletionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion",
		Short: "Generate shell auto-completion for the CLI",
		Long: fmt.Sprintf(`To load completions:

##### Bash:

`+"`"+`$ source <(%[1]s completion bash)`+"`"+`

To load completions for each session, execute once:
* Linux:
  `+"`"+`$ %[1]s completion bash > /etc/bash_completion.d/%[1]s`+"`"+`
* macOS:
  `+"`"+`$ %[1]s completion bash > /usr/local/etc/bash_completion.d/%[1]s`+"`"+`

##### Zsh:

If shell completion is not already enabled in your environment, you will need to enable it. You can execute the following once:
`+"`"+`$ echo "autoload -U compinit; compinit" >> ~/.zshrc`+"`"+`

To load completions for each session, execute once:
`+"`"+`$ %[1]s completion zsh > "${fpath[1]}/_%[1]s"`+"`"+`

You will need to start a new shell for this setup to take effect.

##### fish:

`+"`"+`$ %[1]s completion fish | source`+"`"+`

To load completions for each session, execute once:
`+"`"+`$ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish`+"`"+`

##### PowerShell:

`+"`"+`PS> %[1]s completion powershell | Out-String | Invoke-Expression`+"`"+`

To load completions for every new session, run:
`+"`"+`PS> %[1]s completion powershell > %[1]s.ps1`+"`"+`
and source this file from your PowerShell profile.
`, `infra`),
		DisableFlagsInUseLine: true,
		GroupID:               groupOther,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Run: func(cmd *cobra.Command, args []string) {
			shell := filepath.Base(os.Getenv("SHELL"))
			if len(args) > 0 {
				shell = args[0]
			}

			switch shell {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout) // nolint
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout) // nolint
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true) // nolint
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout) // nolint
			default:
				fmt.Fprintf(os.Stdout, "No completions found for specified shell.\n")
			}
		},
	}
}

// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(zim completion bash)

# To load completions for each session, execute once:
Linux:
  $ zim completion bash > /etc/bash_completion.d/zim
MacOS:
  $ zim completion bash > /usr/local/etc/bash_completion.d/zim

Zsh:

$ source <(zim completion zsh)

# To load completions for each session with oh-my-zsh, execute once:
  $ mkdir -p ~/.oh-my-zsh/completions
  $ zim completion zsh > ~/.oh-my-zsh/completions/_zim
# Then execute to reload for your current session:
  $ exec zsh

# To load completions for each session manually with zsh, place the completions
# in a directory (~/.completions in this example):
  $ mkdir -p ~/.completions
  $ zim completion zsh > ~/.completions/_zim
# Then add the directory to your fpath, for example by adding in ~/.zshrc
  fpath=(~/.completions $fpath)
# Then execute to reload for your current session
  $ exec zsh

Fish:

$ zim completion fish | source

# To load completions for each session, execute once:
$ zim completion fish > ~/.config/fish/completions/zim.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fugue/zim/cache"
	"github.com/fugue/zim/exec"
	"github.com/fugue/zim/project"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewShowKeyCommand returns a command that shows a cache key for a Rule
func NewShowKeyCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "key",
		Short: "Show a rule cache key",
		Run: func(cmd *cobra.Command, args []string) {

			var ruleName, componentName string
			opts := getZimOptions(cmd, args)

			if gitDir, err := gitRoot(opts.Directory); err == nil {
				opts.Directory = gitDir
			}

			if len(opts.Rules) != 1 {
				fatal(errors.New("Must specify exactly one rule name with -r"))
			}

			ruleName = opts.Rules[0]
			if strings.Contains(ruleName, ".") {
				parts := strings.SplitN(ruleName, ".", 2)
				ruleName = parts[1]
				opts.Components = []string{parts[0]}
			}

			if len(opts.Components) != 1 {
				fatal(errors.New("Must specify exactly one component name with -c"))
			}
			componentName = opts.Components[0]

			var executor exec.Executor
			if opts.UseDocker {
				executor = exec.NewDockerExecutor(opts.Directory, opts.Platform)
			} else {
				executor = exec.NewBashExecutor()
			}

			projDef, componentDefs, err := project.Discover(opts.Directory)
			if err != nil {
				fatal(err)
			}

			// Load selected components from the project
			proj, err := project.NewWithOptions(project.Opts{
				Root:          opts.Directory,
				ProjectDef:    projDef,
				ComponentDefs: componentDefs,
				Executor:      executor,
			})
			if err != nil {
				fatal(err)
			}
			c := proj.Components().WithName(componentName).First()
			if c == nil {
				fatal(fmt.Errorf("Unknown component: %s", componentName))
			}
			r, found := c.Rule(ruleName)
			if !found {
				fatal(fmt.Errorf("Unknown rule: %s.%s", componentName, ruleName))
			}

			ctx := context.Background()
			zimCache := cache.New(nil)
			key, err := zimCache.Key(ctx, r)
			if err != nil {
				fatal(err)
			}

			if viper.GetBool("detail") {
				js, err := json.Marshal(key)
				if err != nil {
					fatal(err)
				}
				fmt.Println(string(js))
			} else {
				fmt.Println(key.String())
			}
		},
	}

	cmd.Flags().Bool("detail", false, "Show key details")
	viper.BindPFlag("detail", cmd.Flags().Lookup("detail"))

	return cmd
}

func init() {
	rootCmd.AddCommand(NewShowKeyCommand())
}

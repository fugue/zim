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
	"fmt"

	"github.com/fatih/color"
	"github.com/fugue/zim/format"
	"github.com/spf13/cobra"
)

type listOutputsViewItem struct {
	Component string
	Rule      string
	Output    string
	Exists    bool
}

// NewListArtifactsCommand returns a command that lists all rules in the project
func NewListArtifactsCommand() *cobra.Command {

	defaultCols := []string{
		"Component",
		"Rule",
		"Output",
		"Exists",
	}

	cmd := &cobra.Command{
		Use:     "outputs",
		Short:   "List rule outputs",
		Aliases: []string{"out", "outs", "outputs"},
		Run: func(cmd *cobra.Command, args []string) {

			opts := getZimOptions(cmd, args)
			proj, err := getProject(opts.Directory)
			if err != nil {
				fatal(err)
			}
			comps, err := proj.Select(opts.Components, opts.Kinds)
			if err != nil {
				fatal(err)
			}
			projDir := proj.RootAbsPath()

			var rows []interface{}
			var rowColors []*color.Color
			for _, c := range comps {
				for _, ruleName := range c.RuleNames() {
					r, err := c.Rule(ruleName, nil)
					if err != nil {
						fatal(err)
					}
					missingOutputs, err := r.MissingOutputs().RelativePaths(projDir)
					if err != nil {
						fatal(err)
					}
					missing := map[string]bool{}
					for _, artifact := range missingOutputs {
						missing[artifact] = true
					}
					outputs, err := r.Outputs().RelativePaths(projDir)
					if err != nil {
						fatal(err)
					}
					for _, output := range outputs {
						outputMissing := missing[output]
						if outputMissing {
							rowColors = append(rowColors, color.New(color.FgRed))
						} else {
							rowColors = append(rowColors, color.New(color.FgWhite))
						}
						rows = append(rows, listOutputsViewItem{
							Component: c.Name(),
							Rule:      r.Name(),
							Output:    output,
							Exists:    !outputMissing,
						})
					}
				}
			}
			table, err := format.Table(format.TableOpts{
				Rows:       rows,
				Columns:    defaultCols,
				ShowHeader: true,
				Colors:     rowColors,
			})
			if err != nil {
				fatal(err)
			}
			for _, tableRow := range table {
				fmt.Println(tableRow)
			}
		},
	}
	return cmd
}

func init() {
	listCmd.AddCommand(NewListArtifactsCommand())
}

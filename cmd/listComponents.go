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

	"github.com/fugue/zim/format"
	"github.com/spf13/cobra"
)

type listComponentsViewItem struct {
	Name      string
	Kind      string
	App       string
	Directory string
}

// NewListComponentsCommand returns a command that lists all components in the project
func NewListComponentsCommand() *cobra.Command {

	defaultCols := []string{
		"Name",
		"Kind",
	}

	cmd := &cobra.Command{
		Use:     "components",
		Short:   "List components in the project",
		Aliases: []string{"c"},
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

			var rows []interface{}
			for _, c := range comps {
				rows = append(rows, listComponentsViewItem{
					Name:      c.Name(),
					Kind:      c.Kind(),
					App:       c.App(),
					Directory: c.Directory(),
				})
			}
			table, err := format.Table(format.TableOpts{
				Rows:       rows,
				Columns:    defaultCols,
				ShowHeader: true,
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
	listCmd.AddCommand(NewListComponentsCommand())
}

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

	"github.com/fatih/structs"
	"github.com/fugue/zim/format"
	"github.com/spf13/cobra"
)

type listEnvViewItem struct {
	Key   string
	Value interface{}
}

// listEnvCmd represents the listEnv command
var listEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "List zim environment and configuration",
	Run: func(cmd *cobra.Command, args []string) {

		defaultCols := []string{
			"Key",
			"Value",
		}

		fields := structs.Map(getZimOptions(cmd, args))

		var rows []interface{}
		for _, k := range []string{"URL", "Region", "Debug", "Jobs", "UseDocker"} {
			rows = append(rows, listEnvViewItem{Key: k, Value: fields[k]})
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

func init() {
	listCmd.AddCommand(listEnvCmd)
}

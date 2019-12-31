package cmd

import (
	"fmt"
	"sort"

	"github.com/LuminalHQ/zim/format"
	"github.com/fatih/structs"
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

		fields := structs.Map(getZimOptions())
		var names []string
		for k := range fields {
			names = append(names, k)
		}
		sort.Strings(names)

		var rows []interface{}
		for _, k := range names {
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

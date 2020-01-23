package cmd

import (
	"fmt"

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

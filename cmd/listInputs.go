package cmd

import (
	"fmt"

	"github.com/LuminalHQ/zim/format"
	"github.com/spf13/cobra"
)

type listInputsViewItem struct {
	Component string
	Rule      string
	Input     string
}

// NewListInputsCommand returns a command that lists all rules in the project
func NewListInputsCommand() *cobra.Command {

	defaultCols := []string{
		"Component",
		"Rule",
		"Input",
	}

	cmd := &cobra.Command{
		Use:     "inputs",
		Short:   "List rule inputs",
		Aliases: []string{"in", "ins", "inputs"},
		Run: func(cmd *cobra.Command, args []string) {

			opts := getZimOptions()
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
				for _, r := range c.Rules() {
					inputs, err := r.Inputs()
					if err != nil {
						fatal(err)
					}
					relInputs, err := inputs.RelativePaths(proj.RootAbsPath())
					if err != nil {
						fatal(err)
					}
					for _, input := range relInputs {
						rows = append(rows, listInputsViewItem{
							Component: c.Name(),
							Rule:      r.Name(),
							Input:     input,
						})
					}
				}
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
	listCmd.AddCommand(NewListInputsCommand())
}

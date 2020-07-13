package cmd

import (
	"fmt"

	"github.com/fugue/zim/format"
	"github.com/spf13/cobra"
)

type listRulesViewItem struct {
	Component string
	Rule      string
}

// NewListRulesCommand returns a command that lists all rules in the project
func NewListRulesCommand() *cobra.Command {

	defaultCols := []string{
		"Component",
		"Rule",
	}

	cmd := &cobra.Command{
		Use:     "rules",
		Short:   "List rules in the project",
		Aliases: []string{"r", "rule", "rules"},
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
				for _, r := range c.Rules() {
					rows = append(rows, listRulesViewItem{
						Component: c.Name(),
						Rule:      r.Name(),
					})
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
	listCmd.AddCommand(NewListRulesCommand())
}

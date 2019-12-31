package cmd

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List items within a project",
}

func init() {
	rootCmd.AddCommand(listCmd)
}

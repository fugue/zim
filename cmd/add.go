package cmd

import (
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add subcommands",
}

func init() {
	rootCmd.AddCommand(addCmd)
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "zim",
	Short:   "The caching build tool",
	Version: fmt.Sprintf("%s, build %s", Version, GitCommit),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags available to all subcommands
	rootCmd.PersistentFlags().StringP("url", "u", "", "Zim API URL")
	rootCmd.PersistentFlags().StringP("dir", "d", ".", "Working directory")
	rootCmd.PersistentFlags().String("region", "us-east-2", "AWS region")
	rootCmd.PersistentFlags().Bool("docker", true, "Use Docker when running rules")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringSliceP("kinds", "k", nil, "Select kinds of components to operate on")
	rootCmd.PersistentFlags().StringSliceP("components", "c", nil, "Select components to operate on by name")
	rootCmd.PersistentFlags().StringSliceP("rules", "r", nil, "Rules to run against components")
	rootCmd.PersistentFlags().String("cache-mode", "", "Cache mode (WRITE_ONLY)")

	// Bind flags to environment variables if they are present
	viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("docker", rootCmd.PersistentFlags().Lookup("docker"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("kinds", rootCmd.PersistentFlags().Lookup("kinds"))
	viper.BindPFlag("components", rootCmd.PersistentFlags().Lookup("components"))
	viper.BindPFlag("rules", rootCmd.PersistentFlags().Lookup("rules"))
	viper.BindPFlag("cache-mode", rootCmd.PersistentFlags().Lookup("cache-mode"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	// Environment variables will be prefixed with "ZIM_"
	viper.SetEnvPrefix("zim")

	home, err := os.UserHomeDir()
	if err != nil {
		fatal(err)
	}
	// Search config in home directory with name ".zim" (without extension)
	viper.AddConfigPath(home)
	viper.SetConfigName(".zim")

	viper.AutomaticEnv()
	viper.ReadInConfig()
}

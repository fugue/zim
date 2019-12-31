package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zim",
	Short: "The caching build tool",
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
	rootCmd.PersistentFlags().StringP("dir", "d", ".", "Working directory")
	rootCmd.PersistentFlags().StringP("bucket", "b", "", "S3 bucket used for storage")
	rootCmd.PersistentFlags().String("region", "us-east-2", "AWS region")
	rootCmd.PersistentFlags().String("cache", "", "Local cache directory")
	rootCmd.PersistentFlags().Bool("docker", false, "Use Docker when running rules")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringSliceP("kinds", "k", nil, "Select kinds of components to operate on")
	rootCmd.PersistentFlags().StringSliceP("components", "c", nil, "Select components to operate on by name")
	rootCmd.PersistentFlags().StringSliceP("rules", "r", nil, "Rules to run against components")

	// Bind flags to environment variables if they are present
	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("bucket", rootCmd.PersistentFlags().Lookup("bucket"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("cache", rootCmd.PersistentFlags().Lookup("cache"))
	viper.BindPFlag("docker", rootCmd.PersistentFlags().Lookup("docker"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("kinds", rootCmd.PersistentFlags().Lookup("kinds"))
	viper.BindPFlag("components", rootCmd.PersistentFlags().Lookup("components"))
	viper.BindPFlag("rules", rootCmd.PersistentFlags().Lookup("rules"))
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

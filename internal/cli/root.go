// Package cli provides command-line interface functionality for the soba tool.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	Version string
	Commit  string
	Date    string
)

var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "soba",
		Short: "GitHub to Claude Code workflow automation",
		Long: `Soba is an autonomous CLI tool that fully automates GitHub Issue-driven
development workflows through seamless integration with Claude Code AI.`,
	}

	// Add subcommands
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newInitCmd())

	return cmd
}

func Execute(version, commit, date string) error {
	Version = version
	Commit = commit
	Date = date
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".soba")
		viper.SetConfigName("config")
		viper.SetConfigType("yml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

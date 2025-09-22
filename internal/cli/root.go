// Package cli provides command-line interface functionality for the soba tool.
package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/douhashi/soba/pkg/logging"
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
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newOpenCmd())
	cmd.AddCommand(newLogCmd())

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

	// Initialize logging factory
	logConfig := logging.Config{
		Level:  "info",
		Format: "text",
	}
	if verbose {
		logConfig.Level = "debug"
	}

	factory, err := logging.NewFactory(logConfig)
	if err != nil {
		// Fallback to mock logger if initialization fails
		factory = &logging.Factory{}
	}

	log := factory.CreateComponentLogger("cli")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".soba")
		viper.SetConfigName("config")
		viper.SetConfigType("yml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Debug(context.Background(), "Using config file", logging.Field{Key: "path", Value: viper.ConfigFileUsed()})
	} else if verbose {
		log.Debug(context.Background(), "No config file found", logging.Field{Key: "error", Value: err.Error()})
	}
}

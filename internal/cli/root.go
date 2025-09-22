// Package cli provides command-line interface functionality for the soba tool.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/douhashi/soba/pkg/logging"
)

var (
	cfgFile    string
	verbose    bool
	logLevel   string
	Version    string
	Commit     string
	Date       string
	logFactory *logging.Factory
)

var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "soba",
		Short: "GitHub to Claude Code workflow automation",
		Long: `Soba is an autonomous CLI tool that fully automates GitHub Issue-driven
development workflows through seamless integration with Claude Code AI.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLogLevel()
		},
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
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "set log level (debug, info, warn, error)")
}

func initConfig() {
	// Initialize logging factory
	logConfig := logging.Config{
		Level:  getEffectiveLogLevel(),
		Format: "text",
	}

	var err error
	logFactory, err = logging.NewFactory(logConfig)
	if err != nil {
		// Fallback to mock logger if initialization fails
		logFactory = &logging.Factory{}
	}

	log := logFactory.CreateComponentLogger("cli")

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

// validateLogLevel validates the log level flag
func validateLogLevel() error {
	if logLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		isValid := false
		for _, level := range validLevels {
			if logLevel == level {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid log level: %s. Valid levels are: debug, info, warn, error", logLevel)
		}
	}
	return nil
}

// getEffectiveLogLevel returns the effective log level based on flags priority
func getEffectiveLogLevel() string {
	// Priority: --log-level > --verbose > default
	if logLevel != "" {
		return logLevel
	}
	if verbose {
		return "debug"
	}
	return "warn" // Default level
}

// GetLogFactory returns the global log factory instance
func GetLogFactory() *logging.Factory {
	if logFactory == nil {
		// Create a default factory if not initialized
		logFactory, _ = logging.NewFactory(logging.Config{
			Level:  getEffectiveLogLevel(),
			Format: "text",
		})
		if logFactory == nil {
			logFactory = &logging.Factory{}
		}
	}
	return logFactory
}

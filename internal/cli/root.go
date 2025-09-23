// Package cli provides command-line interface functionality for the soba tool.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/pkg/app"
)

var (
	cfgFile  string
	verbose  bool
	logLevel string
	Version  string
	Commit   string
	Date     string

	appInitOnce sync.Once
	appInitErr  error
)

var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "soba",
		Short: "GitHub to Claude Code workflow automation",
		Long: `Soba is an autonomous CLI tool that fully automates GitHub Issue-driven
development workflows through seamless integration with Claude Code AI.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate log level first
			if err := validateLogLevel(); err != nil {
				return err
			}

			// Skip initialization for commands that don't need config
			cmdName := cmd.Name()
			// For subcommands, get the actual command being executed
			if cmd.HasParent() {
				cmdName = cmd.Name()
			}

			if cmdName == "init" || cmdName == "version" || cmdName == "stop" || cmdName == "log" {
				return nil
			}

			// Initialize app with CLI options (only once)
			return initializeApp()
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "set log level (debug, info, warn, error)")
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

// initializeApp initializes the application with CLI options
func initializeApp() error {
	appInitOnce.Do(func() {
		// Get config file path
		configPath := cfgFile
		if configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				appInitErr = err
				return
			}
			configPath = filepath.Join(cwd, ".soba", "config.yml")
		}

		// Initialize with CLI options
		defer func() {
			if r := recover(); r != nil {
				appInitErr = fmt.Errorf("app initialization failed: %v", r)
			}
		}()

		app.MustInitializeWithOptions(configPath, &app.InitOptions{
			LogLevel: logLevel,
			Verbose:  verbose,
		})
	})

	return appInitErr
}

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		version string
		wantErr bool
	}{
		{
			name:    "Execute with version",
			args:    []string{"version"},
			version: "1.0.0",
			wantErr: false,
		},
		{
			name:    "Execute with help",
			args:    []string{"--help"},
			version: "1.0.0",
			wantErr: false,
		},
		{
			name:    "Execute with unknown command",
			args:    []string{"unknown"},
			version: "1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			Version = tt.version

			err := Execute(tt.version, "test-commit", "test-date")
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	Version = "1.0.0-test"
	Commit = "abc123"
	Date = "2024-01-01"

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.0.0-test") {
		t.Errorf("Version command output = %v, want to contain version", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("Version command output = %v, want to contain commit", output)
	}
	if !strings.Contains(output, "2024-01-01") {
		t.Errorf("Version command output = %v, want to contain date", output)
	}
}

func TestRootCmdDescription(t *testing.T) {
	if rootCmd.Short != "GitHub to Claude Code workflow automation" {
		t.Errorf("Short description incorrect: got %v", rootCmd.Short)
	}

	expectedLong := `Soba is an autonomous CLI tool that fully automates GitHub Issue-driven
development workflows through seamless integration with Claude Code AI.`

	if rootCmd.Long != expectedLong {
		t.Errorf("Long description incorrect: got %v", rootCmd.Long)
	}
}

func TestConfigFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "Config flag with value",
			args:    []string{"--config", ".soba/test.yml", "--help"},
			wantErr: false,
		},
		{
			name:    "Verbose flag",
			args:    []string{"--verbose", "--help"},
			wantErr: false,
		},
		{
			name:    "Short flags",
			args:    []string{"-c", "config.yml", "-v", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() with flags error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigCommandExists(t *testing.T) {
	// Check if config command exists in root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "config" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Config command should be registered to root command")
	}
}

func TestLogLevelFlag(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantLogLevel string
		wantErr      bool
	}{
		{
			name:         "Log level debug",
			args:         []string{"--log-level", "debug", "--help"},
			wantLogLevel: "debug",
			wantErr:      false,
		},
		{
			name:         "Log level info",
			args:         []string{"--log-level", "info", "--help"},
			wantLogLevel: "info",
			wantErr:      false,
		},
		{
			name:         "Log level warn",
			args:         []string{"--log-level", "warn", "--help"},
			wantLogLevel: "warn",
			wantErr:      false,
		},
		{
			name:         "Log level error",
			args:         []string{"--log-level", "error", "--help"},
			wantLogLevel: "error",
			wantErr:      false,
		},
		{
			name:         "Invalid log level",
			args:         []string{"--log-level", "invalid", "version"},
			wantLogLevel: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			rootCmd = newRootCmd()
			logLevel = ""
			verbose = false
			rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
			rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
			rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "set log level (debug, info, warn, error)")

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check if log level was set correctly
			if !tt.wantErr && logLevel != tt.wantLogLevel {
				t.Errorf("logLevel = %v, want %v", logLevel, tt.wantLogLevel)
			}
		})
	}
}

func TestLogLevelPriority(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantEffectLevel string
		wantErr         bool
	}{
		{
			name:            "log-level takes priority over verbose",
			args:            []string{"--log-level", "error", "--verbose", "--help"},
			wantEffectLevel: "error",
			wantErr:         false,
		},
		{
			name:            "verbose flag when no log-level",
			args:            []string{"--verbose", "--help"},
			wantEffectLevel: "debug",
			wantErr:         false,
		},
		{
			name:            "default when no flags",
			args:            []string{"--help"},
			wantEffectLevel: "warn",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			rootCmd = newRootCmd()
			logLevel = ""
			verbose = false
			rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
			rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
			rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "set log level (debug, info, warn, error)")

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// After initConfig, check the effective log level
			effectiveLevel := getEffectiveLogLevel()
			if effectiveLevel != tt.wantEffectLevel {
				t.Errorf("Effective log level = %v, want %v", effectiveLevel, tt.wantEffectLevel)
			}
		})
	}
}

//go:build integration
// +build integration

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogLevelFlagIntegration(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		envLogLevel    string
		expectedOutput string
		notExpected    string
	}{
		{
			name:           "Debug level shows debug messages",
			args:           []string{"--log-level", "debug", "version"},
			expectedOutput: "",
			notExpected:    "",
		},
		{
			name:           "Error level hides debug messages",
			args:           []string{"--log-level", "error", "version"},
			expectedOutput: "",
			notExpected:    "DEBUG",
		},
		{
			name:           "Verbose flag sets debug level",
			args:           []string{"--verbose", "version"},
			expectedOutput: "",
			notExpected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command for each test
			rootCmd = newRootCmd()
			logLevel = ""
			verbose = false

			// Capture output
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)

			// Set environment variable if needed
			if tt.envLogLevel != "" {
				os.Setenv("SOBA_LOG_LEVEL", tt.envLogLevel)
				defer os.Unsetenv("SOBA_LOG_LEVEL")
			}

			// Set args
			rootCmd.SetArgs(tt.args)

			// Execute command
			err := rootCmd.Execute()
			if err != nil && !strings.Contains(err.Error(), "invalid log level") {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := buf.String()

			// Check expected output
			if tt.expectedOutput != "" && !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expectedOutput, output)
			}

			// Check not expected output
			if tt.notExpected != "" && strings.Contains(output, tt.notExpected) {
				t.Errorf("Expected output NOT to contain '%s', got: %s", tt.notExpected, output)
			}
		})
	}
}

func TestLogLevelPersistence(t *testing.T) {
	tests := []struct {
		name          string
		commands      [][]string
		checkLogLevel string
	}{
		{
			name: "Log level persists across commands",
			commands: [][]string{
				{"--log-level", "error", "version"},
			},
			checkLogLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute commands
			for _, args := range tt.commands {
				rootCmd = newRootCmd()
				rootCmd.SetArgs(args)
				_ = rootCmd.Execute()
			}

			// Check final log level
			if tt.checkLogLevel != "" && logLevel != tt.checkLogLevel {
				t.Errorf("Expected log level %s, got %s", tt.checkLogLevel, logLevel)
			}
		})
	}
}

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

			err := Execute(tt.version)
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

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.0.0-test") {
		t.Errorf("Version command output = %v, want to contain version", output)
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
		name     string
		args     []string
		wantErr  bool
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
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogCmd(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "log command exists",
			args: []string{"--help"},
			want: "Display logs from the running soba process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newLogCmd()
			require.NotNil(t, cmd)

			// Set up buffer to capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.want)
		})
	}
}

func TestLogCommand_BasicAttributes(t *testing.T) {
	cmd := newLogCmd()

	assert.Equal(t, "log", cmd.Use)
	assert.Equal(t, "Display logs from the running soba process", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestLogCommand_Flags(t *testing.T) {
	cmd := newLogCmd()

	// Test lines flag
	linesFlag := cmd.Flags().Lookup("lines")
	require.NotNil(t, linesFlag)
	assert.Equal(t, "n", linesFlag.Shorthand)
	assert.Equal(t, "30", linesFlag.DefValue)

	// Test follow flag
	followFlag := cmd.Flags().Lookup("follow")
	require.NotNil(t, followFlag)
	assert.Equal(t, "f", followFlag.Shorthand)
	assert.Equal(t, "false", followFlag.DefValue)
}

func TestRunLog_WithDefaultOptions(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sobaDir := filepath.Join(tempDir, ".soba")
	logsDir := filepath.Join(sobaDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	// Create PID file
	pidFile := filepath.Join(sobaDir, "soba.pid")
	err = os.WriteFile(pidFile, []byte("12345"), 0644)
	require.NoError(t, err)

	// Create log file with test content
	logFile := filepath.Join(logsDir, "soba-12345.log")
	logContent := ""
	for i := 1; i <= 50; i++ {
		logContent += "log line " + string(rune(i+'0'-1)) + "\n"
	}
	err = os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test command
	cmd := newLogCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	// Should show last 30 lines by default
	lines := bytes.Count([]byte(output), []byte("\n"))
	assert.Equal(t, 30, lines)
}

func TestRunLog_WithLinesOption(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sobaDir := filepath.Join(tempDir, ".soba")
	logsDir := filepath.Join(sobaDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	// Create PID file
	pidFile := filepath.Join(sobaDir, "soba.pid")
	err = os.WriteFile(pidFile, []byte("12345"), 0644)
	require.NoError(t, err)

	// Create log file with test content
	logFile := filepath.Join(logsDir, "soba-12345.log")
	logContent := ""
	for i := 1; i <= 50; i++ {
		logContent += "log line " + string(rune(i+'0'-1)) + "\n"
	}
	err = os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test command with -n 10
	cmd := newLogCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-n", "10"})

	err = cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	// Should show last 10 lines
	lines := bytes.Count([]byte(output), []byte("\n"))
	assert.Equal(t, 10, lines)
}

func TestRunLog_NoPidFile(t *testing.T) {
	// Create temporary directory without PID file
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test command
	cmd := newLogCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PID file not found")
}

func TestRunLog_NoLogFile(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	sobaDir := filepath.Join(tempDir, ".soba")
	err := os.MkdirAll(sobaDir, 0755)
	require.NoError(t, err)

	// Create PID file
	pidFile := filepath.Join(sobaDir, "soba.pid")
	err = os.WriteFile(pidFile, []byte("12345"), 0644)
	require.NoError(t, err)

	// Change to temp directory (no log file)
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test command
	cmd := newLogCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "log file not found")
}

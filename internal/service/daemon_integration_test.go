//go:build integration
// +build integration

package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
)

func TestDaemonService_BackgroundStartIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	logsDir := filepath.Join(sobaDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// Build test binary
	testBinary := filepath.Join(tmpDir, "soba-test")
	cmd := exec.Command("go", "build", "-o", testBinary, "../../cmd/soba")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	// Test configuration
	cfgPath := filepath.Join(tmpDir, ".soba.yaml")
	cfgContent := `
github:
  repository: "douhashi/soba"
  token: "test-token"
workflow:
  interval: 30
log:
  output_path: ".soba/logs/soba-${PID}.log"
  retention_count: 10
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0644))

	// Start daemon in background
	startCmd := exec.Command(testBinary, "start", "-d", "-c", cfgPath)
	startCmd.Dir = tmpDir
	output, err = startCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v\nOutput: %s", err, output)
	}

	// Give daemon time to start
	time.Sleep(2 * time.Second)

	// Check PID file exists
	pidFile := filepath.Join(sobaDir, "soba.pid")
	assert.FileExists(t, pidFile)

	// Read PID and verify process is running
	pidContent, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(string(pidContent))
	require.NoError(t, err)

	// Verify process exists
	process, err := os.FindProcess(pid)
	require.NoError(t, err)
	assert.NotNil(t, process)

	// Check log file was created
	logFiles, err := filepath.Glob(filepath.Join(logsDir, "soba-*.log"))
	require.NoError(t, err)
	assert.NotEmpty(t, logFiles)

	// Stop the daemon
	stopCmd := exec.Command(testBinary, "stop", "-c", cfgPath)
	stopCmd.Dir = tmpDir
	output, err = stopCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop daemon: %v\nOutput: %s", err, output)
	}

	// Verify PID file was removed
	assert.NoFileExists(t, pidFile)
}

func TestDaemonService_LogRotation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, ".soba", "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// Create old log files to test rotation
	for i := 1; i <= 15; i++ {
		logFile := filepath.Join(logsDir, fmt.Sprintf("soba-%d.log", 1000+i))
		require.NoError(t, os.WriteFile(logFile, []byte("test log"), 0644))
		// Set different modification times
		modTime := time.Now().Add(-time.Duration(i) * time.Hour)
		require.NoError(t, os.Chtimes(logFile, modTime, modTime))
	}

	// Create service with retention count
	service := &daemonService{
		workDir: tmpDir,
	}

	cfg := &config.Config{
		Log: config.LogConfig{
			OutputPath:     ".soba/logs/soba-${PID}.log",
			RetentionCount: 10,
		},
	}

	ctx := context.Background()
	// This will trigger log cleanup
	_ = service.StartDaemon(ctx, cfg)

	// Check that only 10 newest files remain
	logFiles, err := filepath.Glob(filepath.Join(logsDir, "soba-*.log"))
	require.NoError(t, err)
	assert.LessOrEqual(t, len(logFiles), 10)
}

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	var lines int
	var follow bool

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Display logs from the running soba process",
		Long: `Display logs from the running soba process by reading from the log file
associated with the currently running daemon process.

The command reads the PID from .soba/soba.pid and displays the corresponding
log file .soba/logs/soba-{pid}.log.

Options:
- Default: Show last 30 lines (equivalent to tail -n 30)
- -n, --lines: Specify number of lines to show
- -f, --follow: Follow log output in real-time (equivalent to tail -f)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd, lines, follow)
		},
	}

	cmd.Flags().IntVarP(&lines, "lines", "n", 30, "number of lines to display")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")

	return cmd
}

func runLog(cmd *cobra.Command, lines int, follow bool) error {
	// Read PID from .soba/soba.pid
	pidPath := filepath.Join(".soba", "soba.pid")
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("PID file not found at %s. Is soba daemon running?", pidPath)
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Parse PID
	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in file %s: %s", pidPath, pidStr)
	}

	// Construct log file path
	logPath := filepath.Join(".soba", "logs", fmt.Sprintf("soba-%d.log", pid))

	// Check if log file exists
	if _, err := os.Stat(logPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log file not found at %s", logPath)
		}
		return fmt.Errorf("failed to access log file: %w", err)
	}

	if follow {
		return followLog(cmd, logPath)
	}

	return tailLog(cmd, logPath, lines)
}

func tailLog(cmd *cobra.Command, logPath string, lines int) error {
	// Use tail command to display log file
	return executeTail(cmd.OutOrStdout(), logPath, lines, false)
}

func followLog(cmd *cobra.Command, logPath string) error {
	// Use tail -f command to follow log file
	return executeTail(cmd.OutOrStdout(), logPath, 30, true)
}

// executeTail executes tail command to display log file contents
func executeTail(w io.Writer, logPath string, lines int, follow bool) error {
	// Check if tail command is available
	if _, err := exec.LookPath("tail"); err != nil {
		return fmt.Errorf("tail command not found in PATH. Please ensure 'tail' is installed and available")
	}

	// Build tail command arguments
	args := []string{"-n", strconv.Itoa(lines)}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, logPath)

	// Create tail command
	cmd := exec.Command("tail", args...)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if follow {
		return executeTailFollow(cmd, logPath)
	}
	return executeTailNormal(cmd, logPath)
}

// executeTailFollow handles tail -f mode with signal interruption
func executeTailFollow(cmd *exec.Cmd, logPath string) error {
	// Setup signal handler for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		cancel()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Run tail command
	if err := cmd.Start(); err != nil {
		return handleTailError(err, logPath, "failed to start tail command")
	}

	// Wait for command completion or interruption
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-done:
		if err != nil {
			return handleTailError(err, logPath, "tail command failed")
		}
		return nil
	}
}

// executeTailNormal handles normal tail mode
func executeTailNormal(cmd *exec.Cmd, logPath string) error {
	if err := cmd.Run(); err != nil {
		return handleTailError(err, logPath, "tail command failed")
	}
	return nil
}

// handleTailError processes tail command errors
func handleTailError(err error, logPath string, defaultMsg string) error {
	if os.IsPermission(err) {
		return fmt.Errorf("permission denied to execute tail command: %w", err)
	}
	// Check for specific exit codes
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			// tail returns 1 for file not found or permission issues
			return fmt.Errorf("cannot read log file %s: %w", logPath, err)
		}
	}
	return fmt.Errorf("%s: %w", defaultMsg, err)
}

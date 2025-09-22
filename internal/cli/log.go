package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read all lines into memory
	allLines := make([]string, 0, 1000) // Pre-allocate with reasonable capacity
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	// Get last N lines
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	// Output the lines
	for i := start; i < len(allLines); i++ {
		fmt.Fprintln(cmd.OutOrStdout(), allLines[i])
	}

	return nil
}

func followLog(cmd *cobra.Command, logPath string) error {
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// First, output existing content
	if err := tailLog(cmd, logPath, 30); err != nil {
		return err
	}

	// Seek to end of file
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// Follow new content
	scanner := bufio.NewScanner(file)
	for {
		if scanner.Scan() {
			fmt.Fprintln(cmd.OutOrStdout(), scanner.Text())
		} else {
			// No new content, wait a bit
			time.Sleep(100 * time.Millisecond)
		}

		// Check for errors
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading log file: %w", err)
		}
	}
}

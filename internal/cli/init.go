package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/douhashi/soba/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize soba configuration",
		Long:  `Initialize soba configuration by creating a .soba/config.yml file in the current directory`,
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Define paths
	sobaDir := filepath.Join(currentDir, ".soba")
	configPath := filepath.Join(sobaDir, "config.yml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	// Create .soba directory if it doesn't exist
	if err := os.MkdirAll(sobaDir, 0755); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot create directory %s", sobaDir)
		}
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate config template
	configContent := config.GenerateTemplate()

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write to %s", configPath)
		}
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cmd.Printf("Successfully created config file at %s\n", configPath)
	return nil
}

package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
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
	log := logger.GetLogger()

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("Failed to get current directory", "error", err)
		return errors.WrapInternal(err, "failed to get current directory")
	}

	// Define paths
	sobaDir := filepath.Join(currentDir, ".soba")
	configPath := filepath.Join(sobaDir, "config.yml")

	log.Debug("Initializing soba configuration", "directory", sobaDir, "config", configPath)

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		log.Warn("Config file already exists", "path", configPath)
		var conflictErr error = errors.NewConflictError("config file already exists")
		conflictErr = errors.WithContext(conflictErr, "path", configPath)
		return conflictErr
	}

	// Create .soba directory if it doesn't exist
	if err := os.MkdirAll(sobaDir, 0755); err != nil {
		if os.IsPermission(err) {
			log.Error("Permission denied", "directory", sobaDir)
			return infra.NewConfigLoadError(sobaDir, "permission denied: cannot create directory")
		}
		log.Error("Failed to create directory", "error", err)
		return errors.WrapInternal(err, "failed to create directory")
	}

	log.Debug("Created directory", "path", sobaDir)

	// Generate config template
	configContent := config.GenerateTemplate()

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		if os.IsPermission(err) {
			log.Error("Permission denied", "file", configPath)
			return infra.NewConfigLoadError(configPath, "permission denied: cannot write file")
		}
		log.Error("Failed to write config file", "error", err)
		return errors.WrapInternal(err, "failed to write config file")
	}

	log.Info("Successfully created config file", "path", configPath)
	cmd.Printf("Successfully created config file at %s\n", configPath)
	return nil
}

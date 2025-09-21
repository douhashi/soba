// Package cli provides command-line interface functionality for the soba tool.
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logger"
)

// newConfigCmd creates a new config command
func newConfigCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display current configuration",
		Long: `Display the current soba configuration from .soba/config.yml file.
Sensitive information like tokens and webhook URLs will be masked.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(cmd, configPath)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path (default: .soba/config.yml)")

	return cmd
}

// runConfig executes the config command
func runConfig(cmd *cobra.Command, configPath string) error {
	log := logger.GetLogger()

	// 設定ファイルパスを決定
	if configPath == "" {
		configPath = filepath.Join(".soba", "config.yml")
	}

	log.Debug("Loading config file", "path", configPath)

	// 設定ファイルを読み込み
	cfg, err := config.Load(configPath)
	if err != nil {
		cmd.PrintErrf("Error: Failed to load config file: %v\n", err)
		return err
	}

	// 設定内容を表示用に整形
	output, err := config.DisplayConfig(cfg)
	if err != nil {
		cmd.PrintErrf("Error: Failed to format config: %v\n", err)
		return err
	}

	// 設定内容を出力
	fmt.Fprint(cmd.OutOrStdout(), output)

	return nil
}

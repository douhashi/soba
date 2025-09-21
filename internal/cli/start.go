package cli

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/service"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
)

func newStartCmd() *cobra.Command {
	var daemon bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Issue monitoring in foreground or daemon mode",
		Long: `Start Issue monitoring process. By default, runs in foreground mode.
Use -d/--daemon flag to run in daemon mode (background).
Use -v/--verbose flag to enable debug logging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, args, daemon, verbose)
		},
	}

	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run in daemon mode (background)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")

	return cmd
}

func runStart(cmd *cobra.Command, args []string, daemon, verbose bool) error {
	daemonService := service.NewDaemonService()
	return runStartWithService(cmd, args, daemon, verbose, daemonService)
}

// DaemonServiceInterface はデーモンサービスのインターフェース（テスト用）
type DaemonServiceInterface interface {
	StartForeground(ctx context.Context, cfg *config.Config) error
	StartDaemon(ctx context.Context, cfg *config.Config) error
}

// runStartWithService allows dependency injection for testing
func runStartWithService(cmd *cobra.Command, _ []string, daemon, verbose bool, daemonService DaemonServiceInterface) error {
	log := logger.NewLogger(logger.GetLogger())

	// 現在のディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("Failed to get current directory", "error", err)
		return errors.WrapInternal(err, "failed to get current directory")
	}

	// 設定ファイルのパスを構築
	configPath := filepath.Join(currentDir, ".soba", "config.yml")

	// 設定ファイルが存在するかチェック
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		log.Error("Config file not found", "path", configPath)
		return errors.NewNotFoundError("config file not found. Please run 'soba init' first")
	}

	// 設定ファイルを読み込み
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Error("Failed to load config", "error", err)
		return errors.WrapInternal(err, "failed to load config")
	}

	// verboseが指定されている場合はログレベルを調整
	if verbose {
		logger.Init(logger.Config{
			Environment: "development",
			Level:       slog.LevelDebug,
		})
		log = logger.NewLogger(logger.GetLogger())
		log.Debug("Debug logging enabled")
	}

	ctx := context.Background()

	if daemon {
		log.Info("Starting Issue monitoring in daemon mode", "repository", cfg.GitHub.Repository)
		err = daemonService.StartDaemon(ctx, cfg)
		if err == nil {
			cmd.Printf("Successfully started daemon mode\n")
		}
	} else {
		log.Info("Starting Issue monitoring in foreground mode", "repository", cfg.GitHub.Repository)
		err = daemonService.StartForeground(ctx, cfg)
		if err == nil {
			cmd.Printf("Issue monitoring stopped\n")
		}
	}

	return err
}

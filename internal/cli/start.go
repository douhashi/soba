package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/service"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logging"
)

func newStartCmd() *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Issue monitoring in foreground or daemon mode",
		Long: `Start Issue monitoring process. By default, runs in foreground mode.
Use -d/--daemon flag to run in daemon mode (background).
Use -v/--verbose flag to enable debug logging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, args, daemon)
		},
	}

	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run in daemon mode (background)")

	return cmd
}

func runStart(cmd *cobra.Command, args []string, daemon bool) error {
	// Get CLI log level and verbose flags from root command
	root := cmd.Root()
	cliLogLevel, _ := root.Flags().GetString("log-level")
	verboseFlag, _ := root.Flags().GetBool("verbose")

	daemonService := service.NewDaemonServiceWithCLIParams(GetLogFactory(), cliLogLevel, verboseFlag)
	return runStartWithService(cmd, args, daemon, daemonService)
}

// DaemonServiceInterface はデーモンサービスのインターフェース（テスト用）
type DaemonServiceInterface interface {
	StartForeground(ctx context.Context, cfg *config.Config) error
	StartDaemon(ctx context.Context, cfg *config.Config) error
}

// runStartWithService allows dependency injection for testing
func runStartWithService(cmd *cobra.Command, _ []string, daemon bool, daemonService DaemonServiceInterface) error {
	var log logging.Logger = logging.NewMockLogger()

	// 現在のディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error(context.Background(), "Failed to get current directory", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to get current directory")
	}

	// 設定ファイルのパスを構築
	configPath := filepath.Join(currentDir, ".soba", "config.yml")

	// 設定ファイルが存在するかチェック
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		log.Error(context.Background(), "Config file not found", logging.Field{Key: "path", Value: configPath})
		return errors.NewNotFoundError("config file not found. Please run 'soba init' first")
	}

	// 設定ファイルを読み込み
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Error(context.Background(), "Failed to load config", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to load config")
	}

	ctx := context.Background()

	if daemon {
		log.Info(ctx, "Starting Issue monitoring in daemon mode", logging.Field{Key: "repository", Value: cfg.GitHub.Repository})
		err = daemonService.StartDaemon(ctx, cfg)
		if err == nil {
			cmd.Printf("Successfully started daemon mode\n")
		}
	} else {
		log.Info(ctx, "Starting Issue monitoring in foreground mode", logging.Field{Key: "repository", Value: cfg.GitHub.Repository})
		err = daemonService.StartForeground(ctx, cfg)
		if err == nil {
			cmd.Printf("Issue monitoring stopped\n")
		}
	}

	return err
}

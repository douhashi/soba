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

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon process",
		Long:  `Stop the running soba daemon process and clean up associated tmux sessions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd, args)
		},
	}

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	daemonService := service.NewDaemonService(GetLogFactory())
	return runStopWithService(cmd, args, daemonService)
}

// StopServiceInterface はストップサービスのインターフェース（テスト用）
type StopServiceInterface interface {
	Stop(ctx context.Context, repository string) error
}

// runStopWithService allows dependency injection for testing
func runStopWithService(cmd *cobra.Command, _ []string, daemonService StopServiceInterface) error {
	var log logging.Logger = logging.NewMockLogger()

	// verboseが指定されている場合はログレベルを調整

	// 現在のディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error(context.Background(), "Failed to get current directory", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to get current directory")
	}

	// 設定ファイルのパスを構築
	configPath := filepath.Join(currentDir, ".soba", "config.yml")

	// 設定ファイルが存在するかチェック（存在しなくても停止処理は続行）
	var repository string
	if _, statErr := os.Stat(configPath); !os.IsNotExist(statErr) {
		// 設定ファイルが存在する場合は読み込む
		cfg, loadErr := config.Load(configPath)
		if loadErr != nil {
			log.Warn(context.Background(), "Failed to load config, using empty repository", logging.Field{Key: "error", Value: loadErr.Error()})
			repository = ""
		} else {
			repository = cfg.GitHub.Repository
		}
	} else {
		log.Debug(context.Background(), "Config file not found, using empty repository", logging.Field{Key: "path", Value: configPath})
		repository = ""
	}

	ctx := context.Background()

	log.Info(context.Background(), "Stopping daemon process", logging.Field{Key: "repository", Value: repository})
	err = daemonService.Stop(ctx, repository)
	if err != nil {
		log.Error(context.Background(), "Failed to stop daemon", logging.Field{Key: "error", Value: err.Error()})
		return err
	}

	cmd.Printf("Daemon stopped successfully\n")
	return nil
}

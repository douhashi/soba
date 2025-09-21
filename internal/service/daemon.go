package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
)

// DaemonService provides daemon functionality for Issue monitoring
type DaemonService interface {
	StartForeground(ctx context.Context, cfg *config.Config) error
	StartDaemon(ctx context.Context, cfg *config.Config) error
	IsRunning() bool
}

// IssueProcessorInterface はIssue処理のインターフェース
type IssueProcessorInterface interface {
	Process(ctx context.Context, cfg *config.Config) error
}

type daemonService struct {
	workDir   string
	processor IssueProcessorInterface
}

// NewDaemonService creates a new daemon service
func NewDaemonService() DaemonService {
	workDir, _ := os.Getwd()
	return &daemonService{
		workDir:   workDir,
		processor: NewIssueProcessor(),
	}
}

// StartForeground starts Issue monitoring in foreground mode
func (d *daemonService) StartForeground(ctx context.Context, cfg *config.Config) error {
	log := logger.NewLogger(logger.GetLogger())
	log.Info("Starting Issue monitoring in foreground mode")

	ticker := time.NewTicker(time.Duration(cfg.Workflow.Interval) * time.Second)
	defer ticker.Stop()

	// 最初に一度実行
	if err := d.processor.Process(ctx, cfg); err != nil {
		log.Error("Failed to process issues", "error", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping Issue monitoring (context canceled)")
			return nil
		case <-ticker.C:
			if err := d.processor.Process(ctx, cfg); err != nil {
				log.Error("Failed to process issues", "error", err)
				// エラーが発生してもループを継続
			}
		}
	}
}

// StartDaemon starts Issue monitoring in daemon mode
func (d *daemonService) StartDaemon(ctx context.Context, cfg *config.Config) error {
	log := logger.NewLogger(logger.GetLogger())

	// 既に実行中かチェック
	if d.IsRunning() {
		return errors.NewConflictError("daemon is already running")
	}

	// .sobaディレクトリとlogsディレクトリを作成
	sobaDir := filepath.Join(d.workDir, ".soba")
	logsDir := filepath.Join(sobaDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Error("Failed to create logs directory", "error", err)
		return errors.WrapInternal(err, "failed to create logs directory")
	}

	// PIDファイルを作成
	if err := d.createPIDFile(); err != nil {
		log.Error("Failed to create PID file", "error", err)
		return err
	}

	log.Info("Daemon started successfully")

	// フォアグラウンドモードと同じ処理を実行
	// 実際のデーモン化はここでは簡略化（本来はfork等が必要）
	return d.StartForeground(ctx, cfg)
}

// IsRunning checks if daemon is currently running
func (d *daemonService) IsRunning() bool {
	pidFile := filepath.Join(d.workDir, ".soba", "soba.pid")

	// PIDファイルが存在しない場合は実行されていない
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return false
	}

	// PIDファイルを読み込み
	content, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(content))
	if err != nil {
		return false
	}

	// プロセスが実際に存在するかチェック
	if err := syscall.Kill(pid, 0); err != nil {
		// プロセスが存在しない場合はPIDファイルを削除
		os.Remove(pidFile)
		return false
	}

	return true
}

// createPIDFile creates PID file
func (d *daemonService) createPIDFile() error {
	pidFile := filepath.Join(d.workDir, ".soba", "soba.pid")
	pid := os.Getpid()

	content := fmt.Sprintf("%d", pid)
	if err := os.WriteFile(pidFile, []byte(content), 0600); err != nil {
		return errors.WrapInternal(err, "failed to write PID file")
	}

	return nil
}

// removePIDFile removes PID file
func (d *daemonService) removePIDFile() error {
	pidFile := filepath.Join(d.workDir, ".soba", "soba.pid")
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return errors.WrapInternal(err, "failed to remove PID file")
	}
	return nil
}

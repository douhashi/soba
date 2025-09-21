package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/internal/service/builder"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
)

// DaemonService provides daemon functionality for Issue monitoring
type DaemonService interface {
	StartForeground(ctx context.Context, cfg *config.Config) error
	StartDaemon(ctx context.Context, cfg *config.Config) error
	IsRunning() bool
	Stop(ctx context.Context, repository string) error
}

// IssueProcessorInterface はIssue処理のインターフェース
type IssueProcessorInterface interface {
	Process(ctx context.Context, cfg *config.Config) error
	ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error
	UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
	Configure(cfg *config.Config) error
}

type daemonService struct {
	workDir                   string
	processor                 IssueProcessorInterface
	watcher                   *IssueWatcher
	prWatcher                 *PRWatcher
	closedIssueCleanupService *ClosedIssueCleanupService
	tmux                      tmux.TmuxClient
}

// init initializes the service factory
func init() {
	builder.SetServiceFactory(&DefaultServiceFactory{})
}

// NewDaemonService creates a new daemon service using ServiceBuilder
func NewDaemonService() DaemonService {
	serviceBuilder := builder.NewServiceBuilder()
	service, err := serviceBuilder.Build()
	if err != nil {
		log := logger.GetLogger()
		log.Error("Failed to create daemon service", "error", err)
		// Return minimal fallback service
		return createFallbackService()
	}
	return service
}

// NewDaemonServiceWithDependencies creates daemon service with injected dependencies
func NewDaemonServiceWithDependencies(
	workDir string,
	processor IssueProcessorInterface,
	watcher *IssueWatcher,
	prWatcher *PRWatcher,
	cleanupService *ClosedIssueCleanupService,
	tmuxClient tmux.TmuxClient,
) DaemonService {
	return &daemonService{
		workDir:                   workDir,
		processor:                 processor,
		watcher:                   watcher,
		prWatcher:                 prWatcher,
		closedIssueCleanupService: cleanupService,
		tmux:                      tmuxClient,
	}
}

// createFallbackService creates minimal working service for emergency fallback
func createFallbackService() DaemonService {
	workDir, _ := os.Getwd()
	return &daemonService{
		workDir: workDir,
		// All services initialized as nil - configureAndStartWatchers handles nil checks
	}
}

// initializeTmuxSession はtmuxセッションを初期化する
func (d *daemonService) initializeTmuxSession(cfg *config.Config) error {
	log := logger.NewLogger(logger.GetLogger())

	// リポジトリ情報からセッション名を生成
	sessionName := d.generateSessionName(cfg.GitHub.Repository)

	// セッションが存在しない場合は作成
	if !d.tmux.SessionExists(sessionName) {
		if err := d.tmux.CreateSession(sessionName); err != nil {
			log.Error("Failed to create tmux session", "error", err, "session", sessionName)
			return fmt.Errorf("failed to create tmux session %s: %w", sessionName, err)
		}
		log.Info("Created tmux session", "session", sessionName)
	} else {
		log.Debug("Tmux session already exists", "session", sessionName)
	}

	return nil
}

const DefaultSessionPrefix = "soba"

// generateSessionName はリポジトリ情報からセッション名を生成する
func (d *daemonService) generateSessionName(repository string) string {
	if repository == "" {
		return DefaultSessionPrefix
	}

	// スラッシュで分割して所有者とリポジトリ名を結合
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		// 不正な形式の場合はデフォルトに戻る
		return DefaultSessionPrefix
	}

	// "soba-{owner}-{repo}"形式で生成
	sessionName := DefaultSessionPrefix + "-" + strings.Join(parts, "-")
	return sessionName
}

// StartForeground starts Issue monitoring in foreground mode
func (d *daemonService) StartForeground(ctx context.Context, cfg *config.Config) error {
	log := logger.NewLogger(logger.GetLogger())
	log.Info("Starting Issue monitoring in foreground mode")

	// tmuxセッションを初期化
	if err := d.initializeTmuxSession(cfg); err != nil {
		return err
	}

	// watchers設定と起動（共通処理を使用）
	return d.configureAndStartWatchers(ctx, cfg, log)
}

// configureAndStartWatchers はwatchersの設定と起動を行う共通処理
func (d *daemonService) configureAndStartWatchers(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	// IssueWatcherに設定を反映
	if d.watcher != nil {
		d.watcher.config = cfg
		d.watcher.interval = time.Duration(cfg.Workflow.Interval) * time.Second
		d.watcher.SetLogger(log)
	}

	// QueueManagerにowner/repoを設定
	if d.watcher != nil && d.watcher.queueManager != nil && cfg.GitHub.Repository != "" {
		parts := strings.Split(cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			d.watcher.queueManager.owner = parts[0]
			d.watcher.queueManager.repo = parts[1]
			d.watcher.queueManager.SetLogger(log)
		}
	}

	// PRWatcherに設定を反映
	if d.prWatcher != nil {
		d.prWatcher.config = cfg
		d.prWatcher.interval = time.Duration(cfg.Workflow.Interval) * time.Second
		d.prWatcher.SetLogger(log)
	}

	// ClosedIssueCleanupServiceを設定
	if cfg.GitHub.Repository != "" {
		parts := strings.Split(cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			sessionName := fmt.Sprintf("soba-%s", parts[1])
			interval := time.Duration(cfg.Workflow.ClosedIssueCleanupInterval) * time.Second
			d.closedIssueCleanupService.Configure(
				parts[0], parts[1], sessionName,
				cfg.Workflow.ClosedIssueCleanupEnabled, interval,
			)
		}
	}

	// IssueWatcher、PRWatcher、ClosedIssueCleanupServiceを並行して起動
	errCh := make(chan error, 3)

	// IssueWatcherを起動
	go func() {
		if d.watcher != nil {
			errCh <- d.watcher.Start(ctx)
		} else {
			errCh <- nil
		}
	}()

	// PRWatcherを起動
	go func() {
		if d.prWatcher != nil {
			errCh <- d.prWatcher.Start(ctx)
		} else {
			errCh <- nil
		}
	}()

	// ClosedIssueCleanupServiceを起動
	go func() {
		errCh <- d.closedIssueCleanupService.Start(ctx)
	}()

	// どれかがエラーで終了したら全体を終了
	for i := 0; i < 3; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}

	return nil
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

	// tmuxセッションを初期化
	if err := d.initializeTmuxSession(cfg); err != nil {
		return err
	}

	// PIDファイルを作成
	if err := d.createPIDFile(); err != nil {
		log.Error("Failed to create PID file", "error", err)
		return err
	}

	log.Info("Daemon started successfully")

	// watchers設定と起動（共通処理を使用）
	return d.configureAndStartWatchers(ctx, cfg, log)
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

// Stop stops the running daemon process
func (d *daemonService) Stop(ctx context.Context, repository string) error {
	log := logger.NewLogger(logger.GetLogger())
	pidFile := filepath.Join(d.workDir, ".soba", "soba.pid")

	// PIDファイルが存在しない場合はデーモンが実行されていない
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		log.Warn("Daemon is not running (PID file not found)")
		return errors.NewNotFoundError("daemon is not running")
	}

	// PIDファイルを読み込み
	content, err := os.ReadFile(pidFile)
	if err != nil {
		log.Error("Failed to read PID file", "error", err)
		return errors.WrapInternal(err, "failed to read PID file")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		log.Error("Invalid PID in file", "content", string(content))
		return errors.NewValidationError("invalid PID in file")
	}

	// プロセスが存在するかチェック
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Error("Process not found", "pid", pid, "error", err)
		return errors.NewNotFoundError("process not found")
	}

	// プロセスが実際に実行中かチェック（Unix系OSの場合）
	if err := process.Signal(syscall.Signal(0)); err != nil {
		log.Warn("Process is not running", "pid", pid)
		// PIDファイルを削除してエラーを返す
		d.removePIDFile()
		return errors.NewNotFoundError("process not found")
	}

	log.Info("Stopping daemon process", "pid", pid)

	// まずSIGTERMを送信してグレースフルシャットダウンを試みる
	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Warn("Failed to send SIGTERM", "pid", pid, "error", err)
	} else {
		// プロセスが終了するまで最大10秒待つ
		for i := 0; i < 100; i++ {
			time.Sleep(100 * time.Millisecond)
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// プロセスが終了した
				log.Info("Daemon process stopped gracefully", "pid", pid)
				break
			}
			if i == 99 {
				// タイムアウト - SIGKILLを送信
				log.Warn("Process did not stop gracefully, sending SIGKILL", "pid", pid)
				if err := process.Signal(syscall.SIGKILL); err != nil {
					log.Error("Failed to kill process", "pid", pid, "error", err)
				}
			}
		}
	}

	// tmuxセッションのクリーンアップ
	sessionName := d.generateSessionName(repository)
	if d.tmux != nil && d.tmux.SessionExists(sessionName) {
		log.Info("Cleaning up tmux session", "session", sessionName)
		if err := d.tmux.KillSession(sessionName); err != nil {
			// tmuxエラーは警告として扱い、停止処理は継続
			log.Warn("Failed to kill tmux session", "session", sessionName, "error", err)
		}
	}

	// PIDファイルを削除
	if err := d.removePIDFile(); err != nil {
		log.Warn("Failed to remove PID file", "error", err)
	}

	log.Info("Daemon stopped successfully")
	return nil
}

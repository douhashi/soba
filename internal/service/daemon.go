package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	"github.com/douhashi/soba/pkg/logging"
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
	logger                    logging.Logger
}

// init initializes the service factory
func init() {
	builder.SetServiceFactory(&DefaultServiceFactory{})
}

// NewDaemonService creates a new daemon service using ServiceBuilder
func NewDaemonService() DaemonService {
	// デフォルトのlogFactoryを作成
	logFactory, err := logging.NewFactory(logging.Config{
		Level:  "DEBUG",
		Format: "json",
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create log factory: %v", err))
	}

	logger := logFactory.CreateComponentLogger("daemon-new")
	ctx := context.Background()
	logger.Info(ctx, "NewDaemonService called")

	serviceBuilder := builder.NewServiceBuilder(logFactory)
	logger.Info(ctx, "ServiceBuilder created")

	service, err := serviceBuilder.Build(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to create daemon service",
			logging.Field{Key: "error", Value: err.Error()},
		)
		panic(fmt.Sprintf("Failed to build daemon service: %v", err))
	}

	logger.Info(ctx, "ServiceBuilder.Build succeeded")
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
	logger logging.Logger,
) DaemonService {
	return &daemonService{
		workDir:                   workDir,
		processor:                 processor,
		watcher:                   watcher,
		prWatcher:                 prWatcher,
		closedIssueCleanupService: cleanupService,
		tmux:                      tmuxClient,
		logger:                    logger,
	}
}

// initializeTmuxSession はtmuxセッションを初期化する
func (d *daemonService) initializeTmuxSession(cfg *config.Config) error {
	ctx := context.Background()

	// リポジトリ情報からセッション名を生成
	sessionName := d.generateSessionName(cfg.GitHub.Repository)

	// セッションが存在しない場合は作成
	if !d.tmux.SessionExists(sessionName) {
		if err := d.tmux.CreateSession(sessionName); err != nil {
			d.logger.Error(ctx, "Failed to create tmux session",
				logging.Field{Key: "error", Value: err},
				logging.Field{Key: "session", Value: sessionName},
			)
			return fmt.Errorf("failed to create tmux session %s: %w", sessionName, err)
		}
		d.logger.Info(ctx, "Created tmux session",
			logging.Field{Key: "session", Value: sessionName},
		)
	} else {
		d.logger.Debug(ctx, "Tmux session already exists",
			logging.Field{Key: "session", Value: sessionName},
		)
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

// initializeLogging はログ出力を初期化する共通メソッド
func (d *daemonService) initializeLogging(cfg *config.Config, alsoToStdout bool) (string, error) {
	ctx := context.Background()

	// 空のパスの場合は何もしない
	if cfg.Log.OutputPath == "" {
		return "", nil
	}

	// .sobaディレクトリとlogsディレクトリを作成
	sobaDir := filepath.Join(d.workDir, ".soba")
	logsDir := filepath.Join(sobaDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		d.logger.Error(ctx, "Failed to create logs directory",
			logging.Field{Key: "error", Value: err},
		)
		return "", errors.WrapInternal(err, "failed to create logs directory")
	}

	// ログファイルパスを決定（環境変数展開）
	logPath := cfg.Log.OutputPath
	if strings.Contains(logPath, "${PID}") {
		logPath = strings.ReplaceAll(logPath, "${PID}", strconv.Itoa(os.Getpid()))
	}
	logPath = os.ExpandEnv(logPath)

	// 相対パスの場合はworkDirからの相対パスとして解釈
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(d.workDir, logPath)
	}

	// ログファイル出力用の新しいFactoryを作成
	// 設定ファイルからのログレベルを優先し、未設定の場合は環境変数を使用
	logLevel := cfg.Log.Level
	if logLevel == "" {
		logLevel = os.Getenv("LOG_LEVEL")
	}
	fileLogFactory, err := logging.NewFactory(logging.Config{
		Level:        logLevel,
		Format:       "json",
		Output:       logPath,
		AlsoToStdout: alsoToStdout, // フォアグラウンドモードではstdoutにも出力
	})
	if err != nil {
		d.logger.Error(ctx, "Failed to initialize log file",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "path", Value: logPath},
		)
		// ログファイル初期化に失敗してもデーモンは継続（stdout出力のみ）
	} else {
		// ファイル出力用のロガーに切り替え
		d.logger = fileLogFactory.CreateComponentLogger("daemon")
	}

	// 古いログファイルのクリーンアップはlumberjackが自動で行うため不要

	return logPath, nil
}

// StartForeground starts Issue monitoring in foreground mode
func (d *daemonService) StartForeground(ctx context.Context, cfg *config.Config) error {
	// ログ出力を初期化（foregroundモードではstdoutとログファイルへ出力）
	logPath, err := d.initializeLogging(cfg, true) // true = also output to stdout
	if err != nil {
		return err
	}

	// ログ初期化後ログ出力を開始

	if logPath != "" {
		d.logger.Info(ctx, "Starting Issue monitoring in foreground mode",
			logging.Field{Key: "logFile", Value: logPath},
		)
	} else {
		d.logger.Info(ctx, "Starting Issue monitoring in foreground mode")
	}

	// tmuxセッションを初期化
	if err := d.initializeTmuxSession(cfg); err != nil {
		return err
	}

	// watchers設定と起動（共通処理を使用）
	return d.configureAndStartWatchers(ctx, cfg)
}

// configureAndStartWatchers はwatchersの設定と起動を行う共通処理
func (d *daemonService) configureAndStartWatchers(ctx context.Context, cfg *config.Config) error {
	// IssueWatcherに設定を反映
	if d.watcher != nil {
		d.watcher.config = cfg
		d.watcher.interval = time.Duration(cfg.Workflow.Interval) * time.Second
		d.watcher.SetLogger(d.logger)
	}

	// QueueManagerを作成または設定
	if d.watcher != nil && cfg.GitHub.Repository != "" {
		parts := strings.Split(cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			// QueueManagerが存在しない場合は作成
			if d.watcher.queueManager == nil {
				d.logger.Info(ctx, "Creating QueueManager",
					logging.Field{Key: "owner", Value: parts[0]},
					logging.Field{Key: "repo", Value: parts[1]},
				)
				queueManager := NewQueueManager(d.watcher.client, parts[0], parts[1])
				queueManager.SetLogger(d.logger)
				d.watcher.SetQueueManager(queueManager)
			} else {
				// 既存のQueueManagerを設定
				d.watcher.queueManager.owner = parts[0]
				d.watcher.queueManager.repo = parts[1]
				d.watcher.queueManager.SetLogger(d.logger)
			}
		}
	}

	// PRWatcherに設定を反映
	if d.prWatcher != nil {
		d.prWatcher.config = cfg
		d.prWatcher.interval = time.Duration(cfg.Workflow.Interval) * time.Second
		d.prWatcher.SetLogger(d.logger)
	}

	// ClosedIssueCleanupServiceを設定
	if d.closedIssueCleanupService != nil && cfg.GitHub.Repository != "" {
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
		if d.closedIssueCleanupService != nil {
			errCh <- d.closedIssueCleanupService.Start(ctx)
		} else {
			errCh <- nil
		}
	}()

	// どれかがエラーで終了したら全体を終了
	for i := 0; i < 3; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}

	return nil
}

const envBackgroundProcess = "SOBA_BACKGROUND_PROCESS"
const envTestMode = "SOBA_TEST_MODE"
const envValueTrue = "true"

// StartDaemon starts Issue monitoring in daemon mode
func (d *daemonService) StartDaemon(ctx context.Context, cfg *config.Config) error {
	// 既に実行中かチェック
	if d.IsRunning() {
		return errors.NewConflictError("daemon is already running")
	}

	// 環境変数でバックグラウンドプロセスか判定
	if os.Getenv(envBackgroundProcess) != envValueTrue {
		// 親プロセス: 子プロセスを起動
		return d.forkAndExit()
	}

	// 子プロセス: デーモン処理を継続
	// ログ出力を初期化（daemonモードではログファイルのみに出力）
	logPath, err := d.initializeLogging(cfg, false) // false = file only
	if err != nil {
		return err
	}

	// ログ初期化後ログ出力を開始

	// tmuxセッションを初期化
	if err := d.initializeTmuxSession(cfg); err != nil {
		return err
	}

	// PIDファイルを作成
	if err := d.createPIDFile(); err != nil {
		d.logger.Error(ctx, "Failed to create PID file",
			logging.Field{Key: "error", Value: err},
		)
		return err
	}

	d.logger.Info(ctx, "Daemon started successfully",
		logging.Field{Key: "logFile", Value: logPath},
	)

	// watchers設定と起動（共通処理を使用）
	return d.configureAndStartWatchers(ctx, cfg)
}

// forkAndExit forks a child process and exits the parent
func (d *daemonService) forkAndExit() error {
	ctx := context.Background()

	// テスト環境ではos.Exitを呼ばない
	if os.Getenv(envTestMode) == envValueTrue {
		d.logger.Debug(ctx, "Test mode: skipping fork")
		return nil
	}

	// 現在の実行ファイルパスを取得
	execPath, err := os.Executable()
	if err != nil {
		d.logger.Error(ctx, "Failed to get executable path",
			logging.Field{Key: "error", Value: err},
		)
		return errors.WrapInternal(err, "failed to get executable path")
	}

	// 子プロセス用の引数を準備
	args := os.Args[1:] // 元の引数を保持

	// 子プロセスを起動
	cmd := exec.Command(execPath, args...)
	cmd.Env = append(os.Environ(), envBackgroundProcess+"="+envValueTrue)

	// プロセス分離の属性を設定
	cmd.SysProcAttr = d.getSysProcAttr()

	// 標準入出力をnullデバイスにリダイレクト
	if devNull, err := os.Open(os.DevNull); err == nil {
		defer devNull.Close()
		cmd.Stdin = devNull
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	// 子プロセスを起動
	if err := cmd.Start(); err != nil {
		d.logger.Error(ctx, "Failed to start background process",
			logging.Field{Key: "error", Value: err},
		)
		return errors.WrapInternal(err, "failed to start background process")
	}

	d.logger.Info(ctx, "Background process started",
		logging.Field{Key: "pid", Value: cmd.Process.Pid},
	)

	// 親プロセスを終了
	os.Exit(0)
	return nil
}

// getSysProcAttr returns system-specific process attributes for daemon
// This method will have OS-specific implementations
func (d *daemonService) getSysProcAttr() *syscall.SysProcAttr {
	return getSysProcAttr()
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
	pidFile := filepath.Join(d.workDir, ".soba", "soba.pid")

	// PIDファイルが存在しない場合はデーモンが実行されていない
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		d.logger.Warn(ctx, "Daemon is not running (PID file not found)")
		return errors.NewNotFoundError("daemon is not running")
	}

	// PIDファイルを読み込み
	content, err := os.ReadFile(pidFile)
	if err != nil {
		d.logger.Error(ctx, "Failed to read PID file",
			logging.Field{Key: "error", Value: err},
		)
		return errors.WrapInternal(err, "failed to read PID file")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		d.logger.Error(ctx, "Invalid PID in file",
			logging.Field{Key: "content", Value: string(content)},
		)
		return errors.NewValidationError("invalid PID in file")
	}

	// プロセスが存在するかチェック
	process, err := os.FindProcess(pid)
	if err != nil {
		d.logger.Error(ctx, "Process not found",
			logging.Field{Key: "pid", Value: pid},
			logging.Field{Key: "error", Value: err},
		)
		return errors.NewNotFoundError("process not found")
	}

	// プロセスが実際に実行中かチェック（Unix系OSの場合）
	if err := process.Signal(syscall.Signal(0)); err != nil {
		d.logger.Warn(ctx, "Process is not running",
			logging.Field{Key: "pid", Value: pid},
		)
		// PIDファイルを削除してエラーを返す
		d.removePIDFile()
		return errors.NewNotFoundError("process not found")
	}

	d.logger.Info(ctx, "Stopping daemon process",
		logging.Field{Key: "pid", Value: pid},
	)

	// まずSIGTERMを送信してグレースフルシャットダウンを試みる
	if err := process.Signal(syscall.SIGTERM); err != nil {
		d.logger.Warn(ctx, "Failed to send SIGTERM",
			logging.Field{Key: "pid", Value: pid},
			logging.Field{Key: "error", Value: err},
		)
	} else {
		// プロセスが終了するまで最大10秒待つ
		for i := 0; i < 100; i++ {
			time.Sleep(100 * time.Millisecond)
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// プロセスが終了した
				d.logger.Info(ctx, "Daemon process stopped gracefully",
					logging.Field{Key: "pid", Value: pid},
				)
				break
			}
			if i == 99 {
				// タイムアウト - SIGKILLを送信
				d.logger.Warn(ctx, "Process did not stop gracefully, sending SIGKILL",
					logging.Field{Key: "pid", Value: pid},
				)
				if err := process.Signal(syscall.SIGKILL); err != nil {
					d.logger.Error(ctx, "Failed to kill process",
						logging.Field{Key: "pid", Value: pid},
						logging.Field{Key: "error", Value: err},
					)
				}
			}
		}
	}

	// tmuxセッションのクリーンアップ
	sessionName := d.generateSessionName(repository)
	if d.tmux != nil && d.tmux.SessionExists(sessionName) {
		d.logger.Info(ctx, "Cleaning up tmux session",
			logging.Field{Key: "session", Value: sessionName},
		)
		if err := d.tmux.KillSession(sessionName); err != nil {
			// tmuxエラーは警告として扱い、停止処理は継続
			d.logger.Warn(ctx, "Failed to kill tmux session",
				logging.Field{Key: "session", Value: sessionName},
				logging.Field{Key: "error", Value: err},
			)
		}
	}

	// PIDファイルを削除
	if err := d.removePIDFile(); err != nil {
		d.logger.Warn(ctx, "Failed to remove PID file",
			logging.Field{Key: "error", Value: err},
		)
	}

	d.logger.Info(ctx, "Daemon stopped successfully")
	return nil
}

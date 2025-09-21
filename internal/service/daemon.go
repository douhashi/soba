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

// NewDaemonService creates a new daemon service
func NewDaemonService() DaemonService {
	workDir, _ := os.Getwd()

	// 依存関係を初期化
	tokenProvider := github.NewDefaultTokenProvider()
	githubClient, _ := github.NewClient(tokenProvider, &github.ClientOptions{})
	tmuxClient := tmux.NewClient()

	// 一時的にコメントアウト - GitHubClientInterface問題のため
	// defaultCfg := &config.Config{
	//     Git: config.GitConfig{
	//         WorktreeBasePath: ".git/soba/worktrees",
	//         BaseBranch:       "main",
	//     },
	// }
	// gitClient, err := git.NewClient(workDir)
	// if err != nil {
	//     log := logger.GetLogger()
	//     log.Error("Failed to initialize git client", "error", err)
	//     return nil
	// }
	// 型キャスト問題を一時的に回避
	// TODO: GitHubClientInterfaceとgithub.Clientを統一する
	var processor IssueProcessorInterface
	var watcher *IssueWatcher
	var prWatcher *PRWatcher

	// 一時的にnilで初期化
	// processor = NewIssueProcessor(githubClient, nil)
	// executor = NewWorkflowExecutor(tmuxClient, workspace, processor)
	// processorWithDeps := NewIssueProcessor(githubClient, executor)
	// queueManager := NewQueueManager(githubClient, "", "")
	// watcher = NewIssueWatcher(githubClient, &config.Config{})
	// prWatcher = NewPRWatcher(githubClient, &config.Config{})

	// ClosedIssueCleanupServiceを初期化（設定は後で更新）
	closedIssueCleanupService := NewClosedIssueCleanupService(
		githubClient,
		tmuxClient,
		"",            // owner は後で設定
		"",            // repo は後で設定
		"",            // sessionName は後で設定
		false,         // 設定は後で更新
		5*time.Minute, // デフォルトインターバル
	)

	return &daemonService{
		workDir:                   workDir,
		processor:                 processor,
		watcher:                   watcher,
		prWatcher:                 prWatcher,
		closedIssueCleanupService: closedIssueCleanupService,
		tmux:                      tmuxClient,
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

// generateSessionName はリポジトリ情報からセッション名を生成する
func (d *daemonService) generateSessionName(repository string) string {
	if repository == "" {
		return "soba"
	}

	// スラッシュで分割して所有者とリポジトリ名を結合
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		// 不正な形式の場合はデフォルトに戻る
		return "soba"
	}

	// "soba-{owner}-{repo}"形式で生成
	sessionName := "soba-" + strings.Join(parts, "-")
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

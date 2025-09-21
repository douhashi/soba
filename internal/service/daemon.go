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
	"github.com/douhashi/soba/internal/infra/git"
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
	workDir   string
	processor IssueProcessorInterface
	watcher   *IssueWatcher
	prWatcher *PRWatcher
}

// NewDaemonService creates a new daemon service
func NewDaemonService() DaemonService {
	workDir, _ := os.Getwd()

	// デフォルト設定を作成（後で実際の設定で上書きされる）
	defaultCfg := &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
			BaseBranch:       "main",
		},
	}

	// 依存関係を初期化
	tokenProvider := github.NewDefaultTokenProvider()
	githubClient, _ := github.NewClient(tokenProvider, &github.ClientOptions{})
	tmuxClient := tmux.NewClient()

	// Git クライアントとワークスペースマネージャーを初期化
	gitClient, err := git.NewClient(workDir)
	if err != nil {
		log := logger.GetLogger()
		log.Error("Failed to initialize git client", "error", err)
		return nil
	}
	workspace := NewGitWorkspaceManager(defaultCfg, gitClient)

	// GitHubクライアント付きでProcessorを初期化
	processor := NewIssueProcessor(githubClient, nil)

	// ProcessorをExecutorに渡す
	executor := NewWorkflowExecutor(tmuxClient, workspace, processor)

	// ProcessorにExecutorを設定（循環依存を解決）
	processorWithDeps := NewIssueProcessor(githubClient, executor)

	// QueueManagerを初期化（owner/repoは後で設定）
	queueManager := NewQueueManager(githubClient, "", "")

	// IssueWatcherを初期化
	// 注: configは後でStartForeground/StartDaemonで設定される
	watcher := NewIssueWatcher(githubClient, &config.Config{})
	watcher.SetProcessor(processorWithDeps)
	watcher.SetQueueManager(queueManager)
	watcher.SetWorkflowExecutor(executor)

	// PRWatcherを初期化
	prWatcher := NewPRWatcher(githubClient, &config.Config{})

	return &daemonService{
		workDir:   workDir,
		processor: processorWithDeps,
		watcher:   watcher,
		prWatcher: prWatcher,
	}
}

// StartForeground starts Issue monitoring in foreground mode
func (d *daemonService) StartForeground(ctx context.Context, cfg *config.Config) error {
	log := logger.NewLogger(logger.GetLogger())
	log.Info("Starting Issue monitoring in foreground mode")

	// IssueWatcherに設定を反映
	d.watcher.config = cfg
	d.watcher.interval = time.Duration(cfg.Workflow.Interval) * time.Second
	d.watcher.SetLogger(log)

	// QueueManagerにowner/repoを設定
	if d.watcher.queueManager != nil && cfg.GitHub.Repository != "" {
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

	// IssueWatcherとPRWatcherを並行して起動
	errCh := make(chan error, 2)

	// IssueWatcherを起動
	go func() {
		errCh <- d.watcher.Start(ctx)
	}()

	// PRWatcherを起動
	go func() {
		errCh <- d.prWatcher.Start(ctx)
	}()

	// どちらかがエラーで終了したら全体を終了
	for i := 0; i < 2; i++ {
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

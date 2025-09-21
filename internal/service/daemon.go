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
	"github.com/douhashi/soba/internal/domain"
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
}

type daemonService struct {
	workDir   string
	processor IssueProcessorInterface
	watcher   *IssueWatcher
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
	gitClient, _ := git.NewClient(workDir)
	workspace := NewGitWorkspaceManager(defaultCfg, gitClient)

	processor := NewIssueProcessor() // 一時的に既存のコンストラクタを使用
	executor := NewWorkflowExecutor(tmuxClient, workspace, processor)
	strategy := domain.NewDefaultPhaseStrategy()

	// ProcessorをExecutorとStrategyと一緒に再初期化
	processorWithDeps := NewIssueProcessorWithDependencies(githubClient, executor, strategy)

	// IssueWatcherを初期化
	// 注: configは後でStartForeground/StartDaemonで設定される
	watcher := NewIssueWatcher(githubClient, &config.Config{})
	watcher.SetPhaseStrategy(strategy)
	watcher.SetProcessor(processorWithDeps)

	return &daemonService{
		workDir:   workDir,
		processor: processorWithDeps,
		watcher:   watcher,
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

	// IssueWatcherを起動
	return d.watcher.Start(ctx)
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

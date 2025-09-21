package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/pkg/logger"
)

const (
	DefaultMaxPanes    = 3
	DefaultSessionName = "soba"
)

// WorkflowExecutor はワークフロー実行のインターフェース
type WorkflowExecutor interface {
	// ExecutePhase は指定されたフェーズを実行する
	ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase) error
}

// workflowExecutor はWorkflowExecutorの実装
type workflowExecutor struct {
	tmux           tmux.TmuxClient
	workspace      GitWorkspaceManager
	issueProcessor IssueProcessorUpdater
	logger         logger.Logger
	maxPanes       int
}

// IssueProcessorUpdater はラベル更新機能を持つインターフェース
type IssueProcessorUpdater interface {
	UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
	Configure(cfg *config.Config) error
}

// NewWorkflowExecutor は新しいWorkflowExecutorを作成する
func NewWorkflowExecutor(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater) WorkflowExecutor {
	return &workflowExecutor{
		tmux:           tmuxClient,
		workspace:      workspace,
		issueProcessor: processor,
		logger:         logger.NewLogger(logger.GetLogger()),
		maxPanes:       DefaultMaxPanes,
	}
}

// NewWorkflowExecutorWithLogger はロガー付きで新しいWorkflowExecutorを作成する
func NewWorkflowExecutorWithLogger(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater, log logger.Logger) WorkflowExecutor {
	return &workflowExecutor{
		tmux:           tmuxClient,
		workspace:      workspace,
		issueProcessor: processor,
		logger:         log,
		maxPanes:       DefaultMaxPanes,
	}
}

// ExecutePhase は指定されたフェーズを実行する
func (e *workflowExecutor) ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase) error {
	e.logger.Info("Executing phase", "issue", issueNumber, "phase", phase)

	// IssueProcessorに設定を適用
	if err := e.issueProcessor.Configure(cfg); err != nil {
		e.logger.Error("Failed to configure issue processor", "error", err)
		return WrapServiceError(err, "failed to configure issue processor")
	}

	// フェーズ定義を取得
	phaseDef := domain.PhaseDefinitions[string(phase)]
	if phaseDef == nil {
		return NewWorkflowExecutionError("soba", string(phase), "phase not defined")
	}

	// 現在実行されているフェーズに対して、トリガーラベルから実行ラベルへ更新
	if err := e.issueProcessor.UpdateLabels(ctx, issueNumber, phaseDef.TriggerLabel, phaseDef.ExecutionLabel); err != nil {
		e.logger.Error("Failed to update labels", "error", err, "issue", issueNumber, "from", phaseDef.TriggerLabel, "to", phaseDef.ExecutionLabel)
		return WrapServiceError(err, "failed to update labels")
	}

	// 実行タイプに応じた処理
	switch phaseDef.ExecutionType {
	case domain.ExecutionTypeLabelOnly:
		// ラベル更新のみの場合は、ここで完了
		e.logger.Debug("Label-only phase completed", "issue", issueNumber, "phase", phase)
	case domain.ExecutionTypeCommand:
		// コマンド実行が必要な場合
		if err := e.executeCommandPhase(cfg, issueNumber, phase, phaseDef); err != nil {
			return err
		}
	default:
		return NewWorkflowExecutionError("soba", string(phase), fmt.Sprintf("unknown execution type: %s", phaseDef.ExecutionType))
	}

	e.logger.Info("Phase execution completed", "issue", issueNumber, "phase", phase)
	return nil
}

// executeCommandPhase executes a command-based phase
func (e *workflowExecutor) executeCommandPhase(cfg *config.Config, issueNumber int, phase domain.Phase, phaseDef *domain.PhaseDefinition) error {
	// Worktreeを準備（必要な場合）
	if err := e.prepareWorkspaceIfNeeded(issueNumber, phaseDef); err != nil {
		return err
	}

	// tmuxセッション管理
	sessionName := DefaultSessionName
	windowName := fmt.Sprintf("issue-%d", issueNumber)

	// tmuxセッションとウィンドウのセットアップ
	windowCreated, err := e.setupTmuxSession(sessionName, windowName)
	if err != nil {
		return err
	}

	// ペイン管理（必要な場合）
	// windowが新規作成された場合は、既に1ペインがあるのでスキップ
	if phaseDef.RequiresPane && !windowCreated {
		if err := e.managePane(sessionName, windowName); err != nil {
			e.logger.Error("Failed to manage pane", "error", err, "session", sessionName, "window", windowName)
			return err
		}
	}

	// コマンド実行
	if err := e.executeCommand(cfg, issueNumber, phase, sessionName, windowName); err != nil {
		return err
	}

	return nil
}

// prepareWorkspaceIfNeeded はworktreeを準備する（必要な場合）
func (e *workflowExecutor) prepareWorkspaceIfNeeded(issueNumber int, phaseDef *domain.PhaseDefinition) error {
	if !phaseDef.RequiresWorktree || e.workspace == nil {
		return nil
	}

	e.logger.Info("Preparing workspace for issue", "issue", issueNumber)
	if err := e.workspace.PrepareWorkspace(issueNumber); err != nil {
		e.logger.Error("Failed to prepare workspace", "error", err, "issue", issueNumber)
		return WrapServiceError(err, "failed to prepare workspace")
	}
	e.logger.Debug("Workspace prepared", "issue", issueNumber)
	return nil
}

// setupTmuxSession はtmuxセッションとウィンドウをセットアップする
// windowが新規作成された場合はtrueを返す
func (e *workflowExecutor) setupTmuxSession(sessionName, windowName string) (bool, error) {
	// セッションが存在しなければ作成
	if !e.tmux.SessionExists(sessionName) {
		if err := e.tmux.CreateSession(sessionName); err != nil {
			e.logger.Error("Failed to create tmux session", "error", err, "session", sessionName)
			return false, NewTmuxManagementError("create session", sessionName, err.Error())
		}
		e.logger.Debug("Created tmux session", "session", sessionName)
	}

	// ウィンドウが存在しなければ作成
	exists, err := e.tmux.WindowExists(sessionName, windowName)
	if err != nil {
		e.logger.Error("Failed to check window existence", "error", err, "session", sessionName, "window", windowName)
		return false, NewTmuxManagementError("check window", windowName, err.Error())
	}

	windowCreated := false
	if !exists {
		if err := e.tmux.CreateWindow(sessionName, windowName); err != nil {
			e.logger.Error("Failed to create tmux window", "error", err, "session", sessionName, "window", windowName)
			return false, NewTmuxManagementError("create window", windowName, err.Error())
		}
		e.logger.Debug("Created tmux window", "session", sessionName, "window", windowName)
		windowCreated = true
	}

	return windowCreated, nil
}

// executeCommand はフェーズコマンドを実行する
func (e *workflowExecutor) executeCommand(cfg *config.Config, issueNumber int, phase domain.Phase, sessionName, windowName string) error {
	phaseCommand := e.getPhaseCommand(cfg, phase)
	command := e.buildCommand(phaseCommand, issueNumber)

	e.logger.Debug("Phase command details", "issue", issueNumber, "phase", phase, "phaseCommand", phaseCommand, "builtCommand", command)

	if command == "" {
		e.logger.Info("No command defined for phase, skipping execution", "issue", issueNumber, "phase", phase)
		return nil
	}

	// 最後のペインインデックスを取得（新しく作成されたペイン）
	paneIndex, err := e.tmux.GetLastPaneIndex(sessionName, windowName)
	if err != nil {
		e.logger.Error("Failed to get last pane index", "error", err, "session", sessionName, "window", windowName)
		return NewTmuxManagementError("get pane index", windowName, err.Error())
	}

	// worktreeを必要とするフェーズかチェック
	if requiresWorktree(phase) {
		worktreeDir := fmt.Sprintf("%s/issue-%d", cfg.Git.WorktreeBasePath, issueNumber)
		cdCommand := fmt.Sprintf("cd %s && %s", worktreeDir, command)

		// tmuxペインの準備完了を待つ（コマンド実行の直前）
		if cfg.Workflow.TmuxCommandDelay > 0 {
			delay := time.Duration(cfg.Workflow.TmuxCommandDelay) * time.Second
			e.logger.Debug("Waiting for tmux pane to be ready before command execution", "delay", delay, "issue", issueNumber)
			time.Sleep(delay)
		}

		if err := e.tmux.SendCommand(sessionName, windowName, paneIndex, cdCommand); err != nil {
			e.logger.Error("Failed to send command", "error", err, "command", cdCommand, "pane", paneIndex)
			return NewCommandExecutionError(cdCommand, string(phase), issueNumber, err.Error())
		}
		e.logger.Info("Command sent with worktree cd", "issue", issueNumber, "phase", phase, "worktree", worktreeDir, "command", command)
	} else {
		// tmuxペインの準備完了を待つ（コマンド実行の直前）
		if cfg.Workflow.TmuxCommandDelay > 0 {
			delay := time.Duration(cfg.Workflow.TmuxCommandDelay) * time.Second
			e.logger.Debug("Waiting for tmux pane to be ready before command execution", "delay", delay, "issue", issueNumber)
			time.Sleep(delay)
		}

		if err := e.tmux.SendCommand(sessionName, windowName, paneIndex, command); err != nil {
			e.logger.Error("Failed to send command", "error", err, "command", command, "pane", paneIndex)
			return NewCommandExecutionError(command, string(phase), issueNumber, err.Error())
		}
		e.logger.Info("Command sent", "issue", issueNumber, "phase", phase, "command", command)
	}

	return nil
}

// requiresWorktree はフェーズがworktreeを必要とするか判定する
func requiresWorktree(phase domain.Phase) bool {
	phaseDef := domain.PhaseDefinitions[string(phase)]
	if phaseDef == nil {
		return false
	}
	return phaseDef.RequiresWorktree
}

// managePane はペインを管理する（制限数チェック、古いペイン削除、新規作成、リサイズ）
func (e *workflowExecutor) managePane(sessionName, windowName string) error {
	// 現在のペイン数を取得
	paneCount, err := e.tmux.GetPaneCount(sessionName, windowName)
	if err != nil {
		return NewTmuxManagementError("get pane count", windowName, err.Error())
	}

	e.logger.Debug("Current pane count", "session", sessionName, "window", windowName, "count", paneCount)

	// ペイン数が制限に達している場合、最も古いペインを削除
	if paneCount >= e.maxPanes {
		firstPaneIndex, err := e.tmux.GetFirstPaneIndex(sessionName, windowName)
		if err != nil {
			return NewTmuxManagementError("get first pane index", windowName, err.Error())
		}

		if err := e.tmux.DeletePane(sessionName, windowName, firstPaneIndex); err != nil {
			return NewTmuxManagementError("delete pane", windowName, err.Error())
		}
		e.logger.Debug("Deleted oldest pane", "session", sessionName, "window", windowName, "index", firstPaneIndex)
	}

	// 新しいペインを作成
	if err := e.tmux.CreatePane(sessionName, windowName); err != nil {
		return NewTmuxManagementError("create pane", windowName, err.Error())
	}
	e.logger.Debug("Created new pane", "session", sessionName, "window", windowName)

	// ペインをリサイズ
	if err := e.tmux.ResizePanes(sessionName, windowName); err != nil {
		return NewTmuxManagementError("resize panes", windowName, err.Error())
	}
	e.logger.Debug("Resized panes", "session", sessionName, "window", windowName)

	return nil
}

// buildCommand はフェーズコマンドからコマンド文字列を構築する
func (e *workflowExecutor) buildCommand(phaseCommand config.PhaseCommand, issueNumber int) string {
	parts := []string{phaseCommand.Command}
	parts = append(parts, phaseCommand.Options...)

	// パラメータがある場合は追加
	if phaseCommand.Parameter != "" {
		param := phaseCommand.Parameter

		// {{issue-number}}プレースホルダーを置換
		param = strings.ReplaceAll(param, "{{issue-number}}", strconv.Itoa(issueNumber))

		// 後方互換性のために{issue_number}も置換
		param = strings.ReplaceAll(param, "{issue_number}", strconv.Itoa(issueNumber))

		// パラメータ全体をダブルクォートで囲む
		param = `"` + param + `"`

		parts = append(parts, param)
	}

	return strings.Join(parts, " ")
}

// getPhaseCommand は設定からフェーズ用のコマンドを取得する
func (e *workflowExecutor) getPhaseCommand(cfg *config.Config, phase domain.Phase) config.PhaseCommand {
	switch phase {
	case domain.PhasePlan:
		return cfg.Phase.Plan
	case domain.PhaseImplement:
		return cfg.Phase.Implement
	case domain.PhaseReview:
		return cfg.Phase.Review
	case domain.PhaseRevise:
		return cfg.Phase.Revise
	default:
		// Queue, Mergeなどのフェーズはコマンドなし
		return config.PhaseCommand{}
	}
}

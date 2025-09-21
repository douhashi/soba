package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase, strategy domain.PhaseStrategy) error
}

// workflowExecutor はWorkflowExecutorの実装
type workflowExecutor struct {
	tmux           tmux.TmuxClient
	workspace      GitWorkspaceManager
	issueProcessor IssueProcessorUpdater
	maxPanes       int
}

// IssueProcessorUpdater はラベル更新機能を持つインターフェース
type IssueProcessorUpdater interface {
	UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
}

// NewWorkflowExecutor は新しいWorkflowExecutorを作成する
func NewWorkflowExecutor(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater) WorkflowExecutor {
	return &workflowExecutor{
		tmux:           tmuxClient,
		workspace:      workspace,
		issueProcessor: processor,
		maxPanes:       DefaultMaxPanes,
	}
}

// ExecutePhase は指定されたフェーズを実行する
func (e *workflowExecutor) ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase, strategy domain.PhaseStrategy) error {
	log := logger.NewNopLogger()
	log.Info("Executing phase", "issue", issueNumber, "phase", phase)

	// フェーズ遷移情報を取得
	transition := domain.GetTransition(phase)
	if transition == nil {
		return NewWorkflowExecutionError("soba", string(phase), "no transition defined")
	}

	// ラベルを更新
	if err := e.issueProcessor.UpdateLabels(ctx, issueNumber, transition.From, transition.To); err != nil {
		log.Error("Failed to update labels", "error", err, "issue", issueNumber, "from", transition.From, "to", transition.To)
		return WrapServiceError(err, "failed to update labels")
	}

	log.Debug("Updated labels", "issue", issueNumber, "from", transition.From, "to", transition.To)

	// tmuxセッション管理
	sessionName := DefaultSessionName
	windowName := fmt.Sprintf("issue-%d", issueNumber)

	// セッションが存在しなければ作成
	if !e.tmux.SessionExists(sessionName) {
		if err := e.tmux.CreateSession(sessionName); err != nil {
			log.Error("Failed to create tmux session", "error", err, "session", sessionName)
			return NewTmuxManagementError("create session", sessionName, err.Error())
		}
		log.Debug("Created tmux session", "session", sessionName)
	}

	// ウィンドウが存在しなければ作成
	exists, err := e.tmux.WindowExists(sessionName, windowName)
	if err != nil {
		log.Error("Failed to check window existence", "error", err, "session", sessionName, "window", windowName)
		return NewTmuxManagementError("check window", windowName, err.Error())
	}

	if !exists {
		if err := e.tmux.CreateWindow(sessionName, windowName); err != nil {
			log.Error("Failed to create tmux window", "error", err, "session", sessionName, "window", windowName)
			return NewTmuxManagementError("create window", windowName, err.Error())
		}
		log.Debug("Created tmux window", "session", sessionName, "window", windowName)
	}

	// ペイン管理
	if err := e.managePane(sessionName, windowName); err != nil {
		log.Error("Failed to manage pane", "error", err, "session", sessionName, "window", windowName)
		return err
	}

	// コマンド構築と実行
	command := e.buildCommand(e.getPhaseCommand(cfg, phase), issueNumber)
	if command != "" {
		// 最初のペインインデックスを取得
		paneIndex, err := e.tmux.GetFirstPaneIndex(sessionName, windowName)
		if err != nil {
			log.Error("Failed to get first pane index", "error", err, "session", sessionName, "window", windowName)
			return NewTmuxManagementError("get pane index", windowName, err.Error())
		}

		// コマンド送信
		if err := e.tmux.SendCommand(sessionName, windowName, paneIndex, command); err != nil {
			log.Error("Failed to send command", "error", err, "command", command, "pane", paneIndex)
			return NewCommandExecutionError(command, string(phase), issueNumber, err.Error())
		}
		log.Info("Command sent", "issue", issueNumber, "phase", phase, "command", command)
	}

	log.Info("Phase execution completed", "issue", issueNumber, "phase", phase)
	return nil
}

// managePane はペインを管理する（制限数チェック、古いペイン削除、新規作成、リサイズ）
func (e *workflowExecutor) managePane(sessionName, windowName string) error {
	log := logger.NewNopLogger()

	// 現在のペイン数を取得
	paneCount, err := e.tmux.GetPaneCount(sessionName, windowName)
	if err != nil {
		return NewTmuxManagementError("get pane count", windowName, err.Error())
	}

	log.Debug("Current pane count", "session", sessionName, "window", windowName, "count", paneCount)

	// ペイン数が制限に達している場合、最も古いペインを削除
	if paneCount >= e.maxPanes {
		firstPaneIndex, err := e.tmux.GetFirstPaneIndex(sessionName, windowName)
		if err != nil {
			return NewTmuxManagementError("get first pane index", windowName, err.Error())
		}

		if err := e.tmux.DeletePane(sessionName, windowName, firstPaneIndex); err != nil {
			return NewTmuxManagementError("delete pane", windowName, err.Error())
		}
		log.Debug("Deleted oldest pane", "session", sessionName, "window", windowName, "index", firstPaneIndex)
	}

	// 新しいペインを作成
	if err := e.tmux.CreatePane(sessionName, windowName); err != nil {
		return NewTmuxManagementError("create pane", windowName, err.Error())
	}
	log.Debug("Created new pane", "session", sessionName, "window", windowName)

	// ペインをリサイズ
	if err := e.tmux.ResizePanes(sessionName, windowName); err != nil {
		return NewTmuxManagementError("resize panes", windowName, err.Error())
	}
	log.Debug("Resized panes", "session", sessionName, "window", windowName)

	return nil
}

// buildCommand はフェーズコマンドからコマンド文字列を構築する
func (e *workflowExecutor) buildCommand(phaseCommand config.PhaseCommand, issueNumber int) string {
	parts := []string{phaseCommand.Command}
	parts = append(parts, phaseCommand.Options...)

	// パラメータがある場合は追加
	if phaseCommand.Parameter != "" {
		param := phaseCommand.Parameter
		// {issue_number}プレースホルダーを置換
		param = strings.ReplaceAll(param, "{issue_number}", strconv.Itoa(issueNumber))
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

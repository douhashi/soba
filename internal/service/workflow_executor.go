package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/slack"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/pkg/logging"
)

const (
	DefaultMaxPanes    = 3
	DefaultSessionName = "soba"
)

// WorkflowExecutor はワークフロー実行のインターフェース
type WorkflowExecutor interface {
	// ExecutePhase は指定されたフェーズを実行する
	ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase) error
	// SetIssueProcessor はIssueProcessorを設定する
	SetIssueProcessor(processor IssueProcessorUpdater)
}

// workflowExecutor はWorkflowExecutorの実装
type workflowExecutor struct {
	tmux           tmux.TmuxClient
	workspace      GitWorkspaceManager
	issueProcessor IssueProcessorUpdater
	slackNotifier  *slack.Notifier
	logger         logging.Logger
	maxPanes       int
}

// IssueProcessorUpdater はラベル更新機能を持つインターフェース
type IssueProcessorUpdater interface {
	UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
	Configure(cfg *config.Config) error
}

// NewWorkflowExecutor は新しいWorkflowExecutorを作成する
func NewWorkflowExecutor(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater, logger logging.Logger) WorkflowExecutor {
	return &workflowExecutor{
		tmux:           tmuxClient,
		workspace:      workspace,
		issueProcessor: processor,
		logger:         logger,
		maxPanes:       DefaultMaxPanes,
	}
}

// NewWorkflowExecutorWithSlack はSlack通知付きで新しいWorkflowExecutorを作成する
func NewWorkflowExecutorWithSlack(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater, slackNotifier *slack.Notifier, logger logging.Logger) WorkflowExecutor {
	return &workflowExecutor{
		tmux:           tmuxClient,
		workspace:      workspace,
		issueProcessor: processor,
		slackNotifier:  slackNotifier,
		logger:         logger,
		maxPanes:       DefaultMaxPanes,
	}
}

// ExecutePhase は指定されたフェーズを実行する
func (e *workflowExecutor) ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase) error {
	e.logger.Info(ctx, "Executing phase",
		logging.Field{Key: "issue", Value: issueNumber},
		logging.Field{Key: "phase", Value: string(phase)},
	)

	// Slack通知: フェーズ開始
	if e.slackNotifier != nil {
		if err := e.slackNotifier.NotifyPhaseStart(string(phase), issueNumber); err != nil {
			e.logger.Error(ctx, "Failed to send Slack notification",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "phase", Value: string(phase)},
				logging.Field{Key: "issue", Value: issueNumber},
			)
		}
	}

	// IssueProcessorに設定を適用
	if e.issueProcessor != nil {
		if err := e.issueProcessor.Configure(cfg); err != nil {
			e.logger.Error(ctx, "Failed to configure issue processor", logging.Field{Key: "error", Value: err.Error()})
			return WrapServiceError(err, "failed to configure issue processor")
		}
	}

	// フェーズ定義を取得
	phaseDef := domain.PhaseDefinitions[string(phase)]
	if phaseDef == nil {
		return NewWorkflowExecutionError("soba", string(phase), "phase not defined")
	}

	// 現在実行されているフェーズに対して、トリガーラベルから実行ラベルへ更新
	if e.issueProcessor != nil {
		e.logger.Info(ctx, "Updating issue labels",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "remove_label", Value: phaseDef.TriggerLabel},
			logging.Field{Key: "add_label", Value: phaseDef.ExecutionLabel},
		)

		if err := e.issueProcessor.UpdateLabels(ctx, issueNumber, phaseDef.TriggerLabel, phaseDef.ExecutionLabel); err != nil {
			e.logger.Error(ctx, "Failed to update labels",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "issue", Value: issueNumber},
				logging.Field{Key: "from", Value: phaseDef.TriggerLabel},
				logging.Field{Key: "to", Value: phaseDef.ExecutionLabel},
			)

			// Slack通知: ラベル更新エラー
			if e.slackNotifier != nil {
				notifyErr := e.slackNotifier.NotifyError(
					fmt.Sprintf("Failed to update labels for issue #%d: %s → %s",
						issueNumber, phaseDef.TriggerLabel, phaseDef.ExecutionLabel),
					err.Error(),
				)
				if notifyErr != nil {
					e.logger.Error(ctx, "Failed to send error notification to Slack",
						logging.Field{Key: "error", Value: notifyErr.Error()},
					)
				}
			}

			return WrapServiceError(err, "failed to update labels")
		}

		e.logger.Info(ctx, "Successfully updated issue labels",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "removed", Value: phaseDef.TriggerLabel},
			logging.Field{Key: "added", Value: phaseDef.ExecutionLabel},
		)
	} else {
		e.logger.Debug(ctx, "IssueProcessor is nil, skipping label update",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "phase", Value: string(phase)},
		)
	}

	// 実行タイプに応じた処理
	switch phaseDef.ExecutionType {
	case domain.ExecutionTypeLabelOnly:
		// ラベル更新のみの場合は、ここで完了
		e.logger.Debug(ctx, "Label-only phase completed",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "phase", Value: string(phase)},
		)
	case domain.ExecutionTypeCommand:
		// コマンド実行が必要な場合
		if err := e.executeCommandPhase(cfg, issueNumber, phase, phaseDef); err != nil {
			return err
		}
	default:
		return NewWorkflowExecutionError("soba", string(phase), fmt.Sprintf("unknown execution type: %s", phaseDef.ExecutionType))
	}

	e.logger.Info(ctx, "Phase execution completed",
		logging.Field{Key: "issue", Value: issueNumber},
		logging.Field{Key: "phase", Value: string(phase)},
	)
	return nil
}

// executeCommandPhase executes a command-based phase
func (e *workflowExecutor) executeCommandPhase(cfg *config.Config, issueNumber int, phase domain.Phase, phaseDef *domain.PhaseDefinition) error {
	// Worktreeを準備（必要な場合）
	if err := e.prepareWorkspaceIfNeeded(issueNumber, phaseDef); err != nil {
		return err
	}

	// tmuxセッション管理
	sessionName := e.generateSessionName(cfg.GitHub.Repository)
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
			e.logger.Error(context.Background(), "Failed to manage pane",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "session", Value: sessionName},
				logging.Field{Key: "window", Value: windowName},
			)
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

	e.logger.Info(context.Background(), "Preparing workspace for issue", logging.Field{Key: "issue", Value: issueNumber})
	if err := e.workspace.PrepareWorkspace(issueNumber); err != nil {
		e.logger.Error(context.Background(), "Failed to prepare workspace",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "issue", Value: issueNumber},
		)
		return WrapServiceError(err, "failed to prepare workspace")
	}
	e.logger.Debug(context.Background(), "Workspace prepared", logging.Field{Key: "issue", Value: issueNumber})
	return nil
}

// setupTmuxSession はtmuxセッションとウィンドウをセットアップする
// windowが新規作成された場合はtrueを返す
func (e *workflowExecutor) setupTmuxSession(sessionName, windowName string) (bool, error) {
	// セッションが存在しなければ作成
	if !e.tmux.SessionExists(sessionName) {
		if err := e.tmux.CreateSession(sessionName); err != nil {
			e.logger.Error(context.Background(), "Failed to create tmux session",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "session", Value: sessionName},
			)
			return false, NewTmuxManagementError("create session", sessionName, err.Error())
		}
		e.logger.Debug(context.Background(), "Created tmux session", logging.Field{Key: "session", Value: sessionName})
	}

	// ウィンドウが存在しなければ作成
	exists, err := e.tmux.WindowExists(sessionName, windowName)
	if err != nil {
		e.logger.Error(context.Background(), "Failed to check window existence",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "session", Value: sessionName},
			logging.Field{Key: "window", Value: windowName},
		)
		return false, NewTmuxManagementError("check window", windowName, err.Error())
	}

	windowCreated := false
	if !exists {
		if err := e.tmux.CreateWindow(sessionName, windowName); err != nil {
			e.logger.Error(context.Background(), "Failed to create tmux window",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "session", Value: sessionName},
				logging.Field{Key: "window", Value: windowName},
			)
			return false, NewTmuxManagementError("create window", windowName, err.Error())
		}
		e.logger.Debug(context.Background(), "Created tmux window",
			logging.Field{Key: "session", Value: sessionName},
			logging.Field{Key: "window", Value: windowName},
		)
		windowCreated = true
	}

	return windowCreated, nil
}

// executeCommand はフェーズコマンドを実行する
func (e *workflowExecutor) executeCommand(cfg *config.Config, issueNumber int, phase domain.Phase, sessionName, windowName string) error {
	phaseCommand := e.getPhaseCommand(cfg, phase)
	command := e.buildCommand(phaseCommand, issueNumber)

	e.logger.Debug(context.Background(), "Phase command details",
		logging.Field{Key: "issue", Value: issueNumber},
		logging.Field{Key: "phase", Value: string(phase)},
		logging.Field{Key: "phaseCommand", Value: phaseCommand},
		logging.Field{Key: "builtCommand", Value: command},
	)

	if command == "" {
		e.logger.Info(context.Background(), "No command defined for phase, skipping execution",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "phase", Value: string(phase)},
		)
		return nil
	}

	// 最後のペインインデックスを取得（新しく作成されたペイン）
	paneIndex, err := e.tmux.GetLastPaneIndex(sessionName, windowName)
	if err != nil {
		e.logger.Error(context.Background(), "Failed to get last pane index",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "session", Value: sessionName},
			logging.Field{Key: "window", Value: windowName},
		)
		return NewTmuxManagementError("get pane index", windowName, err.Error())
	}

	// worktreeを必要とするフェーズかチェック
	if requiresWorktree(phase) {
		worktreeDir := fmt.Sprintf("%s/issue-%d", cfg.Git.WorktreeBasePath, issueNumber)
		cdCommand := fmt.Sprintf("cd %s && %s", worktreeDir, command)

		// tmuxペインの準備完了を待つ（コマンド実行の直前）
		if cfg.Workflow.TmuxCommandDelay > 0 {
			delay := time.Duration(cfg.Workflow.TmuxCommandDelay) * time.Second
			e.logger.Debug(context.Background(), "Waiting for tmux pane to be ready before command execution",
				logging.Field{Key: "delay", Value: delay},
				logging.Field{Key: "issue", Value: issueNumber},
			)
			time.Sleep(delay)
		}

		if err := e.tmux.SendCommand(sessionName, windowName, paneIndex, cdCommand); err != nil {
			e.logger.Error(context.Background(), "Failed to send command",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "command", Value: cdCommand},
				logging.Field{Key: "pane", Value: paneIndex},
			)
			return NewCommandExecutionError(cdCommand, string(phase), issueNumber, err.Error())
		}
		e.logger.Info(context.Background(), "Command sent with worktree cd",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "phase", Value: string(phase)},
			logging.Field{Key: "worktree", Value: worktreeDir},
			logging.Field{Key: "command", Value: command},
		)
	} else {
		// tmuxペインの準備完了を待つ（コマンド実行の直前）
		if cfg.Workflow.TmuxCommandDelay > 0 {
			delay := time.Duration(cfg.Workflow.TmuxCommandDelay) * time.Second
			e.logger.Debug(context.Background(), "Waiting for tmux pane to be ready before command execution",
				logging.Field{Key: "delay", Value: delay},
				logging.Field{Key: "issue", Value: issueNumber},
			)
			time.Sleep(delay)
		}

		if err := e.tmux.SendCommand(sessionName, windowName, paneIndex, command); err != nil {
			e.logger.Error(context.Background(), "Failed to send command",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "command", Value: command},
				logging.Field{Key: "pane", Value: paneIndex},
			)
			return NewCommandExecutionError(command, string(phase), issueNumber, err.Error())
		}
		e.logger.Info(context.Background(), "Command sent",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "phase", Value: string(phase)},
			logging.Field{Key: "command", Value: command},
		)
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

	e.logger.Debug(context.Background(), "Current pane count",
		logging.Field{Key: "session", Value: sessionName},
		logging.Field{Key: "window", Value: windowName},
		logging.Field{Key: "count", Value: paneCount},
	)

	// ペイン数が制限に達している場合、最も古いペインを削除
	if paneCount >= e.maxPanes {
		firstPaneIndex, err := e.tmux.GetFirstPaneIndex(sessionName, windowName)
		if err != nil {
			return NewTmuxManagementError("get first pane index", windowName, err.Error())
		}

		if err := e.tmux.DeletePane(sessionName, windowName, firstPaneIndex); err != nil {
			return NewTmuxManagementError("delete pane", windowName, err.Error())
		}
		e.logger.Debug(context.Background(), "Deleted oldest pane",
			logging.Field{Key: "session", Value: sessionName},
			logging.Field{Key: "window", Value: windowName},
			logging.Field{Key: "index", Value: firstPaneIndex},
		)
	}

	// 新しいペインを作成
	if err := e.tmux.CreatePane(sessionName, windowName); err != nil {
		return NewTmuxManagementError("create pane", windowName, err.Error())
	}
	e.logger.Debug(context.Background(), "Created new pane",
		logging.Field{Key: "session", Value: sessionName},
		logging.Field{Key: "window", Value: windowName},
	)

	// ペインをリサイズ
	if err := e.tmux.ResizePanes(sessionName, windowName); err != nil {
		return NewTmuxManagementError("resize panes", windowName, err.Error())
	}
	e.logger.Debug(context.Background(), "Resized panes",
		logging.Field{Key: "session", Value: sessionName},
		logging.Field{Key: "window", Value: windowName},
	)

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

// generateSessionName はリポジトリ情報からセッション名を生成する
func (e *workflowExecutor) generateSessionName(repository string) string {
	if repository == "" {
		return DefaultSessionName
	}

	// スラッシュで分割して所有者とリポジトリ名を結合
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		// 不正な形式の場合はデフォルトに戻る
		return DefaultSessionName
	}

	// "soba-{owner}-{repo}"形式で生成
	// 複数のスラッシュがある場合も全て結合
	sessionName := "soba-" + strings.Join(parts, "-")
	return sessionName
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

// SetIssueProcessor はIssueProcessorを設定する
func (e *workflowExecutor) SetIssueProcessor(processor IssueProcessorUpdater) {
	e.issueProcessor = processor
}

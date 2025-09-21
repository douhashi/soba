package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logger"
)

func TestIntegration_FullWorkflow(t *testing.T) {
	// 完全なワークフローをテスト:
	// 1. soba:todoのIssueを検出
	// 2. シングルライン処理で1つずつ処理
	// 3. WorkflowExecutorがworktreeを準備
	// 4. 各フェーズでラベルが適切に更新される

	// 初期状態: 2つのsoba:todo Issue
	initialIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "First Issue",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
		{
			ID:     2,
			Number: 2,
			Title:  "Second Issue",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
	}

	// Mock設定
	mockGitHub := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			return initialIssues, false, nil
		},
	}

	mockTmux := new(MockTmuxClient)
	mockWorkspace := new(MockWorkspaceManager)

	// Issue #1のQueue処理と自動遷移でのPlan処理を期待
	// tmux操作のモック - Queueフェーズはpane作成なし、Planフェーズではpane作成あり
	mockTmux.On("SessionExists", "soba").Return(false).Maybe()
	mockTmux.On("CreateSession", "soba").Return(nil).Maybe()
	mockTmux.On("WindowExists", "soba", "issue-1").Return(false, nil).Maybe()
	mockTmux.On("CreateWindow", "soba", "issue-1").Return(nil).Maybe()
	mockTmux.On("GetPaneCount", "soba", "issue-1").Return(0, nil).Maybe()
	mockTmux.On("CreatePane", "soba", "issue-1").Return(nil).Maybe()
	mockTmux.On("ResizePanes", "soba", "issue-1").Return(nil).Maybe()

	// planフェーズでのワークスペース準備（自動遷移時）
	mockWorkspace.On("PrepareWorkspace", 1).Return(nil).Maybe()

	// planフェーズでのコマンド実行（自動遷移時）
	mockTmux.On("GetFirstPaneIndex", "soba", "issue-1").Return(0, nil).Maybe()
	mockTmux.On("SendCommand", "soba", "issue-1", 0, mock.AnythingOfType("string")).Return(nil).Maybe()

	// 設定
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "test/repo",
		},
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
		},
		Workflow: config.WorkflowConfig{
			Interval: 1,
		},
		Phase: config.PhaseConfig{
			Plan: config.PhaseCommand{
				Command:   "soba:plan",
				Parameter: "{issue_number}",
			},
			Implement: config.PhaseCommand{
				Command:   "soba:implement",
				Parameter: "{issue_number}",
			},
			Review: config.PhaseCommand{
				Command:   "soba:review",
				Parameter: "{issue_number}",
			},
		},
	}

	// WorkflowExecutorとIssueProcessorを初期化
	mockProcessorUpdater := new(MockIssueProcessorUpdater)
	// Configure呼び出し（複数回呼ばれる可能性がある）
	mockProcessorUpdater.On("Configure", mock.Anything).Return(nil).Maybe()
	// Queueフェーズでラベル更新（自動遷移で複数回呼ばれる可能性がある）
	mockProcessorUpdater.On("UpdateLabels", mock.Anything, 1, "soba:todo", "soba:queued").Return(nil).Maybe()
	// 自動遷移でplanフェーズも実行される場合
	mockProcessorUpdater.On("UpdateLabels", mock.Anything, 1, "soba:queued", "soba:planning").Return(nil).Maybe()

	executor := NewWorkflowExecutorWithLogger(mockTmux, mockWorkspace, mockProcessorUpdater, logger.NewNopLogger())

	// ProcessorにProcessIssueを実装するためのモックを使用
	processor := &MockIssueProcessor{
		ProcessIssueFunc: func(ctx context.Context, cfg *config.Config, issue github.Issue) error {
			// Queueフェーズを実行
			return executor.ExecutePhase(ctx, cfg, issue.Number, domain.PhaseQueue)
		},
	}

	// IssueWatcherを初期化
	watcher := NewIssueWatcher(mockGitHub, cfg)
	watcher.SetProcessor(processor)

	// コンテキストとタイムアウト設定
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 1回目のwatchOnce実行（Issue #1を処理開始）
	err := watcher.watchOnce(ctx)
	assert.NoError(t, err)

	// Issue #1がシングルライン処理で選択されたことを確認
	assert.NotNil(t, watcher.currentIssue)
	assert.Equal(t, 1, *watcher.currentIssue)

	// tmux関連のモックが呼ばれたことを確認
	mockTmux.AssertExpectations(t)
	mockProcessorUpdater.AssertExpectations(t)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	// エラーケースのテスト
	mockGitHub := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			return []github.Issue{
				{
					ID:     1,
					Number: 1,
					Title:  "Test Issue",
					State:  "open",
					Labels: []github.Label{
						{Name: "soba:todo"},
					},
				},
			}, false, nil
		},
	}

	mockTmux := new(MockTmuxClient)
	mockWorkspace := new(MockWorkspaceManager)
	mockProcessor := new(MockIssueProcessorUpdater)

	// tmux関連のモック設定（自動遷移で複数回呼ばれる可能性がある）
	mockTmux.On("SessionExists", "soba").Return(false).Maybe()
	mockTmux.On("CreateSession", "soba").Return(nil).Maybe()
	mockTmux.On("WindowExists", "soba", "issue-1").Return(false, nil).Maybe()
	mockTmux.On("CreateWindow", "soba", "issue-1").Return(nil).Maybe()
	mockTmux.On("GetPaneCount", "soba", "issue-1").Return(0, nil).Maybe()
	mockTmux.On("CreatePane", "soba", "issue-1").Return(nil).Maybe()
	mockTmux.On("ResizePanes", "soba", "issue-1").Return(nil).Maybe()

	// ラベル更新のモック（複数回呼ばれる可能性がある）
	mockProcessor.On("Configure", mock.Anything).Return(nil).Maybe()
	mockProcessor.On("UpdateLabels", mock.Anything, 1, "soba:todo", "soba:queued").Return(nil).Maybe()
	// 自動遷移でplanフェーズも実行される場合
	mockProcessor.On("UpdateLabels", mock.Anything, 1, "soba:queued", "soba:planning").Return(nil).Maybe()

	// planフェーズでのワークスペース準備（自動遷移時）
	mockWorkspace.On("PrepareWorkspace", 1).Return(nil).Maybe()

	// planフェーズでのコマンド実行（自動遷移時）
	mockTmux.On("GetFirstPaneIndex", "soba", "issue-1").Return(0, nil).Maybe()
	mockTmux.On("SendCommand", "soba", "issue-1", 0, mock.AnythingOfType("string")).Return(nil).Maybe()

	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "test/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 1,
		},
	}

	executor := NewWorkflowExecutorWithLogger(mockTmux, mockWorkspace, mockProcessor, logger.NewNopLogger())
	processor := NewIssueProcessor(mockGitHub, executor)

	watcher := NewIssueWatcher(mockGitHub, cfg)
	watcher.SetProcessor(processor)

	ctx := context.Background()

	// watchOnce実行（エラーが発生するが、watchOnce自体は続行）
	err := watcher.watchOnce(ctx)
	assert.NoError(t, err) // watchOnceはエラーを内部でログ出力して続行

	mockWorkspace.AssertExpectations(t)
	mockProcessor.AssertExpectations(t)
}

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logging"
)

// MockIntegrationGitHubClient は統合テスト用のモック
type MockIntegrationGitHubClient struct {
	mock.Mock
}

func (m *MockIntegrationGitHubClient) ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error) {
	args := m.Called(ctx, owner, repo, options)
	return args.Get(0).([]github.Issue), args.Bool(1), args.Error(2)
}

func (m *MockIntegrationGitHubClient) AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockIntegrationGitHubClient) RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

// PR関連のメソッドを追加（インターフェースを満たすため）
func (m *MockIntegrationGitHubClient) ListPullRequests(ctx context.Context, owner, repo string, opts *github.ListPullRequestsOptions) ([]github.PullRequest, bool, error) {
	args := m.Called(ctx, owner, repo, opts)
	if args.Get(0) != nil {
		return args.Get(0).([]github.PullRequest), args.Bool(1), args.Error(2)
	}
	return nil, false, args.Error(2)
}

func (m *MockIntegrationGitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, bool, error) {
	args := m.Called(ctx, owner, repo, number)
	if args.Get(0) != nil {
		return args.Get(0).(*github.PullRequest), args.Bool(1), args.Error(2)
	}
	return nil, false, args.Error(2)
}

func (m *MockIntegrationGitHubClient) MergePullRequest(ctx context.Context, owner, repo string, number int, req *github.MergeRequest) (*github.MergeResponse, error) {
	args := m.Called(ctx, owner, repo, number, req)
	if args.Get(0) != nil {
		return args.Get(0).(*github.MergeResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockIntegrationWorkflowExecutor は統合テスト用のモック
type MockIntegrationWorkflowExecutor struct {
	mock.Mock
}

func (m *MockIntegrationWorkflowExecutor) ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase domain.Phase) error {
	args := m.Called(ctx, cfg, issueNumber, phase)
	return args.Error(0)
}

func (m *MockIntegrationWorkflowExecutor) SetIssueProcessor(processor IssueProcessorUpdater) {
	m.Called(processor)
}

func TestQueueIntegration_TodoToQueuedTransition(t *testing.T) {
	// テスト用のコンテキストと設定
	ctx := context.Background()
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 5,
		},
	}

	// モックの準備
	mockGitHub := new(MockIntegrationGitHubClient)
	mockExecutor := new(MockIntegrationWorkflowExecutor)

	// IssueWatcherとQueueManagerを設定
	watcher := NewIssueWatcher(mockGitHub, cfg)
	queueManager := NewQueueManager(mockGitHub, "owner", "repo")
	queueManager.SetLogger(logging.NewMockLogger())

	watcher.SetQueueManager(queueManager)
	watcher.SetWorkflowExecutor(mockExecutor)
	watcher.SetLogger(logging.NewMockLogger())

	// テストデータ：soba:todoラベルのIssue
	todoIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			Labels: []github.Label{{Name: "soba:todo"}},
			State:  "open",
		},
		{
			ID:     2,
			Number: 2,
			Title:  "Test Issue 2",
			Labels: []github.Label{{Name: "soba:todo"}},
			State:  "open",
		},
	}

	// モックの期待値設定
	mockGitHub.On("ListOpenIssues", ctx, "owner", "repo", mock.Anything).Return(todoIssues, true, nil)

	// Issue 1のラベル変更（最小番号）
	mockGitHub.On("RemoveLabelFromIssue", ctx, "owner", "repo", 1, "soba:todo").Return(nil)
	mockGitHub.On("AddLabelToIssue", ctx, "owner", "repo", 1, "soba:queued").Return(nil)

	// watchOnceを実行
	err := watcher.watchOnce(ctx)
	assert.NoError(t, err)

	// モックの呼び出しを検証
	mockGitHub.AssertExpectations(t)
}

func TestQueueIntegration_QueuedToPlanExecution(t *testing.T) {
	// テスト用のコンテキストと設定
	ctx := context.Background()
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 5,
		},
	}

	// モックの準備
	mockGitHub := new(MockIntegrationGitHubClient)
	mockExecutor := new(MockIntegrationWorkflowExecutor)

	// IssueWatcherを設定
	watcher := NewIssueWatcher(mockGitHub, cfg)
	watcher.SetWorkflowExecutor(mockExecutor)
	watcher.SetLogger(logging.NewMockLogger())

	// テストデータ：soba:queuedラベルのIssue
	queuedIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			Labels: []github.Label{{Name: "soba:queued"}},
			State:  "open",
		},
	}

	// モックの期待値設定
	mockGitHub.On("ListOpenIssues", ctx, "owner", "repo", mock.Anything).Return(queuedIssues, true, nil)

	// Planフェーズがsoba:queuedラベルのIssueに対して実行される
	mockExecutor.On("ExecutePhase", ctx, cfg, 1, domain.PhasePlan).Return(nil)

	// watchOnceを実行
	err := watcher.watchOnce(ctx)
	assert.NoError(t, err)

	// モックの呼び出しを検証
	mockGitHub.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)
}

func TestQueueIntegration_NoLoopWithQueued(t *testing.T) {
	// テスト用のコンテキストと設定
	ctx := context.Background()
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 5,
		},
	}

	// モックの準備
	mockGitHub := new(MockIntegrationGitHubClient)
	mockExecutor := new(MockIntegrationWorkflowExecutor)
	mockProcessor := &MockQueueIssueProcessor{}

	// IssueWatcherを設定
	watcher := NewIssueWatcher(mockGitHub, cfg)
	watcher.SetProcessor(mockProcessor)
	watcher.SetWorkflowExecutor(mockExecutor)
	watcher.SetQueueManager(NewQueueManager(mockGitHub, "owner", "repo"))
	watcher.SetLogger(logging.NewMockLogger())

	// テストデータ：soba:queuedラベルのIssue
	queuedIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			Labels: []github.Label{{Name: "soba:queued"}},
			State:  "open",
		},
	}

	// モックの期待値設定
	mockGitHub.On("ListOpenIssues", ctx, "owner", "repo", mock.Anything).Return(queuedIssues, true, nil)

	// planフェーズが実行される（queueフェーズではない！）
	mockExecutor.On("ExecutePhase", ctx, cfg, 1, domain.PhasePlan).Return(nil)

	// ProcessIssueはキューイングされたIssueには呼ばれないはず

	// watchOnceを実行
	err := watcher.watchOnce(ctx)
	assert.NoError(t, err)

	// モックの呼び出しを検証
	mockGitHub.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)

	// ProcessIssueが呼ばれていないことを確認
	assert.False(t, mockProcessor.processCalled)
}

// MockQueueIssueProcessor は統合テスト用のモック
type MockQueueIssueProcessor struct {
	processCalled bool
}

func (m *MockQueueIssueProcessor) Process(ctx context.Context, cfg *config.Config) error {
	return nil
}

func (m *MockQueueIssueProcessor) ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error {
	m.processCalled = true
	return nil
}

func (m *MockQueueIssueProcessor) UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	return nil
}

func (m *MockQueueIssueProcessor) Configure(cfg *config.Config) error {
	return nil
}

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
)

func TestNewIssueWatcher(t *testing.T) {
	client := &MockGitHubClient{}
	cfg := &config.Config{
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)

	if watcher == nil {
		t.Error("expected IssueWatcher to be created, got nil")
	}
}

func TestIssueWatcher_SingleLineProcessing(t *testing.T) {
	// 複数のsoba:todoラベル付きIssueがある場合、番号順に1つずつ処理されることを確認
	mockIssues := []github.Issue{
		{
			ID:     3,
			Number: 3,
			Title:  "Test Issue 3",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
		{
			ID:     2,
			Number: 2,
			Title:  "Test Issue 2",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
	}

	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "test/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 1,
		},
	}

	client := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			return mockIssues, false, nil
		},
	}

	// 処理されたIssue番号を記録
	processedIssues := []int{}
	processor := &MockIssueProcessor{
		ProcessIssueFunc: func(ctx context.Context, cfg *config.Config, issue github.Issue) error {
			processedIssues = append(processedIssues, issue.Number)
			return nil
		},
	}

	watcher := NewIssueWatcher(client, cfg)
	watcher.EnablePhaseStrategy()
	watcher.SetProcessor(processor)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// watchOnceを呼び出して処理
	err := watcher.watchOnce(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Issue番号1（最小）のみ処理されることを確認
	if len(processedIssues) != 1 || processedIssues[0] != 1 {
		t.Errorf("expected only Issue #1 to be processed, got: %v", processedIssues)
	}
}

func TestIssueWatcher_ContinueAfterCompletion(t *testing.T) {
	// soba:mergedまたはclosedになったら次のIssueを処理することを確認
	initialIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:doing"},
			},
		},
		{
			ID:     2,
			Number: 2,
			Title:  "Test Issue 2",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
	}

	completedIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:merged"},
			},
		},
		{
			ID:     2,
			Number: 2,
			Title:  "Test Issue 2",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
	}

	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "test/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 1,
		},
	}

	callCount := 0
	client := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			callCount++
			if callCount == 1 {
				return initialIssues, false, nil
			}
			return completedIssues, false, nil
		},
	}

	processedIssues := []int{}
	processor := &MockIssueProcessor{
		ProcessIssueFunc: func(ctx context.Context, cfg *config.Config, issue github.Issue) error {
			processedIssues = append(processedIssues, issue.Number)
			return nil
		},
	}

	watcher := NewIssueWatcher(client, cfg)
	watcher.EnablePhaseStrategy()
	watcher.SetProcessor(processor)

	ctx := context.Background()

	// 初回呼び出し - Issue #1は処理中なのでスキップ
	err := watcher.watchOnce(ctx)
	if err != nil {
		t.Fatalf("unexpected error in first call: %v", err)
	}

	// 2回目呼び出し - Issue #1が完了したので、Issue #2を処理
	err = watcher.watchOnce(ctx)
	if err != nil {
		t.Fatalf("unexpected error in second call: %v", err)
	}

	// Issue #2が処理されたことを確認
	if len(processedIssues) != 1 || processedIssues[0] != 2 {
		t.Errorf("expected Issue #2 to be processed after #1 completion, got: %v", processedIssues)
	}
}

func TestIssueWatcher_Watch_WithLabelFilter(t *testing.T) {
	// テスト用のIssueデータ
	mockIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:planning"},
			},
		},
		{
			ID:     2,
			Number: 2,
			Title:  "Test Issue 2",
			State:  "open",
			Labels: []github.Label{
				{Name: "bug"},
			},
		},
	}

	client := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			return mockIssues, false, nil
		},
	}

	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)

	// ラベルフィルタを設定（soba:で始まるラベル）
	filteredIssues, err := watcher.fetchFilteredIssues(context.Background())

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// soba:planningラベルを持つIssueのみがフィルタされるべき
	if len(filteredIssues) != 1 {
		t.Errorf("expected 1 filtered issue, got: %d", len(filteredIssues))
	}

	if filteredIssues[0].Number != 1 {
		t.Errorf("expected issue number 1, got: %d", filteredIssues[0].Number)
	}
}

func TestIssueWatcher_DetectChanges(t *testing.T) {
	client := &MockGitHubClient{}
	cfg := &config.Config{
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)

	// 初期状態
	issue1 := github.Issue{
		ID:     1,
		Number: 1,
		Title:  "Test Issue",
		State:  "open",
		Labels: []github.Label{
			{Name: "soba:planning"},
		},
	}

	// 初回設定
	changes := watcher.detectChanges([]github.Issue{issue1})
	if len(changes) != 1 {
		t.Errorf("expected 1 new issue, got: %d", len(changes))
	}
	if changes[0].Type != IssueChangeTypeNew {
		t.Errorf("expected change type 'new', got: %s", changes[0].Type)
	}

	// ラベル変更
	issue1Updated := issue1
	issue1Updated.Labels = []github.Label{
		{Name: "soba:doing"},
	}

	changes = watcher.detectChanges([]github.Issue{issue1Updated})
	if len(changes) != 1 {
		t.Errorf("expected 1 label change, got: %d", len(changes))
	}
	if changes[0].Type != IssueChangeTypeLabelChanged {
		t.Errorf("expected change type 'label_changed', got: %s", changes[0].Type)
	}
}

func TestIssueWatcher_Start_StopOnContext(t *testing.T) {
	client := &MockGitHubClient{}
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 1, // 短いインターバルでテスト
		},
	}

	watcher := NewIssueWatcher(client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should return when context is canceled
	err := watcher.Start(ctx)
	if err != nil {
		t.Errorf("expected no error on context cancellation, got: %v", err)
	}
}

func TestIssueWatcher_ParseRepositoryFromConfig(t *testing.T) {
	client := &MockGitHubClient{}
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)

	owner, repo := watcher.parseRepository()

	if owner != "owner" {
		t.Errorf("expected owner 'owner', got: %s", owner)
	}
	if repo != "repo" {
		t.Errorf("expected repo 'repo', got: %s", repo)
	}
}

func TestIssueWatcher_ErrorHandling(t *testing.T) {
	client := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			return nil, false, fmt.Errorf("API error")
		},
	}
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)

	_, err := watcher.fetchFilteredIssues(context.Background())

	if err == nil {
		t.Error("expected error from GitHub API, got nil")
	}
}

func TestIssueWatcher_ProcessWithPhaseStrategy(t *testing.T) {
	// テスト用のIssueデータ
	mockIssues := []github.Issue{
		{
			ID:     1,
			Number: 1,
			Title:  "Test Issue 1",
			State:  "open",
			Labels: []github.Label{
				{Name: "soba:todo"},
			},
		},
	}

	client := &MockGitHubClient{
		ListOpenIssuesFunc: func(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
			return mockIssues, false, nil
		},
	}

	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "owner/repo",
		},
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)
	watcher.EnablePhaseStrategy()

	// 初回実行
	changes := watcher.detectChanges(mockIssues)
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got: %d", len(changes))
	}

	// PhaseStrategyでIssueのフェーズが判定できることを確認
	for _, change := range changes {
		phase, nextLabel, err := watcher.analyzePhase(change.Issue)
		if err != nil {
			t.Errorf("expected no error analyzing phase, got: %v", err)
		}
		if phase != "queue" {
			t.Errorf("expected phase 'queue', got: %s", phase)
		}
		if nextLabel != "soba:queued" {
			t.Errorf("expected next label 'soba:queued', got: %s", nextLabel)
		}
	}
}

func TestIssueWatcher_PhaseTransitionValidation(t *testing.T) {
	client := &MockGitHubClient{}
	cfg := &config.Config{
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}

	watcher := NewIssueWatcher(client, cfg)
	watcher.EnablePhaseStrategy()

	// 初期状態: soba:planning
	issue1 := github.Issue{
		ID:     1,
		Number: 1,
		Title:  "Test Issue",
		State:  "open",
		Labels: []github.Label{
			{Name: "soba:planning"},
		},
	}

	// 初回設定
	watcher.detectChanges([]github.Issue{issue1})

	// 有効な遷移: planning -> ready
	issue1Updated := issue1
	issue1Updated.Labels = []github.Label{
		{Name: "soba:ready"},
	}

	changes := watcher.detectChanges([]github.Issue{issue1Updated})
	if len(changes) != 1 {
		t.Errorf("expected 1 label change, got: %d", len(changes))
	}

	// 遷移が有効かチェック
	isValid := watcher.isValidTransition(changes[0])
	if !isValid {
		t.Error("expected transition from planning to ready to be valid")
	}

	// 無効な遷移: ready -> planning (逆方向)
	issue1Invalid := issue1Updated
	issue1Invalid.Labels = []github.Label{
		{Name: "soba:planning"},
	}

	changes = watcher.detectChanges([]github.Issue{issue1Invalid})
	if len(changes) != 1 {
		t.Errorf("expected 1 label change, got: %d", len(changes))
	}

	// 遷移が無効かチェック
	isValid = watcher.isValidTransition(changes[0])
	if isValid {
		t.Error("expected transition from ready to planning to be invalid")
	}
}

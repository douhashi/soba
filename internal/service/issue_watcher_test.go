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
		listIssuesFunc: func(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error) {
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
		listIssuesFunc: func(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error) {
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

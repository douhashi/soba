package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logger"
)

// IssueChangeType はIssue変更の種類を表す
type IssueChangeType string

const (
	IssueChangeTypeNew          IssueChangeType = "new"
	IssueChangeTypeLabelChanged IssueChangeType = "label_changed"
	IssueChangeTypeStateChanged IssueChangeType = "state_changed"
)

// IssueChange はIssueの変更を表す
type IssueChange struct {
	Type     IssueChangeType
	Issue    github.Issue
	Previous *github.Issue
}

// IssueWatcher はIssue監視機能を提供する
type IssueWatcher struct {
	client         GitHubClientInterface
	config         *config.Config
	interval       time.Duration
	logger         logger.Logger
	previousIssues map[int64]github.Issue // Issue IDをキーとする前回の状態
}

// NewIssueWatcher は新しいIssueWatcherを作成する
func NewIssueWatcher(client GitHubClientInterface, cfg *config.Config) *IssueWatcher {
	// デフォルト値の設定
	if cfg.Workflow.Interval == 0 {
		cfg.Workflow.Interval = 20
	}

	// ロガーの初期化（テスト環境を考慮）
	log := logger.NewNopLogger() // デフォルトでNopLogger使用

	return &IssueWatcher{
		client:         client,
		config:         cfg,
		interval:       time.Duration(cfg.Workflow.Interval) * time.Second,
		logger:         log,
		previousIssues: make(map[int64]github.Issue),
	}
}

// SetLogger はロガーを設定する（運用時用）
func (w *IssueWatcher) SetLogger(log logger.Logger) {
	w.logger = log
}

// Start はIssue監視を開始する
func (w *IssueWatcher) Start(ctx context.Context) error {
	w.logger.Info("Starting Issue watcher", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 最初に一度実行
	if err := w.watchOnce(ctx); err != nil {
		w.logger.Error("Initial watch failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Issue watcher stopped due to context cancellation")
			return nil
		case <-ticker.C:
			if err := w.watchOnce(ctx); err != nil {
				w.logger.Error("Watch cycle failed", "error", err)
			}
		}
	}
}

// watchOnce は一度だけIssue監視を実行する
func (w *IssueWatcher) watchOnce(ctx context.Context) error {
	w.logger.Debug("Starting watch cycle")

	issues, err := w.fetchFilteredIssues(ctx)
	if err != nil {
		return err
	}

	changes := w.detectChanges(issues)
	if len(changes) > 0 {
		w.logger.Info("Detected issue changes", "count", len(changes))
		for _, change := range changes {
			w.logChange(change)
		}
	}

	return nil
}

// fetchFilteredIssues はフィルタされたIssue一覧を取得する
func (w *IssueWatcher) fetchFilteredIssues(ctx context.Context) ([]github.Issue, error) {
	owner, repo := w.parseRepository()
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid repository configuration: %s", w.config.GitHub.Repository)
	}

	opts := &github.ListIssuesOptions{
		State:   "open",
		Page:    1,
		PerPage: 100,
	}

	issues, _, err := w.client.ListOpenIssues(ctx, owner, repo, opts)
	if err != nil {
		w.logger.Error("Failed to fetch issues from GitHub", "error", err, "owner", owner, "repo", repo)
		return nil, err
	}

	w.logger.Debug("Fetched issues from GitHub", "total_count", len(issues), "owner", owner, "repo", repo)

	// soba:で始まるラベルを持つIssueのみをフィルタ
	var filteredIssues []github.Issue
	for _, issue := range issues {
		if w.hasSobaLabel(issue) {
			filteredIssues = append(filteredIssues, issue)
		}
	}

	w.logger.Debug("Filtered soba issues", "filtered_count", len(filteredIssues), "total_count", len(issues))

	return filteredIssues, nil
}

// hasSobaLabel はIssueがsoba:で始まるラベルを持つかチェックする
func (w *IssueWatcher) hasSobaLabel(issue github.Issue) bool {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.Name, "soba:") {
			return true
		}
	}
	return false
}

// detectChanges はIssueの変更を検知する
func (w *IssueWatcher) detectChanges(currentIssues []github.Issue) []IssueChange {
	var changes []IssueChange

	// 現在のIssueをマップに変換
	currentIssueMap := make(map[int64]github.Issue)
	for _, issue := range currentIssues {
		currentIssueMap[issue.ID] = issue
	}

	// 新しいIssueと変更されたIssueをチェック
	for _, current := range currentIssues {
		if previous, exists := w.previousIssues[current.ID]; exists {
			// 既存のIssue - 変更をチェック
			if w.hasLabelChanged(previous, current) {
				changes = append(changes, IssueChange{
					Type:     IssueChangeTypeLabelChanged,
					Issue:    current,
					Previous: &previous,
				})
			}
			if previous.State != current.State {
				changes = append(changes, IssueChange{
					Type:     IssueChangeTypeStateChanged,
					Issue:    current,
					Previous: &previous,
				})
			}
		} else {
			// 新しいIssue
			changes = append(changes, IssueChange{
				Type:  IssueChangeTypeNew,
				Issue: current,
			})
		}
	}

	// 前回の状態を更新
	w.previousIssues = currentIssueMap

	return changes
}

// hasLabelChanged はラベルが変更されたかチェックする
func (w *IssueWatcher) hasLabelChanged(previous, current github.Issue) bool {
	if len(previous.Labels) != len(current.Labels) {
		return true
	}

	// ラベル名のセットを比較
	prevLabelNames := make(map[string]bool)
	for _, label := range previous.Labels {
		prevLabelNames[label.Name] = true
	}

	for _, label := range current.Labels {
		if !prevLabelNames[label.Name] {
			return true
		}
	}

	return false
}

// parseRepository は設定からowner/repoを分解する
func (w *IssueWatcher) parseRepository() (string, string) {
	repo := w.config.GitHub.Repository
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		w.logger.Error("Invalid repository format", "repository", repo, "expected_format", "owner/repo")
		return "", ""
	}
	return parts[0], parts[1]
}

// logChange は変更をログ出力する
func (w *IssueWatcher) logChange(change IssueChange) {
	switch change.Type {
	case IssueChangeTypeNew:
		w.logger.Info("New issue detected",
			"issue_number", change.Issue.Number,
			"title", change.Issue.Title,
			"labels", w.formatLabels(change.Issue.Labels))
	case IssueChangeTypeLabelChanged:
		w.logger.Info("Issue label changed",
			"issue_number", change.Issue.Number,
			"title", change.Issue.Title,
			"old_labels", w.formatLabels(change.Previous.Labels),
			"new_labels", w.formatLabels(change.Issue.Labels))
	case IssueChangeTypeStateChanged:
		w.logger.Info("Issue state changed",
			"issue_number", change.Issue.Number,
			"title", change.Issue.Title,
			"old_state", change.Previous.State,
			"new_state", change.Issue.State)
	}
}

// formatLabels はラベル一覧を文字列にフォーマットする
func (w *IssueWatcher) formatLabels(labels []github.Label) string {
	labelNames := make([]string, 0, len(labels)) // prealloc対応
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}
	return strings.Join(labelNames, ", ")
}

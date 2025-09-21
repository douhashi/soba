package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
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
	previousIssues map[int64]github.Issue  // Issue IDをキーとする前回の状態
	phaseStrategy  domain.PhaseStrategy    // Phase管理戦略
	processor      IssueProcessorInterface // Issue処理用のプロセッサ
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
		phaseStrategy:  nil, // デフォルトではPhaseStrategyは無効
	}
}

// SetLogger はロガーを設定する（運用時用）
func (w *IssueWatcher) SetLogger(log logger.Logger) {
	w.logger = log
}

// EnablePhaseStrategy はPhaseStrategyを有効にする
func (w *IssueWatcher) EnablePhaseStrategy() {
	w.phaseStrategy = domain.NewDefaultPhaseStrategy()
}

// SetPhaseStrategy はPhaseStrategyを設定する
func (w *IssueWatcher) SetPhaseStrategy(strategy domain.PhaseStrategy) {
	w.phaseStrategy = strategy
}

// SetProcessor はIssueProcessorを設定する
func (w *IssueWatcher) SetProcessor(processor IssueProcessorInterface) {
	w.processor = processor
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
			// PhaseStrategyが有効な場合は、フェーズ分析を行う
			if w.phaseStrategy != nil && change.Type == IssueChangeTypeLabelChanged {
				w.analyzeAndLogPhaseTransition(change)
			}
			// IssueProcessorが設定されている場合、Issueを処理する
			if w.processor != nil && change.Type == IssueChangeTypeLabelChanged {
				if err := w.processor.ProcessIssue(ctx, w.config, change.Issue); err != nil {
					w.logger.Error("Failed to process issue", "error", err, "issue", change.Issue.Number)
				}
			}
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

// analyzePhase はIssueの現在のフェーズを分析する
func (w *IssueWatcher) analyzePhase(issue github.Issue) (string, string, error) {
	if w.phaseStrategy == nil {
		return "", "", fmt.Errorf("phase strategy is not enabled")
	}

	// ラベル名の配列を作成
	labelNames := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labelNames = append(labelNames, label.Name)
	}

	// 現在のフェーズを判定
	phase, err := w.phaseStrategy.GetCurrentPhase(labelNames)
	if err != nil {
		return "", "", err
	}

	// 次のラベルを取得
	nextLabel, err := w.phaseStrategy.GetNextLabel(phase)
	if err != nil {
		// 次の遷移がない場合はエラーではなく空文字を返す
		return string(phase), "", nil
	}

	return string(phase), nextLabel, nil
}

// isValidTransition は遷移が有効かチェックする
func (w *IssueWatcher) isValidTransition(change IssueChange) bool {
	if w.phaseStrategy == nil || change.Previous == nil {
		return true // PhaseStrategyが無効な場合は常に有効とする
	}

	// 前のフェーズを取得
	prevLabelNames := make([]string, 0, len(change.Previous.Labels))
	for _, label := range change.Previous.Labels {
		prevLabelNames = append(prevLabelNames, label.Name)
	}
	prevPhase, err := w.phaseStrategy.GetCurrentPhase(prevLabelNames)
	if err != nil {
		w.logger.Debug("Failed to get previous phase", "error", err)
		return true // エラーの場合は検証をスキップ
	}

	// 現在のフェーズを取得
	currLabelNames := make([]string, 0, len(change.Issue.Labels))
	for _, label := range change.Issue.Labels {
		currLabelNames = append(currLabelNames, label.Name)
	}
	currPhase, err := w.phaseStrategy.GetCurrentPhase(currLabelNames)
	if err != nil {
		w.logger.Debug("Failed to get current phase", "error", err)
		return true // エラーの場合は検証をスキップ
	}

	// 遷移の検証
	err = w.phaseStrategy.ValidateTransition(prevPhase, currPhase)
	return err == nil
}

// analyzeAndLogPhaseTransition はフェーズ遷移を分析してログ出力する
func (w *IssueWatcher) analyzeAndLogPhaseTransition(change IssueChange) {
	if change.Previous == nil {
		return
	}

	// 前のフェーズを取得
	prevLabelNames := make([]string, 0, len(change.Previous.Labels))
	for _, label := range change.Previous.Labels {
		prevLabelNames = append(prevLabelNames, label.Name)
	}
	prevPhase, _ := w.phaseStrategy.GetCurrentPhase(prevLabelNames)

	// 現在のフェーズと次のラベルを取得
	currentPhase, nextLabel, err := w.analyzePhase(change.Issue)
	if err != nil {
		w.logger.Debug("Failed to analyze phase", "error", err, "issue_number", change.Issue.Number)
		return
	}

	// 遷移の検証
	isValid := w.isValidTransition(change)

	// ログ出力
	if isValid {
		w.logger.Info("Phase transition detected",
			"issue_number", change.Issue.Number,
			"from_phase", string(prevPhase),
			"to_phase", currentPhase,
			"next_label", nextLabel,
		)
	} else {
		w.logger.Warn("Invalid phase transition detected",
			"issue_number", change.Issue.Number,
			"from_phase", string(prevPhase),
			"to_phase", currentPhase,
		)
	}
}

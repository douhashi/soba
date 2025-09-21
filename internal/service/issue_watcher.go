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
	client           GitHubClientInterface
	config           *config.Config
	interval         time.Duration
	logger           logger.Logger
	previousIssues   map[int64]github.Issue  // Issue IDをキーとする前回の状態
	processor        IssueProcessorInterface // Issue処理用のプロセッサ
	currentIssue     *int                    // 現在処理中のIssue番号（シングルライン処理用）
	queueManager     *QueueManager           // キュー管理用マネージャー
	workflowExecutor WorkflowExecutor        // ワークフロー実行用エグゼキューター
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

// SetProcessor はIssueProcessorを設定する
func (w *IssueWatcher) SetProcessor(processor IssueProcessorInterface) {
	w.processor = processor
}

// SetQueueManager はQueueManagerを設定する
func (w *IssueWatcher) SetQueueManager(qm *QueueManager) {
	w.queueManager = qm
}

// SetWorkflowExecutor はWorkflowExecutorを設定する
func (w *IssueWatcher) SetWorkflowExecutor(executor WorkflowExecutor) {
	w.workflowExecutor = executor
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

	// 1. キュー管理（soba:todo → soba:queued）
	if w.queueManager != nil {
		if err := w.queueManager.EnqueueNextIssue(ctx, issues); err != nil {
			w.logger.Error("Failed to enqueue", "error", err)
		}
	}

	// 2. キューに入ったIssueの処理（soba:queued → plan実行）
	w.processQueuedIssues(ctx, issues)

	// 3. その他のワークフロー処理
	issueToProcess := w.selectIssueForProcessing(issues)

	// 変更を検知してログ出力
	w.detectAndLogChanges(issues)

	// 選択されたIssueを処理
	if err := w.processSelectedIssue(ctx, issueToProcess); err != nil {
		w.logger.Error("Failed to process selected issue", "error", err)
	}

	// 自動フェーズ遷移を処理（queue以外）
	w.handleAutoTransitions(ctx, issues)

	return nil
}

// detectAndLogChanges は変更を検知してログ出力を行う
func (w *IssueWatcher) detectAndLogChanges(issues []github.Issue) {
	changes := w.detectChanges(issues)
	if len(changes) == 0 {
		return
	}

	w.logger.Info("Detected issue changes", "count", len(changes))
	for _, change := range changes {
		w.logChange(change)
		// PhaseStrategyが有効な場合は、フェーズ分析を行う
		if change.Type == IssueChangeTypeLabelChanged {
			w.analyzeAndLogPhaseTransition(change)
		}
	}
}

// processSelectedIssue は選択されたIssueを処理する
func (w *IssueWatcher) processSelectedIssue(ctx context.Context, issueToProcess *github.Issue) error {
	if issueToProcess == nil || w.workflowExecutor == nil {
		return nil
	}

	// トリガーラベルから実行するフェーズを判定
	var phaseToExecute domain.Phase
	for _, phaseDef := range domain.PhaseDefinitions {
		if w.hasLabel(*issueToProcess, phaseDef.TriggerLabel) {
			phaseToExecute = domain.Phase(phaseDef.Name)
			break
		}
	}

	if phaseToExecute == "" {
		w.logger.Debug("No trigger label found for issue", "issue", issueToProcess.Number)
		return nil
	}

	w.logger.Info("Processing issue in single-line mode", "issue", issueToProcess.Number, "phase", phaseToExecute)
	if err := w.workflowExecutor.ExecutePhase(ctx, w.config, issueToProcess.Number, phaseToExecute); err != nil {
		w.logger.Error("Failed to execute phase", "error", err, "issue", issueToProcess.Number, "phase", phaseToExecute)
		return err
	}

	return nil
}

// processQueuedIssues はキューに入ったIssueを処理する
func (w *IssueWatcher) processQueuedIssues(ctx context.Context, issues []github.Issue) {
	if w.workflowExecutor == nil {
		return
	}

	for _, issue := range issues {
		// soba:queuedラベルがあれば即座にplanフェーズを実行
		if w.hasLabel(issue, domain.LabelQueued) {
			w.logger.Info("Processing queued issue", "issue", issue.Number)

			// 明示的にplanフェーズを実行
			err := w.workflowExecutor.ExecutePhase(ctx, w.config, issue.Number, domain.PhasePlan)
			if err != nil {
				w.logger.Error("Failed to execute plan phase", "error", err, "issue", issue.Number)
			}
			break // シングルライン処理のため1つだけ処理
		}
	}
}

// handleAutoTransitions は自動フェーズ遷移を処理する
func (w *IssueWatcher) handleAutoTransitions(ctx context.Context, issues []github.Issue) {
	if w.processor == nil {
		return
	}

	for _, issue := range issues {
		// 処理中のIssueのみ対象（シングルライン処理を考慮）
		if w.currentIssue != nil && issue.Number != *w.currentIssue {
			continue
		}

		// 自動遷移が必要なフェーズを確認
		if !w.shouldAutoTransition(issue) {
			continue
		}

		w.logger.Info("Auto-transitioning issue phase", "issue", issue.Number)
		if err := w.processor.ProcessIssue(ctx, w.config, issue); err != nil {
			w.logger.Error("Failed to auto-transition issue", "error", err, "issue", issue.Number)
		}
	}
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
	// ラベル名の配列を作成
	labelNames := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labelNames = append(labelNames, label.Name)
	}

	// 現在のフェーズを判定
	phase, err := domain.GetCurrentPhaseFromLabels(labelNames)
	if err != nil {
		return "", "", err
	}

	// 次のラベルを取得（フェーズ定義から）
	phaseDef := domain.PhaseDefinitions[string(phase)]
	if phaseDef == nil {
		return string(phase), "", nil
	}

	// 完了ラベルから最初のものを次のラベルとして使用
	var nextLabel string
	for label := range phaseDef.CompletionLabels {
		nextLabel = label
		break
	}

	return string(phase), nextLabel, nil
}

// isValidTransition は遷移が有効かチェックする
func (w *IssueWatcher) isValidTransition(change IssueChange) bool {
	if change.Previous == nil {
		return true // PhaseStrategyが無効な場合は常に有効とする
	}

	// 前のフェーズを取得
	prevLabelNames := make([]string, 0, len(change.Previous.Labels))
	for _, label := range change.Previous.Labels {
		prevLabelNames = append(prevLabelNames, label.Name)
	}
	prevPhase, err := domain.GetCurrentPhaseFromLabels(prevLabelNames)
	if err != nil {
		w.logger.Debug("Failed to get previous phase", "error", err)
		return true // エラーの場合は検証をスキップ
	}

	// 現在のフェーズを取得
	currLabelNames := make([]string, 0, len(change.Issue.Labels))
	for _, label := range change.Issue.Labels {
		currLabelNames = append(currLabelNames, label.Name)
	}
	currPhase, err := domain.GetCurrentPhaseFromLabels(currLabelNames)
	if err != nil {
		w.logger.Debug("Failed to get current phase", "error", err)
		return true // エラーの場合は検証をスキップ
	}

	// 遷移の検証
	// TODO: 遷移ルールを必要に応じて追加
	_ = prevPhase
	_ = currPhase
	err = nil
	return err == nil
}

// selectIssueForProcessing はシングルライン処理のため、処理するIssueを選択する
func (w *IssueWatcher) selectIssueForProcessing(issues []github.Issue) *github.Issue {
	// 進行中のIssueをチェック
	if inProgressIssue := w.checkInProgressIssues(issues); inProgressIssue != nil {
		return inProgressIssue
	}

	// 現在処理中のIssueをチェック
	if w.checkCurrentIssue(issues) {
		return nil // 処理中Issueが継続中の場合
	}

	// 処理可能なIssueを収集
	processableIssues := w.collectProcessableIssues(issues)
	if len(processableIssues) == 0 {
		return nil
	}

	// 最小番号のIssueを選択して処理開始
	return w.selectMinimumIssue(processableIssues)
}

// checkInProgressIssues は進行中のIssueをチェックし、継続または完了処理を行う
func (w *IssueWatcher) checkInProgressIssues(issues []github.Issue) *github.Issue {
	for _, issue := range issues {
		if w.isInProgressPhase(issue) {
			w.currentIssue = &issue.Number
			w.logger.Debug("Issue still in progress", "issue", issue.Number)
			return nil // シングルライン処理のため、他のIssueは処理しない
		}
	}
	return nil
}

// checkCurrentIssue は現在処理中のIssueの状況をチェックする
func (w *IssueWatcher) checkCurrentIssue(issues []github.Issue) bool {
	if w.currentIssue == nil {
		return false
	}

	currentIssueNumber := *w.currentIssue
	for _, issue := range issues {
		if issue.Number == currentIssueNumber {
			// 進行中ラベルがない場合は、処理が完了したとみなす
			// （例：soba:planning → soba:readyへの遷移）
			if !w.isInProgressPhase(issue) {
				w.logger.Info("Issue phase completed, ready for next phase", "issue", currentIssueNumber)
				w.currentIssue = nil
				return false
			}

			w.logger.Debug("Issue still in progress", "issue", currentIssueNumber)
			return true
		}
	}

	// 処理中のIssueが見つからない場合もクリア（Issue closed の場合）
	w.logger.Info("Processing issue completed (closed)", "issue", currentIssueNumber)
	w.currentIssue = nil
	return false
}

// collectProcessableIssues は処理可能なIssueを収集する
func (w *IssueWatcher) collectProcessableIssues(issues []github.Issue) []github.Issue {
	var processableIssues []github.Issue
	for _, issue := range issues {
		// soba:queuedはprocessQueuedIssuesで処理されるので除外
		if w.hasLabel(issue, domain.LabelQueued) {
			continue
		}

		// QueueManagerが設定されている場合、soba:todoはQueueManagerで処理されるので除外
		if w.queueManager != nil && w.hasLabel(issue, "soba:todo") {
			continue
		}

		hasTodoLabel := w.hasLabel(issue, "soba:todo")
		hasProcessable := w.hasProcessablePhase(issue)
		w.logger.Debug("Checking issue for processing", "issue", issue.Number, "hasTodo", hasTodoLabel, "hasProcessable", hasProcessable)

		if hasTodoLabel || hasProcessable {
			processableIssues = append(processableIssues, issue)
		}
	}
	return processableIssues
}

// selectMinimumIssue は最小番号のIssueを選択して処理開始する
func (w *IssueWatcher) selectMinimumIssue(processableIssues []github.Issue) *github.Issue {
	minIssue := processableIssues[0]
	for _, issue := range processableIssues[1:] {
		if issue.Number < minIssue.Number {
			minIssue = issue
		}
	}

	// 処理開始（まだ処理中のIssueがない場合）
	if w.currentIssue == nil {
		issueNumber := minIssue.Number
		w.currentIssue = &issueNumber
		w.logger.Info("Selected issue for processing", "issue", minIssue.Number)
		return &minIssue
	}

	// 処理中のIssueがある場合は、そのIssueのみ返す
	for _, issue := range processableIssues {
		if issue.Number == *w.currentIssue {
			w.logger.Debug("Continuing processing of current issue", "issue", issue.Number)
			return &issue
		}
	}

	// 処理中のIssueが見つからない場合は新しいIssueを選択
	issueNumber := minIssue.Number
	w.currentIssue = &issueNumber
	w.logger.Info("Selected new issue for processing", "issue", minIssue.Number)
	return &minIssue
}

// hasLabel は指定されたラベルを持つかチェックする
func (w *IssueWatcher) hasLabel(issue github.Issue, labelName string) bool {
	for _, label := range issue.Labels {
		if label.Name == labelName {
			return true
		}
	}
	return false
}

// hasProcessablePhase はIssueが処理可能なフェーズにあるかチェックする
func (w *IssueWatcher) hasProcessablePhase(issue github.Issue) bool {
	// トリガーラベルを持つIssueを処理可能とする
	// (soba:queuedは除外 - processQueuedIssuesで処理される)
	triggerLabels := []string{
		domain.LabelReady,           // implementフェーズのトリガー
		domain.LabelReviewRequested, // reviewフェーズのトリガー
		domain.LabelRequiresChanges, // reviseフェーズのトリガー
		domain.LabelDone,            // mergeフェーズのトリガー
	}

	for _, triggerLabel := range triggerLabels {
		if w.hasLabel(issue, triggerLabel) {
			return true
		}
	}

	// 実行中ラベルを持つIssueも処理継続の対象
	executionLabels := []string{
		domain.LabelPlanning,
		domain.LabelDoing,
		domain.LabelReviewing,
		domain.LabelRevising,
	}

	for _, executionLabel := range executionLabels {
		if w.hasLabel(issue, executionLabel) {
			return true
		}
	}

	return false
}

// isInProgressPhase はIssueが進行中のフェーズにあるかチェックする
func (w *IssueWatcher) isInProgressPhase(issue github.Issue) bool {
	// 進行中とみなすラベル
	inProgressLabels := []string{
		"soba:planning", "soba:doing", "soba:reviewing", "soba:revising",
	}

	for _, label := range inProgressLabels {
		if w.hasLabel(issue, label) {
			return true
		}
	}
	return false
}

// shouldAutoTransition は自動遷移が必要かチェックする
func (w *IssueWatcher) shouldAutoTransition(issue github.Issue) bool {
	// 現在のアーキテクチャではqueueフェーズの自動遷移は
	// processQueuedIssuesで処理されるため、常にfalseを返す
	return false
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
	prevPhase, _ := domain.GetCurrentPhaseFromLabels(prevLabelNames)

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

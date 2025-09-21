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

// PRWatcher はPR監視機能を提供する
type PRWatcher struct {
	client   GitHubClientInterface
	config   *config.Config
	interval time.Duration
	logger   logger.Logger
}

// NewPRWatcher は新しいPRWatcherを作成する
func NewPRWatcher(client GitHubClientInterface, cfg *config.Config) *PRWatcher {
	// デフォルト値の設定
	if cfg.Workflow.Interval == 0 {
		cfg.Workflow.Interval = 20
	}

	// ロガーの初期化（テスト環境を考慮）
	log := logger.NewNopLogger() // デフォルトでNopLogger使用

	return &PRWatcher{
		client:   client,
		config:   cfg,
		interval: time.Duration(cfg.Workflow.Interval) * time.Second,
		logger:   log,
	}
}

// SetLogger はロガーを設定する（運用時用）
func (w *PRWatcher) SetLogger(log logger.Logger) {
	w.logger = log
}

// Start はPR監視を開始する
func (w *PRWatcher) Start(ctx context.Context) error {
	w.logger.Info("Starting PR watcher", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 最初に一度実行
	if err := w.watchOnce(ctx); err != nil {
		w.logger.Error("Initial watch failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("PR watcher stopped due to context cancellation")
			return nil
		case <-ticker.C:
			if err := w.watchOnce(ctx); err != nil {
				w.logger.Error("Watch cycle failed", "error", err)
			}
		}
	}
}

// watchOnce は一度だけPR監視を実行する
func (w *PRWatcher) watchOnce(ctx context.Context) error {
	w.logger.Debug("Starting PR watch cycle")

	prs, err := w.fetchOpenPullRequests(ctx)
	if err != nil {
		return err
	}

	w.logger.Debug("Fetched pull requests", "count", len(prs))

	// soba:lgtmラベルが付いたPRを処理
	for _, pr := range prs {
		if w.hasLGTMLabel(pr) {
			w.logger.Info("Found PR with soba:lgtm label",
				"number", pr.Number,
				"title", pr.Title,
			)

			if err := w.mergePullRequest(ctx, pr); err != nil {
				w.logger.Error("Failed to merge PR",
					"number", pr.Number,
					"error", err,
				)
				// エラーが発生してもサービスは継続
				continue
			}
		}
	}

	return nil
}

// fetchOpenPullRequests はオープンなPR一覧を取得する
func (w *PRWatcher) fetchOpenPullRequests(ctx context.Context) ([]github.PullRequest, error) {
	owner, repo := w.parseRepository()
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid repository configuration: %s", w.config.GitHub.Repository)
	}

	opts := &github.ListPullRequestsOptions{
		State:   "open",
		Page:    1,
		PerPage: 100,
	}

	prs, _, err := w.client.ListPullRequests(ctx, owner, repo, opts)
	if err != nil {
		w.logger.Error("Failed to fetch pull requests from GitHub",
			"error", err,
			"owner", owner,
			"repo", repo,
		)
		return nil, err
	}

	return prs, nil
}

// hasLGTMLabel はPRがsoba:lgtmラベルを持つかチェックする
func (w *PRWatcher) hasLGTMLabel(pr github.PullRequest) bool {
	for _, label := range pr.Labels {
		if label.Name == "soba:lgtm" {
			return true
		}
	}
	return false
}

// mergePullRequest はPRをSquash Mergeする
func (w *PRWatcher) mergePullRequest(ctx context.Context, pr github.PullRequest) error {
	owner, repo := w.parseRepository()
	if owner == "" || repo == "" {
		return fmt.Errorf("invalid repository configuration: %s", w.config.GitHub.Repository)
	}

	// mergeableがnullの場合は、GitHub APIが計算中の可能性があるため、個別にPR情報を再取得
	if pr.MergeableState == "" {
		w.logger.Info("PR mergeable state is unknown, fetching detailed PR info",
			"number", pr.Number,
		)

		// PR情報を個別に取得（最大3回リトライ）
		var detailedPR *github.PullRequest
		var err error
		for i := 0; i < 3; i++ {
			detailedPR, _, err = w.client.GetPullRequest(ctx, owner, repo, pr.Number)
			if err != nil {
				w.logger.Error("Failed to get detailed PR info",
					"number", pr.Number,
					"attempt", i+1,
					"error", err,
				)
				return err
			}

			// mergeableStateが判明したら終了
			if detailedPR.MergeableState != "" {
				pr = *detailedPR
				break
			}

			// まだ計算中の場合は少し待つ
			if i < 2 {
				w.logger.Debug("PR mergeable state still unknown, waiting...",
					"number", pr.Number,
					"attempt", i+1,
				)
				time.Sleep(2 * time.Second)
			}
		}
	}

	// マージ可能な状態かチェック
	if !pr.Mergeable || pr.MergeableState != "clean" {
		w.logger.Info("PR is not in mergeable state",
			"number", pr.Number,
			"mergeable", pr.Mergeable,
			"mergeableState", pr.MergeableState,
		)
		return nil // エラーではなくスキップ
	}

	// Squash Mergeリクエストを作成
	mergeReq := &github.MergeRequest{
		CommitTitle: fmt.Sprintf("feat: %s (#%d)", pr.Title, pr.Number),
		MergeMethod: "squash",
	}

	// マージ実行
	resp, err := w.client.MergePullRequest(ctx, owner, repo, pr.Number, mergeReq)
	if err != nil {
		return fmt.Errorf("failed to merge PR #%d: %w", pr.Number, err)
	}

	if resp.Merged {
		w.logger.Info("Successfully merged PR",
			"number", pr.Number,
			"sha", resp.SHA,
		)
	} else {
		w.logger.Warn("PR merge was not successful",
			"number", pr.Number,
			"message", resp.Message,
		)
	}

	return nil
}

// parseRepository は設定からowner/repoを分解する
func (w *PRWatcher) parseRepository() (string, string) {
	repo := w.config.GitHub.Repository
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		w.logger.Error("Invalid repository format",
			"repository", repo,
			"expected_format", "owner/repo",
		)
		return "", ""
	}
	return parts[0], parts[1]
}

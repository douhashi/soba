package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/slack"
	"github.com/douhashi/soba/pkg/logging"
)

// PRWatcher はPR監視機能を提供する
type PRWatcher struct {
	client   GitHubClientInterface
	config   *config.Config
	interval time.Duration
	logger   logging.Logger
}

// NewPRWatcher は新しいPRWatcherを作成する
func NewPRWatcher(client GitHubClientInterface, cfg *config.Config) *PRWatcher {
	// デフォルト値の設定
	if cfg.Workflow.Interval == 0 {
		cfg.Workflow.Interval = 20
	}

	// ロガーの初期化（テスト環境を考慮）
	log := logging.NewMockLogger() // デフォルトでMockLogger使用

	return &PRWatcher{
		client:   client,
		config:   cfg,
		interval: time.Duration(cfg.Workflow.Interval) * time.Second,
		logger:   log,
	}
}

// SetLogger はロガーを設定する（運用時用）
func (w *PRWatcher) SetLogger(log logging.Logger) {
	w.logger = log
}

// Start はPR監視を開始する
func (w *PRWatcher) Start(ctx context.Context) error {
	w.logger.Info(ctx, "Starting PR watcher", logging.Field{Key: "interval", Value: w.interval})

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 最初に一度実行
	if err := w.watchOnce(ctx); err != nil {
		w.logger.Error(ctx, "Initial watch failed", logging.Field{Key: "error", Value: err.Error()})
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info(ctx, "PR watcher stopped due to context cancellation")
			return nil
		case <-ticker.C:
			if err := w.watchOnce(ctx); err != nil {
				w.logger.Error(ctx, "Watch cycle failed", logging.Field{Key: "error", Value: err.Error()})
			}
		}
	}
}

// watchOnce は一度だけPR監視を実行する
func (w *PRWatcher) watchOnce(ctx context.Context) error {
	w.logger.Info(ctx, "Starting PR watch cycle")

	prs, err := w.fetchOpenPullRequests(ctx)
	if err != nil {
		return err
	}

	w.logger.Debug(ctx, "Fetched pull requests", logging.Field{Key: "count", Value: len(prs)})

	// soba:lgtmラベルが付いたPRを処理
	for _, pr := range prs {
		if w.hasLGTMLabel(pr) {
			w.logger.Info(ctx, "Found PR with soba:lgtm label",
				logging.Field{Key: "number", Value: pr.Number},
				logging.Field{Key: "title", Value: pr.Title},
			)

			if err := w.mergePullRequest(ctx, pr); err != nil {
				w.logger.Error(ctx, "Failed to merge PR",
					logging.Field{Key: "number", Value: pr.Number},
					logging.Field{Key: "error", Value: err.Error()},
				)
				// エラーが発生してもサービスは継続
				continue
			}
		}
	}

	w.logger.Info(ctx, "PR watch cycle completed")
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
		w.logger.Error(ctx, "Failed to fetch pull requests from GitHub",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "owner", Value: owner},
			logging.Field{Key: "repo", Value: repo},
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
		w.logger.Info(ctx, "PR mergeable state is unknown, fetching detailed PR info",
			logging.Field{Key: "number", Value: pr.Number},
		)

		// PR情報を個別に取得（最大3回リトライ）
		var detailedPR *github.PullRequest
		var err error
		for i := 0; i < 3; i++ {
			detailedPR, _, err = w.client.GetPullRequest(ctx, owner, repo, pr.Number)
			if err != nil {
				w.logger.Error(ctx, "Failed to get detailed PR info",
					logging.Field{Key: "number", Value: pr.Number},
					logging.Field{Key: "attempt", Value: i + 1},
					logging.Field{Key: "error", Value: err.Error()},
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
				w.logger.Debug(ctx, "PR mergeable state still unknown, waiting...",
					logging.Field{Key: "number", Value: pr.Number},
					logging.Field{Key: "attempt", Value: i + 1},
				)
				time.Sleep(2 * time.Second)
			}
		}
	}

	// マージ可能な状態かチェック
	if !pr.Mergeable || pr.MergeableState != "clean" {
		w.logger.Info(ctx, "PR is not in mergeable state",
			logging.Field{Key: "number", Value: pr.Number},
			logging.Field{Key: "mergeable", Value: pr.Mergeable},
			logging.Field{Key: "mergeableState", Value: pr.MergeableState},
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
		w.logger.Info(ctx, "Successfully merged PR",
			logging.Field{Key: "number", Value: pr.Number},
			logging.Field{Key: "sha", Value: resp.SHA},
		)

		// Slack通知: PRマージ完了
		// PR番号からIssue番号を抽出 (ファイル名パターンから推測)
		issueNumber := w.extractIssueNumber(pr.Title)
		slack.NotifyPRMerged(pr.Number, issueNumber)
	} else {
		w.logger.Warn(ctx, "PR merge was not successful",
			logging.Field{Key: "number", Value: pr.Number},
			logging.Field{Key: "message", Value: resp.Message},
		)
	}

	return nil
}

// parseRepository は設定からowner/repoを分解する
func (w *PRWatcher) parseRepository() (string, string) {
	repo := w.config.GitHub.Repository
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		w.logger.Error(context.Background(), "Invalid repository format",
			logging.Field{Key: "repository", Value: repo},
			logging.Field{Key: "expected_format", Value: "owner/repo"},
		)
		return "", ""
	}
	return parts[0], parts[1]
}

// extractIssueNumber はPRタイトルからIssue番号を抽出する
func (w *PRWatcher) extractIssueNumber(title string) int {
	// PRタイトルから"(#数字)"パターンを探す
	parts := strings.Split(title, "(#")
	if len(parts) < 2 {
		return 0
	}

	numberPart := strings.Split(parts[1], ")")[0]
	issueNumber, err := strconv.Atoi(numberPart)
	if err != nil {
		w.logger.Debug(context.Background(), "Failed to extract issue number from PR title",
			logging.Field{Key: "title", Value: title},
			logging.Field{Key: "error", Value: err.Error()},
		)
		return 0
	}

	return issueNumber
}

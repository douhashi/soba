package service

import (
	"context"
	"strings"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logging"
)

// GitHubClientInterface はGitHubクライアントのインターフェース
type GitHubClientInterface interface {
	ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error)
	AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	ListPullRequests(ctx context.Context, owner, repo string, opts *github.ListPullRequestsOptions) ([]github.PullRequest, bool, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, bool, error)
	MergePullRequest(ctx context.Context, owner, repo string, number int, req *github.MergeRequest) (*github.MergeResponse, error)
}

type issueProcessor struct {
	githubClient GitHubClientInterface
	owner        string
	repo         string
	executor     WorkflowExecutor
}

// NewIssueProcessor creates a new issue processor with dependencies
func NewIssueProcessor(client GitHubClientInterface, executor WorkflowExecutor) IssueProcessorInterface {
	return &issueProcessor{
		githubClient: client,
		executor:     executor,
	}
}

// ProcessIssue processes a single issue
func (p *issueProcessor) ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error {
	log := logging.NewMockLogger() // テスト環境でのロガー競合を避けるためMockLoggerを使用

	// owner/repoを設定から取得して設定
	if cfg.GitHub.Repository != "" {
		parts := strings.Split(cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			p.owner = parts[0]
			p.repo = parts[1]
		}
	}

	// GitHubクライアントが初期化されていない場合は初期化
	if p.githubClient == nil {
		tokenProvider := github.NewDefaultTokenProvider()
		client, err := github.NewClient(tokenProvider, &github.ClientOptions{
			Logger: log,
		})
		if err != nil {
			log.Error(ctx, "Failed to create GitHub client", logging.Field{Key: "error", Value: err.Error()})
			return errors.WrapInternal(err, "failed to create GitHub client")
		}
		p.githubClient = client
	}

	// ラベル名の配列を取得
	labelNames := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labelNames = append(labelNames, label.Name)
	}

	// 現在のフェーズを判定
	phase, err := domain.GetCurrentPhaseFromLabels(labelNames)
	if err != nil {
		log.Debug(ctx, "Failed to get current phase",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "issue", Value: issue.Number},
		)
		return errors.WrapInternal(err, "failed to get current phase")
	}

	log.Info(ctx, "Processing issue",
		logging.Field{Key: "issue", Value: issue.Number},
		logging.Field{Key: "phase", Value: phase},
		logging.Field{Key: "labels", Value: labelNames},
	)

	// WorkflowExecutorを使ってフェーズを実行
	if err := p.executor.ExecutePhase(ctx, cfg, issue.Number, phase); err != nil {
		log.Error(ctx, "Failed to execute phase",
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "issue", Value: issue.Number},
			logging.Field{Key: "phase", Value: phase},
		)
		return errors.WrapInternal(err, "failed to execute phase")
	}

	return nil
}

// Process processes issues from GitHub repository
func (p *issueProcessor) Process(ctx context.Context, cfg *config.Config) error {
	log := logging.NewMockLogger()

	// リポジトリが設定されているかチェック
	if cfg.GitHub.Repository == "" {
		return errors.NewValidationError("repository not configured")
	}

	// owner/repo形式かチェック
	parts := strings.Split(cfg.GitHub.Repository, "/")
	if len(parts) != 2 {
		return errors.NewValidationError("invalid repository format: expected 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]
	p.owner = owner
	p.repo = repo

	// GitHubクライアントを初期化（まだ設定されていない場合）
	if p.githubClient == nil {
		tokenProvider := github.NewDefaultTokenProvider()
		client, err := github.NewClient(tokenProvider, &github.ClientOptions{
			Logger: log,
		})
		if err != nil {
			log.Error(ctx, "Failed to create GitHub client", logging.Field{Key: "error", Value: err.Error()})
			return errors.WrapInternal(err, "failed to create GitHub client")
		}
		p.githubClient = client
	}

	log.Debug(ctx, "Processing issues", logging.Field{Key: "repository", Value: cfg.GitHub.Repository})

	// Openなissueを取得
	options := &github.ListIssuesOptions{
		State: "open",
	}

	issues, _, err := p.githubClient.ListOpenIssues(ctx, owner, repo, options)
	if err != nil {
		log.Error(ctx, "Failed to list issues", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to list issues")
	}

	log.Info(ctx, "Retrieved issues",
		logging.Field{Key: "count", Value: len(issues)},
		logging.Field{Key: "repository", Value: cfg.GitHub.Repository},
	)

	// ここで各issueに対する処理を実装
	// 現在は取得とログ出力のみ
	for _, issue := range issues {
		log.Debug(ctx, "Found open issue",
			logging.Field{Key: "number", Value: issue.Number},
			logging.Field{Key: "title", Value: issue.Title},
			logging.Field{Key: "state", Value: issue.State},
		)
	}

	return nil
}

// Configure は設定を適用する
func (p *issueProcessor) Configure(cfg *config.Config) error {
	// owner/repoを設定から取得して設定
	if cfg.GitHub.Repository != "" {
		parts := strings.Split(cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			p.owner = parts[0]
			p.repo = parts[1]
		}
	}

	// GitHubクライアントが初期化されていない場合は初期化
	if p.githubClient == nil {
		tokenProvider := github.NewDefaultTokenProvider()
		client, err := github.NewClient(tokenProvider, &github.ClientOptions{})
		if err != nil {
			return errors.WrapInternal(err, "failed to create GitHub client")
		}
		p.githubClient = client
	}

	return nil
}

// UpdateLabels はIssueのラベルを更新する（削除→追加）
func (p *issueProcessor) UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	log := logging.NewMockLogger()

	// GitHubクライアントが初期化されているか確認
	if p.githubClient == nil {
		return errors.NewInternalError("GitHub client not initialized")
	}

	// owner/repoが設定されているか確認
	if p.owner == "" || p.repo == "" {
		return errors.NewInternalError("repository info not set")
	}

	log.Info(ctx, "Starting label update",
		logging.Field{Key: "issue", Value: issueNumber},
		logging.Field{Key: "owner", Value: p.owner},
		logging.Field{Key: "repo", Value: p.repo},
		logging.Field{Key: "remove", Value: removeLabel},
		logging.Field{Key: "add", Value: addLabel},
	)

	// 古いラベルを削除（存在しない場合はスキップ）
	if removeLabel != "" {
		log.Debug(ctx, "Attempting to remove label",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "label", Value: removeLabel},
		)

		if err := p.githubClient.RemoveLabelFromIssue(ctx, p.owner, p.repo, issueNumber, removeLabel); err != nil {
			// エラーメッセージを解析して、ラベルが存在しない場合は警告として扱う
			errMsg := err.Error()
			if strings.Contains(strings.ToLower(errMsg), "not found") ||
				strings.Contains(strings.ToLower(errMsg), "404") ||
				strings.Contains(strings.ToLower(errMsg), "label does not exist") {
				log.Warn(ctx, "Label not found on issue, skipping removal",
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: removeLabel},
				)
				// ラベルが存在しない場合はエラーとせず、処理を続行
			} else {
				log.Error(ctx, "Failed to remove label",
					logging.Field{Key: "error", Value: err.Error()},
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: removeLabel},
				)
				return errors.WrapInternal(err, "failed to remove label")
			}
		} else {
			log.Info(ctx, "Successfully removed label from issue",
				logging.Field{Key: "issue", Value: issueNumber},
				logging.Field{Key: "label", Value: removeLabel},
			)
		}
	}

	// 新しいラベルを追加
	if addLabel != "" {
		log.Debug(ctx, "Attempting to add label",
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "label", Value: addLabel},
		)

		if err := p.githubClient.AddLabelToIssue(ctx, p.owner, p.repo, issueNumber, addLabel); err != nil {
			// エラーメッセージを解析して、既にラベルが存在する場合は警告として扱う
			errMsg := err.Error()
			if strings.Contains(strings.ToLower(errMsg), "already exists") ||
				strings.Contains(strings.ToLower(errMsg), "label already added") {
				log.Warn(ctx, "Label already exists on issue",
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: addLabel},
				)
				// 既にラベルが存在する場合もエラーとせず、成功として扱う
			} else {
				log.Error(ctx, "Failed to add label",
					logging.Field{Key: "error", Value: err.Error()},
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: addLabel},
				)
				return errors.WrapInternal(err, "failed to add label")
			}
		} else {
			log.Info(ctx, "Successfully added label to issue",
				logging.Field{Key: "issue", Value: issueNumber},
				logging.Field{Key: "label", Value: addLabel},
			)
		}
	}

	log.Info(ctx, "Label update completed successfully",
		logging.Field{Key: "issue", Value: issueNumber},
		logging.Field{Key: "removed", Value: removeLabel},
		logging.Field{Key: "added", Value: addLabel},
	)

	return nil
}

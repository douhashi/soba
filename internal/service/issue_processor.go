package service

import (
	"context"
	"strings"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
)

// GitHubClientInterface はGitHubクライアントのインターフェース
type GitHubClientInterface interface {
	ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error)
}

type issueProcessor struct {
	githubClient GitHubClientInterface
}

// NewIssueProcessor creates a new issue processor
func NewIssueProcessor() IssueProcessorInterface {
	return &issueProcessor{}
}

// Process processes issues from GitHub repository
func (p *issueProcessor) Process(ctx context.Context, cfg *config.Config) error {
	log := logger.NewLogger(logger.GetLogger())

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

	// GitHubクライアントを初期化（まだ設定されていない場合）
	if p.githubClient == nil {
		tokenProvider := github.NewDefaultTokenProvider()
		client, err := github.NewClient(tokenProvider, &github.ClientOptions{
			Logger: log,
		})
		if err != nil {
			log.Error("Failed to create GitHub client", "error", err)
			return errors.WrapInternal(err, "failed to create GitHub client")
		}
		p.githubClient = client
	}

	log.Debug("Processing issues", "repository", cfg.GitHub.Repository)

	// Openなissueを取得
	options := &github.ListIssuesOptions{
		State: "open",
	}

	issues, _, err := p.githubClient.ListOpenIssues(ctx, owner, repo, options)
	if err != nil {
		log.Error("Failed to list issues", "error", err)
		return errors.WrapInternal(err, "failed to list issues")
	}

	log.Info("Retrieved issues", "count", len(issues), "repository", cfg.GitHub.Repository)

	// ここで各issueに対する処理を実装
	// 現在は取得とログ出力のみ
	for _, issue := range issues {
		log.Debug("Found open issue",
			"number", issue.Number,
			"title", issue.Title,
			"state", issue.State,
		)
	}

	return nil
}

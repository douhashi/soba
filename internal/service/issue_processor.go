package service

import (
	"context"
	"strings"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
)

// GitHubClientInterface はGitHubクライアントのインターフェース
type GitHubClientInterface interface {
	ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error)
	AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
}

type issueProcessor struct {
	githubClient GitHubClientInterface
	owner        string
	repo         string
	executor     WorkflowExecutor
	strategy     domain.PhaseStrategy
}

// NewIssueProcessor creates a new issue processor
func NewIssueProcessor() IssueProcessorInterface {
	return &issueProcessor{}
}

// NewIssueProcessorWithDependencies creates a new issue processor with dependencies
func NewIssueProcessorWithDependencies(client GitHubClientInterface, executor WorkflowExecutor, strategy domain.PhaseStrategy) IssueProcessorInterface {
	return &issueProcessor{
		githubClient: client,
		executor:     executor,
		strategy:     strategy,
	}
}

// ProcessIssue processes a single issue
func (p *issueProcessor) ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error {
	log := logger.NewNopLogger() // テスト環境でのロガー競合を避けるためNopLoggerを使用

	// ラベル名の配列を取得
	labelNames := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labelNames = append(labelNames, label.Name)
	}

	// 現在のフェーズを判定
	phase, err := p.strategy.GetCurrentPhase(labelNames)
	if err != nil {
		log.Debug("Failed to get current phase", "error", err, "issue", issue.Number)
		return errors.WrapInternal(err, "failed to get current phase")
	}

	log.Info("Processing issue", "issue", issue.Number, "phase", phase, "labels", labelNames)

	// WorkflowExecutorを使ってフェーズを実行
	if err := p.executor.ExecutePhase(ctx, cfg, issue.Number, phase, p.strategy); err != nil {
		log.Error("Failed to execute phase", "error", err, "issue", issue.Number, "phase", phase)
		return errors.WrapInternal(err, "failed to execute phase")
	}

	return nil
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
	p.owner = owner
	p.repo = repo

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

// UpdateLabels はIssueのラベルを更新する（削除→追加）
func (p *issueProcessor) UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	log := logger.NewLogger(logger.GetLogger())

	// GitHubクライアントが初期化されているか確認
	if p.githubClient == nil {
		return errors.NewInternalError("GitHub client not initialized")
	}

	// owner/repoが設定されているか確認
	if p.owner == "" || p.repo == "" {
		return errors.NewInternalError("repository info not set")
	}

	// 古いラベルを削除（存在しない場合はスキップ）
	if removeLabel != "" {
		if err := p.githubClient.RemoveLabelFromIssue(ctx, p.owner, p.repo, issueNumber, removeLabel); err != nil {
			log.Error("Failed to remove label", "error", err, "issue", issueNumber, "label", removeLabel)
			return errors.WrapInternal(err, "failed to remove label")
		}
		log.Debug("Removed label from issue", "issue", issueNumber, "label", removeLabel)
	}

	// 新しいラベルを追加
	if addLabel != "" {
		if err := p.githubClient.AddLabelToIssue(ctx, p.owner, p.repo, issueNumber, addLabel); err != nil {
			log.Error("Failed to add label", "error", err, "issue", issueNumber, "label", addLabel)
			return errors.WrapInternal(err, "failed to add label")
		}
		log.Debug("Added label to issue", "issue", issueNumber, "label", addLabel)
	}

	return nil
}

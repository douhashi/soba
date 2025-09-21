package service

import (
	"context"
	"strings"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logger"
)

// QueueManager はキュー管理機能を提供する
type QueueManager struct {
	client GitHubClientInterface
	owner  string
	repo   string
	logger logger.Logger
}

// NewQueueManager は新しいQueueManagerを作成する
func NewQueueManager(client GitHubClientInterface, owner, repo string) *QueueManager {
	return &QueueManager{
		client: client,
		owner:  owner,
		repo:   repo,
		logger: logger.NewNopLogger(),
	}
}

// SetLogger はロガーを設定する
func (q *QueueManager) SetLogger(log logger.Logger) {
	q.logger = log
}

// EnqueueNextIssue は次のIssueをキューに入れる
func (q *QueueManager) EnqueueNextIssue(ctx context.Context, issues []github.Issue) error {
	// 1. アクティブなタスクがあるか確認
	if q.hasActiveTask(issues) {
		q.logger.Debug("Active task exists, skipping enqueue")
		return nil
	}

	// 2. soba:todoのIssueを収集
	todoIssues := q.collectTodoIssues(issues)
	if len(todoIssues) == 0 {
		q.logger.Debug("No todo issues found")
		return nil
	}

	// 3. 最小番号のIssueを選択
	targetIssue := q.selectMinimumIssue(todoIssues)

	// 4. ラベル変更（soba:todo → soba:queued）
	q.logger.Info("Enqueueing issue", "issue", targetIssue.Number)
	return q.updateLabels(ctx, targetIssue.Number, "soba:todo", "soba:queued")
}

// hasActiveTask はアクティブなタスクがあるかチェック
func (q *QueueManager) hasActiveTask(issues []github.Issue) bool {
	for _, issue := range issues {
		if q.hasSobaLabel(issue) && !q.hasLabel(issue, "soba:todo") {
			return true // soba:todo以外のsobaラベルがある
		}
	}
	return false
}

// collectTodoIssues はtodoラベルを持つIssueを収集する
func (q *QueueManager) collectTodoIssues(issues []github.Issue) []github.Issue {
	var todoIssues []github.Issue
	for _, issue := range issues {
		if q.hasLabel(issue, "soba:todo") {
			todoIssues = append(todoIssues, issue)
		}
	}
	return todoIssues
}

// selectMinimumIssue は最小番号のIssueを選択する
func (q *QueueManager) selectMinimumIssue(issues []github.Issue) *github.Issue {
	if len(issues) == 0 {
		return nil
	}

	minIssue := issues[0]
	for _, issue := range issues[1:] {
		if issue.Number < minIssue.Number {
			minIssue = issue
		}
	}
	return &minIssue
}

// updateLabels はラベルを更新する（削除→追加）
func (q *QueueManager) updateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	// 古いラベルを削除
	if removeLabel != "" {
		if err := q.client.RemoveLabelFromIssue(ctx, q.owner, q.repo, issueNumber, removeLabel); err != nil {
			q.logger.Error("Failed to remove label", "error", err, "issue", issueNumber, "label", removeLabel)
			return errors.WrapInternal(err, "failed to remove label")
		}
		q.logger.Debug("Removed label from issue", "issue", issueNumber, "label", removeLabel)
	}

	// 新しいラベルを追加
	if addLabel != "" {
		if err := q.client.AddLabelToIssue(ctx, q.owner, q.repo, issueNumber, addLabel); err != nil {
			q.logger.Error("Failed to add label", "error", err, "issue", issueNumber, "label", addLabel)
			return errors.WrapInternal(err, "failed to add label")
		}
		q.logger.Debug("Added label to issue", "issue", issueNumber, "label", addLabel)
	}

	return nil
}

// hasLabel は指定されたラベルを持つかチェックする
func (q *QueueManager) hasLabel(issue github.Issue, labelName string) bool {
	for _, label := range issue.Labels {
		if label.Name == labelName {
			return true
		}
	}
	return false
}

// hasSobaLabel はIssueがsoba:で始まるラベルを持つかチェックする
func (q *QueueManager) hasSobaLabel(issue github.Issue) bool {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.Name, "soba:") {
			return true
		}
	}
	return false
}

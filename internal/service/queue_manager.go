package service

import (
	"context"
	"strings"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logging"
)

// QueueManager はキュー管理機能を提供する
type QueueManager struct {
	client GitHubClientInterface
	owner  string
	repo   string
	logger logging.Logger
}

// NewQueueManager は新しいQueueManagerを作成する
func NewQueueManager(client GitHubClientInterface, owner, repo string) *QueueManager {
	return &QueueManager{
		client: client,
		owner:  owner,
		repo:   repo,
		logger: logging.NewMockLogger(),
	}
}

// SetLogger はロガーを設定する
func (q *QueueManager) SetLogger(log logging.Logger) {
	q.logger = log
}

// EnqueueNextIssue は次のIssueをキューに入れる
func (q *QueueManager) EnqueueNextIssue(ctx context.Context, issues []github.Issue) error {
	q.logger.Info(ctx, "Starting queue management",
		logging.Field{Key: "issue_count", Value: len(issues)})

	// 1. アクティブなタスクがあるか確認
	if q.hasActiveTask(issues) {
		q.logger.Debug(ctx, "Active task exists, skipping enqueue")
		q.logger.Info(ctx, "Queue management completed",
			logging.Field{Key: "result", Value: "skipped_active_task"})
		return nil
	}

	// 2. soba:todoのIssueを収集
	todoIssues := q.collectTodoIssues(issues)
	if len(todoIssues) == 0 {
		q.logger.Debug(ctx, "No todo issues found")
		q.logger.Info(ctx, "Queue management completed",
			logging.Field{Key: "result", Value: "no_todo_issues"})
		return nil
	}

	// 3. 最小番号のIssueを選択
	targetIssue := q.selectMinimumIssue(todoIssues)

	// 4. ラベル変更（soba:todo → soba:queued）
	q.logger.Info(ctx, "Enqueueing issue", logging.Field{Key: "issue", Value: targetIssue.Number})
	err := q.updateLabels(ctx, targetIssue.Number, "soba:todo", "soba:queued")
	if err != nil {
		q.logger.Info(ctx, "Queue management completed",
			logging.Field{Key: "result", Value: "failed"},
			logging.Field{Key: "error", Value: err.Error()})
		return err
	}

	q.logger.Info(ctx, "Queue management completed",
		logging.Field{Key: "result", Value: "enqueued"},
		logging.Field{Key: "issue", Value: targetIssue.Number})
	return nil
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
	q.logger.Info(ctx, "Updating labels for queue management",
		logging.Field{Key: "issue", Value: issueNumber},
		logging.Field{Key: "remove", Value: removeLabel},
		logging.Field{Key: "add", Value: addLabel},
	)

	// 古いラベルを削除
	if removeLabel != "" {
		if err := q.client.RemoveLabelFromIssue(ctx, q.owner, q.repo, issueNumber, removeLabel); err != nil {
			// エラーメッセージを解析して、ラベルが存在しない場合は警告として扱う
			errMsg := err.Error()
			if strings.Contains(strings.ToLower(errMsg), "not found") ||
				strings.Contains(strings.ToLower(errMsg), "404") ||
				strings.Contains(strings.ToLower(errMsg), "label does not exist") {
				q.logger.Warn(ctx, "Label not found on issue, skipping removal",
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: removeLabel},
				)
				// ラベルが存在しない場合はエラーとせず、処理を続行
			} else {
				q.logger.Error(ctx, "Failed to remove label",
					logging.Field{Key: "error", Value: err.Error()},
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: removeLabel},
				)
				return errors.WrapInternal(err, "failed to remove label")
			}
		} else {
			q.logger.Info(ctx, "Successfully removed label from issue",
				logging.Field{Key: "issue", Value: issueNumber},
				logging.Field{Key: "label", Value: removeLabel},
			)
		}
	}

	// 新しいラベルを追加
	if addLabel != "" {
		if err := q.client.AddLabelToIssue(ctx, q.owner, q.repo, issueNumber, addLabel); err != nil {
			// エラーメッセージを解析して、既にラベルが存在する場合は警告として扱う
			errMsg := err.Error()
			if strings.Contains(strings.ToLower(errMsg), "already exists") ||
				strings.Contains(strings.ToLower(errMsg), "label already added") {
				q.logger.Warn(ctx, "Label already exists on issue",
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: addLabel},
				)
				// 既にラベルが存在する場合もエラーとせず、成功として扱う
			} else {
				q.logger.Error(ctx, "Failed to add label",
					logging.Field{Key: "error", Value: err.Error()},
					logging.Field{Key: "issue", Value: issueNumber},
					logging.Field{Key: "label", Value: addLabel},
				)
				return errors.WrapInternal(err, "failed to add label")
			}
		} else {
			q.logger.Info(ctx, "Successfully added label to issue",
				logging.Field{Key: "issue", Value: issueNumber},
				logging.Field{Key: "label", Value: addLabel},
			)
		}
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

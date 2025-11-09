package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logging"
)

func TestQueueManager_collectTodoIssues_WithPriority(t *testing.T) {
	qm := &QueueManager{
		logger: logging.NewMockLogger(),
	}

	tests := []struct {
		name     string
		issues   []github.Issue
		expected []int // Expected issue numbers
	}{
		{
			name: "soba:todo:highラベルを持つIssueも収集される",
			issues: []github.Issue{
				{Number: 1, Labels: []github.Label{{Name: "soba:todo:high"}}},
				{Number: 2, Labels: []github.Label{{Name: "soba:todo"}}},
				{Number: 3, Labels: []github.Label{{Name: "soba:planning"}}},
			},
			expected: []int{1, 2},
		},
		{
			name: "soba:todo:lowラベルを持つIssueも収集される",
			issues: []github.Issue{
				{Number: 1, Labels: []github.Label{{Name: "soba:todo:low"}}},
				{Number: 2, Labels: []github.Label{{Name: "soba:todo"}}},
				{Number: 3, Labels: []github.Label{{Name: "bug"}}},
			},
			expected: []int{1, 2},
		},
		{
			name: "soba:todo:highとsoba:todoの両方を持つIssueは1回だけ収集される",
			issues: []github.Issue{
				{Number: 1, Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}}},
				{Number: 2, Labels: []github.Label{{Name: "soba:todo"}}},
			},
			expected: []int{1, 2},
		},
		{
			name: "優先度ラベルのみ（soba:todoなし）でも収集される",
			issues: []github.Issue{
				{Number: 1, Labels: []github.Label{{Name: "soba:todo:high"}}},
				{Number: 2, Labels: []github.Label{{Name: "soba:todo:low"}}},
				{Number: 3, Labels: []github.Label{{Name: "soba:todo"}}},
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "優先度ラベル付きIssueが全て収集される",
			issues: []github.Issue{
				{Number: 1, Labels: []github.Label{{Name: "soba:todo:high"}}},
				{Number: 2, Labels: []github.Label{{Name: "soba:todo"}}},
				{Number: 3, Labels: []github.Label{{Name: "soba:todo:low"}}},
				{Number: 4, Labels: []github.Label{{Name: "soba:planning"}}},
				{Number: 5, Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}}},
			},
			expected: []int{1, 2, 3, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qm.collectTodoIssues(tt.issues)

			// Issue番号のリストを作成
			var resultNumbers []int
			for _, issue := range result {
				resultNumbers = append(resultNumbers, issue.Number)
			}

			assert.Equal(t, tt.expected, resultNumbers)
		})
	}
}

func TestQueueManager_hasActiveTask_WithPriority(t *testing.T) {
	qm := &QueueManager{
		logger: logging.NewMockLogger(),
	}

	tests := []struct {
		name     string
		issues   []github.Issue
		expected bool
	}{
		{
			name: "soba:todo:highのみはアクティブタスクではない",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo:high"}}},
			},
			expected: false,
		},
		{
			name: "soba:todo:lowのみはアクティブタスクではない",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo:low"}}},
			},
			expected: false,
		},
		{
			name: "soba:todo:highとsoba:planningが混在（アクティブタスクあり）",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo:high"}}},
				{Labels: []github.Label{{Name: "soba:planning"}}},
			},
			expected: true,
		},
		{
			name: "優先度ラベルのみの場合はアクティブタスクではない",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo:high"}}},
				{Labels: []github.Label{{Name: "soba:todo"}}},
				{Labels: []github.Label{{Name: "soba:todo:low"}}},
			},
			expected: false,
		},
		{
			name: "soba:queuedがある場合はアクティブタスクあり",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo:high"}}},
				{Labels: []github.Label{{Name: "soba:queued"}}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qm.hasActiveTask(tt.issues)
			assert.Equal(t, tt.expected, result)
		})
	}
}

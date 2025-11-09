package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logging"
)

func TestQueueManager_getPriority(t *testing.T) {
	qm := &QueueManager{
		logger: logging.NewMockLogger(),
	}

	tests := []struct {
		name     string
		issue    github.Issue
		expected int
	}{
		{
			name: "高優先度のIssue",
			issue: github.Issue{
				Number: 1,
				Labels: []github.Label{
					{Name: "soba:todo:high"},
					{Name: "soba:todo"},
				},
			},
			expected: 0, // 最高優先度
		},
		{
			name: "通常優先度のIssue",
			issue: github.Issue{
				Number: 2,
				Labels: []github.Label{
					{Name: "soba:todo"},
				},
			},
			expected: 1, // 通常優先度
		},
		{
			name: "低優先度のIssue",
			issue: github.Issue{
				Number: 3,
				Labels: []github.Label{
					{Name: "soba:todo:low"},
					{Name: "soba:todo"},
				},
			},
			expected: 2, // 低優先度
		},
		{
			name: "高優先度と低優先度の両方を持つIssue（最高優先度を採用）",
			issue: github.Issue{
				Number: 4,
				Labels: []github.Label{
					{Name: "soba:todo:high"},
					{Name: "soba:todo:low"},
					{Name: "soba:todo"},
				},
			},
			expected: 0, // 最高優先度を採用
		},
		{
			name: "優先度ラベルがないIssue",
			issue: github.Issue{
				Number: 5,
				Labels: []github.Label{
					{Name: "bug"},
				},
			},
			expected: 1, // デフォルトは通常優先度
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qm.getPriority(tt.issue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueueManager_sortByPriority(t *testing.T) {
	qm := &QueueManager{
		logger: logging.NewMockLogger(),
	}

	issues := []github.Issue{
		{
			Number: 5,
			Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
		},
		{
			Number: 2,
			Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
		},
		{
			Number: 4,
			Labels: []github.Label{{Name: "soba:todo"}},
		},
		{
			Number: 1,
			Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
		},
		{
			Number: 3,
			Labels: []github.Label{{Name: "soba:todo"}},
		},
		{
			Number: 6,
			Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
		},
	}

	sorted := qm.sortByPriority(issues)

	// 優先度順に並んでいることを確認
	// 高優先度（Issue 1, 2）
	assert.Equal(t, 1, sorted[0].Number)
	assert.Equal(t, 2, sorted[1].Number)

	// 通常優先度（Issue 3, 4）
	assert.Equal(t, 3, sorted[2].Number)
	assert.Equal(t, 4, sorted[3].Number)

	// 低優先度（Issue 5, 6）
	assert.Equal(t, 5, sorted[4].Number)
	assert.Equal(t, 6, sorted[5].Number)
}

func TestQueueManager_selectPriorityIssue(t *testing.T) {
	qm := &QueueManager{
		logger: logging.NewMockLogger(),
	}

	tests := []struct {
		name           string
		issues         []github.Issue
		expectedNumber int
	}{
		{
			name: "高優先度のIssueが優先される",
			issues: []github.Issue{
				{
					Number: 10,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
				{
					Number: 20,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
				{
					Number: 30,
					Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
				},
			},
			expectedNumber: 20, // 高優先度のIssue
		},
		{
			name: "同じ優先度の場合はIssue番号が小さいものが選ばれる",
			issues: []github.Issue{
				{
					Number: 15,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
				{
					Number: 10,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
				{
					Number: 20,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
			},
			expectedNumber: 10, // 同じ高優先度の中で最小番号
		},
		{
			name: "通常優先度のみの場合",
			issues: []github.Issue{
				{
					Number: 30,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
				{
					Number: 10,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
				{
					Number: 20,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
			},
			expectedNumber: 10, // 最小番号
		},
		{
			name: "低優先度のみの場合",
			issues: []github.Issue{
				{
					Number: 25,
					Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
				},
				{
					Number: 15,
					Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
				},
			},
			expectedNumber: 15, // 低優先度の中で最小番号
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qm.selectPriorityIssue(tt.issues)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedNumber, result.Number)
		})
	}
}

func TestQueueManager_EnqueueNextIssue_WithPriority(t *testing.T) {
	tests := []struct {
		name             string
		issues           []github.Issue
		expectedIssueNum int
		setupMock        func(*MockQueueGitHubClient, int)
		shouldEnqueue    bool
	}{
		{
			name: "高優先度のIssueが通常優先度より先に処理される",
			issues: []github.Issue{
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
				{
					Number: 2,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
				{
					Number: 3,
					Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
				},
			},
			expectedIssueNum: 2, // 高優先度のIssue
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:high").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			shouldEnqueue: true,
		},
		{
			name: "高優先度が複数ある場合は番号が小さいものが優先",
			issues: []github.Issue{
				{
					Number: 5,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
				{
					Number: 3,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
			},
			expectedIssueNum: 3, // 高優先度の中で最小番号
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:high").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			shouldEnqueue: true,
		},
		{
			name: "低優先度のみの場合でも正しく処理される",
			issues: []github.Issue{
				{
					Number: 10,
					Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
				},
				{
					Number: 5,
					Labels: []github.Label{{Name: "soba:todo:low"}, {Name: "soba:todo"}},
				},
			},
			expectedIssueNum: 5, // 低優先度の中で最小番号
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:low").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			shouldEnqueue: true,
		},
		{
			name: "アクティブタスクがある場合は優先度に関わらずスキップ",
			issues: []github.Issue{
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:planning"}},
				},
				{
					Number: 2,
					Labels: []github.Label{{Name: "soba:todo:high"}, {Name: "soba:todo"}},
				},
			},
			expectedIssueNum: 0,
			setupMock:        func(m *MockQueueGitHubClient, issueNum int) {},
			shouldEnqueue:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockQueueGitHubClient)
			if tt.shouldEnqueue {
				tt.setupMock(mockClient, tt.expectedIssueNum)
			}

			qm := NewQueueManager(mockClient, "owner", "repo")
			qm.SetLogger(logging.NewMockLogger())

			err := qm.EnqueueNextIssue(context.Background(), tt.issues)
			assert.NoError(t, err)

			mockClient.AssertExpectations(t)
		})
	}
}

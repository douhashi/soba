package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logger"
)

// MockQueueGitHubClient はテスト用のモッククライアント
type MockQueueGitHubClient struct {
	mock.Mock
}

func (m *MockQueueGitHubClient) ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error) {
	args := m.Called(ctx, owner, repo, options)
	return args.Get(0).([]github.Issue), args.Bool(1), args.Error(2)
}

func (m *MockQueueGitHubClient) AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockQueueGitHubClient) RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func TestQueueManager_EnqueueNextIssue(t *testing.T) {
	tests := []struct {
		name          string
		issues        []github.Issue
		setupMock     func(*MockQueueGitHubClient)
		expectedError bool
		expectedLog   string
	}{
		{
			name: "アクティブタスクが存在する場合はスキップ",
			issues: []github.Issue{
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:planning"}},
				},
				{
					Number: 2,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
			},
			setupMock:     func(m *MockQueueGitHubClient) {},
			expectedError: false,
			expectedLog:   "Active task exists, skipping enqueue",
		},
		{
			name: "todoイssueがない場合はスキップ",
			issues: []github.Issue{
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:done"}},
				},
			},
			setupMock:     func(m *MockQueueGitHubClient) {},
			expectedError: false,
		},
		{
			name: "最小番号のtodoイssueをキューに入れる",
			issues: []github.Issue{
				{
					Number: 3,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
				{
					Number: 2,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
			},
			setupMock: func(m *MockQueueGitHubClient) {
				// Issue番号1のラベルを更新
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", 1, "soba:todo").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", 1, "soba:queued").Return(nil)
			},
			expectedError: false,
			expectedLog:   "Enqueueing issue",
		},
		{
			name: "todoとqueuedが混在する場合、アクティブタスクとみなす",
			issues: []github.Issue{
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:queued"}},
				},
				{
					Number: 2,
					Labels: []github.Label{{Name: "soba:todo"}},
				},
			},
			setupMock:     func(m *MockQueueGitHubClient) {},
			expectedError: false,
			expectedLog:   "Active task exists, skipping enqueue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockQueueGitHubClient)
			tt.setupMock(mockClient)

			qm := NewQueueManager(mockClient, "owner", "repo")
			qm.SetLogger(logger.NewNopLogger())

			err := qm.EnqueueNextIssue(context.Background(), tt.issues)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestQueueManager_hasActiveTask(t *testing.T) {
	qm := &QueueManager{
		logger: logger.NewNopLogger(),
	}

	tests := []struct {
		name     string
		issues   []github.Issue
		expected bool
	}{
		{
			name:     "空のIssueリスト",
			issues:   []github.Issue{},
			expected: false,
		},
		{
			name: "todoラベルのみ",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo"}}},
			},
			expected: false,
		},
		{
			name: "planningラベルがある",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:planning"}}},
			},
			expected: true,
		},
		{
			name: "queuedラベルがある",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:queued"}}},
			},
			expected: true,
		},
		{
			name: "todoとplanningが混在",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "soba:todo"}}},
				{Labels: []github.Label{{Name: "soba:planning"}}},
			},
			expected: true,
		},
		{
			name: "sobaラベル以外のみ",
			issues: []github.Issue{
				{Labels: []github.Label{{Name: "bug"}}},
				{Labels: []github.Label{{Name: "enhancement"}}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qm.hasActiveTask(tt.issues)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueueManager_collectTodoIssues(t *testing.T) {
	qm := &QueueManager{
		logger: logger.NewNopLogger(),
	}

	issues := []github.Issue{
		{Number: 1, Labels: []github.Label{{Name: "soba:todo"}}},
		{Number: 2, Labels: []github.Label{{Name: "soba:planning"}}},
		{Number: 3, Labels: []github.Label{{Name: "soba:todo"}}},
		{Number: 4, Labels: []github.Label{{Name: "bug"}}},
		{Number: 5, Labels: []github.Label{{Name: "soba:todo"}, {Name: "enhancement"}}},
	}

	result := qm.collectTodoIssues(issues)

	assert.Len(t, result, 3)
	assert.Equal(t, 1, result[0].Number)
	assert.Equal(t, 3, result[1].Number)
	assert.Equal(t, 5, result[2].Number)
}

func TestQueueManager_selectMinimumIssue(t *testing.T) {
	qm := &QueueManager{
		logger: logger.NewNopLogger(),
	}

	issues := []github.Issue{
		{Number: 5},
		{Number: 2},
		{Number: 8},
		{Number: 1},
		{Number: 3},
	}

	result := qm.selectMinimumIssue(issues)

	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Number)
}

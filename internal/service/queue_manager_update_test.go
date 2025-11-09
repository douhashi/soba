package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logging"
)

func TestQueueManager_updateTodoLabelsToQueued(t *testing.T) {
	tests := []struct {
		name      string
		issue     github.Issue
		setupMock func(*MockQueueGitHubClient, int)
		wantError bool
	}{
		{
			name: "soba:todoラベルのみを削除",
			issue: github.Issue{
				Number: 1,
				Labels: []github.Label{{Name: "soba:todo"}},
			},
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			wantError: false,
		},
		{
			name: "soba:todo:highラベルを削除",
			issue: github.Issue{
				Number: 2,
				Labels: []github.Label{{Name: "soba:todo:high"}},
			},
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:high").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			wantError: false,
		},
		{
			name: "soba:todo:lowラベルを削除",
			issue: github.Issue{
				Number: 3,
				Labels: []github.Label{{Name: "soba:todo:low"}},
			},
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:low").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			wantError: false,
		},
		{
			name: "複数のtodo系ラベルを全て削除",
			issue: github.Issue{
				Number: 4,
				Labels: []github.Label{
					{Name: "soba:todo:high"},
					{Name: "soba:todo"},
				},
			},
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:high").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			wantError: false,
		},
		{
			name: "todo系ラベルとその他のラベルが混在",
			issue: github.Issue{
				Number: 5,
				Labels: []github.Label{
					{Name: "soba:todo:low"},
					{Name: "bug"},
					{Name: "enhancement"},
				},
			},
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				// bugとenhancementラベルは削除されない
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:low").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			wantError: false,
		},
		{
			name: "すべてのtodo系ラベルを削除",
			issue: github.Issue{
				Number: 6,
				Labels: []github.Label{
					{Name: "soba:todo:high"},
					{Name: "soba:todo"},
					{Name: "soba:todo:low"},
				},
			},
			setupMock: func(m *MockQueueGitHubClient, issueNum int) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:high").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", issueNum, "soba:todo:low").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", issueNum, "soba:queued").Return(nil)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockQueueGitHubClient)
			tt.setupMock(mockClient, tt.issue.Number)

			qm := NewQueueManager(mockClient, "owner", "repo")
			qm.SetLogger(logging.NewMockLogger())

			err := qm.updateTodoLabelsToQueued(context.Background(), &tt.issue)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestQueueManager_EnqueueNextIssue_RemovePriorityLabels(t *testing.T) {
	tests := []struct {
		name      string
		issues    []github.Issue
		setupMock func(*MockQueueGitHubClient)
	}{
		{
			name: "soba:todo:highラベルが削除されてsoba:queuedが追加される",
			issues: []github.Issue{
				{
					Number: 1,
					Labels: []github.Label{{Name: "soba:todo:high"}},
				},
			},
			setupMock: func(m *MockQueueGitHubClient) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", 1, "soba:todo:high").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", 1, "soba:queued").Return(nil)
			},
		},
		{
			name: "soba:todo:lowラベルが削除されてsoba:queuedが追加される",
			issues: []github.Issue{
				{
					Number: 2,
					Labels: []github.Label{{Name: "soba:todo:low"}},
				},
			},
			setupMock: func(m *MockQueueGitHubClient) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", 2, "soba:todo:low").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", 2, "soba:queued").Return(nil)
			},
		},
		{
			name: "高優先度Issue（複数ラベル）のすべてのtodo系ラベルが削除される",
			issues: []github.Issue{
				{
					Number: 3,
					Labels: []github.Label{
						{Name: "soba:todo:high"},
						{Name: "soba:todo"},
						{Name: "bug"},
					},
				},
			},
			setupMock: func(m *MockQueueGitHubClient) {
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", 3, "soba:todo").Return(nil)
				m.On("RemoveLabelFromIssue", mock.Anything, "owner", "repo", 3, "soba:todo:high").Return(nil)
				m.On("AddLabelToIssue", mock.Anything, "owner", "repo", 3, "soba:queued").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockQueueGitHubClient)
			tt.setupMock(mockClient)

			qm := NewQueueManager(mockClient, "owner", "repo")
			qm.SetLogger(logging.NewMockLogger())

			err := qm.EnqueueNextIssue(context.Background(), tt.issues)
			assert.NoError(t, err)

			mockClient.AssertExpectations(t)
		})
	}
}

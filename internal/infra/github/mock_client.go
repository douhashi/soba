package github

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockClient はGitHub APIクライアントのモック実装
type MockClient struct {
	mock.Mock
}

// NewMockClient は新しいMockClientを作成する
func NewMockClient() *MockClient {
	return &MockClient{}
}

// ListIssues のモック実装
func (m *MockClient) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]Issue, error) {
	args := m.Called(ctx, owner, repo, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Issue), args.Error(1)
}

// ListOpenIssues のモック実装
func (m *MockClient) ListOpenIssues(ctx context.Context, owner, repo string, opts *ListIssuesOptions) ([]Issue, bool, error) {
	args := m.Called(ctx, owner, repo, opts)
	if args.Get(0) == nil {
		return nil, false, args.Error(2)
	}
	return args.Get(0).([]Issue), args.Bool(1), args.Error(2)
}

// Label関連のモック実装
func (m *MockClient) CreateLabel(ctx context.Context, owner, repo string, label CreateLabelRequest) (*Label, error) {
	args := m.Called(ctx, owner, repo, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Label), args.Error(1)
}

func (m *MockClient) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Label), args.Error(1)
}

func (m *MockClient) GetIssueLabels(ctx context.Context, owner, repo string, issueNumber int) ([]Label, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Label), args.Error(1)
}

func (m *MockClient) UpdateIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []string) error {
	args := m.Called(ctx, owner, repo, issueNumber, labels)
	return args.Error(0)
}

func (m *MockClient) AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockClient) RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

// Pull Request関連のモック実装
func (m *MockClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, bool, error) {
	args := m.Called(ctx, owner, repo, number)
	if args.Get(0) == nil {
		return nil, false, args.Error(2)
	}
	return args.Get(0).(*PullRequest), args.Bool(1), args.Error(2)
}

func (m *MockClient) ListPullRequests(ctx context.Context, owner, repo string, opts *ListPullRequestsOptions) ([]PullRequest, bool, error) {
	args := m.Called(ctx, owner, repo, opts)
	if args.Get(0) == nil {
		return nil, false, args.Error(2)
	}
	return args.Get(0).([]PullRequest), args.Bool(1), args.Error(2)
}

func (m *MockClient) MergePullRequest(ctx context.Context, owner, repo string, number int, mergeReq *MergeRequest) (*MergeResponse, error) {
	args := m.Called(ctx, owner, repo, number, mergeReq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MergeResponse), args.Error(1)
}

// Comment関連のモック実装
func (m *MockClient) CreateComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	args := m.Called(ctx, owner, repo, issueNumber, body)
	return args.Error(0)
}

func (m *MockClient) ListComments(ctx context.Context, owner, repo string, issueNumber int, opts *ListCommentsOptions) ([]IssueComment, error) {
	args := m.Called(ctx, owner, repo, issueNumber, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]IssueComment), args.Error(1)
}

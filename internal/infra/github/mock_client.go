package github

import (
	"context"
	"fmt"

	"github.com/douhashi/soba/internal/infra"
)

// MockClient はGitHub APIクライアントのモック実装
type MockClient struct {
	// ListOpenIssuesのモック設定
	ListOpenIssuesFunc  func(ctx context.Context, owner, repo string, opts *ListIssuesOptions) ([]Issue, bool, error)
	ListOpenIssuesCalls []struct {
		Owner string
		Repo  string
		Opts  *ListIssuesOptions
	}

	// エラーを返すかどうか
	ShouldError bool
	Error       error

	// 返すデータ
	Issues  []Issue
	HasNext bool
}

// NewMockClient は新しいMockClientを作成する
func NewMockClient() *MockClient {
	return &MockClient{
		Issues: []Issue{},
	}
}

// ListOpenIssues のモック実装
func (m *MockClient) ListOpenIssues(ctx context.Context, owner, repo string, opts *ListIssuesOptions) ([]Issue, bool, error) {
	// 呼び出しを記録
	m.ListOpenIssuesCalls = append(m.ListOpenIssuesCalls, struct {
		Owner string
		Repo  string
		Opts  *ListIssuesOptions
	}{
		Owner: owner,
		Repo:  repo,
		Opts:  opts,
	})

	// カスタム関数が設定されている場合はそれを使用
	if m.ListOpenIssuesFunc != nil {
		return m.ListOpenIssuesFunc(ctx, owner, repo, opts)
	}

	// エラーを返す設定の場合
	if m.ShouldError {
		if m.Error != nil {
			return nil, false, m.Error
		}
		return nil, false, infra.NewGitHubAPIError(500, "/repos/"+owner+"/"+repo+"/issues", "mock error")
	}

	// 正常なレスポンスを返す
	return m.Issues, m.HasNext, nil
}

// SetupSuccessResponse はモックの成功レスポンスを設定する
func (m *MockClient) SetupSuccessResponse(issues []Issue, hasNext bool) {
	m.Issues = issues
	m.HasNext = hasNext
	m.ShouldError = false
	m.Error = nil
}

// SetupErrorResponse はモックのエラーレスポンスを設定する
func (m *MockClient) SetupErrorResponse(statusCode int, message string) {
	m.ShouldError = true
	m.Error = infra.NewGitHubAPIError(statusCode, "/repos/test/test/issues", message)
}

// Reset はモックの状態をリセットする
func (m *MockClient) Reset() {
	m.ListOpenIssuesCalls = nil
	m.ListOpenIssuesFunc = nil
	m.ShouldError = false
	m.Error = nil
	m.Issues = []Issue{}
	m.HasNext = false
}

// AssertCalled は指定された呼び出しがされたか確認する
func (m *MockClient) AssertCalled(owner, repo string) error {
	for _, call := range m.ListOpenIssuesCalls {
		if call.Owner == owner && call.Repo == repo {
			return nil
		}
	}
	return fmt.Errorf("ListOpenIssues was not called with owner=%s, repo=%s", owner, repo)
}

// CallCount は呼び出し回数を返す
func (m *MockClient) CallCount() int {
	return len(m.ListOpenIssuesCalls)
}

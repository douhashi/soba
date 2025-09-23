package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logging"
)

// MockGitHubClientForPR はPR Watcher用のモッククライアント
type MockGitHubClientForPR struct {
	prs           []github.PullRequest
	mergeRequests []struct {
		owner  string
		repo   string
		number int
		req    *github.MergeRequest
	}
	mergeError error
}

func (m *MockGitHubClientForPR) ListPullRequests(ctx context.Context, owner, repo string, opts *github.ListPullRequestsOptions) ([]github.PullRequest, bool, error) {
	return m.prs, false, nil
}

func (m *MockGitHubClientForPR) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, bool, error) {
	for _, pr := range m.prs {
		if pr.Number == number {
			return &pr, false, nil
		}
	}
	return nil, false, &github.ErrorResponse{Message: "Not Found"}
}

func (m *MockGitHubClientForPR) MergePullRequest(ctx context.Context, owner, repo string, number int, req *github.MergeRequest) (*github.MergeResponse, error) {
	m.mergeRequests = append(m.mergeRequests, struct {
		owner  string
		repo   string
		number int
		req    *github.MergeRequest
	}{owner, repo, number, req})

	if m.mergeError != nil {
		return nil, m.mergeError
	}

	return &github.MergeResponse{
		SHA:     "abc123",
		Merged:  true,
		Message: "Pull Request successfully merged",
	}, nil
}

// その他のインターフェースメソッドのスタブ実装
func (m *MockGitHubClientForPR) ListOpenIssues(ctx context.Context, owner, repo string, opts *github.ListIssuesOptions) ([]github.Issue, bool, error) {
	return nil, false, nil
}

func (m *MockGitHubClientForPR) AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	return nil
}

func (m *MockGitHubClientForPR) RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	return nil
}

func TestNewPRWatcher(t *testing.T) {
	t.Run("デフォルトの設定でPRWatcherを作成できる", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
			Workflow: config.WorkflowConfig{
				Interval: 30,
			},
		}
		client := &MockGitHubClientForPR{}

		watcher := NewPRWatcher(client, cfg)
		require.NotNil(t, watcher)
		assert.Equal(t, 30*time.Second, watcher.interval)
		assert.NotNil(t, watcher.logger)
	})

	t.Run("interval未設定の場合デフォルト値が使用される", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
			Workflow: config.WorkflowConfig{
				Interval: 0,
			},
		}
		client := &MockGitHubClientForPR{}

		watcher := NewPRWatcher(client, cfg)
		assert.Equal(t, 20*time.Second, watcher.interval)
	})
}

func TestWatchOnce(t *testing.T) {
	t.Run("soba:lgtmラベル付きのPRをマージする", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
			Workflow: config.WorkflowConfig{
				Interval: 1,
			},
		}

		now := time.Now()
		mockClient := &MockGitHubClientForPR{
			prs: []github.PullRequest{
				{
					ID:     1,
					Number: 10,
					Title:  "Test PR with LGTM",
					State:  "open",
					Labels: []github.Label{
						{Name: "soba:lgtm"},
					},
					CreatedAt:      now,
					UpdatedAt:      now,
					Mergeable:      true,
					MergeableState: "clean",
				},
				{
					ID:     2,
					Number: 11,
					Title:  "Test PR without LGTM",
					State:  "open",
					Labels: []github.Label{
						{Name: "other-label"},
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		watcher := NewPRWatcher(mockClient, cfg)
		watcher.SetLogger(logging.NewMockLogger())

		ctx := context.Background()
		err := watcher.watchOnce(ctx)
		require.NoError(t, err)

		// PR #10がマージされたことを確認
		assert.Len(t, mockClient.mergeRequests, 1)
		assert.Equal(t, "owner", mockClient.mergeRequests[0].owner)
		assert.Equal(t, "repo", mockClient.mergeRequests[0].repo)
		assert.Equal(t, 10, mockClient.mergeRequests[0].number)
		assert.Equal(t, "squash", mockClient.mergeRequests[0].req.MergeMethod)
	})

	t.Run("複数のsoba:lgtm付きPRがある場合は全てマージする", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
		}

		now := time.Now()
		mockClient := &MockGitHubClientForPR{
			prs: []github.PullRequest{
				{
					ID:     1,
					Number: 10,
					Title:  "First PR with LGTM",
					State:  "open",
					Labels: []github.Label{
						{Name: "soba:lgtm"},
					},
					CreatedAt:      now,
					UpdatedAt:      now,
					Mergeable:      true,
					MergeableState: "clean",
				},
				{
					ID:     2,
					Number: 11,
					Title:  "Second PR with LGTM",
					State:  "open",
					Labels: []github.Label{
						{Name: "soba:lgtm"},
					},
					CreatedAt:      now,
					UpdatedAt:      now,
					Mergeable:      true,
					MergeableState: "clean",
				},
			},
		}

		watcher := NewPRWatcher(mockClient, cfg)
		watcher.SetLogger(logging.NewMockLogger())

		ctx := context.Background()
		err := watcher.watchOnce(ctx)
		require.NoError(t, err)

		// 両方のPRがマージされたことを確認
		assert.Len(t, mockClient.mergeRequests, 2)
		numbers := []int{
			mockClient.mergeRequests[0].number,
			mockClient.mergeRequests[1].number,
		}
		assert.Contains(t, numbers, 10)
		assert.Contains(t, numbers, 11)
	})

	t.Run("マージできない状態のPRはスキップする", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
		}

		now := time.Now()
		mockClient := &MockGitHubClientForPR{
			prs: []github.PullRequest{
				{
					ID:     1,
					Number: 10,
					Title:  "PR with merge conflict",
					State:  "open",
					Labels: []github.Label{
						{Name: "soba:lgtm"},
					},
					CreatedAt:      now,
					UpdatedAt:      now,
					Mergeable:      false,
					MergeableState: "dirty", // マージ競合
				},
			},
		}

		watcher := NewPRWatcher(mockClient, cfg)
		watcher.SetLogger(logging.NewMockLogger())

		ctx := context.Background()
		err := watcher.watchOnce(ctx)
		require.NoError(t, err)

		// マージされていないことを確認
		assert.Len(t, mockClient.mergeRequests, 0)
	})

	t.Run("マージエラー時もサービスは継続する", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
		}

		now := time.Now()
		mockClient := &MockGitHubClientForPR{
			prs: []github.PullRequest{
				{
					ID:     1,
					Number: 10,
					Title:  "PR with LGTM",
					State:  "open",
					Labels: []github.Label{
						{Name: "soba:lgtm"},
					},
					CreatedAt:      now,
					UpdatedAt:      now,
					Mergeable:      true,
					MergeableState: "clean",
				},
			},
			mergeError: &github.ErrorResponse{Message: "Merge failed"},
		}

		watcher := NewPRWatcher(mockClient, cfg)
		watcher.SetLogger(logging.NewMockLogger())

		ctx := context.Background()
		err := watcher.watchOnce(ctx)
		// エラーは返さない（ログ出力のみ）
		require.NoError(t, err)
	})
}

func TestParseRepository(t *testing.T) {
	t.Run("正常なリポジトリ形式をパースできる", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
		}
		watcher := NewPRWatcher(&MockGitHubClientForPR{}, cfg)

		owner, repo := watcher.parseRepository()
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("不正な形式の場合空文字を返す", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "invalid-format",
			},
		}
		watcher := NewPRWatcher(&MockGitHubClientForPR{}, cfg)
		watcher.SetLogger(logging.NewMockLogger())

		owner, repo := watcher.parseRepository()
		assert.Equal(t, "", owner)
		assert.Equal(t, "", repo)
	})
}

func TestPRWatcher_WatchCycleLogs(t *testing.T) {
	t.Run("watchOnce開始時と完了時にINFOログが出力される", func(t *testing.T) {
		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "owner/repo",
			},
			Workflow: config.WorkflowConfig{
				Interval: 1,
			},
		}

		mockClient := &MockGitHubClientForPR{
			prs: []github.PullRequest{
				{
					ID:     1,
					Number: 10,
					Title:  "Test PR",
					State:  "open",
					Labels: []github.Label{
						{Name: "other-label"},
					},
				},
			},
		}

		watcher := NewPRWatcher(mockClient, cfg)
		mockLogger := logging.NewMockLogger()
		watcher.SetLogger(mockLogger)

		ctx := context.Background()
		err := watcher.watchOnce(ctx)
		require.NoError(t, err)

		// "Starting PR watch cycle" がDEBUGからINFOに変更されたことを確認
		foundStartLog := false
		for _, msg := range mockLogger.Messages {
			if msg.Message == "Starting PR watch cycle" && msg.Level == "INFO" {
				foundStartLog = true
				break
			}
		}
		assert.True(t, foundStartLog, "expected 'Starting PR watch cycle' INFO log")

		// "PR watch cycle completed" INFOログが追加されたことを確認
		foundCompleteLog := false
		for _, msg := range mockLogger.Messages {
			if msg.Message == "PR watch cycle completed" && msg.Level == "INFO" {
				foundCompleteLog = true
				break
			}
		}
		assert.True(t, foundCompleteLog, "expected 'PR watch cycle completed' INFO log")
	})
}

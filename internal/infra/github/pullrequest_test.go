package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/douhashi/soba/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のmockTokenProviderヘルパー（token_provider_test.goから利用）
func newMockTokenProvider(token string) TokenProvider {
	return &mockTokenProvider{token: token}
}

func TestListPullRequests(t *testing.T) {
	t.Run("正常にPR一覧を取得できる", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			// クエリパラメータのチェック
			query := r.URL.Query()
			assert.Equal(t, "open", query.Get("state"))
			assert.Equal(t, "1", query.Get("page"))
			assert.Equal(t, "100", query.Get("per_page"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]PullRequest{
				{
					ID:     1,
					Number: 10,
					Title:  "Test PR",
					State:  "open",
					Labels: []Label{
						{Name: "soba:lgtm"},
					},
					MergeableState: "clean",
				},
			})
		}))
		defer server.Close()

		client := &ClientImpl{
			httpClient:    http.DefaultClient,
			tokenProvider: newMockTokenProvider("test-token"),
			baseURL:       server.URL,
			logger:        logger.NewNopLogger(),
		}

		opts := &ListPullRequestsOptions{
			State:   "open",
			Page:    1,
			PerPage: 100,
		}

		prs, _, err := client.ListPullRequests(context.Background(), "owner", "repo", opts)
		require.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, 10, prs[0].Number)
		assert.Equal(t, "Test PR", prs[0].Title)
	})

	t.Run("APIエラー時にエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Message: "Not Found",
			})
		}))
		defer server.Close()

		client := &ClientImpl{
			httpClient:    http.DefaultClient,
			tokenProvider: newMockTokenProvider("test-token"),
			baseURL:       server.URL,
			logger:        logger.NewNopLogger(),
		}

		_, _, err := client.ListPullRequests(context.Background(), "owner", "repo", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Not Found")
	})
}

func TestMergePullRequest(t *testing.T) {
	t.Run("正常にPRをマージできる", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/pulls/10/merge", r.URL.Path)
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			var req MergeRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "Merge PR #10", req.CommitTitle)
			assert.Equal(t, "squash", req.MergeMethod)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MergeResponse{
				SHA:     "abc123",
				Merged:  true,
				Message: "Pull Request successfully merged",
			})
		}))
		defer server.Close()

		client := &ClientImpl{
			httpClient:    http.DefaultClient,
			tokenProvider: newMockTokenProvider("test-token"),
			baseURL:       server.URL,
			logger:        logger.NewNopLogger(),
		}

		req := &MergeRequest{
			CommitTitle: "Merge PR #10",
			MergeMethod: "squash",
		}

		resp, err := client.MergePullRequest(context.Background(), "owner", "repo", 10, req)
		require.NoError(t, err)
		assert.True(t, resp.Merged)
		assert.Equal(t, "Pull Request successfully merged", resp.Message)
	})

	t.Run("マージ競合時にエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Message: "Merge conflict",
			})
		}))
		defer server.Close()

		client := &ClientImpl{
			httpClient:    http.DefaultClient,
			tokenProvider: newMockTokenProvider("test-token"),
			baseURL:       server.URL,
			logger:        logger.NewNopLogger(),
		}

		_, err := client.MergePullRequest(context.Background(), "owner", "repo", 10, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Merge conflict")
	})
}

func TestGetPullRequest(t *testing.T) {
	t.Run("正常にPR詳細を取得できる", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/pulls/10", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			now := time.Now()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(PullRequest{
				ID:        1,
				Number:    10,
				Title:     "Test PR",
				Body:      "This is a test PR",
				State:     "open",
				HTMLURL:   "https://github.com/owner/repo/pull/10",
				CreatedAt: now,
				UpdatedAt: now,
				Labels: []Label{
					{Name: "soba:lgtm"},
				},
				Mergeable:      true,
				MergeableState: "clean",
			})
		}))
		defer server.Close()

		client := &ClientImpl{
			httpClient:    http.DefaultClient,
			tokenProvider: newMockTokenProvider("test-token"),
			baseURL:       server.URL,
			logger:        logger.NewNopLogger(),
		}

		pr, _, err := client.GetPullRequest(context.Background(), "owner", "repo", 10)
		require.NoError(t, err)
		assert.Equal(t, 10, pr.Number)
		assert.Equal(t, "Test PR", pr.Title)
		assert.True(t, pr.Mergeable)
		assert.Equal(t, "clean", pr.MergeableState)
	})
}

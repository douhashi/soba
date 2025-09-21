package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("ListOpenIssues", func(t *testing.T) {
		t.Run("returns list of open issues", func(t *testing.T) {
			expectedIssues := []Issue{
				{
					Number:  1,
					Title:   "First Issue",
					State:   "open",
					HTMLURL: "https://github.com/owner/repo/issues/1",
				},
				{
					Number:  2,
					Title:   "Second Issue",
					State:   "open",
					HTMLURL: "https://github.com/owner/repo/issues/2",
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// URLパスとクエリパラメータの検証
				assert.Equal(t, "/repos/owner/repo/issues", r.URL.Path)
				assert.Equal(t, "open", r.URL.Query().Get("state"))
				assert.Equal(t, "1", r.URL.Query().Get("page"))
				assert.Equal(t, "30", r.URL.Query().Get("per_page"))

				// レスポンスヘッダーの設定（ページネーション情報）
				w.Header().Set("Link", `<https://api.github.com/repos/owner/repo/issues?page=2>; rel="next"`)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(expectedIssues)
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			issues, hasNext, err := client.ListOpenIssues(ctx, "owner", "repo", nil)
			require.NoError(t, err)
			assert.Equal(t, expectedIssues, issues)
			assert.True(t, hasNext)
		})

		t.Run("handles options correctly", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// オプションが正しくクエリパラメータに変換されているか確認
				assert.Equal(t, "/repos/owner/repo/issues", r.URL.Path)

				query := r.URL.Query()
				assert.Equal(t, "all", query.Get("state"))
				assert.Equal(t, "bug,enhancement", query.Get("labels"))
				assert.Equal(t, "updated", query.Get("sort"))
				assert.Equal(t, "desc", query.Get("direction"))
				assert.Equal(t, "2", query.Get("page"))
				assert.Equal(t, "50", query.Get("per_page"))

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]Issue{})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			opts := &ListIssuesOptions{
				State:     "all",
				Labels:    []string{"bug", "enhancement"},
				Sort:      "updated",
				Direction: "desc",
				Page:      2,
				PerPage:   50,
			}

			_, _, err = client.ListOpenIssues(ctx, "owner", "repo", opts)
			require.NoError(t, err)
		})

		t.Run("handles empty response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]Issue{})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			issues, hasNext, err := client.ListOpenIssues(ctx, "owner", "repo", nil)
			require.NoError(t, err)
			assert.Empty(t, issues)
			assert.False(t, hasNext)
		})

		t.Run("handles API error", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "Not Found",
				})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			issues, _, err := client.ListOpenIssues(ctx, "owner", "repo", nil)
			assert.Error(t, err)
			assert.Nil(t, issues)
			assert.Contains(t, err.Error(), "Not Found")
		})

		t.Run("validates required parameters", func(t *testing.T) {
			tokenProvider := &mockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, nil)
			require.NoError(t, err)

			// オーナーが空の場合
			issues, _, err := client.ListOpenIssues(ctx, "", "repo", nil)
			assert.Error(t, err)
			assert.Nil(t, issues)
			assert.Contains(t, err.Error(), "owner is required")

			// レポジトリが空の場合
			issues, _, err = client.ListOpenIssues(ctx, "owner", "", nil)
			assert.Error(t, err)
			assert.Nil(t, issues)
			assert.Contains(t, err.Error(), "repo is required")
		})
	})

	t.Run("parseLinkHeader", func(t *testing.T) {
		t.Run("parses link header with next", func(t *testing.T) {
			link := `<https://api.github.com/repos/owner/repo/issues?page=2>; rel="next", <https://api.github.com/repos/owner/repo/issues?page=10>; rel="last"`
			hasNext := parseLinkHeader(link)
			assert.True(t, hasNext)
		})

		t.Run("parses link header without next", func(t *testing.T) {
			link := `<https://api.github.com/repos/owner/repo/issues?page=1>; rel="prev", <https://api.github.com/repos/owner/repo/issues?page=10>; rel="last"`
			hasNext := parseLinkHeader(link)
			assert.False(t, hasNext)
		})

		t.Run("handles empty link header", func(t *testing.T) {
			hasNext := parseLinkHeader("")
			assert.False(t, hasNext)
		})
	})

	t.Run("buildIssuesURL", func(t *testing.T) {
		client, err := NewClient(&mockTokenProvider{token: "test"}, nil)
		require.NoError(t, err)

		t.Run("builds basic URL", func(t *testing.T) {
			url := client.buildIssuesURL("owner", "repo", nil)
			assert.Equal(t, "https://api.github.com/repos/owner/repo/issues?page=1&per_page=30&state=open", url)
		})

		t.Run("builds URL with options", func(t *testing.T) {
			opts := &ListIssuesOptions{
				State:     "closed",
				Labels:    []string{"bug", "help wanted"},
				Sort:      "created",
				Direction: "asc",
				Page:      2,
				PerPage:   50,
			}

			urlStr := client.buildIssuesURL("owner", "repo", opts)
			parsedURL, err := url.Parse(urlStr)
			require.NoError(t, err)

			query := parsedURL.Query()
			assert.Equal(t, "closed", query.Get("state"))
			assert.Equal(t, "bug,help wanted", query.Get("labels"))
			assert.Equal(t, "created", query.Get("sort"))
			assert.Equal(t, "asc", query.Get("direction"))
			assert.Equal(t, "2", query.Get("page"))
			assert.Equal(t, "50", query.Get("per_page"))
		})

		t.Run("handles Since parameter", func(t *testing.T) {
			since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			opts := &ListIssuesOptions{
				Since: &since,
			}

			urlStr := client.buildIssuesURL("owner", "repo", opts)
			parsedURL, err := url.Parse(urlStr)
			require.NoError(t, err)

			query := parsedURL.Query()
			assert.Equal(t, "2024-01-01T00:00:00Z", query.Get("since"))
		})
	})
}
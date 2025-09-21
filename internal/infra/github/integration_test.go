package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	// 実APIテストのスキップフラグ
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping integration test: GITHUB_TOKEN not set")
	}

	ctx := context.Background()

	t.Run("Real API Integration", func(t *testing.T) {
		t.Run("fetches issues from real repository", func(t *testing.T) {
			// 環境変数からトークンを取得
			tokenProvider := NewEnvTokenProvider("GITHUB_TOKEN")

			// クライアントを作成
			client, err := NewClient(tokenProvider, nil)
			require.NoError(t, err)

			// GitHub公開リポジトリから Issue を取得
			issues, hasNext, err := client.ListOpenIssues(ctx, "golang", "go", &ListIssuesOptions{
				PerPage: 5,
			})

			// エラーチェック
			require.NoError(t, err)

			// 結果の検証
			assert.NotNil(t, issues)
			assert.GreaterOrEqual(t, len(issues), 0) // 0件以上
			assert.NotNil(t, hasNext)

			// 各 Issue の基本的なフィールドをチェック
			for _, issue := range issues {
				assert.NotZero(t, issue.ID)
				assert.NotZero(t, issue.Number)
				assert.NotEmpty(t, issue.Title)
				assert.NotEmpty(t, issue.State)
				assert.NotEmpty(t, issue.URL)
				assert.NotEmpty(t, issue.HTMLURL)
			}
		})
	})
}

func TestEndToEnd(t *testing.T) {
	ctx := context.Background()

	t.Run("Complete workflow with mock server", func(t *testing.T) {
		// テストデータを取得
		testIssues, err := GetTestIssues()
		require.NoError(t, err)

		// モックサーバーの作成
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			// 認証ヘッダーのチェック
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "Bad credentials",
				})
				return
			}

			// パスのチェック
			if r.URL.Path != "/repos/test/repo/issues" {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "Not Found",
				})
				return
			}

			// ページングのシミュレーション
			page := r.URL.Query().Get("page")
			switch page {
			case "1":
				w.Header().Set("Link", `<https://api.github.com/repos/test/repo/issues?page=2>; rel="next"`)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(testIssues[:2])
			case "2":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(testIssues[2:])
			default:
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]Issue{})
			}
		}))
		defer server.Close()

		// トークンプロバイダーの設定
		tokenProvider := &mockTokenProvider{
			token: "test-token",
		}

		// クライアントの作成
		client, err := NewClient(tokenProvider, &ClientOptions{
			BaseURL: server.URL,
		})
		require.NoError(t, err)

		// 1ページ目を取得
		issues1, hasNext1, err := client.ListOpenIssues(ctx, "test", "repo", &ListIssuesOptions{
			Page:    1,
			PerPage: 2,
		})
		require.NoError(t, err)
		assert.Len(t, issues1, 2)
		assert.True(t, hasNext1)

		// 2ページ目を取得
		issues2, hasNext2, err := client.ListOpenIssues(ctx, "test", "repo", &ListIssuesOptions{
			Page:    2,
			PerPage: 2,
		})
		require.NoError(t, err)
		assert.Len(t, issues2, 1)
		assert.False(t, hasNext2)

		// リクエスト回数の確認
		assert.Equal(t, 2, requestCount)
	})

	t.Run("Error handling and retry", func(t *testing.T) {
		// エラー回数をカウント
		errorCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errorCount++
			if errorCount < 3 {
				// 最初の2回は500エラー
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "Internal Server Error",
				})
			} else {
				// 3回目で成功
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]Issue{
					{
						Number: 1,
						Title:  "Success after retry",
						State:  "open",
					},
				})
			}
		}))
		defer server.Close()

		tokenProvider := &mockTokenProvider{
			token: "test-token",
		}

		// リトライ機能付きクライアントの作成
		client, err := NewClient(tokenProvider, &ClientOptions{
			BaseURL: server.URL,
		})
		require.NoError(t, err)

		// リトライクライアントでラップ
		retryClient := NewRetryableClient(&RetryOptions{
			MaxRetries:  3,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
		})

		var issues []Issue
		var hasNext bool

		// リトライ付きでリクエストを実行
		_, err = retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
			req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/repos/test/repo/issues", nil)
			resp, err := client.doRequest(ctx, req)
			if err != nil {
				return nil, err
			}

			if resp.StatusCode == http.StatusOK {
				var tmpIssues []Issue
				json.NewDecoder(resp.Body).Decode(&tmpIssues)
				issues = tmpIssues
				hasNext = false
				resp.Body.Close()
			}
			return resp, nil
		})

		require.NoError(t, err)
		assert.Len(t, issues, 1)
		assert.Equal(t, "Success after retry", issues[0].Title)
		assert.False(t, hasNext)
		assert.Equal(t, 3, errorCount) // 2回失敗 + 1回成功
	})

	t.Run("Authentication flow", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 認証ヘッダーを確認
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "Requires authentication",
				})
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]Issue{})
		}))
		defer server.Close()

		t.Run("with valid token", func(t *testing.T) {
			tokenProvider := &mockTokenProvider{
				token: "valid-token",
			}

			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			issues, _, err := client.ListOpenIssues(ctx, "test", "repo", nil)
			require.NoError(t, err)
			assert.Empty(t, issues)
		})

		t.Run("with token provider error", func(t *testing.T) {
			tokenProvider := &mockTokenProvider{
				token: "",
				err:   assert.AnError,
			}

			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			issues, _, err := client.ListOpenIssues(ctx, "test", "repo", nil)
			assert.Error(t, err)
			assert.Nil(t, issues)
			assert.Contains(t, err.Error(), "failed to get token")
		})
	})
}
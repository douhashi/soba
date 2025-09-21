package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	ctx := context.Background()

	t.Run("NewClient", func(t *testing.T) {
		t.Run("creates client with token provider", func(t *testing.T) {
			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}

			client, err := NewClient(tokenProvider, nil)
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tokenProvider, client.tokenProvider)
			assert.Equal(t, defaultBaseURL, client.baseURL)
		})

		t.Run("creates client with custom options", func(t *testing.T) {
			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}

			opts := &ClientOptions{
				BaseURL: "https://github.enterprise.com/api/v3",
				Timeout: 30 * time.Second,
			}

			client, err := NewClient(tokenProvider, opts)
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, opts.BaseURL, client.baseURL)
		})

		t.Run("returns error when token provider is nil", func(t *testing.T) {
			client, err := NewClient(nil, nil)
			assert.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "token provider is required")
		})
	})

	t.Run("doRequest", func(t *testing.T) {
		t.Run("sends request with authentication header", func(t *testing.T) {
			// テストサーバーを作成
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 認証ヘッダーをチェック
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				assert.Equal(t, "/repos/test/repo/issues", r.URL.Path)

				// レスポンスを返す
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]Issue{
					{
						Number: 1,
						Title:  "Test Issue",
						State:  "open",
					},
				})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}

			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/repos/test/repo/issues", nil)
			require.NoError(t, err)

			resp, err := client.doRequest(ctx, req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("handles authentication failure", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "Bad credentials",
				})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{
				token: "invalid-token",
			}

			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/repos/test/repo/issues", nil)
			require.NoError(t, err)

			resp, err := client.doRequest(ctx, req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})

		t.Run("handles token provider error", func(t *testing.T) {
			tokenProvider := &mockTokenProvider{
				token: "",
				err:   assert.AnError,
			}

			client, err := NewClient(tokenProvider, nil)
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "GET", defaultBaseURL+"/repos/test/repo/issues", nil)
			require.NoError(t, err)

			_, err = client.doRequest(ctx, req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to get token")
		})
	})

	t.Run("parseErrorResponse", func(t *testing.T) {
		t.Run("parses GitHub error response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message:          "Not Found",
					DocumentationURL: "https://docs.github.com/rest",
				})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}

			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/repos/test/repo/issues", nil)
			require.NoError(t, err)

			resp, err := client.doRequest(ctx, req)
			require.NoError(t, err)
			defer resp.Body.Close()

			err = client.parseErrorResponse(resp)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Not Found")
			assert.Contains(t, err.Error(), "404")
		})

		t.Run("handles rate limit response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-RateLimit-Limit", "60")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", "1234567890")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(ErrorResponse{
					Message: "API rate limit exceeded",
				})
			}))
			defer server.Close()

			tokenProvider := &mockTokenProvider{
				token: "test-token",
			}

			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
			})
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/repos/test/repo/issues", nil)
			require.NoError(t, err)

			resp, err := client.doRequest(ctx, req)
			require.NoError(t, err)
			defer resp.Body.Close()

			err = client.parseErrorResponse(resp)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "rate limit")
		})
	})
}

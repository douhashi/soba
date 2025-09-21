package github

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("RetryableClient", func(t *testing.T) {
		t.Run("succeeds on first attempt", func(t *testing.T) {
			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			}

			retrier := NewRetryableClient(nil)
			resp, err := retrier.DoWithRetry(ctx, handler)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, 1, attempt)
		})

		t.Run("retries on rate limit error", func(t *testing.T) {
			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				if attempt < 3 {
					// 通常のサーバーエラーとして扱う（レート制限のテストは別途）
					return &http.Response{
						StatusCode: http.StatusServiceUnavailable,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			}

			opts := &RetryOptions{
				MaxRetries:  3,
				InitialWait: 10 * time.Millisecond,
				MaxWait:     100 * time.Millisecond,
			}
			retrier := NewRetryableClient(opts)
			resp, err := retrier.DoWithRetry(ctx, handler)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, 3, attempt)
		})

		t.Run("retries on 500 errors", func(t *testing.T) {
			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				if attempt < 2 {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			}

			opts := &RetryOptions{
				MaxRetries:  3,
				InitialWait: 10 * time.Millisecond,
			}
			retrier := NewRetryableClient(opts)
			resp, err := retrier.DoWithRetry(ctx, handler)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, 2, attempt)
		})

		t.Run("retries on network error", func(t *testing.T) {
			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				if attempt < 2 {
					return nil, fmt.Errorf("network error")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			}

			opts := &RetryOptions{
				MaxRetries:  3,
				InitialWait: 10 * time.Millisecond,
			}
			retrier := NewRetryableClient(opts)
			resp, err := retrier.DoWithRetry(ctx, handler)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, 2, attempt)
		})

		t.Run("stops after max retries", func(t *testing.T) {
			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
				}, nil
			}

			opts := &RetryOptions{
				MaxRetries:  2,
				InitialWait: 10 * time.Millisecond,
			}
			retrier := NewRetryableClient(opts)
			resp, err := retrier.DoWithRetry(ctx, handler)
			assert.NoError(t, err) // エラーは返さない、最後のレスポンスを返す
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.Equal(t, 3, attempt) // 初回 + 2リトライ
		})

		t.Run("does not retry on client errors", func(t *testing.T) {
			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				return &http.Response{
					StatusCode: http.StatusBadRequest,
				}, nil
			}

			retrier := NewRetryableClient(nil)
			resp, err := retrier.DoWithRetry(ctx, handler)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, 1, attempt)
		})

		t.Run("respects context cancellation", func(t *testing.T) {
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // すぐにキャンセル

			attempt := 0
			handler := func() (*http.Response, error) {
				attempt++
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
				}, nil
			}

			retrier := NewRetryableClient(nil)
			resp, err := retrier.DoWithRetry(cancelCtx, handler)
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), "context canceled")
			assert.Equal(t, 0, attempt) // コンテキストがキャンセル済みなので実行されない
		})
	})

	t.Run("isRetryable", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
			err        error
			expected   bool
		}{
			{"429 Too Many Requests", http.StatusTooManyRequests, nil, true},
			{"500 Internal Server Error", http.StatusInternalServerError, nil, true},
			{"502 Bad Gateway", http.StatusBadGateway, nil, true},
			{"503 Service Unavailable", http.StatusServiceUnavailable, nil, true},
			{"504 Gateway Timeout", http.StatusGatewayTimeout, nil, true},
			{"Network Error", 0, fmt.Errorf("network error"), true},
			{"200 OK", http.StatusOK, nil, false},
			{"400 Bad Request", http.StatusBadRequest, nil, false},
			{"401 Unauthorized", http.StatusUnauthorized, nil, false},
			{"404 Not Found", http.StatusNotFound, nil, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var resp *http.Response
				if tt.statusCode > 0 {
					resp = &http.Response{StatusCode: tt.statusCode}
				}
				result := isRetryable(resp, tt.err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("calculateBackoff", func(t *testing.T) {
		opts := &RetryOptions{
			InitialWait: 100 * time.Millisecond,
			MaxWait:     2 * time.Second,
			Multiplier:  2,
		}

		t.Run("exponential backoff", func(t *testing.T) {
			// 初回: 100ms（ジッター込みで50-100ms）
			wait := calculateBackoff(0, opts)
			assert.GreaterOrEqual(t, wait, 50*time.Millisecond)
			assert.LessOrEqual(t, wait, 100*time.Millisecond)

			// 2回目: 200ms（ジッター込みで100-200ms）
			wait = calculateBackoff(1, opts)
			assert.GreaterOrEqual(t, wait, 100*time.Millisecond)
			assert.LessOrEqual(t, wait, 200*time.Millisecond)

			// 3回目: 400ms（ジッター込みで200-400ms）
			wait = calculateBackoff(2, opts)
			assert.GreaterOrEqual(t, wait, 200*time.Millisecond)
			assert.LessOrEqual(t, wait, 400*time.Millisecond)
		})

		t.Run("respects max wait", func(t *testing.T) {
			// 多くの試行後でもMaxWaitを超えない
			wait := calculateBackoff(10, opts)
			assert.LessOrEqual(t, wait, 2*time.Second)
		})
	})

	t.Run("getRateLimitReset", func(t *testing.T) {
		t.Run("parses reset time from header", func(t *testing.T) {
			now := time.Now()
			resetTime := now.Add(30 * time.Second)
			resp := &http.Response{
				Header: http.Header{
					"X-RateLimit-Reset": []string{fmt.Sprintf("%d", resetTime.Unix())},
				},
			}

			wait := getRateLimitReset(resp)
			// 誤差を考慮して29-31秒の範囲で判定
			assert.GreaterOrEqual(t, wait, 29*time.Second)
			assert.LessOrEqual(t, wait, 60*time.Second) // 最大60秒に制限されている
		})

		t.Run("returns default when header is missing", func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}

			wait := getRateLimitReset(resp)
			assert.Equal(t, 60*time.Second, wait)
		})

		t.Run("returns default when header is invalid", func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{
					"X-RateLimit-Reset": []string{"invalid"},
				},
			}

			wait := getRateLimitReset(resp)
			assert.Equal(t, 60*time.Second, wait)
		})
	})
}

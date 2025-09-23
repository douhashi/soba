package github

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/douhashi/soba/internal/infra"
	"github.com/douhashi/soba/pkg/logging"
)

// RetryOptions はリトライの設定
type RetryOptions struct {
	MaxRetries  int           // 最大リトライ回数
	InitialWait time.Duration // 初回待機時間
	MaxWait     time.Duration // 最大待機時間
	Multiplier  float64       // 待機時間の倍率
	Logger      logging.Logger // ロガー
}

// デフォルトのリトライ設定
var defaultRetryOptions = &RetryOptions{
	MaxRetries:  3,
	InitialWait: 1 * time.Second,
	MaxWait:     30 * time.Second,
	Multiplier:  2.0,
}

// RetryableClient はリトライ機能を持つHTTPクライアント
type RetryableClient struct {
	options *RetryOptions
	logger  logging.Logger
}

// NewRetryableClient は新しいRetryableClientを作成する
func NewRetryableClient(opts *RetryOptions) *RetryableClient {
	if opts == nil || opts.Logger == nil {
		panic("RetryOptions with Logger is required")
	}

	if opts.Multiplier == 0 {
		opts.Multiplier = 2.0
	}
	if opts.InitialWait == 0 {
		opts.InitialWait = 1 * time.Second
	}
	if opts.MaxWait == 0 {
		opts.MaxWait = 30 * time.Second
	}

	return &RetryableClient{
		options: opts,
		logger:  opts.Logger,
	}
}

// DoWithRetry はリトライ付きでHTTPリクエストを実行する
func (r *RetryableClient) DoWithRetry(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= r.options.MaxRetries; attempt++ {
		// コンテキストのキャンセルチェック
		select {
		case <-ctx.Done():
			return nil, infra.WrapInfraError(ctx.Err(), "context canceled during retry")
		default:
		}

		// リクエスト実行
		resp, err := fn()
		lastResp = resp
		lastErr = err

		// 成功またはリトライ不可能な場合は終了
		if err == nil && resp != nil && !isRetryable(resp, err) {
			return resp, nil
		}

		// 最大リトライ回数に達した場合
		if attempt >= r.options.MaxRetries {
			r.logger.Debug(ctx, "Max retries reached",
				logging.Field{Key: "attempt", Value: attempt + 1},
				logging.Field{Key: "max_retries", Value: r.options.MaxRetries},
			)
			break
		}

		// 待機時間の計算
		var waitTime time.Duration
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			// レート制限の場合は Reset 時刻まで待機
			waitTime = getRateLimitReset(resp)
			r.logger.Info(ctx, "Rate limited, waiting until reset",
				logging.Field{Key: "wait_seconds", Value: waitTime.Seconds()},
				logging.Field{Key: "attempt", Value: attempt + 1},
			)
		} else {
			// 指数バックオフ
			waitTime = calculateBackoff(attempt, r.options)
			r.logger.Debug(ctx, "Retrying after backoff",
				logging.Field{Key: "wait_seconds", Value: waitTime.Seconds()},
				logging.Field{Key: "attempt", Value: attempt + 1},
				logging.Field{Key: "status_code", Value: getStatusCode(resp)},
			)
		}

		// 待機
		select {
		case <-time.After(waitTime):
			// 待機完了
		case <-ctx.Done():
			return nil, infra.WrapInfraError(ctx.Err(), "context canceled during backoff")
		}
	}

	// 最後のレスポンス/エラーを返す
	if lastErr != nil {
		return nil, lastErr
	}
	return lastResp, nil
}

// isRetryable はリトライ可能なエラーかどうか判定する
func isRetryable(resp *http.Response, err error) bool {
	// ネットワークエラーの場合はリトライ
	if err != nil {
		return true
	}

	// レスポンスがない場合はリトライしない
	if resp == nil {
		return false
	}

	// ステータスコードで判定
	switch resp.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// calculateBackoff は指数バックオフの待機時間を計算する
func calculateBackoff(attempt int, opts *RetryOptions) time.Duration {
	// 指数バックオフの計算
	wait := float64(opts.InitialWait) * math.Pow(opts.Multiplier, float64(attempt))

	// 最大待機時間で制限
	if wait > float64(opts.MaxWait) {
		wait = float64(opts.MaxWait)
	}

	// ジッターの追加 (0.5〜1.0倍のランダム)
	jitter := 0.5 + rand.Float64()*0.5
	wait = wait * jitter

	return time.Duration(wait)
}

// getRateLimitReset はレート制限のリセット時刻までの待機時間を取得する
func getRateLimitReset(resp *http.Response) time.Duration {
	if resp == nil || resp.Header == nil {
		return 60 * time.Second // デフォルト値
	}

	resetStr := resp.Header.Get("X-RateLimit-Reset")
	if resetStr == "" {
		return 60 * time.Second
	}

	resetTime, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil {
		return 60 * time.Second
	}

	// リセット時刻までの待機時間を計算
	wait := time.Until(time.Unix(resetTime, 0))
	if wait <= 0 {
		return 100 * time.Millisecond // 過去の時刻の場合は短い待機時間
	}

	// 最大待機時間を制限（テストのため）
	if wait > 60*time.Second {
		return 60 * time.Second
	}

	return wait
}

// getStatusCode はレスポンスからステータスコードを安全に取得する
func getStatusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

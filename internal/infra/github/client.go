package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/douhashi/soba/internal/infra"
	"github.com/douhashi/soba/pkg/logging"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultTimeout = 30 * time.Second
)

// ClientImpl はGitHub APIクライアントの実装
type ClientImpl struct {
	httpClient    *http.Client
	tokenProvider TokenProvider
	baseURL       string
	logger        logging.Logger
}

// ClientOptions はクライアントのオプション
type ClientOptions struct {
	BaseURL string         // GitHub Enterprise用のカスタムURL
	Timeout time.Duration  // HTTPクライアントのタイムアウト
	Logger  logging.Logger // ロガー (必須)
}

// NewClient は新しいGitHub APIクライアントを作成する
func NewClient(tokenProvider TokenProvider, opts *ClientOptions) (*ClientImpl, error) {
	if tokenProvider == nil {
		return nil, infra.NewGitHubAPIError(0, "", "token provider is required")
	}

	if opts == nil || opts.Logger == nil {
		return nil, infra.NewGitHubAPIError(0, "", "logger is required")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &ClientImpl{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		tokenProvider: tokenProvider,
		baseURL:       baseURL,
		logger:        opts.Logger,
	}, nil
}

// doRequest は認証付きHTTPリクエストを実行する
func (c *ClientImpl) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// トークンを取得
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to get token")
	}

	// ヘッダーを設定
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// リクエスト情報をログ出力
	c.logger.Debug(ctx, "GitHub API request",
		logging.Field{Key: "method", Value: req.Method},
		logging.Field{Key: "url", Value: req.URL.String()},
	)

	// リクエスト実行
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to execute HTTP request")
	}

	// レスポンス情報をログ出力
	c.logger.Debug(ctx, "GitHub API response",
		logging.Field{Key: "status", Value: resp.StatusCode},
		logging.Field{Key: "url", Value: req.URL.String()},
	)

	return resp, nil
}

// parseErrorResponse はエラーレスポンスを解析する
func (c *ClientImpl) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return infra.NewGitHubAPIError(resp.StatusCode, resp.Request.URL.String(), "failed to read error response")
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// JSONパースに失敗した場合は生のボディを使用
		return infra.NewGitHubAPIError(resp.StatusCode, resp.Request.URL.String(), string(body))
	}

	// レート制限エラーの特別処理
	if resp.StatusCode == http.StatusTooManyRequests {
		resetTime := resp.Header.Get("X-RateLimit-Reset")
		return infra.NewGitHubAPIError(
			resp.StatusCode,
			resp.Request.URL.String(),
			fmt.Sprintf("API rate limit exceeded. Reset at: %s. Message: %s", resetTime, errResp.Message),
		)
	}

	return infra.NewGitHubAPIError(resp.StatusCode, resp.Request.URL.String(), errResp.Message)
}

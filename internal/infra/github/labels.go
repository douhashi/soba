package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/douhashi/soba/internal/infra"
)

// CreateLabel は新しいラベルを作成する
func (c *Client) CreateLabel(ctx context.Context, owner, repo string, request CreateLabelRequest) (*Label, error) {
	// リクエストボディの作成
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to marshal request body")
	}

	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/labels", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to create request")
	}

	// リクエスト実行（リトライ付き）
	retryClient := NewRetryableClient(&RetryOptions{
		Logger: c.logger,
	})
	resp, err := retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
		return c.doRequest(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスの処理
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp)
	}

	// レスポンスのパース
	var label Label
	if err := json.NewDecoder(resp.Body).Decode(&label); err != nil {
		return nil, infra.WrapInfraError(err, "failed to decode response")
	}

	return &label, nil
}

// ListLabels はリポジトリのラベル一覧を取得する
func (c *Client) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/labels", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to create request")
	}

	// リクエスト実行（リトライ付き）
	retryClient := NewRetryableClient(&RetryOptions{
		Logger: c.logger,
	})
	resp, err := retryClient.DoWithRetry(ctx, func() (*http.Response, error) {
		return c.doRequest(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスの処理
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp)
	}

	// レスポンスのパース
	var labels []Label
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		return nil, infra.WrapInfraError(err, "failed to decode response")
	}

	return labels, nil
}

// GetSobaLabels はsobaワークフローで使用するラベル定義を返す
func GetSobaLabels() []CreateLabelRequest {
	return []CreateLabelRequest{
		{
			Name:        "soba:todo",
			Color:       "e1e4e8",
			Description: "New issue awaiting processing",
		},
		{
			Name:        "soba:queued",
			Color:       "fbca04",
			Description: "Selected for processing",
		},
		{
			Name:        "soba:planning",
			Color:       "d4c5f9",
			Description: "Claude creating implementation plan",
		},
		{
			Name:        "soba:ready",
			Color:       "0e8a16",
			Description: "Plan complete, awaiting implementation",
		},
		{
			Name:        "soba:doing",
			Color:       "1d76db",
			Description: "Claude working on implementation",
		},
		{
			Name:        "soba:review-requested",
			Color:       "f9d71c",
			Description: "PR created, awaiting review",
		},
		{
			Name:        "soba:reviewing",
			Color:       "a2eeef",
			Description: "Claude reviewing PR",
		},
		{
			Name:        "soba:done",
			Color:       "0e8a16",
			Description: "Review approved, ready to merge",
		},
		{
			Name:        "soba:requires-changes",
			Color:       "d93f0b",
			Description: "Review requested modifications",
		},
		{
			Name:        "soba:revising",
			Color:       "ff6347",
			Description: "Claude applying requested changes",
		},
		{
			Name:        "soba:merged",
			Color:       "6f42c1",
			Description: "PR merged and issue closed",
		},
	}
}
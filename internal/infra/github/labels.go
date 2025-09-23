package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/douhashi/soba/internal/infra"
	"github.com/douhashi/soba/pkg/logging"
)

// CreateLabel は新しいラベルを作成する
func (c *ClientImpl) CreateLabel(ctx context.Context, owner, repo string, request CreateLabelRequest) (*Label, error) {
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
func (c *ClientImpl) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
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

// AddLabelToIssue はIssueにラベルを追加する
func (c *ClientImpl) AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	// リクエストボディの作成
	labels := []string{label}
	reqBody, err := json.Marshal(labels)
	if err != nil {
		return infra.WrapInfraError(err, "failed to marshal request body")
	}

	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return infra.WrapInfraError(err, "failed to create request")
	}

	// リクエスト実行
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// レスポンスの処理
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseErrorResponse(resp)
	}

	return nil
}

// RemoveLabelFromIssue はIssueからラベルを削除する
func (c *ClientImpl) RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels/%s", c.baseURL, owner, repo, issueNumber, label)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return infra.WrapInfraError(err, "failed to create request")
	}

	// リクエスト実行
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// レスポンスの処理
	// ラベルが存在しない場合は404が返るが、それはエラーとしない
	if resp.StatusCode == http.StatusNotFound {
		c.logger.Debug(ctx, "Label not found on issue",
			logging.Field{Key: "owner", Value: owner},
			logging.Field{Key: "repo", Value: repo},
			logging.Field{Key: "issue", Value: issueNumber},
			logging.Field{Key: "label", Value: label},
		)
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseErrorResponse(resp)
	}

	return nil
}

// GetIssueLabels はIssueのラベル一覧を取得する
func (c *ClientImpl) GetIssueLabels(ctx context.Context, owner, repo string, issueNumber int) ([]Label, error) {
	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to create request")
	}

	// リクエスト実行
	resp, err := c.doRequest(ctx, req)
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

// UpdateIssueLabels はIssueのラベルを更新する
func (c *ClientImpl) UpdateIssueLabels(ctx context.Context, owner, repo string, issueNumber int, labels []string) error {
	// リクエストボディの作成
	reqBody, err := json.Marshal(labels)
	if err != nil {
		return infra.WrapInfraError(err, "failed to marshal request body")
	}

	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.baseURL, owner, repo, issueNumber)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return infra.WrapInfraError(err, "failed to create request")
	}

	// リクエスト実行
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// レスポンスの処理
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseErrorResponse(resp)
	}

	return nil
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
	}
}

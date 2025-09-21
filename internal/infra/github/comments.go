package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/douhashi/soba/internal/infra"
)

// CreateComment はIssueにコメントを作成する
func (c *ClientImpl) CreateComment(ctx context.Context, owner, repo string, issueNumber int, body string) error {
	// リクエストボディの作成
	requestBody := map[string]string{
		"body": body,
	}
	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return infra.WrapInfraError(err, "failed to marshal request body")
	}

	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
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

// ListComments はIssueのコメント一覧を取得する
func (c *ClientImpl) ListComments(ctx context.Context, owner, repo string, issueNumber int, opts *ListCommentsOptions) ([]IssueComment, error) {
	// HTTPリクエストの作成
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, issueNumber)
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
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	// レスポンスのパース
	var comments []IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, infra.WrapInfraError(err, "failed to decode response")
	}

	return comments, nil
}
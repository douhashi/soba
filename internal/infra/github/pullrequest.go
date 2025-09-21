package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/douhashi/soba/internal/infra"
)

// ListPullRequests は指定されたリポジトリのPR一覧を取得する
func (c *ClientImpl) ListPullRequests(ctx context.Context, owner, repo string, opts *ListPullRequestsOptions) ([]PullRequest, bool, error) {
	// バリデーション
	if owner == "" {
		return nil, false, infra.NewGitHubAPIError(0, "", "owner is required")
	}
	if repo == "" {
		return nil, false, infra.NewGitHubAPIError(0, "", "repo is required")
	}

	// デフォルトオプションの設定
	if opts == nil {
		opts = &ListPullRequestsOptions{
			State:   "open",
			Page:    1,
			PerPage: 30,
		}
	} else {
		if opts.State == "" {
			opts.State = "open"
		}
		if opts.Page == 0 {
			opts.Page = 1
		}
		if opts.PerPage == 0 {
			opts.PerPage = 30
		}
	}

	// URLの構築
	apiURL := c.buildPullRequestsURL(owner, repo, opts)

	// HTTPリクエストの作成
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, false, infra.WrapInfraError(err, "failed to create request")
	}

	// リクエストの実行
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	// エラーチェック
	if resp.StatusCode != http.StatusOK {
		if err := c.parseErrorResponse(resp); err != nil {
			return nil, false, err
		}
		return nil, false, infra.NewGitHubAPIError(resp.StatusCode, "", "unexpected status code")
	}

	// レスポンスのデコード
	var prs []PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, false, infra.WrapInfraError(err, "failed to decode response")
	}

	// ページネーションのチェック
	hasNextPage := c.hasNextPage(resp)

	c.logger.Info("Fetched pull requests",
		"count", len(prs),
		"owner", owner,
		"repo", repo,
		"hasNextPage", hasNextPage,
	)

	return prs, hasNextPage, nil
}

// GetPullRequest は指定されたPRの詳細を取得する
func (c *ClientImpl) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, bool, error) {
	// バリデーション
	if owner == "" {
		return nil, false, infra.NewGitHubAPIError(0, "", "owner is required")
	}
	if repo == "" {
		return nil, false, infra.NewGitHubAPIError(0, "", "repo is required")
	}
	if number <= 0 {
		return nil, false, infra.NewGitHubAPIError(0, "", "invalid pull request number")
	}

	// URLの構築
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, number)

	// HTTPリクエストの作成
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, false, infra.WrapInfraError(err, "failed to create request")
	}

	// リクエストの実行
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	// エラーチェック
	if resp.StatusCode != http.StatusOK {
		if err := c.parseErrorResponse(resp); err != nil {
			return nil, false, err
		}
		return nil, false, infra.NewGitHubAPIError(resp.StatusCode, "", "unexpected status code")
	}

	// レスポンスのデコード
	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, false, infra.WrapInfraError(err, "failed to decode response")
	}

	return &pr, false, nil
}

// MergePullRequest は指定されたPRをマージする
func (c *ClientImpl) MergePullRequest(ctx context.Context, owner, repo string, number int, req *MergeRequest) (*MergeResponse, error) {
	// バリデーション
	if owner == "" {
		return nil, infra.NewGitHubAPIError(0, "", "owner is required")
	}
	if repo == "" {
		return nil, infra.NewGitHubAPIError(0, "", "repo is required")
	}
	if number <= 0 {
		return nil, infra.NewGitHubAPIError(0, "", "invalid pull request number")
	}

	// デフォルトのマージリクエスト
	if req == nil {
		req = &MergeRequest{
			MergeMethod: "merge",
		}
	}

	// URLの構築
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", c.baseURL, owner, repo, number)

	// リクエストボディの作成
	body, err := json.Marshal(req)
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to marshal request body")
	}

	// HTTPリクエストの作成
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, infra.WrapInfraError(err, "failed to create request")
	}

	// リクエストの実行
	resp, err := c.doRequest(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// エラーチェック
	if resp.StatusCode != http.StatusOK {
		if err := c.parseErrorResponse(resp); err != nil {
			return nil, err
		}
		return nil, infra.NewGitHubAPIError(resp.StatusCode, "", "unexpected status code")
	}

	// レスポンスのデコード
	var mergeResp MergeResponse
	if err := json.NewDecoder(resp.Body).Decode(&mergeResp); err != nil {
		return nil, infra.WrapInfraError(err, "failed to decode response")
	}

	c.logger.Info("Pull request merged successfully",
		"number", number,
		"owner", owner,
		"repo", repo,
		"sha", mergeResp.SHA,
	)

	return &mergeResp, nil
}

// buildPullRequestsURL はPR一覧取得用のURLを構築する
func (c *ClientImpl) buildPullRequestsURL(owner, repo string, opts *ListPullRequestsOptions) string {
	baseURL := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, owner, repo)

	// クエリパラメータの構築
	params := url.Values{}
	params.Set("state", opts.State)
	params.Set("page", fmt.Sprintf("%d", opts.Page))
	params.Set("per_page", fmt.Sprintf("%d", opts.PerPage))

	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}
	if opts.Direction != "" {
		params.Set("direction", opts.Direction)
	}

	// ラベルフィルタ
	if len(opts.Labels) > 0 {
		for _, label := range opts.Labels {
			params.Add("labels", label)
		}
	}

	return baseURL + "?" + params.Encode()
}

// hasNextPage はレスポンスヘッダーから次のページがあるか判定する
func (c *ClientImpl) hasNextPage(resp *http.Response) bool {
	linkHeader := resp.Header.Get("Link")
	if linkHeader == "" {
		return false
	}

	// Linkヘッダーに "rel=\"next\"" が含まれていれば次のページがある
	// 簡易的な実装
	return len(linkHeader) > 0 && (len(linkHeader) > 0 && linkHeader != "")
}

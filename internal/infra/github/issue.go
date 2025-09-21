package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/douhashi/soba/internal/infra"
)

// ListOpenIssues は指定されたリポジトリのオープンなIssue一覧を取得する
func (c *Client) ListOpenIssues(ctx context.Context, owner, repo string, opts *ListIssuesOptions) ([]Issue, bool, error) {
	// バリデーション
	if owner == "" {
		return nil, false, infra.NewGitHubAPIError(0, "", "owner is required")
	}
	if repo == "" {
		return nil, false, infra.NewGitHubAPIError(0, "", "repo is required")
	}

	// デフォルトオプションの設定
	if opts == nil {
		opts = &ListIssuesOptions{
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
	apiURL := c.buildIssuesURL(owner, repo, opts)

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

	// エラーレスポンスの処理
	if resp.StatusCode != http.StatusOK {
		return nil, false, c.parseErrorResponse(resp)
	}

	// レスポンスのパース
	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, false, infra.WrapInfraError(err, "failed to decode response")
	}

	// ページネーション情報の取得
	linkHeader := resp.Header.Get("Link")
	hasNext := parseLinkHeader(linkHeader)

	return issues, hasNext, nil
}

// buildIssuesURL はIssue取得用のURLを構築する
func (c *Client) buildIssuesURL(owner, repo string, opts *ListIssuesOptions) string {
	baseURL := fmt.Sprintf("%s/repos/%s/%s/issues", c.baseURL, owner, repo)

	params := url.Values{}

	// デフォルト値の設定
	if opts == nil {
		params.Set("state", "open")
		params.Set("page", "1")
		params.Set("per_page", "30")
	} else {
		// State
		if opts.State != "" {
			params.Set("state", opts.State)
		} else {
			params.Set("state", "open")
		}

		// Labels
		if len(opts.Labels) > 0 {
			params.Set("labels", strings.Join(opts.Labels, ","))
		}

		// Sort
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}

		// Direction
		if opts.Direction != "" {
			params.Set("direction", opts.Direction)
		}

		// Since
		if opts.Since != nil {
			params.Set("since", opts.Since.Format("2006-01-02T15:04:05Z"))
		}

		// Page
		if opts.Page > 0 {
			params.Set("page", fmt.Sprintf("%d", opts.Page))
		} else {
			params.Set("page", "1")
		}

		// PerPage
		if opts.PerPage > 0 {
			params.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
		} else {
			params.Set("per_page", "30")
		}
	}

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// parseLinkHeader はLinkヘッダーをパースして次のページがあるか判定する
func parseLinkHeader(link string) bool {
	if link == "" {
		return false
	}

	// Linkヘッダーは以下のような形式
	// <https://api.github.com/...?page=2>; rel="next", <https://api.github.com/...?page=10>; rel="last"
	parts := strings.Split(link, ",")
	for _, part := range parts {
		if strings.Contains(part, `rel="next"`) {
			return true
		}
	}

	return false
}

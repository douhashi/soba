package github

import "time"

// Issue はGitHub IssueのAPI応答を表す
type Issue struct {
	ID        int64      `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	URL       string     `json:"url"`
	HTMLURL   string     `json:"html_url"`
	Labels    []Label    `json:"labels"`
	Assignees []User     `json:"assignees"`
	User      User       `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

// Label はGitHub Labelを表す
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// User はGitHub Userを表す
type User struct {
	ID      int64  `json:"id"`
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
}

// ListIssuesOptions はIssue一覧取得時のオプション
type ListIssuesOptions struct {
	State     string   // open, closed, all
	Labels    []string // ラベルフィルタ
	Sort      string   // created, updated, comments
	Direction string   // asc, desc
	Since     *time.Time
	Page      int
	PerPage   int
}

// CreateLabelRequest はラベル作成時のリクエスト
type CreateLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// ErrorResponse はGitHub APIのエラーレスポンス
type ErrorResponse struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
}

// Error はerrorインターフェースを実装
func (e ErrorResponse) Error() string {
	return e.Message
}

// PullRequest はGitHub Pull Requestを表す
type PullRequest struct {
	ID             int64      `json:"id"`
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	State          string     `json:"state"` // open, closed
	URL            string     `json:"url"`
	HTMLURL        string     `json:"html_url"`
	Labels         []Label    `json:"labels"`
	User           User       `json:"user"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClosedAt       *time.Time `json:"closed_at"`
	MergedAt       *time.Time `json:"merged_at"`
	Mergeable      bool       `json:"mergeable"`
	MergeableState string     `json:"mergeable_state"` // clean, dirty, unknown, etc.
}

// ListPullRequestsOptions はPR一覧取得時のオプション
type ListPullRequestsOptions struct {
	State     string   // open, closed, all
	Labels    []string // ラベルフィルタ
	Sort      string   // created, updated
	Direction string   // asc, desc
	Page      int
	PerPage   int
}

// MergeRequest はPRマージ時のリクエスト
type MergeRequest struct {
	CommitTitle   string `json:"commit_title,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	SHA           string `json:"sha,omitempty"`
	MergeMethod   string `json:"merge_method,omitempty"` // merge, squash, rebase
}

// MergeResponse はPRマージ時のレスポンス
type MergeResponse struct {
	SHA     string `json:"sha"`
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

// IssueComment はIssueコメントを表す
type IssueComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListCommentsOptions はコメント一覧取得時のオプション
type ListCommentsOptions struct {
	Sort      string // created, updated
	Direction string // asc, desc
	Since     *time.Time
	Page      int
	PerPage   int
}

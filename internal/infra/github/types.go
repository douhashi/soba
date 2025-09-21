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

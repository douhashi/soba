package github

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// GetTestIssues はテスト用のIssue一覧を返す
func GetTestIssues() ([]Issue, error) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	data, err := os.ReadFile(filepath.Join(dir, "testdata", "issues.json"))
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

// GetSampleIssue はテスト用の単一Issueを返す
func GetSampleIssue() Issue {
	createdAt, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2024-01-16T14:22:00Z")

	return Issue{
		ID:      1234567890,
		Number:  42,
		Title:   "Feature: Add support for multiple configurations",
		Body:    "We need to support multiple configuration files to make the system more flexible.",
		State:   "open",
		URL:     "https://api.github.com/repos/example/repo/issues/42",
		HTMLURL: "https://github.com/example/repo/issues/42",
		Labels: []Label{
			{
				ID:    208045946,
				Name:  "enhancement",
				Color: "a2eeef",
			},
			{
				ID:    208045947,
				Name:  "priority:high",
				Color: "d73a4a",
			},
		},
		Assignees: []User{
			{
				ID:      583231,
				Login:   "octocat",
				HTMLURL: "https://github.com/octocat",
			},
		},
		User: User{
			ID:      1,
			Login:   "developer",
			HTMLURL: "https://github.com/developer",
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ClosedAt:  nil,
	}
}

// GetClosedIssue はクローズ済みのテストIssueを返す
func GetClosedIssue() Issue {
	createdAt, _ := time.Parse(time.RFC3339, "2024-01-10T10:00:00Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2024-01-12T15:00:00Z")
	closedAt, _ := time.Parse(time.RFC3339, "2024-01-12T15:00:00Z")

	return Issue{
		ID:      1234567889,
		Number:  41,
		Title:   "Bug: Fixed memory leak",
		Body:    "There was a memory leak in the parser module.",
		State:   "closed",
		URL:     "https://api.github.com/repos/example/repo/issues/41",
		HTMLURL: "https://github.com/example/repo/issues/41",
		Labels: []Label{
			{
				ID:    208045948,
				Name:  "bug",
				Color: "d73a4a",
			},
		},
		Assignees: []User{},
		User: User{
			ID:      2,
			Login:   "contributor",
			HTMLURL: "https://github.com/contributor",
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ClosedAt:  &closedAt,
	}
}

// GetEmptyIssueList は空のIssue一覧を返す
func GetEmptyIssueList() []Issue {
	return []Issue{}
}

// GetLargeIssueList は大量のテストIssueを生成して返す
func GetLargeIssueList(count int) []Issue {
	issues := make([]Issue, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		issues[i] = Issue{
			ID:        int64(1000000 + i),
			Number:    100 + i,
			Title:     "Test Issue " + string(rune(i)),
			Body:      "This is a test issue body",
			State:     "open",
			URL:       "https://api.github.com/repos/test/repo/issues/" + string(rune(100+i)),
			HTMLURL:   "https://github.com/test/repo/issues/" + string(rune(100+i)),
			Labels:    []Label{},
			Assignees: []User{},
			User: User{
				ID:      int64(1000 + i),
				Login:   "user" + string(rune(i)),
				HTMLURL: "https://github.com/user" + string(rune(i)),
			},
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
			UpdatedAt: now.Add(-time.Duration(i/2) * time.Hour),
			ClosedAt:  nil,
		}
	}

	return issues
}
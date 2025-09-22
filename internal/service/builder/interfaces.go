package builder

import (
	"context"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
)

// GitWorkspaceManager manages Git workspaces for issues
type GitWorkspaceManager interface {
	PrepareWorkspace(issueNumber int) error
	CleanupWorkspace(issueNumber int) error
}

// IssueProcessorInterface handles issue processing
type IssueProcessorInterface interface {
	Process(ctx context.Context, cfg *config.Config) error
	ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error
	UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
	Configure(cfg *config.Config) error
}

// IssueProcessorUpdater handles label updates
type IssueProcessorUpdater interface {
	UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
	Configure(cfg *config.Config) error
}

// WorkflowExecutor executes workflow phases
type WorkflowExecutor interface {
	ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase interface{}) error
	SetIssueProcessor(processor IssueProcessorUpdater)
}

// IssueWatcher watches for issue changes
type IssueWatcher interface {
	Start(ctx context.Context) error
	SetProcessor(processor IssueProcessorInterface)
	SetQueueManager(manager interface{})
	SetLogger(logger interface{})
	SetWorkflowExecutor(executor WorkflowExecutor)
}

// PRWatcher watches for PR changes
type PRWatcher interface {
	Start(ctx context.Context) error
	SetLogger(logger interface{})
}

// ErrorHandler handles various errors in the system
type ErrorHandler interface {
	HandleGitHubClientError(err error) (GitHubClientInterface, error)
	HandleGitClientError(workDir string, err error) (*MockGitClient, error)
	ShouldContinueOnError(component string, err error) bool
	LogError(ctx context.Context, msg string, err error)
	LogWarning(ctx context.Context, msg string)
	LogInfo(ctx context.Context, msg string)
}

// ClosedIssueCleanupService cleans up closed issues
type ClosedIssueCleanupService interface {
	Start(ctx context.Context) error
	Configure(owner, repo, sessionName string, enabled bool, interval interface{})
}

// DaemonService provides daemon functionality
type DaemonService interface {
	StartForeground(ctx context.Context, cfg *config.Config) error
	StartDaemon(ctx context.Context, cfg *config.Config) error
	IsRunning() bool
	Stop(ctx context.Context, repository string) error
}

// StatusService provides status information about soba
type StatusService interface {
	GetStatus(ctx context.Context) (*Status, error)
}

// Status represents the current status of soba
type Status struct {
	Daemon *DaemonStatus `json:"daemon"`
	Tmux   *TmuxStatus   `json:"tmux"`
	Issues []IssueStatus `json:"issues"`
}

// DaemonStatus represents daemon process status
type DaemonStatus struct {
	Running bool   `json:"running"`
	PID     int    `json:"pid,omitempty"`
	Uptime  string `json:"uptime,omitempty"`
}

// TmuxStatus represents tmux session status
type TmuxStatus struct {
	SessionName string       `json:"session_name"`
	Windows     []TmuxWindow `json:"windows"`
}

// TmuxWindow represents a tmux window
type TmuxWindow struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	IssueNumber int    `json:"issue_number,omitempty"`
	Title       string `json:"title,omitempty"`
}

// IssueStatus represents an issue's processing status
type IssueStatus struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
	State  string   `json:"state"`
}

// GitClientInterface defines Git client interface
type GitClientInterface interface {
	GetCurrentBranch() (string, error)
	CreateBranch(branchName string, baseBranch string) error
	SwitchBranch(branchName string) error
	DeleteBranch(branchName string, force bool) error
	BranchExists(branchName string) (bool, error)
}

// GitHubClientInterface defines GitHub client interface
type GitHubClientInterface interface {
	ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error)
	AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	ListPullRequests(ctx context.Context, owner, repo string, opts *github.ListPullRequestsOptions) ([]github.PullRequest, bool, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, bool, error)
	MergePullRequest(ctx context.Context, owner, repo string, number int, req *github.MergeRequest) (*github.MergeResponse, error)
}

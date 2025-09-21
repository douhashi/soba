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

// GitHubClientInterface defines GitHub client interface
type GitHubClientInterface interface {
	ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error)
	AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error
	ListPullRequests(ctx context.Context, owner, repo string, opts *github.ListPullRequestsOptions) ([]github.PullRequest, bool, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, bool, error)
	MergePullRequest(ctx context.Context, owner, repo string, number int, req *github.MergeRequest) (*github.MergeResponse, error)
}

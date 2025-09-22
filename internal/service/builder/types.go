package builder

import (
	"github.com/douhashi/soba/internal/infra/git"
	"github.com/douhashi/soba/internal/infra/slack"
	"github.com/douhashi/soba/internal/infra/tmux"
)

// Clients represents the available clients
type Clients = ResolvedClients

// ResolvedClients represents resolved client dependencies
type ResolvedClients struct {
	GitHubClient  GitHubClientInterface
	GitClient     *git.Client
	TmuxClient    tmux.TmuxClient
	SlackNotifier *slack.Notifier
}

// ResolvedServices represents resolved service dependencies
type ResolvedServices struct {
	IssueProcessor   IssueProcessorInterface
	WorkflowExecutor WorkflowExecutor
	IssueWatcher     IssueWatcher
	PRWatcher        PRWatcher
	QueueManager     interface{}
	CleanupService   ClosedIssueCleanupService
	WorkspaceManager GitWorkspaceManager
}

// DefaultServiceFactory interface
type DefaultServiceFactory interface {
	CreateWorkflowExecutorWithSlack(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater, slackNotifier interface{}) WorkflowExecutor
	CreatePRWatcherWithSlack(githubClient GitHubClientInterface, cfg interface{}, slackNotifier interface{}) PRWatcher
}

package builder

import (
	"github.com/douhashi/soba/internal/infra/git"
	"github.com/douhashi/soba/internal/infra/tmux"
)

// Clients represents the available clients
type Clients = ResolvedClients

// ResolvedClients represents resolved client dependencies
type ResolvedClients struct {
	GitHubClient GitHubClientInterface
	GitClient    *git.Client
	TmuxClient   tmux.TmuxClient
	// SlackNotifier removed - using singleton SlackManager instead
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
	// WithSlackメソッドは削除済み - 新しいSlackManagerシングルトンを使用
}

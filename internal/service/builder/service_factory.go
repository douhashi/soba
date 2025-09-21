package builder

import (
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/tmux"
)

// ServiceFactory creates service instances
type ServiceFactory interface {
	CreateGitWorkspaceManager(cfg *config.Config, gitClient interface{}) GitWorkspaceManager
	CreateMockGitWorkspaceManager() GitWorkspaceManager
	CreateWorkflowExecutor(tmuxClient tmux.TmuxClient, workspace GitWorkspaceManager, processor IssueProcessorUpdater) WorkflowExecutor
	CreateIssueProcessor(githubClient GitHubClientInterface, executor WorkflowExecutor) IssueProcessorInterface
	CreateIssueWatcher(githubClient GitHubClientInterface, cfg *config.Config) IssueWatcher
	CreateQueueManager(githubClient GitHubClientInterface, owner, repo string) interface{}
	CreatePRWatcher(githubClient GitHubClientInterface, cfg *config.Config) PRWatcher
	CreateClosedIssueCleanupService(githubClient GitHubClientInterface, tmuxClient tmux.TmuxClient, owner, repo, sessionName string, enabled bool, interval time.Duration) ClosedIssueCleanupService
	CreateDaemonServiceWithDependencies(workDir string, processor IssueProcessorInterface, watcher IssueWatcher, prWatcher PRWatcher, cleanupService ClosedIssueCleanupService, tmuxClient tmux.TmuxClient) DaemonService
}

// SetServiceFactory sets the service factory
var serviceFactory ServiceFactory

func SetServiceFactory(factory ServiceFactory) {
	serviceFactory = factory
}

// GetServiceFactory returns the current service factory
func GetServiceFactory() ServiceFactory {
	return serviceFactory
}

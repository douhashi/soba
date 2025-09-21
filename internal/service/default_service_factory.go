package service

import (
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/git"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/internal/service/builder"
)

// DefaultServiceFactory implements ServiceFactory interface
type DefaultServiceFactory struct{}

// CreateGitWorkspaceManager creates git workspace manager
func (f *DefaultServiceFactory) CreateGitWorkspaceManager(cfg *config.Config, gitClient interface{}) builder.GitWorkspaceManager {
	if gc, ok := gitClient.(*git.Client); ok {
		return &GitWorkspaceManagerAdapter{NewGitWorkspaceManager(cfg, gc)}
	}
	return &GitWorkspaceManagerAdapter{NewMockGitWorkspaceManager()}
}

// CreateMockGitWorkspaceManager creates mock git workspace manager
func (f *DefaultServiceFactory) CreateMockGitWorkspaceManager() builder.GitWorkspaceManager {
	return &GitWorkspaceManagerAdapter{NewMockGitWorkspaceManager()}
}

// CreateWorkflowExecutor creates workflow executor
func (f *DefaultServiceFactory) CreateWorkflowExecutor(tmuxClient tmux.TmuxClient, workspace builder.GitWorkspaceManager, processor builder.IssueProcessorUpdater) builder.WorkflowExecutor {
	// Convert builder interface back to concrete type for NewWorkflowExecutor
	var concreteWorkspace GitWorkspaceManager
	if adapter, ok := workspace.(*GitWorkspaceManagerAdapter); ok {
		concreteWorkspace = adapter.GitWorkspaceManager
	}
	var concreteProcessor IssueProcessorUpdater
	if adapter, ok := processor.(*IssueProcessorAdapter); ok {
		concreteProcessor = adapter.IssueProcessorInterface
	}
	return &WorkflowExecutorAdapter{NewWorkflowExecutor(tmuxClient, concreteWorkspace, concreteProcessor)}
}

// CreateIssueProcessor creates issue processor
func (f *DefaultServiceFactory) CreateIssueProcessor(githubClient builder.GitHubClientInterface, executor builder.WorkflowExecutor) builder.IssueProcessorInterface {
	var concreteExecutor WorkflowExecutor
	if adapter, ok := executor.(*WorkflowExecutorAdapter); ok {
		concreteExecutor = adapter.WorkflowExecutor
	}
	// Convert builder interface to concrete type
	var concreteGithubClient GitHubClientInterface
	if impl, ok := githubClient.(GitHubClientInterface); ok {
		concreteGithubClient = impl
	}
	return &IssueProcessorAdapter{NewIssueProcessor(concreteGithubClient, concreteExecutor)}
}

// CreateIssueWatcher creates issue watcher
func (f *DefaultServiceFactory) CreateIssueWatcher(githubClient builder.GitHubClientInterface, cfg *config.Config) builder.IssueWatcher {
	var concreteGithubClient GitHubClientInterface
	if impl, ok := githubClient.(GitHubClientInterface); ok {
		concreteGithubClient = impl
	}
	return &IssueWatcherAdapter{NewIssueWatcher(concreteGithubClient, cfg)}
}

// CreateQueueManager creates queue manager
func (f *DefaultServiceFactory) CreateQueueManager(githubClient builder.GitHubClientInterface, owner, repo string) interface{} {
	var concreteGithubClient GitHubClientInterface
	if impl, ok := githubClient.(GitHubClientInterface); ok {
		concreteGithubClient = impl
	}
	return NewQueueManager(concreteGithubClient, owner, repo)
}

// CreatePRWatcher creates PR watcher
func (f *DefaultServiceFactory) CreatePRWatcher(githubClient builder.GitHubClientInterface, cfg *config.Config) builder.PRWatcher {
	var concreteGithubClient GitHubClientInterface
	if impl, ok := githubClient.(GitHubClientInterface); ok {
		concreteGithubClient = impl
	}
	return &PRWatcherAdapter{NewPRWatcher(concreteGithubClient, cfg)}
}

// CreateClosedIssueCleanupService creates cleanup service
func (f *DefaultServiceFactory) CreateClosedIssueCleanupService(githubClient builder.GitHubClientInterface, tmuxClient tmux.TmuxClient, owner, repo, sessionName string, enabled bool, interval time.Duration) builder.ClosedIssueCleanupService {
	// Type assert to *github.ClientImpl which is what NewClosedIssueCleanupService expects
	if impl, ok := githubClient.(*github.ClientImpl); ok {
		return &ClosedIssueCleanupServiceAdapter{NewClosedIssueCleanupService(impl, tmuxClient, owner, repo, sessionName, enabled, interval)}
	}
	// Fallback: return a mock implementation or nil
	return &ClosedIssueCleanupServiceAdapter{NewClosedIssueCleanupService(nil, tmuxClient, owner, repo, sessionName, enabled, interval)}
}

// CreateDaemonServiceWithDependencies creates daemon service with dependencies
func (f *DefaultServiceFactory) CreateDaemonServiceWithDependencies(workDir string, processor builder.IssueProcessorInterface, watcher builder.IssueWatcher, prWatcher builder.PRWatcher, cleanupService builder.ClosedIssueCleanupService, tmuxClient tmux.TmuxClient) builder.DaemonService {
	// Convert builder interfaces back to concrete types
	var concreteProcessor IssueProcessorInterface
	if adapter, ok := processor.(*IssueProcessorAdapter); ok {
		concreteProcessor = adapter.IssueProcessorInterface
	}

	var concreteWatcher *IssueWatcher
	if adapter, ok := watcher.(*IssueWatcherAdapter); ok {
		concreteWatcher = adapter.IssueWatcher
	}

	var concretePRWatcher *PRWatcher
	if adapter, ok := prWatcher.(*PRWatcherAdapter); ok {
		concretePRWatcher = adapter.PRWatcher
	}

	var concreteCleanupService *ClosedIssueCleanupService
	if adapter, ok := cleanupService.(*ClosedIssueCleanupServiceAdapter); ok {
		concreteCleanupService = adapter.ClosedIssueCleanupService
	}

	service := NewDaemonServiceWithDependencies(workDir, concreteProcessor, concreteWatcher, concretePRWatcher, concreteCleanupService, tmuxClient)
	return &DaemonServiceAdapter{service.(*daemonService)}
}
package builder

import (
	"fmt"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/git"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/pkg/logger"
)

// DependencyResolver resolves component dependencies with proper error handling
type DependencyResolver struct {
	config       *config.Config
	workDir      string
	logger       logger.Logger
	errorHandler ErrorHandler
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(cfg *config.Config, workDir string, log logger.Logger, handler ErrorHandler) *DependencyResolver {
	return &DependencyResolver{
		config:       cfg,
		workDir:      workDir,
		logger:       log,
		errorHandler: handler,
	}
}

// ResolveClients resolves all client dependencies
func (r *DependencyResolver) ResolveClients() (*ResolvedClients, error) {
	clients := &ResolvedClients{}

	// GitHub Client (必須)
	tokenProvider := github.NewDefaultTokenProvider()
	githubClientImpl, err := github.NewClient(tokenProvider, &github.ClientOptions{})
	if err != nil {
		githubClient, handlerErr := r.errorHandler.HandleGitHubClientError(err)
		if handlerErr != nil {
			return nil, fmt.Errorf("failed to initialize GitHub client: %w", handlerErr)
		}
		clients.GitHubClient = githubClient
	} else {
		clients.GitHubClient = githubClientImpl
	}

	// Git Client (オプショナル、フォールバック可能)
	gitClient, err := git.NewClient(r.workDir)
	if err != nil {
		if !r.errorHandler.ShouldContinueOnError("git_client", err) {
			return nil, fmt.Errorf("failed to initialize git client: %w", err)
		}
		mockClient, handlerErr := r.errorHandler.HandleGitClientError(r.workDir, err)
		if handlerErr != nil {
			return nil, handlerErr
		}
		clients.GitClient = mockClient
	} else {
		clients.GitClient = gitClient
	}

	// Tmux Client (必須)
	clients.TmuxClient = tmux.NewClient()

	return clients, nil
}

// ResolveServices resolves service dependencies using clean dependency injection
func (r *DependencyResolver) ResolveServices(clients *ResolvedClients) (*ResolvedServices, error) {
	if serviceFactory == nil {
		return nil, fmt.Errorf("service factory not set")
	}

	services := &ResolvedServices{}

	// Phase 1: Create workspace manager
	workspace := r.createWorkspaceManager(clients)
	services.WorkspaceManager = workspace

	// Phase 2: Create workflow executor with nil processor (will be set later)
	workflowExecutor := serviceFactory.CreateWorkflowExecutor(clients.TmuxClient, workspace, nil)
	services.WorkflowExecutor = workflowExecutor

	// Phase 3: Create issue processor with workflow executor
	issueProcessor := serviceFactory.CreateIssueProcessor(clients.GitHubClient, workflowExecutor)
	services.IssueProcessor = issueProcessor

	// Phase 3.5: Set issue processor to workflow executor (completes the circular dependency)
	workflowExecutor.SetIssueProcessor(issueProcessor)

	// Phase 4: Create watchers and other services
	issueWatcher := r.createIssueWatcher(clients, issueProcessor, workflowExecutor)
	services.IssueWatcher = issueWatcher

	prWatcher := r.createPRWatcher(clients)
	services.PRWatcher = prWatcher

	cleanupService := r.createCleanupService(clients)
	services.CleanupService = cleanupService

	return services, nil
}

// createWorkspaceManager creates workspace manager based on available git client
func (r *DependencyResolver) createWorkspaceManager(clients *ResolvedClients) GitWorkspaceManager {
	if clients.GitClient != nil {
		return serviceFactory.CreateGitWorkspaceManager(r.config, clients.GitClient)
	}
	return serviceFactory.CreateMockGitWorkspaceManager()
}

// createIssueWatcher creates and configures issue watcher
func (r *DependencyResolver) createIssueWatcher(clients *ResolvedClients, processor IssueProcessorInterface, workflowExecutor WorkflowExecutor) IssueWatcher {
	watcher := serviceFactory.CreateIssueWatcher(clients.GitHubClient, r.config)
	watcher.SetProcessor(processor)
	watcher.SetWorkflowExecutor(workflowExecutor)

	queueManager := serviceFactory.CreateQueueManager(clients.GitHubClient, "", "")
	watcher.SetQueueManager(queueManager)

	return watcher
}

// createPRWatcher creates PR watcher
func (r *DependencyResolver) createPRWatcher(clients *ResolvedClients) PRWatcher {
	return serviceFactory.CreatePRWatcher(clients.GitHubClient, r.config)
}

// createCleanupService creates cleanup service
func (r *DependencyResolver) createCleanupService(clients *ResolvedClients) ClosedIssueCleanupService {
	return serviceFactory.CreateClosedIssueCleanupService(
		clients.GitHubClient,
		clients.TmuxClient,
		"", "", "",
		false,
		5*time.Minute,
	)
}

// ResolvedClients holds resolved client dependencies
type ResolvedClients struct {
	GitHubClient GitHubClientInterface
	GitClient    interface{} // Can be *git.Client or *MockGitClient
	TmuxClient   tmux.TmuxClient
}

// ResolvedServices holds resolved service dependencies
type ResolvedServices struct {
	WorkspaceManager GitWorkspaceManager
	IssueProcessor   IssueProcessorInterface
	WorkflowExecutor WorkflowExecutor
	IssueWatcher     IssueWatcher
	PRWatcher        PRWatcher
	CleanupService   ClosedIssueCleanupService
}

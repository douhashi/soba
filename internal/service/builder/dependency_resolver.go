package builder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/git"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/slack"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/pkg/logging"
)

// DependencyResolver resolves component dependencies with new logging
type DependencyResolver struct {
	config       *config.Config
	workDir      string
	logFactory   *logging.Factory
	logger       logging.Logger
	errorHandler ErrorHandler
}

// NewDependencyResolver creates a new dependency resolver with new logging
func NewDependencyResolver(cfg *config.Config, workDir string, logFactory *logging.Factory, handler ErrorHandler) *DependencyResolver {
	return &DependencyResolver{
		config:       cfg,
		workDir:      workDir,
		logFactory:   logFactory,
		logger:       logFactory.CreateComponentLogger("dependency-resolver"),
		errorHandler: handler,
	}
}

// ResolveClients resolves all client dependencies
func (r *DependencyResolver) ResolveClients(ctx context.Context) (*ResolvedClients, error) {
	r.logger.Info(ctx, "Resolving client dependencies")

	clients := &ResolvedClients{}

	// GitHub Client (必須)
	r.logger.Debug(ctx, "Initializing GitHub client")
	tokenProvider := github.NewDefaultTokenProvider()
	githubClientImpl, err := github.NewClient(tokenProvider, &github.ClientOptions{
		Logger: r.logFactory.CreateComponentLogger("github-client"),
	})
	if err != nil {
		r.logger.Error(ctx, "Failed to initialize GitHub client",
			logging.Field{Key: "error", Value: err.Error()},
		)
		githubClient, handlerErr := r.errorHandler.HandleGitHubClientError(err)
		if handlerErr != nil {
			return nil, fmt.Errorf("failed to initialize GitHub client: %w", handlerErr)
		}
		clients.GitHubClient = githubClient
	} else {
		r.logger.Debug(ctx, "GitHub client initialized successfully")
		clients.GitHubClient = githubClientImpl
	}

	// Git Client (オプショナル、フォールバック可能)
	r.logger.Debug(ctx, "Initializing Git client",
		logging.Field{Key: "workDir", Value: r.workDir},
	)
	gitClient, err := git.NewClient(r.workDir)
	if err != nil {
		r.logger.Warn(ctx, "Failed to initialize Git client, using fallback",
			logging.Field{Key: "error", Value: err.Error()},
		)
		if !r.errorHandler.ShouldContinueOnError("git_client", err) {
			return nil, fmt.Errorf("failed to initialize git client: %w", err)
		}
		mockClient, handlerErr := r.errorHandler.HandleGitClientError(r.workDir, err)
		if handlerErr != nil {
			return nil, handlerErr
		}
		// MockGitClient can be used as *git.Client due to embedded type
		if mockClient != nil && mockClient.Client == nil {
			// If embedded Client is nil, we use the mock as is
			clients.GitClient = (*git.Client)(nil)
		} else {
			clients.GitClient = mockClient.Client
		}
	} else {
		r.logger.Debug(ctx, "Git client initialized successfully")
		clients.GitClient = gitClient
	}

	// Tmux Client (必須)
	r.logger.Debug(ctx, "Initializing Tmux client")
	clients.TmuxClient = tmux.NewClient()

	// Slack Client (オプショナル)
	if r.config.Slack.NotificationsEnabled && r.config.Slack.WebhookURL != "" {
		r.logger.Info(ctx, "Initializing Slack client for notifications")
		slackClient := slack.NewClient(r.config.Slack.WebhookURL, 10*time.Second)
		slackLogger := r.logFactory.CreateComponentLogger("slack-notifier")
		clients.SlackNotifier = slack.NewNotifier(slackClient, &r.config.Slack, slackLogger)
	} else {
		r.logger.Debug(ctx, "Slack notifications not configured")
	}

	r.logger.Info(ctx, "Client dependencies resolved successfully")
	return clients, nil
}

// ResolveServices resolves service dependencies
func (r *DependencyResolver) ResolveServices(ctx context.Context, clients *ResolvedClients) (*ResolvedServices, error) {
	r.logger.Info(ctx, "Resolving service dependencies")

	if serviceFactory == nil {
		r.logger.Error(ctx, "Service factory not set")
		return nil, fmt.Errorf("service factory not set")
	}

	services := &ResolvedServices{}

	// Phase 1: Create workspace manager
	r.logger.Debug(ctx, "Creating workspace manager")
	workspace := r.createWorkspaceManager(clients)
	services.WorkspaceManager = workspace

	// Phase 2: Create workflow executor with nil processor (will be set later)
	r.logger.Debug(ctx, "Creating workflow executor")
	var workflowExecutor WorkflowExecutor
	if clients.SlackNotifier != nil {
		// Create workflow executor with Slack notifications if available
		if factory, ok := serviceFactory.(DefaultServiceFactory); ok {
			r.logger.Debug(ctx, "Creating workflow executor with Slack support")
			workflowExecutor = factory.CreateWorkflowExecutorWithSlack(
				clients.TmuxClient,
				workspace,
				nil, // Will be set later
				clients.SlackNotifier,
			)
		} else {
			r.logger.Debug(ctx, "Creating standard workflow executor")
			workflowExecutor = serviceFactory.CreateWorkflowExecutor(
				clients.TmuxClient,
				workspace,
				nil, // Will be set later
			)
		}
	} else {
		r.logger.Debug(ctx, "Creating workflow executor without Slack")
		workflowExecutor = serviceFactory.CreateWorkflowExecutor(
			clients.TmuxClient,
			workspace,
			nil, // Will be set later
		)
	}
	services.WorkflowExecutor = workflowExecutor

	// Phase 3: Create issue processor
	r.logger.Debug(ctx, "Creating issue processor")
	issueProcessor := serviceFactory.CreateIssueProcessor(
		clients.GitHubClient,
		workflowExecutor,
	)
	services.IssueProcessor = issueProcessor

	// Phase 4: Update workflow executor with processor
	r.logger.Debug(ctx, "Updating workflow executor with processor")
	// Set the issue processor on the workflow executor
	workflowExecutor.SetIssueProcessor(issueProcessor)
	r.logger.Info(ctx, "Successfully set IssueProcessor on WorkflowExecutor")

	// Parse repository for owner and repo (needed for multiple services)
	owner, repo := parseRepository(r.config.GitHub.Repository)

	// Phase 5: Create watchers
	r.logger.Debug(ctx, "Creating issue watcher")
	services.IssueWatcher = serviceFactory.CreateIssueWatcher(
		clients.GitHubClient,
		r.config,
	)

	// Configure issue watcher with queue manager and other services
	r.logger.Info(ctx, "Repository config",
		logging.Field{Key: "repository", Value: r.config.GitHub.Repository},
		logging.Field{Key: "owner", Value: owner},
		logging.Field{Key: "repo", Value: repo},
	)
	if owner == "" || repo == "" {
		r.logger.Error(ctx, "Repository configuration is required but not provided",
			logging.Field{Key: "repository", Value: r.config.GitHub.Repository},
			logging.Field{Key: "owner", Value: owner},
			logging.Field{Key: "repo", Value: repo},
		)
		return nil, fmt.Errorf("repository configuration is required: owner=%q repo=%q (repository=%q)", owner, repo, r.config.GitHub.Repository)
	}

	r.logger.Info(ctx, "Creating queue manager",
		logging.Field{Key: "owner", Value: owner},
		logging.Field{Key: "repo", Value: repo},
	)
	queueManager := serviceFactory.CreateQueueManager(clients.GitHubClient, owner, repo)
	services.IssueWatcher.SetQueueManager(queueManager)
	r.logger.Info(ctx, "Queue manager set to IssueWatcher")
	services.IssueWatcher.SetProcessor(issueProcessor)
	services.IssueWatcher.SetWorkflowExecutor(workflowExecutor)

	r.logger.Debug(ctx, "Creating PR watcher")
	if clients.SlackNotifier != nil && r.config.Slack.NotificationsEnabled {
		// Create PR watcher with Slack notifications if available
		if factory, ok := serviceFactory.(DefaultServiceFactory); ok {
			r.logger.Debug(ctx, "Creating PR watcher with Slack support")
			services.PRWatcher = factory.CreatePRWatcherWithSlack(
				clients.GitHubClient,
				r.config,
				clients.SlackNotifier,
			)
		} else {
			r.logger.Debug(ctx, "Creating standard PR watcher")
			services.PRWatcher = serviceFactory.CreatePRWatcher(
				clients.GitHubClient,
				r.config,
			)
		}
	} else {
		r.logger.Debug(ctx, "Creating PR watcher without Slack")
		services.PRWatcher = serviceFactory.CreatePRWatcher(
			clients.GitHubClient,
			r.config,
		)
	}

	// Phase 6: Create cleanup service
	r.logger.Debug(ctx, "Creating cleanup service")
	services.CleanupService = serviceFactory.CreateClosedIssueCleanupService(
		clients.GitHubClient,
		clients.TmuxClient,
		owner,
		repo,
		"soba",
		r.config.Workflow.ClosedIssueCleanupEnabled,
		time.Duration(r.config.Workflow.ClosedIssueCleanupInterval)*time.Second,
	)

	r.logger.Info(ctx, "Service dependencies resolved successfully")
	return services, nil
}

// createWorkspaceManager creates a workspace manager based on config
func (r *DependencyResolver) createWorkspaceManager(clients *ResolvedClients) GitWorkspaceManager {
	r.logger.Debug(context.Background(), "Creating workspace manager",
		logging.Field{Key: "workDir", Value: r.workDir},
		logging.Field{Key: "worktreeBasePath", Value: r.config.Git.WorktreeBasePath},
	)

	return serviceFactory.CreateGitWorkspaceManager(
		r.config,
		clients.GitClient,
	)
}

// parseRepository parses repository string to owner and repo
func parseRepository(repository string) (string, string) {
	parts := strings.Split(repository, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", repository
}

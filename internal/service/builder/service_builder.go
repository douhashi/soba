package builder

import (
	"context"
	"fmt"
	"os"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

// ServiceBuilder builds DaemonService with new logging system
type ServiceBuilder struct {
	config       *config.Config
	workDir      string
	logFactory   *logging.Factory
	errorHandler ErrorHandler
	resolver     *DependencyResolver
	clients      *Clients
	cliLogLevel  string // CLI log level flag
	verbose      bool   // CLI verbose flag
}

// NewServiceBuilder creates a new service builder with new logging system
func NewServiceBuilder(logFactory *logging.Factory) *ServiceBuilder {
	workDir, _ := os.Getwd()

	return &ServiceBuilder{
		workDir:      workDir,
		logFactory:   logFactory,
		errorHandler: NewProductionErrorHandler(logFactory.CreateComponentLogger("error-handler")),
	}
}

// WithConfig sets configuration
func (b *ServiceBuilder) WithConfig(cfg *config.Config) *ServiceBuilder {
	b.config = cfg
	return b
}

// WithWorkDir sets working directory
func (b *ServiceBuilder) WithWorkDir(workDir string) *ServiceBuilder {
	b.workDir = workDir
	return b
}

// WithErrorHandler sets error handling strategy
func (b *ServiceBuilder) WithErrorHandler(handler ErrorHandler) *ServiceBuilder {
	b.errorHandler = handler
	return b
}

// WithCLILogLevel sets CLI log level
func (b *ServiceBuilder) WithCLILogLevel(level string) *ServiceBuilder {
	b.cliLogLevel = level
	return b
}

// WithVerbose sets verbose flag
func (b *ServiceBuilder) WithVerbose(verbose bool) *ServiceBuilder {
	b.verbose = verbose
	return b
}

// Build creates and configures DaemonService with proper DI
func (b *ServiceBuilder) Build(ctx context.Context) (DaemonService, error) {
	// Use default config if not provided
	if b.config == nil {
		b.config = b.createDefaultConfig()
	}

	// Create component loggers
	builderLogger := b.logFactory.CreateComponentLogger("builder")
	builderLogger.Info(ctx, "Building daemon service",
		logging.Field{Key: "repository", Value: b.config.GitHub.Repository},
	)

	// Create dependency resolver with new logging
	resolver := NewDependencyResolver(b.config, b.workDir, b.logFactory, b.errorHandler)

	// Resolve clients
	clients, err := resolver.ResolveClients(ctx)
	if err != nil {
		builderLogger.Error(ctx, "Failed to resolve clients",
			logging.Field{Key: "error", Value: err.Error()},
		)
		return nil, fmt.Errorf("failed to resolve clients: %w", err)
	}

	// Resolve services
	services, err := resolver.ResolveServices(ctx, clients)
	if err != nil {
		builderLogger.Error(ctx, "Failed to resolve services",
			logging.Field{Key: "error", Value: err.Error()},
		)
		return nil, fmt.Errorf("failed to resolve services: %w", err)
	}

	builderLogger.Info(ctx, "Successfully built daemon service")

	// Create daemon service with resolved dependencies
	return serviceFactory.CreateDaemonServiceWithDependencies(
		b.workDir,
		services.IssueProcessor,
		services.IssueWatcher,
		services.PRWatcher,
		services.CleanupService,
		clients.TmuxClient,
	), nil
}

// BuildDefault builds with default configuration
func (b *ServiceBuilder) BuildDefault(ctx context.Context) error {
	// Use default config if not provided
	if b.config == nil {
		b.config = b.createDefaultConfig()
	}

	// Create dependency resolver
	b.resolver = NewDependencyResolver(b.config, b.workDir, b.logFactory, b.errorHandler)

	// Resolve clients
	clients, err := b.resolver.ResolveClients(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve clients: %w", err)
	}
	b.clients = clients

	return nil
}

// GetServiceFactory returns the service factory
func (b *ServiceBuilder) GetServiceFactory() ServiceFactory {
	return serviceFactory
}

// GetClients returns the resolved clients
func (b *ServiceBuilder) GetClients() *Clients {
	return b.clients
}

// GetConfig returns the configuration
func (b *ServiceBuilder) GetConfig() *config.Config {
	return b.config
}

// createDefaultConfig creates default configuration
func (b *ServiceBuilder) createDefaultConfig() *config.Config {
	return &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
			BaseBranch:       "main",
		},
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
	}
}

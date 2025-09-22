package builder

import (
	"context"
	"fmt"
	"os"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

// ServiceBuilderV2 builds DaemonService with new logging system
type ServiceBuilderV2 struct {
	config       *config.Config
	workDir      string
	logFactory   *logging.Factory
	errorHandler ErrorHandler
	resolver     *DependencyResolverV2
	clients      *Clients
}

// NewServiceBuilderV2 creates a new service builder with new logging system
func NewServiceBuilderV2(logFactory *logging.Factory) *ServiceBuilderV2 {
	workDir, _ := os.Getwd()

	return &ServiceBuilderV2{
		workDir:      workDir,
		logFactory:   logFactory,
		errorHandler: NewProductionErrorHandlerV2(logFactory.CreateComponentLogger("error-handler")),
	}
}

// WithConfig sets configuration
func (b *ServiceBuilderV2) WithConfig(cfg *config.Config) *ServiceBuilderV2 {
	b.config = cfg
	return b
}

// WithWorkDir sets working directory
func (b *ServiceBuilderV2) WithWorkDir(workDir string) *ServiceBuilderV2 {
	b.workDir = workDir
	return b
}

// WithErrorHandler sets error handling strategy
func (b *ServiceBuilderV2) WithErrorHandler(handler ErrorHandler) *ServiceBuilderV2 {
	b.errorHandler = handler
	return b
}

// Build creates and configures DaemonService with proper DI
func (b *ServiceBuilderV2) Build(ctx context.Context) (DaemonService, error) {
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
	resolver := NewDependencyResolverV2(b.config, b.workDir, b.logFactory, b.errorHandler)

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
func (b *ServiceBuilderV2) BuildDefault(ctx context.Context) error {
	// Use default config if not provided
	if b.config == nil {
		b.config = b.createDefaultConfig()
	}

	// Create dependency resolver
	b.resolver = NewDependencyResolverV2(b.config, b.workDir, b.logFactory, b.errorHandler)

	// Resolve clients
	clients, err := b.resolver.ResolveClients(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve clients: %w", err)
	}
	b.clients = clients

	return nil
}

// GetServiceFactory returns the service factory
func (b *ServiceBuilderV2) GetServiceFactory() ServiceFactory {
	return serviceFactory
}

// GetClients returns the resolved clients
func (b *ServiceBuilderV2) GetClients() *Clients {
	return b.clients
}

// GetConfig returns the configuration
func (b *ServiceBuilderV2) GetConfig() *config.Config {
	return b.config
}

// createDefaultConfig creates default configuration
func (b *ServiceBuilderV2) createDefaultConfig() *config.Config {
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

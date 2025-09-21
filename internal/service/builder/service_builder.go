package builder

import (
	"fmt"
	"os"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logger"
)

// ServiceBuilder builds DaemonService with proper dependency management
type ServiceBuilder struct {
	config       *config.Config
	workDir      string
	logger       logger.Logger
	errorHandler ErrorHandler
}

// NewServiceBuilder creates a new service builder
func NewServiceBuilder() *ServiceBuilder {
	workDir, _ := os.Getwd()
	log := logger.NewLogger(logger.GetLogger())

	return &ServiceBuilder{
		workDir:      workDir,
		logger:       log,
		errorHandler: NewProductionErrorHandler(log),
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

// WithLogger sets logger
func (b *ServiceBuilder) WithLogger(log logger.Logger) *ServiceBuilder {
	b.logger = log
	return b
}

// WithErrorHandler sets error handling strategy
func (b *ServiceBuilder) WithErrorHandler(handler ErrorHandler) *ServiceBuilder {
	b.errorHandler = handler
	return b
}

// Build creates and configures DaemonService
func (b *ServiceBuilder) Build() (DaemonService, error) {
	// Use default config if not provided
	if b.config == nil {
		b.config = b.createDefaultConfig()
	}

	// Resolve dependencies
	resolver := NewDependencyResolver(b.config, b.workDir, b.logger, b.errorHandler)

	clients, err := resolver.ResolveClients()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve clients: %w", err)
	}

	services, err := resolver.ResolveServices(clients)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve services: %w", err)
	}

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
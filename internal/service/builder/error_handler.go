package builder

import (
	"context"
	"fmt"

	"github.com/douhashi/soba/pkg/logging"
)

// ProductionErrorHandler implements production error handling with new logging
type ProductionErrorHandler struct {
	logger logging.Logger
}

// NewProductionErrorHandler creates a new production error handler with new logging
func NewProductionErrorHandler(logger logging.Logger) ErrorHandler {
	return &ProductionErrorHandler{
		logger: logger,
	}
}

// HandleGitHubClientError handles GitHub client initialization errors
func (h *ProductionErrorHandler) HandleGitHubClientError(err error) (GitHubClientInterface, error) {
	ctx := context.Background()
	h.logger.Error(ctx, "GitHub client initialization failed",
		logging.Field{Key: "error", Value: err.Error()},
	)
	return nil, fmt.Errorf("GitHub client required: %w", err)
}

// HandleGitClientError handles Git client initialization errors
func (h *ProductionErrorHandler) HandleGitClientError(workDir string, err error) (*MockGitClient, error) {
	ctx := context.Background()
	h.logger.Warn(ctx, "Git client initialization failed, using mock",
		logging.Field{Key: "workDir", Value: workDir},
		logging.Field{Key: "error", Value: err.Error()},
	)
	// Return a mock git client for non-critical operations
	return NewMockGitClient(), nil
}

// ShouldContinueOnError determines if processing should continue after an error
func (h *ProductionErrorHandler) ShouldContinueOnError(component string, err error) bool {
	ctx := context.Background()

	// Git client errors are recoverable
	if component == "git_client" {
		h.logger.Info(ctx, "Continuing with degraded functionality",
			logging.Field{Key: "component", Value: component},
			logging.Field{Key: "error", Value: err.Error()},
		)
		return true
	}

	// Other components are critical
	h.logger.Error(ctx, "Critical component failure",
		logging.Field{Key: "component", Value: component},
		logging.Field{Key: "error", Value: err.Error()},
	)
	return false
}

// LogError logs an error with context
func (h *ProductionErrorHandler) LogError(ctx context.Context, msg string, err error) {
	h.logger.Error(ctx, msg,
		logging.Field{Key: "error", Value: err.Error()},
	)
}

// LogWarning logs a warning with context
func (h *ProductionErrorHandler) LogWarning(ctx context.Context, msg string) {
	h.logger.Warn(ctx, msg)
}

// LogInfo logs an info message with context
func (h *ProductionErrorHandler) LogInfo(ctx context.Context, msg string) {
	h.logger.Info(ctx, msg)
}

package builder

import (
	"github.com/douhashi/soba/pkg/logger"
)

// ErrorHandler defines strategy for handling initialization errors
type ErrorHandler interface {
	HandleGitClientError(workDir string, err error) (*MockGitClient, error)
	HandleGitHubClientError(err error) (GitHubClientInterface, error)
	ShouldContinueOnError(component string, err error) bool
}

// ProductionErrorHandler provides fallback strategies for production
type ProductionErrorHandler struct {
	logger logger.Logger
}

// NewProductionErrorHandler creates production error handler
func NewProductionErrorHandler(log logger.Logger) ErrorHandler {
	return &ProductionErrorHandler{logger: log}
}

// HandleGitClientError provides fallback for git client creation failure
func (h *ProductionErrorHandler) HandleGitClientError(workDir string, err error) (*MockGitClient, error) {
	h.logger.Warn("Git client initialization failed, using mock", "error", err)
	return NewMockGitClient(), nil
}

// HandleGitHubClientError handles GitHub client creation failure
func (h *ProductionErrorHandler) HandleGitHubClientError(err error) (GitHubClientInterface, error) {
	h.logger.Error("GitHub client initialization failed", "error", err)
	return nil, err
}

// ShouldContinueOnError determines if service should continue despite error
func (h *ProductionErrorHandler) ShouldContinueOnError(component string, err error) bool {
	criticalComponents := []string{"github_client", "tmux_client"}
	for _, critical := range criticalComponents {
		if component == critical {
			return false
		}
	}
	return true
}

// MockGitClient provides mock implementation for git operations
type MockGitClient struct{}

// NewMockGitClient creates a new mock git client
func NewMockGitClient() *MockGitClient {
	return &MockGitClient{}
}

// CreateWorktree implements git.Client interface
func (m *MockGitClient) CreateWorktree(path, branchName string) error {
	return nil
}

// RemoveWorktree implements git.Client interface
func (m *MockGitClient) RemoveWorktree(path string) error {
	return nil
}

// WorktreeExists implements git.Client interface
func (m *MockGitClient) WorktreeExists(path string) bool {
	return false
}

// CreateBranch implements git.Client interface
func (m *MockGitClient) CreateBranch(branchName, baseBranch string) error {
	return nil
}

// BranchExists implements git.Client interface
func (m *MockGitClient) BranchExists(branchName string) bool {
	return false
}

// DeleteBranch implements git.Client interface
func (m *MockGitClient) DeleteBranch(branchName string) error {
	return nil
}

// GetCurrentBranch implements git.Client interface
func (m *MockGitClient) GetCurrentBranch() (string, error) {
	return "main", nil
}

// SwitchBranch implements git.Client interface
func (m *MockGitClient) SwitchBranch(branchName string) error {
	return nil
}

// FetchOrigin implements git.Client interface
func (m *MockGitClient) FetchOrigin() error {
	return nil
}

// PullOrigin implements git.Client interface
func (m *MockGitClient) PullOrigin(branchName string) error {
	return nil
}

// PushOrigin implements git.Client interface
func (m *MockGitClient) PushOrigin(branchName string, force bool) error {
	return nil
}

// CommitChanges implements git.Client interface
func (m *MockGitClient) CommitChanges(message string) error {
	return nil
}

// HasUncommittedChanges implements git.Client interface
func (m *MockGitClient) HasUncommittedChanges() (bool, error) {
	return false, nil
}

// GetRemoteURL implements git.Client interface
func (m *MockGitClient) GetRemoteURL(remote string) (string, error) {
	return "https://github.com/mock/repo", nil
}

// GetRepository implements git.Client interface
func (m *MockGitClient) GetRepository() (string, error) {
	return "mock/repo", nil
}

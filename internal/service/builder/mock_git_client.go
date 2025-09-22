package builder

import "github.com/douhashi/soba/internal/infra/git"

// MockGitClient is a mock implementation of git client
type MockGitClient struct {
	*git.Client
}

// NewMockGitClient creates a new mock git client
func NewMockGitClient() *MockGitClient {
	return &MockGitClient{}
}

// GetCurrentBranch returns mock current branch
func (m *MockGitClient) GetCurrentBranch() (string, error) {
	return "main", nil
}

// CreateBranch creates a mock branch
func (m *MockGitClient) CreateBranch(branchName string, baseBranch string) error {
	return nil
}

// SwitchBranch switches to a mock branch
func (m *MockGitClient) SwitchBranch(branchName string) error {
	return nil
}

// DeleteBranch deletes a mock branch
func (m *MockGitClient) DeleteBranch(branchName string, force bool) error {
	return nil
}

// BranchExists checks if a mock branch exists
func (m *MockGitClient) BranchExists(branchName string) (bool, error) {
	return false, nil
}

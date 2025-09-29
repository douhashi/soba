package service

// MockGitWorkspaceManager はGitWorkspaceManagerのモック実装
// 実際のワークスペース管理は行わず、テスト用の動作のみを提供する
type MockGitWorkspaceManager struct{}

// NewMockGitWorkspaceManager は新しいMockGitWorkspaceManagerを作成する
func NewMockGitWorkspaceManager() GitWorkspaceManager {
	return &MockGitWorkspaceManager{}
}

// PrepareWorkspace は成功を返す（モック実装）
func (m *MockGitWorkspaceManager) PrepareWorkspace(issueNumber int) error {
	return nil
}

// CleanupWorkspace は成功を返す（モック実装）
func (m *MockGitWorkspaceManager) CleanupWorkspace(issueNumber int) error {
	return nil
}

// MockGitClient はGitClientのモック実装
type MockGitClient struct{}

// NewMockGitClient は新しいMockGitClientを作成する
func NewMockGitClient() GitClient {
	return &MockGitClient{}
}

// CreateWorktree は成功を返す（モック実装）
func (m *MockGitClient) CreateWorktree(worktreePath, branchName, baseBranch string) error {
	return nil
}

// RemoveWorktree は成功を返す（モック実装）
func (m *MockGitClient) RemoveWorktree(worktreePath string) error {
	return nil
}

// UpdateBaseBranch は成功を返す（モック実装）
func (m *MockGitClient) UpdateBaseBranch(branch string) error {
	return nil
}

// WorktreeExists は常にfalseを返す（モック実装）
func (m *MockGitClient) WorktreeExists(worktreePath string) bool {
	return false
}

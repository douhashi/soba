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

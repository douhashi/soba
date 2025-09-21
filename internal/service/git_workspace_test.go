package service

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
)

// mockGitClient is a mock implementation of git.Client
type mockGitClient struct {
	mock.Mock
}

func (m *mockGitClient) CreateWorktree(worktreePath, branchName, baseBranch string) error {
	args := m.Called(worktreePath, branchName, baseBranch)
	return args.Error(0)
}

func (m *mockGitClient) RemoveWorktree(worktreePath string) error {
	args := m.Called(worktreePath)
	return args.Error(0)
}

func (m *mockGitClient) UpdateBaseBranch(branch string) error {
	args := m.Called(branch)
	return args.Error(0)
}

func (m *mockGitClient) WorktreeExists(worktreePath string) bool {
	args := m.Called(worktreePath)
	return args.Bool(0)
}

func TestNewGitWorkspaceManager(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
			BaseBranch:       "main",
		},
	}
	mockClient := new(mockGitClient)

	manager := NewGitWorkspaceManager(cfg, mockClient)
	assert.NotNil(t, manager)
}

func TestGitWorkspaceManager_PrepareWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		setupMocks  func(*mockGitClient)
		wantErr     bool
	}{
		{
			name:        "Successfully prepare new workspace",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(false)
				mc.On("UpdateBaseBranch", "main").Return(nil)
				mc.On("CreateWorktree", expectedPath, "soba/33", "main").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "Workspace already exists",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(true)
			},
			wantErr: false,
		},
		{
			name:        "Failed to update base branch",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(false)
				mc.On("UpdateBaseBranch", "main").Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:        "Failed to create worktree",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(false)
				mc.On("UpdateBaseBranch", "main").Return(nil)
				mc.On("CreateWorktree", expectedPath, "soba/33", "main").Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:        "Invalid issue number",
			issueNumber: 0,
			setupMocks:  func(mc *mockGitClient) {},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Git: config.GitConfig{
					WorktreeBasePath: ".git/soba/worktrees",
					BaseBranch:       "main",
				},
			}
			mockClient := new(mockGitClient)
			tt.setupMocks(mockClient)

			manager := NewGitWorkspaceManager(cfg, mockClient)
			err := manager.PrepareWorkspace(tt.issueNumber)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockClient.AssertExpectations(t)
		})
	}
}

func TestGitWorkspaceManager_CleanupWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		setupMocks  func(*mockGitClient)
		wantErr     bool
	}{
		{
			name:        "Successfully cleanup workspace",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(true)
				mc.On("RemoveWorktree", expectedPath).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "Workspace does not exist",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(false)
			},
			wantErr: false,
		},
		{
			name:        "Failed to remove worktree",
			issueNumber: 33,
			setupMocks: func(mc *mockGitClient) {
				expectedPath := filepath.Join(".git/soba/worktrees", "issue-33")
				mc.On("WorktreeExists", expectedPath).Return(true)
				mc.On("RemoveWorktree", expectedPath).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:        "Invalid issue number",
			issueNumber: 0,
			setupMocks:  func(mc *mockGitClient) {},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Git: config.GitConfig{
					WorktreeBasePath: ".git/soba/worktrees",
					BaseBranch:       "main",
				},
			}
			mockClient := new(mockGitClient)
			tt.setupMocks(mockClient)

			manager := NewGitWorkspaceManager(cfg, mockClient)
			err := manager.CleanupWorkspace(tt.issueNumber)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockClient.AssertExpectations(t)
		})
	}
}

func TestGitWorkspaceManager_GetWorkspacePath(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
			BaseBranch:       "main",
		},
	}
	mockClient := new(mockGitClient)
	manager := NewGitWorkspaceManager(cfg, mockClient).(*gitWorkspaceManager)

	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "Valid issue number",
			issueNumber: 33,
			want:        filepath.Join(".git/soba/worktrees", "issue-33"),
		},
		{
			name:        "Large issue number",
			issueNumber: 999,
			want:        filepath.Join(".git/soba/worktrees", "issue-999"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.getWorkspacePath(tt.issueNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitWorkspaceManager_GetBranchName(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
			BaseBranch:       "main",
		},
	}
	mockClient := new(mockGitClient)
	manager := NewGitWorkspaceManager(cfg, mockClient).(*gitWorkspaceManager)

	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "Valid issue number",
			issueNumber: 33,
			want:        "soba/33",
		},
		{
			name:        "Large issue number",
			issueNumber: 999,
			want:        "soba/999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.getBranchName(tt.issueNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitWorkspaceManager_WithCustomBaseBranch(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
			BaseBranch:       "develop",
		},
	}
	mockClient := new(mockGitClient)

	expectedPath := filepath.Join(".git/soba/worktrees", "issue-42")
	mockClient.On("WorktreeExists", expectedPath).Return(false)
	mockClient.On("UpdateBaseBranch", "develop").Return(nil)
	mockClient.On("CreateWorktree", expectedPath, "soba/42", "develop").Return(nil)

	manager := NewGitWorkspaceManager(cfg, mockClient)
	err := manager.PrepareWorkspace(42)

	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

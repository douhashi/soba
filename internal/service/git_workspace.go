package service

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/douhashi/soba/internal/config"
)

// GitWorkspaceManager はGitワークスペース管理のインターフェース
// Issue毎のworktree作成やブランチ管理を行う
type GitWorkspaceManager interface {
	// PrepareWorkspace は指定されたissue番号に対応するワークスペースを準備する
	PrepareWorkspace(issueNumber int) error

	// CleanupWorkspace は指定されたissue番号に対応するワークスペースをクリーンアップする
	CleanupWorkspace(issueNumber int) error
}

// GitClient はGit操作を行うクライアントのインターフェース
type GitClient interface {
	CreateWorktree(worktreePath, branchName, baseBranch string) error
	RemoveWorktree(worktreePath string) error
	UpdateBaseBranch(branch string) error
	WorktreeExists(worktreePath string) bool
}

// gitWorkspaceManager は GitWorkspaceManager の実装
type gitWorkspaceManager struct {
	config    *config.Config
	gitClient GitClient
}

// NewGitWorkspaceManager は新しいGitWorkspaceManagerを作成する
func NewGitWorkspaceManager(cfg *config.Config, gitClient GitClient) GitWorkspaceManager {
	return &gitWorkspaceManager{
		config:    cfg,
		gitClient: gitClient,
	}
}

// PrepareWorkspace は指定されたissue番号に対応するワークスペースを準備する
func (g *gitWorkspaceManager) PrepareWorkspace(issueNumber int) error {
	if issueNumber <= 0 {
		return errors.New("invalid issue number")
	}

	worktreePath := g.getWorkspacePath(issueNumber)
	branchName := g.getBranchName(issueNumber)

	// Check if worktree already exists
	if g.gitClient.WorktreeExists(worktreePath) {
		// Worktree already exists, nothing to do
		return nil
	}

	// Update base branch to latest
	if err := g.gitClient.UpdateBaseBranch(g.config.Git.BaseBranch); err != nil {
		return fmt.Errorf("failed to update base branch: %w", err)
	}

	// Create new worktree with branch
	if err := g.gitClient.CreateWorktree(worktreePath, branchName, g.config.Git.BaseBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// CleanupWorkspace は指定されたissue番号に対応するワークスペースをクリーンアップする
func (g *gitWorkspaceManager) CleanupWorkspace(issueNumber int) error {
	if issueNumber <= 0 {
		return errors.New("invalid issue number")
	}

	worktreePath := g.getWorkspacePath(issueNumber)

	// Check if worktree exists
	if !g.gitClient.WorktreeExists(worktreePath) {
		// Worktree does not exist, nothing to cleanup
		return nil
	}

	// Remove worktree
	if err := g.gitClient.RemoveWorktree(worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// getWorkspacePath はissue番号に対応するワークスペースのパスを生成する
func (g *gitWorkspaceManager) getWorkspacePath(issueNumber int) string {
	return filepath.Join(g.config.Git.WorktreeBasePath, fmt.Sprintf("issue-%d", issueNumber))
}

// getBranchName はissue番号に対応するブランチ名を生成する
func (g *gitWorkspaceManager) getBranchName(issueNumber int) string {
	return fmt.Sprintf("soba/%d", issueNumber)
}

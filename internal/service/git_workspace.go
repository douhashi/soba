package service

// GitWorkspaceManager はGitワークスペース管理のインターフェース
// Issue毎のworktree作成やブランチ管理を行う（実装は後続Issue）
type GitWorkspaceManager interface {
	// PrepareWorkspace は指定されたissue番号に対応するワークスペースを準備する
	PrepareWorkspace(issueNumber int) error

	// CleanupWorkspace は指定されたissue番号に対応するワークスペースをクリーンアップする
	CleanupWorkspace(issueNumber int) error
}

// gitWorkspaceManager は GitWorkspaceManager の実装
type gitWorkspaceManager struct {
	workDir string
}

// NewGitWorkspaceManager は新しいGitWorkspaceManagerを作成する
func NewGitWorkspaceManager(workDir string) GitWorkspaceManager {
	return &gitWorkspaceManager{
		workDir: workDir,
	}
}

// PrepareWorkspace は指定されたissue番号に対応するワークスペースを準備する
func (g *gitWorkspaceManager) PrepareWorkspace(issueNumber int) error {
	// TODO: 実装は後続Issueで行う
	return nil
}

// CleanupWorkspace は指定されたissue番号に対応するワークスペースをクリーンアップする
func (g *gitWorkspaceManager) CleanupWorkspace(issueNumber int) error {
	// TODO: 実装は後続Issueで行う
	return nil
}

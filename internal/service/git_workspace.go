package service

// GitWorkspaceManager はGitワークスペース管理のインターフェース
// Issue毎のworktree作成やブランチ管理を行う（実装は後続Issue）
type GitWorkspaceManager interface {
	// PrepareWorkspace は指定されたissue番号に対応するワークスペースを準備する
	PrepareWorkspace(issueNumber int) error

	// CleanupWorkspace は指定されたissue番号に対応するワークスペースをクリーンアップする
	CleanupWorkspace(issueNumber int) error
}

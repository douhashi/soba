//go:build integration
// +build integration

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Integration(t *testing.T) {
	// 統合テストはスキップ可能にする
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 一時ディレクトリを作成
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	// テスト用のGitリポジトリを作成
	setupTestGitRepo(t, repoDir)

	// Clientを作成
	client, err := NewClient(repoDir)
	require.NoError(t, err, "Failed to create Git client")

	t.Run("Full worktree lifecycle", func(t *testing.T) {
		worktreePath := filepath.Join(tempDir, "worktree-lifecycle")
		branchName := "soba/99"
		baseBranch := "main"

		// 1. ベースブランチを最新化
		err := client.UpdateBaseBranch(baseBranch)
		assert.NoError(t, err, "Failed to update base branch")

		// 2. Worktreeが存在しないことを確認
		exists := client.WorktreeExists(worktreePath)
		assert.False(t, exists, "Worktree should not exist initially")

		// 3. Worktreeを作成
		err = client.CreateWorktree(worktreePath, branchName, baseBranch)
		assert.NoError(t, err, "Failed to create worktree")

		// 4. Worktreeが存在することを確認
		exists = client.WorktreeExists(worktreePath)
		assert.True(t, exists, "Worktree should exist after creation")

		// 5. Worktree内でファイルを作成し、変更が反映されることを確認
		testFile := filepath.Join(worktreePath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		assert.NoError(t, err, "Failed to create test file")

		// ファイルが存在することを確認
		_, err = os.Stat(testFile)
		assert.NoError(t, err, "Test file should exist")

		// 6. Worktreeを削除
		err = client.RemoveWorktree(worktreePath)
		assert.NoError(t, err, "Failed to remove worktree")

		// 7. Worktreeが存在しないことを確認
		exists = client.WorktreeExists(worktreePath)
		assert.False(t, exists, "Worktree should not exist after removal")

		// Worktreeディレクトリも削除されていることを確認
		_, err = os.Stat(worktreePath)
		assert.True(t, os.IsNotExist(err), "Worktree directory should be removed")
	})

	t.Run("Multiple worktrees", func(t *testing.T) {
		// 複数のWorktreeを同時に作成・管理できることを確認
		worktree1 := filepath.Join(tempDir, "worktree-1")
		worktree2 := filepath.Join(tempDir, "worktree-2")

		// 両方のWorktreeを作成
		err := client.CreateWorktree(worktree1, "soba/1", "main")
		assert.NoError(t, err, "Failed to create first worktree")

		err = client.CreateWorktree(worktree2, "soba/2", "main")
		assert.NoError(t, err, "Failed to create second worktree")

		// 両方存在することを確認
		assert.True(t, client.WorktreeExists(worktree1), "First worktree should exist")
		assert.True(t, client.WorktreeExists(worktree2), "Second worktree should exist")

		// 一つずつ削除
		err = client.RemoveWorktree(worktree1)
		assert.NoError(t, err, "Failed to remove first worktree")

		assert.False(t, client.WorktreeExists(worktree1), "First worktree should be removed")
		assert.True(t, client.WorktreeExists(worktree2), "Second worktree should still exist")

		err = client.RemoveWorktree(worktree2)
		assert.NoError(t, err, "Failed to remove second worktree")

		assert.False(t, client.WorktreeExists(worktree2), "Second worktree should be removed")
	})

	t.Run("Error handling", func(t *testing.T) {
		// 存在しないWorktreeの削除
		err := client.RemoveWorktree("/nonexistent/path")
		assert.Error(t, err, "Should error when removing non-existent worktree")
		assert.Contains(t, err.Error(), "worktree not found")

		// 同じブランチ名で再作成しようとした場合
		worktreePath := filepath.Join(tempDir, "worktree-duplicate")
		err = client.CreateWorktree(worktreePath, "soba/duplicate", "main")
		assert.NoError(t, err, "Should create first worktree")

		// 同じブランチで別の場所にWorktreeを作成（これはエラーになるはず）
		worktreePath2 := filepath.Join(tempDir, "worktree-duplicate2")
		err = client.CreateWorktree(worktreePath2, "soba/duplicate", "main")
		assert.Error(t, err, "Should not allow duplicate branch name")

		// クリーンアップ
		_ = client.RemoveWorktree(worktreePath)
		_ = client.RemoveWorktree(worktreePath2)
	})
}

func TestClient_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// 一時ディレクトリを作成
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "concurrent-repo")

	// テスト用のGitリポジトリを作成
	setupTestGitRepo(t, repoDir)

	// Clientを作成
	client, err := NewClient(repoDir)
	require.NoError(t, err)

	// 並行してWorktreeを作成
	done := make(chan bool, 3)

	for i := 1; i <= 3; i++ {
		go func(id int) {
			worktreePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-worktree-%d", id))
			branchName := fmt.Sprintf("soba/concurrent-%d", id)

			err := client.CreateWorktree(worktreePath, branchName, "main")
			assert.NoError(t, err, "Failed to create worktree %d", id)

			exists := client.WorktreeExists(worktreePath)
			assert.True(t, exists, "Worktree %d should exist", id)

			done <- true
		}(i)
	}

	// すべての goroutine が完了するのを待つ
	for i := 0; i < 3; i++ {
		<-done
	}

	// クリーンアップ
	for i := 1; i <= 3; i++ {
		worktreePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-worktree-%d", i))
		_ = client.RemoveWorktree(worktreePath)
	}
}

// setupTestGitRepo は統合テスト用の実際のGitリポジトリを作成
func setupTestGitRepo(t *testing.T, repoDir string) {
	t.Helper()

	// ディレクトリ作成
	err := os.MkdirAll(repoDir, 0755)
	require.NoError(t, err)

	// Git初期化
	cmd := exec.Command("git", "init", repoDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to init repo: %s", string(output))

	// Git設定
	cmd = exec.Command("git", "-C", repoDir, "config", "user.email", "test@example.com")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("git", "-C", repoDir, "config", "user.name", "Test User")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	// README作成
	readmePath := filepath.Join(repoDir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repository\n\nThis is a test repository for integration testing.\n"), 0644)
	require.NoError(t, err)

	// .gitignore作成
	gitignorePath := filepath.Join(repoDir, ".gitignore")
	err = os.WriteFile(gitignorePath, []byte("*.tmp\n.DS_Store\n"), 0644)
	require.NoError(t, err)

	// 初回コミット
	cmd = exec.Command("git", "-C", repoDir, "add", ".")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("git", "-C", repoDir, "commit", "-m", "Initial commit")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to commit: %s", string(output))

	// developブランチも作成しておく（テスト用）
	cmd = exec.Command("git", "-C", repoDir, "branch", "develop")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)
}

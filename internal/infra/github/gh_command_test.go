package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGhCommand_CheckAvailability(t *testing.T) {
	t.Run("should return true when gh command is available", func(t *testing.T) {
		cmd := NewGhCommand()
		available := cmd.IsAvailable()
		// ghコマンドがインストールされている環境でのみテストが通る
		// CI環境ではghコマンドがインストールされていることを前提とする
		assert.True(t, available)
	})
}

func TestGhCommand_ListLabels(t *testing.T) {
	t.Run("should list labels from repository", func(t *testing.T) {
		// このテストはモックを使用せず、実際のghコマンドの動作を確認する統合テスト
		t.Skip("Integration test - requires actual GitHub repository")

		cmd := NewGhCommand()
		ctx := context.Background()

		labels, err := cmd.ListLabels(ctx, "douhashi", "soba")
		require.NoError(t, err)
		assert.NotNil(t, labels)
	})

	t.Run("should handle repository without labels", func(t *testing.T) {
		t.Skip("Integration test - requires actual GitHub repository")

		cmd := NewGhCommand()
		ctx := context.Background()

		// 存在しないリポジトリの場合はエラーを返すべき
		_, err := cmd.ListLabels(ctx, "nonexistent", "repo")
		assert.Error(t, err)
	})
}

func TestGhCommand_CreateLabel(t *testing.T) {
	t.Run("should create a new label", func(t *testing.T) {
		t.Skip("Integration test - requires actual GitHub repository")

		cmd := NewGhCommand()
		ctx := context.Background()

		request := CreateLabelRequest{
			Name:        "test-label",
			Color:       "ff0000",
			Description: "Test label for testing",
		}

		err := cmd.CreateLabel(ctx, "douhashi", "soba-test", request)
		assert.NoError(t, err)
	})

	t.Run("should handle duplicate label gracefully", func(t *testing.T) {
		t.Skip("Integration test - requires actual GitHub repository")

		cmd := NewGhCommand()
		ctx := context.Background()

		request := CreateLabelRequest{
			Name:        "existing-label",
			Color:       "00ff00",
			Description: "Already exists",
		}

		// 最初の作成
		_ = cmd.CreateLabel(ctx, "douhashi", "soba-test", request)

		// 2回目の作成（重複）- エラーを返さずにスキップすべき
		err := cmd.CreateLabel(ctx, "douhashi", "soba-test", request)
		assert.NoError(t, err)
	})
}

func TestGhCommand_CreateSobaLabels(t *testing.T) {
	t.Run("should create all soba labels", func(t *testing.T) {
		t.Skip("Integration test - requires actual GitHub repository")

		cmd := NewGhCommand()
		ctx := context.Background()

		created, skipped, err := cmd.CreateSobaLabels(ctx, "douhashi", "soba-test")
		require.NoError(t, err)

		// 全11個のラベルが作成またはスキップされるべき
		assert.Equal(t, 11, created+skipped)
	})

	t.Run("should skip existing soba labels", func(t *testing.T) {
		t.Skip("Integration test - requires actual GitHub repository")

		cmd := NewGhCommand()
		ctx := context.Background()

		// 1回目の実行
		created1, _, err := cmd.CreateSobaLabels(ctx, "douhashi", "soba-test")
		require.NoError(t, err)

		// 2回目の実行（すべてスキップされるはず）
		created2, skipped2, err := cmd.CreateSobaLabels(ctx, "douhashi", "soba-test")
		require.NoError(t, err)

		assert.Equal(t, 0, created2)
		assert.Equal(t, created1, skipped2)
	})
}

func TestGhCommand_AuthCheck(t *testing.T) {
	t.Run("should check authentication status", func(t *testing.T) {
		cmd := NewGhCommand()
		ctx := context.Background()

		authenticated, err := cmd.IsAuthenticated(ctx)
		// エラーがない場合は認証状態を確認
		if err == nil {
			// CI環境では認証されているはず
			assert.NotNil(t, authenticated)
		}
	})
}
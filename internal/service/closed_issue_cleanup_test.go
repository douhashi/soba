package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/logging"
)

func TestClosedIssueCleanupService_CleanupCycleLogs(t *testing.T) {
	t.Run("cleanupOnce開始時と完了時にINFOログが出力される", func(t *testing.T) {
		// GitHub client mock (nil to skip GitHub API calls)
		var githubClient *github.ClientImpl = nil

		// Tmux client mock
		tmuxClient := &MockTmuxClient{}
		tmuxClient.On("WindowExists", "soba", "issue-1").Return(true, nil)
		tmuxClient.On("WindowExists", "soba", "issue-2").Return(false, nil)
		tmuxClient.On("DeleteWindow", "soba", "issue-1").Return(nil)

		// Service作成
		service := NewClosedIssueCleanupService(
			githubClient,
			tmuxClient,
			"owner",
			"repo",
			"soba",
			true,
			60,
		)

		// Mock logger設定
		mockLogger := logging.NewMockLogger()
		service.SetLogger(mockLogger)

		ctx := context.Background()
		err := service.cleanupOnce(ctx)
		require.NoError(t, err)

		// "Starting cleanup of closed issues" がDEBUGからINFOに変更されたことを確認
		foundStartLog := false
		for _, msg := range mockLogger.Messages {
			if msg.Message == "Starting cleanup of closed issues" && msg.Level == "INFO" {
				foundStartLog = true
				break
			}
		}
		assert.True(t, foundStartLog, "expected 'Starting cleanup of closed issues' INFO log")

		// "Cleanup of closed issues completed" INFOログが追加されたことを確認
		foundCompleteLog := false
		for _, msg := range mockLogger.Messages {
			if msg.Message == "Cleanup of closed issues completed" && msg.Level == "INFO" {
				foundCompleteLog = true
				break
			}
		}
		assert.True(t, foundCompleteLog, "expected 'Cleanup of closed issues completed' INFO log")
	})
}

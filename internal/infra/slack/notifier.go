package slack

import (
	"context"
	"fmt"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

type SlackClient interface {
	SendMessage(message string) error
}

type Notifier struct {
	client SlackClient
	config *config.SlackConfig
	logger logging.Logger
	async  bool // テスト用
}

func NewNotifier(client SlackClient, config *config.SlackConfig, logger logging.Logger) *Notifier {
	return &Notifier{
		client: client,
		config: config,
		logger: logger,
		async:  true, // デフォルトは非同期
	}
}

func NewSyncNotifier(client SlackClient, config *config.SlackConfig, logger logging.Logger) *Notifier {
	return &Notifier{
		client: client,
		config: config,
		logger: logger,
		async:  false, // テスト用同期版
	}
}

func (n *Notifier) NotifyPhaseStart(phase string, issueNumber int) error {
	if !n.config.NotificationsEnabled {
		return nil
	}

	message := fmt.Sprintf("🚀 フェーズ開始: %s\nIssue #%d", phase, issueNumber)
	return n.sendAsync(message)
}

func (n *Notifier) NotifyPRMerged(prNumber, issueNumber int) error {
	if !n.config.NotificationsEnabled {
		return nil
	}

	message := fmt.Sprintf("✅ PR マージ完了\nPR #%d (Issue #%d)", prNumber, issueNumber)
	return n.sendAsync(message)
}

func (n *Notifier) NotifyError(title, errorMessage string) error {
	if !n.config.NotificationsEnabled {
		return nil
	}

	message := fmt.Sprintf("❌ エラー: %s\n%s", title, errorMessage)
	return n.sendAsync(message)
}

func (n *Notifier) sendAsync(message string) error {
	if n.async {
		go func() {
			if err := n.client.SendMessage(message); err != nil {
				n.logger.Warn(context.Background(), "Failed to send Slack notification",
					logging.Field{Key: "error", Value: err.Error()},
					logging.Field{Key: "message", Value: message},
				)
			} else {
				n.logger.Debug(context.Background(), "Slack notification sent successfully",
					logging.Field{Key: "message", Value: message},
				)
			}
		}()
		return nil
	} else {
		// 同期実行（テスト用）
		if err := n.client.SendMessage(message); err != nil {
			n.logger.Warn(context.Background(), "Failed to send Slack notification",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "message", Value: message},
			)
		} else {
			n.logger.Debug(context.Background(), "Slack notification sent successfully",
				logging.Field{Key: "message", Value: message},
			)
		}
		return nil
	}
}

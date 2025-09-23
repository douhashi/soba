package slack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

var (
	instance Manager
	once     sync.Once
)

// SlackManager implements Manager interface
type SlackManager struct {
	client SlackClient
	config config.SlackConfig
	logger logging.Logger
}

// Initialize initializes the global Slack manager based on config
func Initialize(cfg *config.Config, logger logging.Logger) {
	once.Do(func() {
		if !cfg.Slack.NotificationsEnabled || cfg.Slack.WebhookURL == "" {
			// Use NoOpManager when Slack is disabled
			instance = &NoOpManager{}
			logger.Info(context.Background(), "Slack notifications disabled")
			return
		}

		// Use default timeout of 30 seconds
		timeout := 30 * time.Second
		client := NewClient(cfg.Slack.WebhookURL, timeout)
		instance = &SlackManager{
			client: client,
			config: cfg.Slack,
			logger: logger,
		}
		logger.Info(context.Background(), "Slack notifications enabled")
	})
}

// GetManager returns the global Slack manager instance
func GetManager() Manager {
	if instance == nil {
		// Return NoOpManager if not initialized
		return &NoOpManager{}
	}
	return instance
}

// Reset resets the singleton (for testing)
func Reset() {
	instance = nil
	once = sync.Once{}
}

// Implementation methods
func (s *SlackManager) NotifyPhaseStart(phase string, issueNumber int) {
	message := fmt.Sprintf("🚀 フェーズ開始: %s\nIssue #%d", phase, issueNumber)
	s.sendAsync(message)
}

func (s *SlackManager) NotifyPRMerged(prNumber, issueNumber int) {
	message := fmt.Sprintf("✅ PR マージ完了\nPR #%d (Issue #%d)", prNumber, issueNumber)
	s.sendAsync(message)
}

func (s *SlackManager) NotifyError(title, errorMessage string) {
	message := fmt.Sprintf("❌ エラー: %s\n%s", title, errorMessage)
	s.sendAsync(message)
}

func (s *SlackManager) IsEnabled() bool {
	return true
}

func (s *SlackManager) sendAsync(message string) {
	go func() {
		if err := s.client.SendMessage(message); err != nil {
			s.logger.Warn(context.Background(), "Failed to send Slack notification",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "message", Value: message},
			)
		} else {
			s.logger.Debug(context.Background(), "Slack notification sent successfully",
				logging.Field{Key: "message", Value: message},
			)
		}
	}()
}

// Package-level convenience functions
func NotifyPhaseStart(phase string, issueNumber int) {
	GetManager().NotifyPhaseStart(phase, issueNumber)
}

func NotifyPRMerged(prNumber, issueNumber int) {
	GetManager().NotifyPRMerged(prNumber, issueNumber)
}

func NotifyError(title, errorMessage string) {
	GetManager().NotifyError(title, errorMessage)
}

func IsEnabled() bool {
	return GetManager().IsEnabled()
}

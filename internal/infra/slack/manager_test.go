package slack

import (
	"testing"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
	"github.com/stretchr/testify/assert"
)

// MockSlackClient for testing
type MockSlackClient struct {
	messages []string
	err      error
}

func (m *MockSlackClient) SendMessage(message string) error {
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, message)
	return nil
}

func TestSlackManagerEnabled(t *testing.T) {
	// Reset singleton for testing
	Reset()

	cfg := &config.Config{
		Slack: config.SlackConfig{
			NotificationsEnabled: true,
			WebhookURL:           "https://hooks.slack.com/test",
		},
	}

	logger := logging.NewMockLogger()
	Initialize(cfg, logger)

	manager := GetManager()
	assert.True(t, manager.IsEnabled())

	// Test global convenience functions
	assert.True(t, IsEnabled())

	// Test notifications (these will be sent asynchronously)
	NotifyPhaseStart("test-phase", 123)
	NotifyPRMerged(456, 123)
	NotifyError("Test Error", "Error details")
}

func TestSlackManagerDisabled(t *testing.T) {
	// Reset singleton for testing
	Reset()

	cfg := &config.Config{
		Slack: config.SlackConfig{
			NotificationsEnabled: false,
		},
	}

	logger := logging.NewMockLogger()
	Initialize(cfg, logger)

	manager := GetManager()
	assert.False(t, manager.IsEnabled())

	// Test global convenience functions
	assert.False(t, IsEnabled())

	// These should be no-ops
	NotifyPhaseStart("test-phase", 123)
	NotifyPRMerged(456, 123)
	NotifyError("Test Error", "Error details")
}

func TestSlackManagerMissingConfig(t *testing.T) {
	// Reset singleton for testing
	Reset()

	cfg := &config.Config{
		Slack: config.SlackConfig{}, // Empty Slack config
	}

	logger := logging.NewMockLogger()
	Initialize(cfg, logger)

	manager := GetManager()
	assert.False(t, manager.IsEnabled())
}

func TestSlackManagerEmptyWebhookURL(t *testing.T) {
	// Reset singleton for testing
	Reset()

	cfg := &config.Config{
		Slack: config.SlackConfig{
			NotificationsEnabled: true,
			WebhookURL:           "", // Empty webhook URL
		},
	}

	logger := logging.NewMockLogger()
	Initialize(cfg, logger)

	manager := GetManager()
	assert.False(t, manager.IsEnabled())
}

func TestGetManagerWithoutInitialization(t *testing.T) {
	// Reset singleton for testing
	Reset()

	// Don't call Initialize()
	manager := GetManager()
	assert.False(t, manager.IsEnabled())

	// These should be no-ops
	manager.NotifyPhaseStart("test", 123)
	manager.NotifyPRMerged(456, 123)
	manager.NotifyError("Error", "Details")
}

func TestSlackManagerSingleton(t *testing.T) {
	// Reset singleton for testing
	Reset()

	cfg := &config.Config{
		Slack: config.SlackConfig{
			NotificationsEnabled: true,
			WebhookURL:           "https://hooks.slack.com/test",
		},
	}

	logger := logging.NewMockLogger()

	// Initialize multiple times
	Initialize(cfg, logger)
	Initialize(cfg, logger)
	Initialize(cfg, logger)

	// Should return the same instance
	manager1 := GetManager()
	manager2 := GetManager()

	assert.Equal(t, manager1, manager2)
}

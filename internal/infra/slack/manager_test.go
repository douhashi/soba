package slack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

// MockSlackClient for testing
type MockSlackClient struct {
	messages  []string
	blockData [][]byte
	err       error
	blockErr  error
}

func (m *MockSlackClient) SendMessage(message string) error {
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockSlackClient) SendBlockMessage(blockData []byte) error {
	if m.blockErr != nil {
		return m.blockErr
	}
	m.blockData = append(m.blockData, blockData)
	return nil
}

func TestSlackManagerEnabled(t *testing.T) {
	// Reset singleton for testing
	Reset()

	// Create temporary templates directory for testing
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates", "slack")
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create test templates
	testTemplates := map[string]string{
		"notify.json": `{"blocks": [{"type": "section", "text": {"type": "mrkdwn", "text": "{{.Text}}"}}]}`,
	}

	for filename, content := range testTemplates {
		writeErr := os.WriteFile(filepath.Join(templateDir, filename), []byte(content), 0644)
		require.NoError(t, writeErr)
	}

	// Change working directory to temp dir for testing
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	cfg := &config.Config{
		Slack: config.SlackConfig{
			NotificationsEnabled: true,
			WebhookURL:           "https://hooks.slack.com/test",
		},
		GitHub: config.GitHubConfig{
			Repository: "douhashi/soba",
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
	Notify("Test notification")
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
	Notify("Test notification")
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
	manager.Notify("Test notification")
}

func TestSlackManagerSingleton(t *testing.T) {
	// Reset singleton for testing
	Reset()

	// Create temporary templates directory for testing
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates", "slack")
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create test templates
	testTemplates := map[string]string{
		"notify.json": `{"blocks": [{"type": "section", "text": {"type": "mrkdwn", "text": "{{.Text}}"}}]}`,
	}

	for filename, content := range testTemplates {
		writeErr := os.WriteFile(filepath.Join(templateDir, filename), []byte(content), 0644)
		require.NoError(t, writeErr)
	}

	// Change working directory to temp dir for testing
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	cfg := &config.Config{
		Slack: config.SlackConfig{
			NotificationsEnabled: true,
			WebhookURL:           "https://hooks.slack.com/test",
		},
		GitHub: config.GitHubConfig{
			Repository: "douhashi/soba",
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

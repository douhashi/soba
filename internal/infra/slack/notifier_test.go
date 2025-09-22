package slack

import (
	"errors"
	"sync"
	"testing"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

type mockSlackClient struct {
	messages []string
	err      error
}

func (m *mockSlackClient) SendMessage(message string) error {
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, message)
	return nil
}

type mockSlackClientWithSync struct {
	mockSlackClient
	mu sync.Mutex
	wg *sync.WaitGroup
}

func (m *mockSlackClientWithSync) SendMessage(message string) error {
	defer func() {
		if m.wg != nil {
			m.wg.Done()
		}
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, message)
	return nil
}

func TestNotifier_NotifyPhaseStart(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		phase          string
		issueNumber    int
		clientError    error
		wantError      bool
		expectedCount  int
		expectedPrefix string
	}{
		{
			name:           "通知有効・正常",
			enabled:        true,
			phase:          "plan",
			issueNumber:    123,
			clientError:    nil,
			wantError:      false,
			expectedCount:  1,
			expectedPrefix: "🚀 フェーズ開始: plan",
		},
		{
			name:           "通知無効",
			enabled:        false,
			phase:          "implement",
			issueNumber:    456,
			clientError:    nil,
			wantError:      false,
			expectedCount:  0,
			expectedPrefix: "",
		},
		{
			name:           "クライアントエラー",
			enabled:        true,
			phase:          "review",
			issueNumber:    789,
			clientError:    errors.New("network error"),
			wantError:      false, // エラーはログのみ、メイン処理は継続
			expectedCount:  0,
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockSlackClient{err: tt.clientError}
			config := &config.SlackConfig{
				NotificationsEnabled: tt.enabled,
			}

			logger := logging.NewMockLogger()
			notifier := NewSyncNotifier(mockClient, config, logger)
			err := notifier.NotifyPhaseStart(tt.phase, tt.issueNumber)

			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(mockClient.messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(mockClient.messages))
			}

			if tt.expectedCount > 0 && len(mockClient.messages) > 0 {
				if len(mockClient.messages[0]) < len(tt.expectedPrefix) ||
					mockClient.messages[0][:len(tt.expectedPrefix)] != tt.expectedPrefix {
					t.Errorf("Expected message to start with '%s', got '%s'",
						tt.expectedPrefix, mockClient.messages[0])
				}
			}
		})
	}
}

func TestNotifier_NotifyPRMerged(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		prNumber       int
		issueNumber    int
		clientError    error
		wantError      bool
		expectedCount  int
		expectedPrefix string
	}{
		{
			name:           "通知有効・正常",
			enabled:        true,
			prNumber:       42,
			issueNumber:    123,
			clientError:    nil,
			wantError:      false,
			expectedCount:  1,
			expectedPrefix: "✅ PR マージ完了",
		},
		{
			name:           "通知無効",
			enabled:        false,
			prNumber:       43,
			issueNumber:    456,
			clientError:    nil,
			wantError:      false,
			expectedCount:  0,
			expectedPrefix: "",
		},
		{
			name:           "クライアントエラー",
			enabled:        true,
			prNumber:       44,
			issueNumber:    789,
			clientError:    errors.New("network error"),
			wantError:      false, // エラーはログのみ、メイン処理は継続
			expectedCount:  0,
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockSlackClient{err: tt.clientError}
			config := &config.SlackConfig{
				NotificationsEnabled: tt.enabled,
			}

			logger := logging.NewMockLogger()
			notifier := NewSyncNotifier(mockClient, config, logger)
			err := notifier.NotifyPRMerged(tt.prNumber, tt.issueNumber)

			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(mockClient.messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(mockClient.messages))
			}

			if tt.expectedCount > 0 && len(mockClient.messages) > 0 {
				if len(mockClient.messages[0]) < len(tt.expectedPrefix) ||
					mockClient.messages[0][:len(tt.expectedPrefix)] != tt.expectedPrefix {
					t.Errorf("Expected message to start with '%s', got '%s'",
						tt.expectedPrefix, mockClient.messages[0])
				}
			}
		})
	}
}

func TestNotifier_AsyncNotify(t *testing.T) {
	var wg sync.WaitGroup
	mockClient := &mockSlackClientWithSync{
		wg: &wg,
	}
	config := &config.SlackConfig{
		NotificationsEnabled: true,
	}

	logger := logging.NewMockLogger()
	notifier := NewNotifier(mockClient, config, logger)

	// 非同期通知をテスト
	wg.Add(1) // 1つのメッセージを待機
	err := notifier.NotifyPhaseStart("plan", 123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 非同期処理の完了を待機
	wg.Wait()

	mockClient.mu.Lock()
	messageCount := len(mockClient.messages)
	mockClient.mu.Unlock()

	if messageCount != 1 {
		t.Errorf("Expected 1 message, got %d", messageCount)
	}
}

func TestNewNotifier(t *testing.T) {
	mockClient := &mockSlackClient{}
	config := &config.SlackConfig{
		NotificationsEnabled: true,
	}

	logger := logging.NewMockLogger()
	notifier := NewNotifier(mockClient, config, logger)

	if notifier == nil {
		t.Error("Expected notifier to be created, got nil")
	}
}

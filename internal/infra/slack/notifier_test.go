package slack

import (
	"errors"
	"testing"
	"time"

	"github.com/douhashi/soba/internal/config"
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

			notifier := NewSyncNotifier(mockClient, config)
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

			notifier := NewSyncNotifier(mockClient, config)
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
	mockClient := &mockSlackClient{}
	config := &config.SlackConfig{
		NotificationsEnabled: true,
	}

	notifier := NewNotifier(mockClient, config)

	// 非同期通知をテスト
	err := notifier.NotifyPhaseStart("plan", 123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 少し待機して非同期処理が完了することを確認
	time.Sleep(10 * time.Millisecond)

	if len(mockClient.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(mockClient.messages))
	}
}

func TestNewNotifier(t *testing.T) {
	mockClient := &mockSlackClient{}
	config := &config.SlackConfig{
		NotificationsEnabled: true,
	}

	notifier := NewNotifier(mockClient, config)

	if notifier == nil {
		t.Error("Expected notifier to be created, got nil")
	}
}

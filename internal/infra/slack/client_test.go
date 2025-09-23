package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_SendMessage(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		serverResponse int
		serverBody     string
		wantError      bool
		errorContains  string
	}{
		{
			name:           "正常なメッセージ送信",
			message:        "テストメッセージ",
			serverResponse: http.StatusOK,
			serverBody:     "ok",
			wantError:      false,
		},
		{
			name:           "空のメッセージ",
			message:        "",
			serverResponse: http.StatusOK,
			serverBody:     "ok",
			wantError:      true,
			errorContains:  "message cannot be empty",
		},
		{
			name:           "サーバーエラー",
			message:        "テストメッセージ",
			serverResponse: http.StatusInternalServerError,
			serverBody:     "server error",
			wantError:      true,
			errorContains:  "failed to send message",
		},
		{
			name:           "不正なレスポンス",
			message:        "テストメッセージ",
			serverResponse: http.StatusBadRequest,
			serverBody:     "invalid_payload",
			wantError:      true,
			errorContains:  "failed to send message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				var payload map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && tt.message != "" {
					t.Errorf("Failed to decode request body: %v", err)
				}

				w.WriteHeader(tt.serverResponse)
				w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, 5*time.Second)
			err := client.SendMessage(tt.message)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestClient_SendMessage_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient(server.URL, 50*time.Millisecond)
	err := client.SendMessage("テストメッセージ")

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout/deadline error, got %v", err)
	}
}

func TestClient_SendMessage_InvalidURL(t *testing.T) {
	client := NewClient("invalid-url", 5*time.Second)
	err := client.SendMessage("テストメッセージ")

	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestClient_SendBlockMessage(t *testing.T) {
	blockData := []byte(`{
		"blocks": [
			{
				"type": "section",
				"text": {
					"type": "mrkdwn",
					"text": "Test block message"
				}
			}
		]
	}`)

	tests := []struct {
		name           string
		blockData      []byte
		serverResponse int
		serverBody     string
		wantError      bool
		errorContains  string
	}{
		{
			name:           "正常なブロックメッセージ送信",
			blockData:      blockData,
			serverResponse: http.StatusOK,
			serverBody:     "ok",
			wantError:      false,
		},
		{
			name:           "空のブロックデータ",
			blockData:      []byte{},
			serverResponse: http.StatusOK,
			serverBody:     "ok",
			wantError:      true,
			errorContains:  "block data cannot be empty",
		},
		{
			name:           "サーバーエラー",
			blockData:      blockData,
			serverResponse: http.StatusInternalServerError,
			serverBody:     "server error",
			wantError:      true,
			errorContains:  "failed to send block message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.serverResponse)
				w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, 5*time.Second)
			err := client.SendBlockMessage(tt.blockData)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	webhookURL := "https://hooks.slack.com/services/T0J5GSMNH/B09G3KFHYSX/test"
	timeout := 10 * time.Second

	client := NewClient(webhookURL, timeout)

	if client == nil {
		t.Error("Expected client to be created, got nil")
	}
}

package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logging"
)

// MockTokenProvider はテスト用のTokenProvider
type MockTokenProvider struct {
	token string
	err   error
}

func (m *MockTokenProvider) GetToken(ctx context.Context) (string, error) {
	return m.token, m.err
}

func TestClient_CreateLabel(t *testing.T) {
	tests := []struct {
		name          string
		label         CreateLabelRequest
		statusCode    int
		responseBody  string
		expectedError bool
		expectedLabel *Label
		errorType     string
	}{
		{
			name: "正常系: ラベル作成成功",
			label: CreateLabelRequest{
				Name:        "soba:todo",
				Color:       "e1e4e8",
				Description: "New issue awaiting processing",
			},
			statusCode: http.StatusCreated,
			responseBody: `{
				"id": 1234567890,
				"name": "soba:todo",
				"color": "e1e4e8",
				"description": "New issue awaiting processing"
			}`,
			expectedError: false,
			expectedLabel: &Label{
				ID:          1234567890,
				Name:        "soba:todo",
				Color:       "e1e4e8",
				Description: "New issue awaiting processing",
			},
		},
		{
			name: "異常系: ラベル重複エラー",
			label: CreateLabelRequest{
				Name:        "existing-label",
				Color:       "ffffff",
				Description: "Test label",
			},
			statusCode: http.StatusUnprocessableEntity,
			responseBody: `{
				"message": "Validation Failed",
				"errors": [
					{
						"code": "already_exists",
						"field": "name"
					}
				]
			}`,
			expectedError: true,
			errorType:     "GitHubAPIError",
		},
		{
			name: "異常系: 権限不足エラー",
			label: CreateLabelRequest{
				Name:        "test-label",
				Color:       "ffffff",
				Description: "Test label",
			},
			statusCode: http.StatusForbidden,
			responseBody: `{
				"message": "Must have push access to repository"
			}`,
			expectedError: true,
			errorType:     "GitHubAPIError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサーバーの設定
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/repos/test-owner/test-repo/labels", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

				// リクエストボディの検証
				var reqBody CreateLabelRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.label, reqBody)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// クライアントの作成
			tokenProvider := &MockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
				Logger:  logging.NewMockLogger(),
			})
			require.NoError(t, err)

			// テスト実行
			ctx := context.Background()
			result, err := client.CreateLabel(ctx, "test-owner", "test-repo", tt.label)

			// 結果の検証
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorType == "GitHubAPIError" {
					// エラータイプの検証（BaseErrorベースのエラーかどうか）
					assert.True(t, errors.IsExternalError(err))
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedLabel, result)
			}
		})
	}
}

func TestClient_ListLabels(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  bool
		expectedLabels []Label
	}{
		{
			name:       "正常系: ラベル一覧取得成功",
			statusCode: http.StatusOK,
			responseBody: `[
				{
					"id": 1234567890,
					"name": "soba:todo",
					"color": "e1e4e8",
					"description": "New issue awaiting processing"
				},
				{
					"id": 1234567891,
					"name": "soba:doing",
					"color": "1d76db",
					"description": "Claude working on implementation"
				}
			]`,
			expectedError: false,
			expectedLabels: []Label{
				{
					ID:          1234567890,
					Name:        "soba:todo",
					Color:       "e1e4e8",
					Description: "New issue awaiting processing",
				},
				{
					ID:          1234567891,
					Name:        "soba:doing",
					Color:       "1d76db",
					Description: "Claude working on implementation",
				},
			},
		},
		{
			name:           "正常系: 空のラベル一覧",
			statusCode:     http.StatusOK,
			responseBody:   `[]`,
			expectedError:  false,
			expectedLabels: []Label{},
		},
		{
			name:           "異常系: リポジトリが見つからない",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"message": "Not Found"}`,
			expectedError:  true,
			expectedLabels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサーバーの設定
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/repos/test-owner/test-repo/labels", r.URL.Path)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// クライアントの作成
			tokenProvider := &MockTokenProvider{token: "test-token"}
			client, err := NewClient(tokenProvider, &ClientOptions{
				BaseURL: server.URL,
				Logger:  logging.NewMockLogger(),
			})
			require.NoError(t, err)

			// テスト実行
			ctx := context.Background()
			result, err := client.ListLabels(ctx, "test-owner", "test-repo")

			// 結果の検証
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLabels, result)
			}
		})
	}
}

func TestGetSobaLabels(t *testing.T) {
	labels := GetSobaLabels()

	// 10個のラベルが定義されていることを確認
	assert.Len(t, labels, 10)

	// 各ラベルの内容を検証
	expectedLabels := map[string]struct {
		color       string
		description string
	}{
		"soba:todo":             {"e1e4e8", "New issue awaiting processing"},
		"soba:queued":           {"fbca04", "Selected for processing"},
		"soba:planning":         {"d4c5f9", "Claude creating implementation plan"},
		"soba:ready":            {"0e8a16", "Plan complete, awaiting implementation"},
		"soba:doing":            {"1d76db", "Claude working on implementation"},
		"soba:review-requested": {"f9d71c", "PR created, awaiting review"},
		"soba:reviewing":        {"a2eeef", "Claude reviewing PR"},
		"soba:done":             {"0e8a16", "Review approved, ready to merge"},
		"soba:requires-changes": {"d93f0b", "Review requested modifications"},
		"soba:revising":         {"ff6347", "Claude applying requested changes"},
	}

	for _, label := range labels {
		expected, exists := expectedLabels[label.Name]
		assert.True(t, exists, "Unexpected label: %s", label.Name)
		assert.Equal(t, expected.color, label.Color, "Color mismatch for label: %s", label.Name)
		assert.Equal(t, expected.description, label.Description, "Description mismatch for label: %s", label.Name)
	}
}

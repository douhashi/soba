package cli

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTmuxClient はTmuxClientのモック実装
type MockTmuxClient struct {
	mock.Mock
}

func (m *MockTmuxClient) CreateSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockTmuxClient) DeleteSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockTmuxClient) SessionExists(sessionName string) bool {
	args := m.Called(sessionName)
	return args.Bool(0)
}

func (m *MockTmuxClient) CreateWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) DeleteWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

func (m *MockTmuxClient) CreatePane(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) DeletePane(sessionName, windowName string, paneIndex int) error {
	args := m.Called(sessionName, windowName, paneIndex)
	return args.Error(0)
}

func (m *MockTmuxClient) GetPaneCount(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *MockTmuxClient) GetFirstPaneIndex(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *MockTmuxClient) GetLastPaneIndex(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *MockTmuxClient) ResizePanes(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) SendCommand(sessionName, windowName string, paneIndex int, command string) error {
	args := m.Called(sessionName, windowName, paneIndex, command)
	return args.Error(0)
}

func (m *MockTmuxClient) KillSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func TestGenerateSessionName(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		expected   string
	}{
		{
			name:       "Normal repository format",
			repository: "douhashi/soba",
			expected:   "soba-douhashi-soba",
		},
		{
			name:       "Empty repository",
			repository: "",
			expected:   "soba",
		},
		{
			name:       "Invalid format (no slash)",
			repository: "invalid-repo",
			expected:   "soba",
		},
		{
			name:       "Invalid format (slash only)",
			repository: "/",
			expected:   "soba",
		},
		{
			name:       "Long repository name",
			repository: "very-long-owner/very-long-repository-name",
			expected:   "soba-very-long-owner-very-long-repository-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &openCmd{}
			result := cmd.generateSessionName(tt.repository)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunOpen_SessionExists(t *testing.T) {
	mockTmux := new(MockTmuxClient)
	cmd := &openCmd{
		tmuxClient: mockTmux,
		attachToSession: func(sessionName string) error {
			// テスト環境では実際のtmux attachを実行せずに成功を返す
			return nil
		},
	}

	// 設定をセットアップ
	viper.Set("github.repository", "douhashi/soba")

	// セッションが既に存在する場合
	mockTmux.On("SessionExists", "soba-douhashi-soba").Return(true)

	err := cmd.runOpen(nil, []string{})

	assert.NoError(t, err)
	mockTmux.AssertExpectations(t)
}

func TestRunOpen_CreateNewSession(t *testing.T) {
	mockTmux := new(MockTmuxClient)
	cmd := &openCmd{
		tmuxClient: mockTmux,
		attachToSession: func(sessionName string) error {
			// テスト環境では実際のtmux attachを実行せずに成功を返す
			return nil
		},
	}

	// 設定をセットアップ
	viper.Set("github.repository", "douhashi/soba")

	// セッションが存在しない場合
	mockTmux.On("SessionExists", "soba-douhashi-soba").Return(false)
	mockTmux.On("CreateSession", "soba-douhashi-soba").Return(nil)

	err := cmd.runOpen(nil, []string{})

	assert.NoError(t, err)
	mockTmux.AssertExpectations(t)
}

func TestRunOpen_CreateSessionError(t *testing.T) {
	mockTmux := new(MockTmuxClient)
	cmd := &openCmd{
		tmuxClient: mockTmux,
		attachToSession: func(sessionName string) error {
			// テスト環境では実際のtmux attachを実行せずに成功を返す
			return nil
		},
	}

	// 設定をセットアップ
	viper.Set("github.repository", "douhashi/soba")

	// セッション作成でエラーが発生する場合
	mockTmux.On("SessionExists", "soba-douhashi-soba").Return(false)
	mockTmux.On("CreateSession", "soba-douhashi-soba").Return(errors.New("tmux error"))

	err := cmd.runOpen(nil, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create session")
	mockTmux.AssertExpectations(t)
}

func TestRunOpen_NoRepository(t *testing.T) {
	mockTmux := new(MockTmuxClient)
	cmd := &openCmd{
		tmuxClient: mockTmux,
		attachToSession: func(sessionName string) error {
			// テスト環境では実際のtmux attachを実行せずに成功を返す
			return nil
		},
	}

	// リポジトリが設定されていない場合
	viper.Set("github.repository", "")

	// デフォルトのセッション名を使用
	mockTmux.On("SessionExists", "soba").Return(true)

	err := cmd.runOpen(nil, []string{})

	assert.NoError(t, err)
	mockTmux.AssertExpectations(t)
}

func TestNewOpenCmd(t *testing.T) {
	cmd := newOpenCmd()

	assert.Equal(t, "open", cmd.Use)
	assert.Equal(t, "Open tmux session", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

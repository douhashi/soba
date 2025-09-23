package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/pkg/app"
)

// MockStopService はStopServiceのモック実装
type MockStopService struct {
	mock.Mock
}

func (m *MockStopService) Stop(ctx context.Context, repository string) error {
	args := m.Called(ctx, repository)
	return args.Error(0)
}

func TestStopCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupApp       func()
		setupMock      func(*MockStopService)
		expectedOutput string
		wantError      bool
	}{
		{
			name: "Stop daemon successfully",
			args: []string{},
			setupApp: func() {
				// Initialize app for testing
				helper := app.NewTestHelper(t)
				helper.InitializeForTest()
			},
			setupMock: func(daemon *MockStopService) {
				daemon.On("Stop", mock.Anything, mock.Anything).Return(nil)
			},
			expectedOutput: "Daemon stopped successfully\n",
			wantError:      false,
		},
		{
			name: "Stop daemon with error",
			args: []string{},
			setupApp: func() {
				// Initialize app for testing
				helper := app.NewTestHelper(t)
				helper.InitializeForTest()
			},
			setupMock: func(daemon *MockStopService) {
				daemon.On("Stop", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			expectedOutput: "",
			wantError:      true,
		},
		{
			name: "Stop daemon without config file",
			args: []string{},
			setupApp: func() {
				// Reset app to ensure no config is loaded
				app.Reset()
				// Initialize minimal app without config
				app.MustInitializeMinimal()
			},
			setupMock: func(daemon *MockStopService) {
				// Repository情報が空でも動作することを確認
				daemon.On("Stop", mock.Anything, "").Return(nil)
			},
			expectedOutput: "Daemon stopped successfully\n",
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup app state
			tt.setupApp()

			// モックの準備
			mockDaemon := new(MockStopService)
			tt.setupMock(mockDaemon)

			// バッファを使用して出力をキャプチャ
			var buf bytes.Buffer

			// コマンドの実行（configファイルが存在しない前提でテスト）
			cmd := newStopCmd()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			// runStopWithServiceを直接呼び出し
			err := runStopWithService(cmd, tt.args, mockDaemon)

			// アサーション
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, buf.String())
			}

			// モックの検証
			mockDaemon.AssertExpectations(t)

			// Clean up
			app.Reset()
		})
	}
}

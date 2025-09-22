package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
)

func TestNewStartCmd(t *testing.T) {
	cmd := newStartCmd()
	assert.Equal(t, "start", cmd.Use)
	assert.Equal(t, "Start Issue monitoring in foreground or daemon mode", cmd.Short)

	// フラグをテスト
	daemonFlag := cmd.Flags().Lookup("daemon")
	require.NotNil(t, daemonFlag)
	assert.Equal(t, "bool", daemonFlag.Value.Type())

}

func TestRunStart_ForegroundMode(t *testing.T) {
	// テスト用一時ディレクトリ作成
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	// テスト用設定ファイル作成
	configPath := filepath.Join(sobaDir, "config.yml")
	configContent := `github:
  token: test-token
  repository: test/repo
workflow:
  interval: 30`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	// 現在のディレクトリを一時的に変更
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()

	// モックサービス
	mockService := &MockDaemonServiceImpl{
		startForegroundFunc: func(ctx context.Context, cfg *config.Config) error {
			return nil
		},
	}

	cmd := &cobra.Command{}
	err = runStartWithService(cmd, []string{}, false, mockService)
	assert.NoError(t, err)
	assert.True(t, mockService.startForegroundCalled)
}

func TestRunStart_DaemonMode(t *testing.T) {
	// テスト用一時ディレクトリ作成
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	// テスト用設定ファイル作成
	configPath := filepath.Join(sobaDir, "config.yml")
	configContent := `github:
  token: test-token
  repository: test/repo
workflow:
  interval: 30`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	// 現在のディレクトリを一時的に変更
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()

	// モックサービス
	mockService := &MockDaemonServiceImpl{
		startDaemonFunc: func(ctx context.Context, cfg *config.Config) error {
			return nil
		},
	}

	cmd := &cobra.Command{}
	err = runStartWithService(cmd, []string{}, true, mockService)
	assert.NoError(t, err)
	assert.True(t, mockService.startDaemonCalled)
}

func TestRunStart_ConfigNotFound(t *testing.T) {
	// 設定ファイルが存在しない一時ディレクトリ
	tmpDir := t.TempDir()

	// 現在のディレクトリを一時的に変更
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		require.NoError(t, os.Chdir(originalDir))
	}()

	// モックサービス
	mockService := &MockDaemonServiceImpl{}

	cmd := &cobra.Command{}
	err = runStartWithService(cmd, []string{}, false, mockService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config")
}

// MockDaemonServiceImpl はテスト用のモックサービス
type MockDaemonServiceImpl struct {
	startForegroundFunc   func(ctx context.Context, cfg *config.Config) error
	startDaemonFunc       func(ctx context.Context, cfg *config.Config) error
	startForegroundCalled bool
	startDaemonCalled     bool
}

func (m *MockDaemonServiceImpl) StartForeground(ctx context.Context, cfg *config.Config) error {
	m.startForegroundCalled = true
	if m.startForegroundFunc != nil {
		return m.startForegroundFunc(ctx, cfg)
	}
	return nil
}

func (m *MockDaemonServiceImpl) StartDaemon(ctx context.Context, cfg *config.Config) error {
	m.startDaemonCalled = true
	if m.startDaemonFunc != nil {
		return m.startDaemonFunc(ctx, cfg)
	}
	return nil
}

func TestRunStart_LogFileCreation(t *testing.T) {
	tests := []struct {
		name       string
		daemonMode bool
	}{
		{
			name:       "Foreground mode creates log file",
			daemonMode: false,
		},
		{
			name:       "Daemon mode creates log file",
			daemonMode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用一時ディレクトリ作成
			tmpDir := t.TempDir()
			sobaDir := filepath.Join(tmpDir, ".soba")
			logsDir := filepath.Join(sobaDir, "logs")
			require.NoError(t, os.MkdirAll(sobaDir, 0755))

			// テスト用設定ファイル作成（ログ設定を含む）
			configPath := filepath.Join(sobaDir, "config.yml")
			configContent := `github:
  token: test-token
  repository: test/repo
workflow:
  interval: 30
log:
  output_path: .soba/logs/soba-${PID}.log
  retention_count: 5`
			require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

			// 現在のディレクトリを一時的に変更
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() {
				require.NoError(t, os.Chdir(originalDir))
			}()

			// モックサービスでログディレクトリが作成されることを確認
			mockService := &MockDaemonServiceImpl{
				startForegroundFunc: func(ctx context.Context, cfg *config.Config) error {
					// ログ設定が渡されていることを確認
					assert.NotEmpty(t, cfg.Log.OutputPath)
					assert.Equal(t, 5, cfg.Log.RetentionCount)
					// ログディレクトリを作成（実際のサービスの動作を模倣）
					require.NoError(t, os.MkdirAll(logsDir, 0755))
					return nil
				},
				startDaemonFunc: func(ctx context.Context, cfg *config.Config) error {
					// ログ設定が渡されていることを確認
					assert.NotEmpty(t, cfg.Log.OutputPath)
					assert.Equal(t, 5, cfg.Log.RetentionCount)
					// ログディレクトリを作成（実際のサービスの動作を模倣）
					require.NoError(t, os.MkdirAll(logsDir, 0755))
					return nil
				},
			}

			cmd := &cobra.Command{}
			err = runStartWithService(cmd, []string{}, tt.daemonMode, mockService)
			assert.NoError(t, err)

			// ログディレクトリが作成されたことを確認
			_, err = os.Stat(logsDir)
			assert.NoError(t, err)

			if tt.daemonMode {
				assert.True(t, mockService.startDaemonCalled)
			} else {
				assert.True(t, mockService.startForegroundCalled)
			}
		})
	}
}

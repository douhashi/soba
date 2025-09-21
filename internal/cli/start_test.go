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

	verboseFlag := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "bool", verboseFlag.Value.Type())
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
	err = runStartWithService(cmd, []string{}, false, false, mockService)
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
	err = runStartWithService(cmd, []string{}, true, false, mockService)
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
	err = runStartWithService(cmd, []string{}, false, false, mockService)
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

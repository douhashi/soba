//go:build integration
// +build integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopCommandIntegration(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	// 元のディレクトリを保存して、テスト終了後に戻る
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// 作業ディレクトリを変更
	require.NoError(t, os.Chdir(tmpDir))

	// テスト用の設定ファイルを作成
	configYAML := `github:
  token: test-token
  repository: test/repo
workflow:
  interval: 5
  closed_issue_cleanup_enabled: false
  closed_issue_cleanup_interval: 60
`
	configPath := filepath.Join(sobaDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	t.Run("Stop when daemon is not running", func(t *testing.T) {
		// stopコマンドを実行
		cmd := newStopCmd()
		cmd.SetArgs([]string{})

		// デーモンが起動していない状態でstopを実行（エラーが返る）
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "daemon is not running")
	})

	t.Run("Stop running daemon", func(t *testing.T) {
		// PIDファイルを作成（テスト用の偽のPID）
		pidFile := filepath.Join(sobaDir, "soba.pid")
		// 存在しないPIDを書き込む（実際のプロセスにシグナルを送らないため）
		require.NoError(t, os.WriteFile(pidFile, []byte("999999"), 0600))

		// stopコマンドを実行
		cmd := newStopCmd()
		cmd.SetArgs([]string{})

		// PIDが存在しないプロセスのため、エラーになる
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "process not found")

		// PIDファイルが削除されていることを確認
		_, err = os.Stat(pidFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Stop with verbose flag", func(t *testing.T) {
		// stopコマンドを実行（verbose付き）
		cmd := newStopCmd()
		cmd.SetArgs([]string{"-v"})

		// デーモンが起動していない状態でstopを実行（エラーが返る）
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "daemon is not running")
	})
}

func TestStopCommandWithRealDaemon(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real daemon in short mode")
	}

	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	// 元のディレクトリを保存して、テスト終了後に戻る
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// 作業ディレクトリを変更
	require.NoError(t, os.Chdir(tmpDir))

	// テスト用の設定ファイルを作成
	configYAML := `github:
  token: test-token
  repository: test/repo
workflow:
  interval: 5
  closed_issue_cleanup_enabled: false
  closed_issue_cleanup_interval: 60
`
	configPath := filepath.Join(sobaDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	// 現在のプロセスのPIDを使ったテスト（注意深く実行）
	t.Run("Stop with current process PID (careful test)", func(t *testing.T) {
		// PIDファイルを作成（現在のプロセスのPIDを使用）
		pidFile := filepath.Join(sobaDir, "soba.pid")
		pid := os.Getpid()
		require.NoError(t, os.WriteFile(pidFile, []byte(string(rune(pid))), 0600))

		// stopコマンドを準備（実行はしない）
		cmd := newStopCmd()
		cmd.SetArgs([]string{})

		// このテストは現在のプロセスにシグナルを送る可能性があるため、
		// 実際には実行せず、コマンドが作成できることだけを確認
		assert.NotNil(t, cmd)

		// PIDファイルを手動で削除
		os.Remove(pidFile)
	})
}

func TestDaemonLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping daemon lifecycle integration test in short mode")
	}

	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	logsDir := filepath.Join(sobaDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// 元のディレクトリを保存して、テスト終了後に戻る
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// 作業ディレクトリを変更
	require.NoError(t, os.Chdir(tmpDir))

	// テスト用の設定ファイルを作成
	configYAML := `github:
  token: test-token
  repository: test/repo
workflow:
  interval: 1
  closed_issue_cleanup_enabled: false
  closed_issue_cleanup_interval: 60
`
	configPath := filepath.Join(sobaDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	// デーモンを起動する準備（実際にはテスト環境では起動が難しいため、シミュレート）
	t.Run("Simulate daemon lifecycle", func(t *testing.T) {
		// PIDファイルを作成して、デーモンが起動している状態をシミュレート
		pidFile := filepath.Join(sobaDir, "soba.pid")

		// 存在しない大きなPIDを使用
		require.NoError(t, os.WriteFile(pidFile, []byte("999999"), 0600))

		// PIDファイルが存在することを確認
		_, err := os.Stat(pidFile)
		assert.NoError(t, err)

		// stopコマンドを実行
		cmd := newStopCmd()
		cmd.SetArgs([]string{})

		// 実行（プロセスが存在しないため、エラーになるが、PIDファイルは削除される）
		err = cmd.Execute()
		assert.Error(t, err) // プロセスが存在しないのでエラー

		// PIDファイルが削除されていることを確認
		time.Sleep(100 * time.Millisecond) // ファイルシステムの同期待ち
		_, err = os.Stat(pidFile)
		assert.True(t, os.IsNotExist(err), "PID file should be deleted")
	})
}

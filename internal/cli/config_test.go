package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := newConfigCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "config", cmd.Use)
	assert.Contains(t, cmd.Short, "Display current configuration")
}

func TestRunConfig_Success(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".soba")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// テスト用の設定ファイルを作成
	configContent := `github:
  token: ghp_test_token
  repository: douhashi/soba
  auth_method: token
workflow:
  interval: 30
  use_tmux: true
  auto_merge_enabled: false
  closed_issue_cleanup_enabled: true
  closed_issue_cleanup_interval: 300
  tmux_command_delay: 3
slack:
  webhook_url: https://hooks.slack.com/test
  notifications_enabled: false
git:
  worktree_base_path: .git/soba/worktrees
phase:
  plan:
    command: plan.sh
    options:
      - -v
    parameter: issue_number
  implement:
    command: implement.sh
    options:
      - -v
    parameter: issue_number`

	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// コマンドを実行
	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config", configPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	// マスキングされているか確認
	assert.Contains(t, output, "***MASKED***")
	assert.NotContains(t, output, "ghp_test_token")
	assert.NotContains(t, output, "https://hooks.slack.com/test")
	// その他の設定が表示されているか確認
	assert.Contains(t, output, "repository: douhashi/soba")
	assert.Contains(t, output, "interval: 30")
	assert.Contains(t, output, "use_tmux: true")
}

func TestRunConfig_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, ".soba", "config.yml")

	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config", nonExistentPath})

	err := cmd.Execute()
	require.Error(t, err)

	output := buf.String()
	assert.Contains(t, strings.ToLower(output), "not found")
}

func TestRunConfig_InvalidYAML(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".soba")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// 不正なYAMLファイルを作成
	invalidContent := `github:
  token: test
  repository: [invalid yaml
  `

	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(invalidContent), 0644))

	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config", configPath})

	err := cmd.Execute()
	require.Error(t, err)

	output := buf.String()
	assert.Contains(t, strings.ToLower(output), "invalid")
}

func TestRunConfig_DefaultPath(t *testing.T) {
	// 現在のディレクトリを保存
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	// 一時ディレクトリを作成して移動
	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))

	// .soba/config.ymlを作成
	configDir := filepath.Join(tempDir, ".soba")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `github:
  token: test_token
  repository: test/repo
workflow:
  interval: 20`

	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// コマンドを実行（--configフラグなし）
	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "repository: test/repo")
	assert.Contains(t, output, "interval: 20")
}

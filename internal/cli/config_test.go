package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/app"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := newConfigCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "config", cmd.Use)
	assert.Contains(t, cmd.Short, "Display current configuration")
}

func TestRunConfig_Success(t *testing.T) {
	// Initialize app with test config
	helper := app.NewTestHelper(t)
	testConfig := &config.Config{
		GitHub: config.GitHubConfig{
			Token:      "ghp_test_token",
			Repository: "douhashi/soba",
			AuthMethod: "token",
		},
		Workflow: config.WorkflowConfig{
			Interval:                   30,
			UseTmux:                    true,
			AutoMergeEnabled:           false,
			ClosedIssueCleanupEnabled:  true,
			ClosedIssueCleanupInterval: 300,
			TmuxCommandDelay:           3,
		},
		Slack: config.SlackConfig{
			WebhookURL:           "https://hooks.slack.com/test",
			NotificationsEnabled: false,
		},
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
		},
		Phase: config.PhaseConfig{
			Plan: config.PhaseCommand{
				Command:   "plan.sh",
				Options:   []string{"-v"},
				Parameter: "issue_number",
			},
			Implement: config.PhaseCommand{
				Command:   "implement.sh",
				Options:   []string{"-v"},
				Parameter: "issue_number",
			},
		},
		Log: config.LogConfig{
			Level:      "warn",
			OutputPath: "stdout",
		},
	}
	helper.InitializeForTestWithConfig(testConfig)

	// コマンドを実行
	cmd := newConfigCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

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
	// This test needs to skip app initialization since we're testing
	// the behavior when a config file doesn't exist
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, ".soba", "config.yml")

	// Create minimal config for app initialization
	minimalConfigPath := filepath.Join(tempDir, "minimal.yml")
	configData := []byte(`github:
  repository: test/repo
log:
  level: warn`)
	require.NoError(t, os.WriteFile(minimalConfigPath, configData, 0644))

	// Initialize app with the minimal config
	helper := app.NewTestHelper(t)
	helper.InitializeForTestWithOptions(minimalConfigPath, nil)

	// Now test that accessing a non-existent config returns default config
	cfg, err := config.Load(nonExistentPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	// Verify defaults are set
	assert.Equal(t, 20, cfg.Workflow.Interval)
	assert.Equal(t, ".git/soba/worktrees", cfg.Git.WorktreeBasePath)
}

func TestRunConfig_InvalidYAML(t *testing.T) {
	// Initialize app for testing
	helper := app.NewTestHelper(t)
	helper.InitializeForTest()

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

	// Test that loading invalid YAML returns an error
	_, err := config.Load(configPath)
	require.Error(t, err)
	// The error message should contain either "unmarshal" or "yaml"
	errMsg := strings.ToLower(err.Error())
	assert.True(t, strings.Contains(errMsg, "unmarshal") || strings.Contains(errMsg, "yaml"),
		"Error should mention unmarshal or yaml, got: %s", err.Error())
}

func TestRunConfig_DefaultPath(t *testing.T) {
	// Initialize app for testing
	helper := app.NewTestHelper(t)

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
  interval: 20
log:
  level: warn`

	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Initialize app with the test config
	helper.InitializeForTestWithOptions(configPath, nil)

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

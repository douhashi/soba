package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
github:
  token: test-token
  repository: owner/repo
  auth_method: token

workflow:
  interval: 30
  use_tmux: true
  auto_merge_enabled: false
  closed_issue_cleanup_enabled: true
  closed_issue_cleanup_interval: 600
  tmux_command_delay: 5

slack:
  webhook_url: https://hooks.slack.com/services/test
  notifications_enabled: true

git:
  worktree_base_path: .git/test/worktrees

phase:
  plan:
    command: /soba:plan
    options: []
    parameter: "{{ISSUE_NUMBER}}"
  implement:
    command: /soba:implement
    options: []
    parameter: "{{ISSUE_NUMBER}}"
  review:
    command: /soba:review
    options: []
    parameter: "{{PR_NUMBER}}"
  revise:
    command: /soba:revise
    options: []
    parameter: "{{ISSUE_NUMBER}}"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GitHub.Token != "test-token" {
		t.Errorf("GitHub token = %v, want test-token", cfg.GitHub.Token)
	}

	if cfg.GitHub.Repository != "owner/repo" {
		t.Errorf("GitHub repository = %v, want owner/repo", cfg.GitHub.Repository)
	}

	if cfg.Workflow.Interval != 30 {
		t.Errorf("Workflow interval = %v, want 30", cfg.Workflow.Interval)
	}

	if !cfg.Workflow.UseTmux {
		t.Errorf("Workflow use_tmux = %v, want true", cfg.Workflow.UseTmux)
	}

	if cfg.Slack.WebhookURL != "https://hooks.slack.com/services/test" {
		t.Errorf("Slack webhook URL = %v, want https://hooks.slack.com/services/test", cfg.Slack.WebhookURL)
	}

	if cfg.Git.WorktreeBasePath != ".git/test/worktrees" {
		t.Errorf("Git worktree base path = %v, want .git/test/worktrees", cfg.Git.WorktreeBasePath)
	}

	if cfg.Phase.Plan.Command != "/soba:plan" {
		t.Errorf("Phase plan command = %v, want /soba:plan", cfg.Phase.Plan.Command)
	}
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	os.Setenv("TEST_GITHUB_TOKEN", "env-token")
	defer os.Unsetenv("TEST_GITHUB_TOKEN")

	configContent := `
github:
  token: ${TEST_GITHUB_TOKEN}
  repository: owner/repo
  auth_method: token

workflow:
  interval: 20
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GitHub.Token != "env-token" {
		t.Errorf("GitHub token = %v, want env-token", cfg.GitHub.Token)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
github:
  token: test-token
  repository: owner/repo
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Workflow.Interval != 20 {
		t.Errorf("Default workflow interval = %v, want 20", cfg.Workflow.Interval)
	}

	if cfg.Workflow.ClosedIssueCleanupInterval != 300 {
		t.Errorf("Default closed issue cleanup interval = %v, want 300", cfg.Workflow.ClosedIssueCleanupInterval)
	}

	if cfg.Workflow.TmuxCommandDelay != 3 {
		t.Errorf("Default tmux command delay = %v, want 3", cfg.Workflow.TmuxCommandDelay)
	}

	if cfg.Git.WorktreeBasePath != ".git/soba/worktrees" {
		t.Errorf("Default git worktree base path = %v, want .git/soba/worktrees", cfg.Git.WorktreeBasePath)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yml")
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got nil")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
github:
  token: test-token
  repository: [this is invalid yaml
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Errorf("Expected error for invalid YAML, got nil")
	}
}

func TestConfigStructFields(t *testing.T) {
	cfg := Config{
		GitHub: GitHubConfig{
			Token:      "test",
			Repository: "test/repo",
			AuthMethod: "token",
		},
		Workflow: WorkflowConfig{
			Interval:                   20,
			UseTmux:                    true,
			AutoMergeEnabled:           true,
			ClosedIssueCleanupEnabled:  true,
			ClosedIssueCleanupInterval: 300,
			TmuxCommandDelay:           3,
		},
		Slack: SlackConfig{
			WebhookURL:           "https://test.com",
			NotificationsEnabled: true,
		},
		Git: GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
		},
		Phase: PhaseConfig{
			Plan: PhaseCommand{
				Command:   "/soba:plan",
				Options:   []string{},
				Parameter: "{{ISSUE_NUMBER}}",
			},
		},
	}

	if reflect.TypeOf(cfg.GitHub).Kind() != reflect.Struct {
		t.Errorf("GitHub config should be a struct")
	}

	if reflect.TypeOf(cfg.Workflow).Kind() != reflect.Struct {
		t.Errorf("Workflow config should be a struct")
	}

	if reflect.TypeOf(cfg.Slack).Kind() != reflect.Struct {
		t.Errorf("Slack config should be a struct")
	}

	if reflect.TypeOf(cfg.Git).Kind() != reflect.Struct {
		t.Errorf("Git config should be a struct")
	}

	if reflect.TypeOf(cfg.Phase).Kind() != reflect.Struct {
		t.Errorf("Phase config should be a struct")
	}
}

func TestExpandEnvVarsWithConditionalWarnings(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		content        string
		expectedWarn   []string
		unexpectedWarn []string
	}{
		{
			name: "GITHUB_TOKEN warning when auth_method is env",
			config: Config{
				GitHub: GitHubConfig{AuthMethod: "env"},
			},
			content:        "token: ${GITHUB_TOKEN}",
			expectedWarn:   []string{"Warning: undefined environment variable: GITHUB_TOKEN"},
			unexpectedWarn: []string{},
		},
		{
			name: "No GITHUB_TOKEN warning when auth_method is gh",
			config: Config{
				GitHub: GitHubConfig{AuthMethod: "gh"},
			},
			content:        "token: ${GITHUB_TOKEN}",
			expectedWarn:   []string{},
			unexpectedWarn: []string{"Warning: undefined environment variable: GITHUB_TOKEN"},
		},
		{
			name: "SLACK_WEBHOOK_URL warning when notifications_enabled is true",
			config: Config{
				Slack: SlackConfig{NotificationsEnabled: true},
			},
			content:        "webhook_url: ${SLACK_WEBHOOK_URL}",
			expectedWarn:   []string{"Warning: undefined environment variable: SLACK_WEBHOOK_URL"},
			unexpectedWarn: []string{},
		},
		{
			name: "No SLACK_WEBHOOK_URL warning when notifications_enabled is false",
			config: Config{
				Slack: SlackConfig{NotificationsEnabled: false},
			},
			content:        "webhook_url: ${SLACK_WEBHOOK_URL}",
			expectedWarn:   []string{},
			unexpectedWarn: []string{"Warning: undefined environment variable: SLACK_WEBHOOK_URL"},
		},
		{
			name: "Both warnings when both conditions are met",
			config: Config{
				GitHub: GitHubConfig{AuthMethod: "env"},
				Slack:  SlackConfig{NotificationsEnabled: true},
			},
			content: "token: ${GITHUB_TOKEN}\nwebhook_url: ${SLACK_WEBHOOK_URL}",
			expectedWarn: []string{
				"Warning: undefined environment variable: GITHUB_TOKEN",
				"Warning: undefined environment variable: SLACK_WEBHOOK_URL",
			},
			unexpectedWarn: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Unset environment variables to trigger warnings
			os.Unsetenv("GITHUB_TOKEN")
			os.Unsetenv("SLACK_WEBHOOK_URL")

			// Call expandEnvVarsWithConfig (function we'll implement)
			result := expandEnvVarsWithConfig(tt.content, &tt.config)

			// Restore stderr and capture output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderrOutput := buf.String()

			// Check for expected warnings
			for _, expectedWarn := range tt.expectedWarn {
				if !strings.Contains(stderrOutput, expectedWarn) {
					t.Errorf("Expected warning '%s' not found in stderr: %s", expectedWarn, stderrOutput)
				}
			}

			// Check for unexpected warnings
			for _, unexpectedWarn := range tt.unexpectedWarn {
				if strings.Contains(stderrOutput, unexpectedWarn) {
					t.Errorf("Unexpected warning '%s' found in stderr: %s", unexpectedWarn, stderrOutput)
				}
			}

			// Verify result contains unexpanded variables when warnings occur
			if len(tt.expectedWarn) > 0 {
				if !strings.Contains(result, "${") {
					t.Errorf("Expected unexpanded variables in result when warnings occur: %s", result)
				}
			}
		})
	}
}

func TestLoadConfigWithConditionalWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	tests := []struct {
		name          string
		configContent string
		expectedWarn  []string
	}{
		{
			name: "No warnings with default config (auth_method=gh, notifications_enabled=false)",
			configContent: `
github:
  auth_method: gh
  token: ${GITHUB_TOKEN}
  repository: owner/repo

slack:
  webhook_url: ${SLACK_WEBHOOK_URL}
  notifications_enabled: false
`,
			expectedWarn: []string{},
		},
		{
			name: "GITHUB_TOKEN warning with auth_method=env",
			configContent: `
github:
  auth_method: env
  token: ${GITHUB_TOKEN}
  repository: owner/repo

slack:
  webhook_url: ${SLACK_WEBHOOK_URL}
  notifications_enabled: false
`,
			expectedWarn: []string{"Warning: undefined environment variable: GITHUB_TOKEN"},
		},
		{
			name: "SLACK_WEBHOOK_URL warning with notifications_enabled=true",
			configContent: `
github:
  auth_method: gh
  token: ${GITHUB_TOKEN}
  repository: owner/repo

slack:
  webhook_url: ${SLACK_WEBHOOK_URL}
  notifications_enabled: true
`,
			expectedWarn: []string{"Warning: undefined environment variable: SLACK_WEBHOOK_URL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Unset environment variables to trigger warnings
			os.Unsetenv("GITHUB_TOKEN")
			os.Unsetenv("SLACK_WEBHOOK_URL")

			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			_, err = Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Restore stderr and capture output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderrOutput := buf.String()

			// Check for expected warnings
			for _, expectedWarn := range tt.expectedWarn {
				if !strings.Contains(stderrOutput, expectedWarn) {
					t.Errorf("Expected warning '%s' not found in stderr: %s", expectedWarn, stderrOutput)
				}
			}

			// Remove test file for next iteration
			os.Remove(configPath)
		})
	}
}

func TestLoadConfigWithLogSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
github:
  token: test-token
  repository: owner/repo

log:
  output_path: /custom/path/logs/soba.log
  retention_count: 5
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Log.OutputPath != "/custom/path/logs/soba.log" {
		t.Errorf("Log output_path = %v, want /custom/path/logs/soba.log", cfg.Log.OutputPath)
	}

	if cfg.Log.RetentionCount != 5 {
		t.Errorf("Log retention_count = %v, want 5", cfg.Log.RetentionCount)
	}
}

func TestLoadConfigWithDefaultLogSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
github:
  token: test-token
  repository: owner/repo
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	expectedPath := fmt.Sprintf(".soba/logs/soba-%d.log", os.Getpid())
	if cfg.Log.OutputPath != expectedPath {
		t.Errorf("Default log output_path = %v, want %v", cfg.Log.OutputPath, expectedPath)
	}

	if cfg.Log.RetentionCount != 10 {
		t.Errorf("Default log retention_count = %v, want 10", cfg.Log.RetentionCount)
	}
}

func TestPIDVariableNoWarning(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// PIDを含む設定ファイルを作成
	configContent := `
github:
  token: test-token
  repository: owner/repo

log:
  output_path: .soba/logs/soba-${PID}.log
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// stderrをキャプチャ
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// PID環境変数を未設定にする
	os.Unsetenv("PID")

	// 設定を読み込む
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// stderrを復元して出力をキャプチャ
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// PIDに対する警告が出ていないことを確認
	if strings.Contains(stderrOutput, "Warning: undefined environment variable: PID") {
		t.Errorf("Unexpected warning for PID variable found in stderr: %s", stderrOutput)
	}

	// ${PID}が置換されずに残っていることを確認（daemonで置換される）
	if !strings.Contains(cfg.Log.OutputPath, "${PID}") {
		t.Errorf("PID variable should not be expanded in config.Load, got: %s", cfg.Log.OutputPath)
	}
}

func TestOtherEnvVarWarningStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// 他の環境変数を含む設定ファイルを作成
	configContent := `
github:
  token: test-token
  repository: owner/repo

log:
  output_path: ${CUSTOM_LOG_PATH}/soba-${PID}.log
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// stderrをキャプチャ
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// 環境変数を未設定にする
	os.Unsetenv("CUSTOM_LOG_PATH")
	os.Unsetenv("PID")

	// 設定を読み込む
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// stderrを復元して出力をキャプチャ
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// CUSTOM_LOG_PATHに対する警告が出ていることを確認
	if !strings.Contains(stderrOutput, "Warning: undefined environment variable: CUSTOM_LOG_PATH") {
		t.Errorf("Expected warning for CUSTOM_LOG_PATH not found in stderr: %s", stderrOutput)
	}

	// PIDに対する警告が出ていないことを確認
	if strings.Contains(stderrOutput, "Warning: undefined environment variable: PID") {
		t.Errorf("Unexpected warning for PID variable found in stderr: %s", stderrOutput)
	}

	// 両方の変数が未展開で残っていることを確認
	if !strings.Contains(cfg.Log.OutputPath, "${CUSTOM_LOG_PATH}") {
		t.Errorf("CUSTOM_LOG_PATH should not be expanded when undefined, got: %s", cfg.Log.OutputPath)
	}
	if !strings.Contains(cfg.Log.OutputPath, "${PID}") {
		t.Errorf("PID should not be expanded in config.Load, got: %s", cfg.Log.OutputPath)
	}
}

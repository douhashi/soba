package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMaskSensitiveConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected []string
		notWant  []string
	}{
		{
			name: "マスク対象フィールドが正しくマスクされること",
			config: Config{
				GitHub: GitHubConfig{
					Token:      "ghp_secret_token_123",
					Repository: "douhashi/soba",
					AuthMethod: "token",
				},
				Slack: SlackConfig{
					WebhookURL:           "https://hooks.slack.com/services/T123/B456/xxx",
					NotificationsEnabled: true,
				},
			},
			expected: []string{
				"token: '***MASKED***'",
				"repository: douhashi/soba",
				"webhook_url: '***MASKED***'",
			},
			notWant: []string{
				"ghp_secret_token_123",
				"https://hooks.slack.com",
			},
		},
		{
			name: "空の機密情報も正しく処理されること",
			config: Config{
				GitHub: GitHubConfig{
					Token:      "",
					Repository: "test/repo",
				},
				Slack: SlackConfig{
					WebhookURL: "",
				},
			},
			expected: []string{
				"token: '***MASKED***'",
				"webhook_url: '***MASKED***'",
			},
		},
		{
			name: "環境変数プレースホルダーもマスクされること",
			config: Config{
				GitHub: GitHubConfig{
					Token: "${GITHUB_TOKEN}",
				},
				Slack: SlackConfig{
					WebhookURL: "${SLACK_WEBHOOK_URL}",
				},
			},
			expected: []string{
				"token: '***MASKED***'",
				"webhook_url: '***MASKED***'",
			},
			notWant: []string{
				"${GITHUB_TOKEN}",
				"${SLACK_WEBHOOK_URL}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := MaskSensitiveConfig(&tt.config)

			// YAMLに変換
			data, err := yaml.Marshal(masked)
			require.NoError(t, err)

			output := string(data)

			// 期待される文字列が含まれていることを確認
			for _, expected := range tt.expected {
				assert.Contains(t, output, expected)
			}

			// 含まれてはいけない文字列が含まれていないことを確認
			for _, notWant := range tt.notWant {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}

func TestDisplayConfig(t *testing.T) {
	config := &Config{
		GitHub: GitHubConfig{
			Token:      "secret_token",
			Repository: "test/repo",
			AuthMethod: "token",
		},
		Workflow: WorkflowConfig{
			Interval:                   30,
			UseTmux:                    true,
			AutoMergeEnabled:           false,
			ClosedIssueCleanupEnabled:  true,
			ClosedIssueCleanupInterval: 300,
			TmuxCommandDelay:           3,
		},
		Slack: SlackConfig{
			WebhookURL:           "https://slack.webhook.url",
			NotificationsEnabled: true,
		},
		Git: GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
		},
		Phase: PhaseConfig{
			Plan: PhaseCommand{
				Command:   "plan.sh",
				Options:   []string{"-v"},
				Parameter: "issue_number",
			},
		},
	}

	output, err := DisplayConfig(config)
	require.NoError(t, err)

	// トークンとWebhook URLがマスクされていることを確認
	assert.Contains(t, output, "***MASKED***")
	assert.NotContains(t, output, "secret_token")
	assert.NotContains(t, output, "https://slack.webhook.url")

	// その他の設定が正しく表示されていることを確認
	assert.Contains(t, output, "repository: test/repo")
	assert.Contains(t, output, "interval: 30")
	assert.Contains(t, output, "use_tmux: true")
	assert.Contains(t, output, "worktree_base_path: .git/soba/worktrees")

	// YAML形式であることを確認
	lines := strings.Split(output, "\n")
	assert.True(t, strings.HasPrefix(lines[0], "github:"))
}

func TestDisplayConfig_NilConfig(t *testing.T) {
	output, err := DisplayConfig(nil)
	assert.Error(t, err)
	assert.Empty(t, output)
	assert.Contains(t, err.Error(), "config is nil")
}

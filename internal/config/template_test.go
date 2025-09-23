package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGenerateTemplate(t *testing.T) {
	t.Run("should generate valid YAML template", func(t *testing.T) {
		// Execute
		template := GenerateTemplate()

		// Assert - template is not empty
		assert.NotEmpty(t, template)

		// Assert - contains expected sections
		assert.Contains(t, template, "github:")
		assert.Contains(t, template, "workflow:")
		assert.Contains(t, template, "slack:")
		assert.Contains(t, template, "git:")
		assert.Contains(t, template, "phase:")

		// Assert - valid YAML structure
		var config Config
		err := yaml.Unmarshal([]byte(template), &config)
		assert.NoError(t, err, "Template should be valid YAML")
	})

	t.Run("should include default values", func(t *testing.T) {
		template := GenerateTemplate()

		// Check default values are present
		assert.Contains(t, template, "auth_method: gh")
		assert.Contains(t, template, "repository: ")
		assert.NotContains(t, template, "douhashi/soba-cli")
		assert.Contains(t, template, "interval: 20")
		assert.Contains(t, template, "use_tmux: true")
		assert.Contains(t, template, "auto_merge_enabled: true")
		assert.Contains(t, template, "closed_issue_cleanup_enabled: true")
		assert.Contains(t, template, "closed_issue_cleanup_interval: 300")
		assert.Contains(t, template, "tmux_command_delay: 3")
		assert.Contains(t, template, "notifications_enabled: false")
		assert.NotContains(t, template, "setup_workspace")
		assert.Contains(t, template, "worktree_base_path: .git/soba/worktrees")
	})

	t.Run("should include environment variable placeholders", func(t *testing.T) {
		template := GenerateTemplate()

		// Check environment variable placeholders
		assert.Contains(t, template, "${SLACK_WEBHOOK_URL}")
		// Check comment about GITHUB_TOKEN
		assert.Contains(t, template, "${GITHUB_TOKEN}")
	})

	t.Run("should include helpful comments", func(t *testing.T) {
		template := GenerateTemplate()

		// Check for helpful comments
		assert.Contains(t, template, "# GitHub settings")
		assert.Contains(t, template, "# Authentication method:")
		assert.Contains(t, template, "# Workflow settings")
		assert.Contains(t, template, "# Slack notifications")
		assert.Contains(t, template, "# Git settings")
		assert.Contains(t, template, "# Phase commands")
	})

	t.Run("should include phase commands", func(t *testing.T) {
		template := GenerateTemplate()

		// Check phase commands
		assert.Contains(t, template, "plan:")
		assert.Contains(t, template, "implement:")
		assert.Contains(t, template, "review:")
		assert.Contains(t, template, "revise:")

		// Check command structure
		assert.Contains(t, template, "command: claude")
		assert.Contains(t, template, "options:")
		assert.Contains(t, template, "--dangerously-skip-permissions")
		assert.Contains(t, template, "parameter:")
		assert.Contains(t, template, "{{issue-number}}")
	})

	t.Run("generated template should be loadable by Load", func(t *testing.T) {
		template := GenerateTemplate()

		// Parse the YAML
		var config Config
		err := yaml.Unmarshal([]byte(template), &config)
		assert.NoError(t, err)

		// Verify structure is correct
		assert.Equal(t, "gh", config.GitHub.AuthMethod)
		assert.Equal(t, "", config.GitHub.Repository)
		assert.Equal(t, 20, config.Workflow.Interval)
		assert.True(t, config.Workflow.UseTmux)
		assert.True(t, config.Workflow.AutoMergeEnabled)
		assert.True(t, config.Workflow.ClosedIssueCleanupEnabled)
		assert.Equal(t, 300, config.Workflow.ClosedIssueCleanupInterval)
		assert.Equal(t, 3, config.Workflow.TmuxCommandDelay)
		assert.False(t, config.Slack.NotificationsEnabled)
		assert.Equal(t, ".git/soba/worktrees", config.Git.WorktreeBasePath)

		// Verify phase commands
		assert.NotNil(t, config.Phase)
		assert.NotNil(t, config.Phase.Plan)
		assert.Equal(t, "claude", config.Phase.Plan.Command)
		assert.Contains(t, config.Phase.Plan.Options, "--dangerously-skip-permissions")
		assert.Equal(t, "/soba:plan {{issue-number}}", config.Phase.Plan.Parameter)
	})

	t.Run("should handle multiline format correctly", func(t *testing.T) {
		template := GenerateTemplate()

		// Ensure proper line breaks
		lines := strings.Split(template, "\n")
		assert.Greater(t, len(lines), 50, "Template should have multiple lines")

		// Check that we have proper YAML structure
		// Count top-level keys
		topLevelKeys := 0
		for _, line := range lines {
			if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "#") {
				topLevelKeys++
			}
		}
		assert.GreaterOrEqual(t, topLevelKeys, 5, "Should have at least 5 top-level keys (github, workflow, slack, git, phase)")
	})
}

func TestGenerateTemplateWithOptions(t *testing.T) {
	t.Run("should use custom repository", func(t *testing.T) {
		opts := &TemplateOptions{
			Repository: "myorg/myrepo",
		}
		template := GenerateTemplateWithOptions(opts)

		// Assert custom repository is used
		assert.Contains(t, template, "repository: myorg/myrepo")
		assert.NotContains(t, template, "douhashi/soba-cli")
	})

	t.Run("should use empty repository when opts is nil", func(t *testing.T) {
		template := GenerateTemplateWithOptions(nil)

		// Assert empty repository is used
		assert.Contains(t, template, "repository: ")
		assert.NotContains(t, template, "douhashi/soba-cli")
	})

	t.Run("should use empty values when repository is empty", func(t *testing.T) {
		opts := &TemplateOptions{
			Repository: "",
		}
		template := GenerateTemplateWithOptions(opts)

		// Assert empty repository is used
		assert.Contains(t, template, "repository: ")
		assert.NotContains(t, template, "douhashi/soba-cli")
	})

	t.Run("generated template with custom options should be valid YAML", func(t *testing.T) {
		opts := &TemplateOptions{
			Repository: "test/repo",
		}
		template := GenerateTemplateWithOptions(opts)

		// Parse the YAML
		var config Config
		err := yaml.Unmarshal([]byte(template), &config)
		assert.NoError(t, err)

		// Verify custom repository is set
		assert.Equal(t, "test/repo", config.GitHub.Repository)
	})
}

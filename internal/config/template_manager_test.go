package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigTemplateManager_NewTemplateManager(t *testing.T) {
	manager := NewTemplateManager()
	assert.NotNil(t, manager)
}

func TestConfigTemplateManager_RenderTemplate(t *testing.T) {
	tests := []struct {
		name           string
		options        *TemplateOptions
		wantContain    []string
		wantNotContain []string
	}{
		{
			name: "default values",
			options: &TemplateOptions{
				Repository: "douhashi/soba-cli",
				LogLevel:   "info",
			},
			wantContain: []string{
				"repository: douhashi/soba-cli",
				"level: info",
				"# GitHub settings",
				"# Workflow settings",
				"# Slack notifications",
				"# Git settings",
				"# Logging settings",
				"# Phase commands",
			},
			wantNotContain: []string{
				"{{.Repository}}",
				"{{.LogLevel}}",
				"level: warn", // should not contain old default log level value
			},
		},
		{
			name: "custom repository",
			options: &TemplateOptions{
				Repository: "myorg/myrepo",
				LogLevel:   "info",
			},
			wantContain: []string{
				"repository: myorg/myrepo",
				"level: info",
			},
			wantNotContain: []string{
				"douhashi/soba-cli",
			},
		},
		{
			name: "debug log level",
			options: &TemplateOptions{
				Repository: "test/repo",
				LogLevel:   "debug",
			},
			wantContain: []string{
				"repository: test/repo",
				"level: debug",
			},
			wantNotContain: []string{
				"level: info",
				"level: warn",
			},
		},
		{
			name: "empty repository should remain empty",
			options: &TemplateOptions{
				Repository: "",
				LogLevel:   "info",
			},
			wantContain: []string{
				"repository: ",
				"level: info",
			},
			wantNotContain: []string{
				"douhashi/soba-cli",
			},
		},
		{
			name: "empty log level fallback to default",
			options: &TemplateOptions{
				Repository: "test/repo",
				LogLevel:   "",
			},
			wantContain: []string{
				"repository: test/repo",
				"level: info",
			},
		},
		{
			name:    "nil options uses empty repository and default log level",
			options: nil,
			wantContain: []string{
				"repository: ",
				"level: info",
			},
			wantNotContain: []string{
				"douhashi/soba-cli",
			},
		},
	}

	manager := NewTemplateManager()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.RenderTemplate(tt.options)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			// Check that wanted strings are present
			for _, want := range tt.wantContain {
				assert.Contains(t, result, want, "Should contain: %s", want)
			}

			// Check that unwanted strings are not present
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant, "Should not contain: %s", notWant)
			}
		})
	}
}

func TestConfigTemplateManager_RenderTemplate_Structure(t *testing.T) {
	manager := NewTemplateManager()

	result, err := manager.RenderTemplate(&TemplateOptions{
		Repository: "test/repo",
		LogLevel:   "info",
	})
	require.NoError(t, err)

	// Verify that all major sections are present
	sections := []string{
		"# GitHub settings",
		"# Workflow settings",
		"# Slack notifications",
		"# Git settings",
		"# Logging settings",
		"# Phase commands",
	}

	for _, section := range sections {
		assert.Contains(t, result, section, "Config should contain section: %s", section)
	}

	// Verify that phase commands are properly escaped
	assert.Contains(t, result, "/soba:plan {{issue-number}}", "Phase commands should contain proper placeholder")
	assert.Contains(t, result, "/soba:implement {{issue-number}}", "Phase commands should contain proper placeholder")
	assert.Contains(t, result, "/soba:review {{issue-number}}", "Phase commands should contain proper placeholder")
	assert.Contains(t, result, "/soba:revise {{issue-number}}", "Phase commands should contain proper placeholder")
}

func TestConfigTemplateManager_RenderTemplate_YAMLValidity(t *testing.T) {
	manager := NewTemplateManager()

	result, err := manager.RenderTemplate(&TemplateOptions{
		Repository: "test/repo",
		LogLevel:   "info",
	})
	require.NoError(t, err)

	// Basic YAML structure checks
	lines := strings.Split(result, "\n")

	// Check indentation is consistent (uses spaces, not tabs)
	for i, line := range lines {
		if strings.HasPrefix(line, "\t") {
			t.Errorf("Line %d should not start with tab: %s", i+1, line)
		}
	}

	// Check that key-value pairs are properly formatted
	assert.Regexp(t, `auth_method:\s+gh`, result)
	assert.Regexp(t, `repository:\s+test/repo`, result)
	assert.Regexp(t, `interval:\s+20`, result)
	assert.Regexp(t, `level:\s+info`, result)
}

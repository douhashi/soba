// Package config provides configuration loading and management functionality.
package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"

	"github.com/douhashi/soba/internal/infra"
)

type Config struct {
	GitHub   GitHubConfig   `yaml:"github"`
	Workflow WorkflowConfig `yaml:"workflow"`
	Slack    SlackConfig    `yaml:"slack"`
	Git      GitConfig      `yaml:"git"`
	Phase    PhaseConfig    `yaml:"phase"`
}

type GitHubConfig struct {
	Token      string `yaml:"token"`
	Repository string `yaml:"repository"`
	AuthMethod string `yaml:"auth_method"`
}

type WorkflowConfig struct {
	Interval                   int  `yaml:"interval"`
	UseTmux                    bool `yaml:"use_tmux"`
	AutoMergeEnabled           bool `yaml:"auto_merge_enabled"`
	ClosedIssueCleanupEnabled  bool `yaml:"closed_issue_cleanup_enabled"`
	ClosedIssueCleanupInterval int  `yaml:"closed_issue_cleanup_interval"`
	TmuxCommandDelay           int  `yaml:"tmux_command_delay"`
}

type SlackConfig struct {
	WebhookURL           string `yaml:"webhook_url"`
	NotificationsEnabled bool   `yaml:"notifications_enabled"`
}

type GitConfig struct {
	WorktreeBasePath string `yaml:"worktree_base_path"`
	SetupWorkspace   bool   `yaml:"setup_workspace"`
	BaseBranch       string `yaml:"base_branch"`
}

type PhaseConfig struct {
	Plan      PhaseCommand `yaml:"plan"`
	Implement PhaseCommand `yaml:"implement"`
	Review    PhaseCommand `yaml:"review"`
	Revise    PhaseCommand `yaml:"revise"`
}

type PhaseCommand struct {
	Command   string   `yaml:"command"`
	Options   []string `yaml:"options"`
	Parameter string   `yaml:"parameter"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, infra.NewConfigLoadError(path, "file not found")
		}
		if os.IsPermission(err) {
			return nil, infra.NewConfigLoadError(path, "permission denied")
		}
		return nil, infra.WrapInfraError(err, "failed to read config file")
	}

	content := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, infra.NewConfigLoadError(path, "invalid YAML format")
	}

	cfg.setDefaults()

	return &cfg, nil
}

func expandEnvVars(content string) string {
	return os.Expand(content, func(key string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		// Log warning for undefined environment variable
		fmt.Fprintf(os.Stderr, "Warning: undefined environment variable: %s\n", key)
		return "${" + key + "}"
	})
}

func (c *Config) setDefaults() {
	if c.Workflow.Interval == 0 {
		c.Workflow.Interval = 20
	}
	if c.Workflow.ClosedIssueCleanupInterval == 0 {
		c.Workflow.ClosedIssueCleanupInterval = 300
	}
	if c.Workflow.TmuxCommandDelay == 0 {
		c.Workflow.TmuxCommandDelay = 3
	}
	if c.Git.WorktreeBasePath == "" {
		c.Git.WorktreeBasePath = ".git/soba/worktrees"
	}
	if c.Git.BaseBranch == "" {
		c.Git.BaseBranch = "main"
	}
}

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
	Log      LogConfig      `yaml:"log"`
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

type LogConfig struct {
	OutputPath     string `yaml:"output_path"`
	RetentionCount int    `yaml:"retention_count"`
	Level          string `yaml:"level"`
	Format         string `yaml:"format"` // "json" or "text"
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config when file doesn't exist
			cfg := &Config{}
			cfg.setDefaults()
			return cfg, nil
		}
		if os.IsPermission(err) {
			return nil, infra.NewConfigLoadError(path, "permission denied")
		}
		return nil, infra.WrapInfraError(err, "failed to read config file")
	}

	// First pass: parse config without environment variable expansion to get conditional settings
	var tempCfg Config
	if err := yaml.Unmarshal(data, &tempCfg); err != nil {
		return nil, infra.NewConfigLoadError(path, "invalid YAML format")
	}
	tempCfg.setDefaults()

	// Second pass: expand environment variables with conditional warnings based on parsed config
	content := expandEnvVarsWithConfig(string(data), &tempCfg)

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, infra.NewConfigLoadError(path, "invalid YAML format")
	}

	cfg.setDefaults()

	return &cfg, nil
}

// expandEnvVarsWithConfig expands environment variables with conditional warnings
// based on the configuration settings
func expandEnvVarsWithConfig(content string, cfg *Config) string {
	return os.Expand(content, func(key string) string {
		// Special handling for PID - don't expand it here, it's replaced at daemon startup
		if key == "PID" {
			return "${PID}"
		}

		if value, ok := os.LookupEnv(key); ok {
			return value
		}

		// Check if warning should be shown based on configuration
		shouldWarn := shouldWarnForEnvVar(key, cfg)
		if shouldWarn {
			fmt.Fprintf(os.Stderr, "Warning: undefined environment variable: %s\n", key)
		}

		return "${" + key + "}"
	})
}

// shouldWarnForEnvVar determines if a warning should be shown for a missing environment variable
// This function uses the EnvVarClassifier to categorize variables and apply appropriate warning logic
func shouldWarnForEnvVar(key string, cfg *Config) bool {
	classifier := NewEnvVarClassifier()
	return classifier.ShouldWarn(key, cfg)
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
		c.Git.WorktreeBasePath = DefaultWorktreeBasePath
	}
	if c.Git.BaseBranch == "" {
		c.Git.BaseBranch = "main"
	}
	if c.Log.OutputPath == "" {
		c.Log.OutputPath = fmt.Sprintf(".soba/logs/soba-%d.log", os.Getpid())
	}
	if c.Log.RetentionCount == 0 {
		c.Log.RetentionCount = 10
	}
	if c.Log.Format == "" {
		c.Log.Format = "text" // Default to text format
	}
}

package config

import (
	"testing"
)

func TestEnvVarCategoryString(t *testing.T) {
	tests := []struct {
		category EnvVarCategory
		expected string
	}{
		{SystemVariable, "system"},
		{ConditionalVariable, "conditional"},
		{UserVariable, "user"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.category.String(); got != tt.expected {
				t.Errorf("EnvVarCategory.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClassifyEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		config   *Config
		expected EnvVarCategory
	}{
		{
			name:     "PID is system variable",
			envVar:   "PID",
			config:   &Config{},
			expected: SystemVariable,
		},
		{
			name:     "GITHUB_TOKEN is conditional variable",
			envVar:   "GITHUB_TOKEN",
			config:   &Config{},
			expected: ConditionalVariable,
		},
		{
			name:     "SLACK_WEBHOOK_URL is conditional variable",
			envVar:   "SLACK_WEBHOOK_URL",
			config:   &Config{},
			expected: ConditionalVariable,
		},
		{
			name:     "Custom variable is user variable",
			envVar:   "CUSTOM_VAR",
			config:   &Config{},
			expected: UserVariable,
		},
		{
			name:     "Empty string is user variable",
			envVar:   "",
			config:   &Config{},
			expected: UserVariable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewEnvVarClassifier()
			if got := classifier.Classify(tt.envVar, tt.config); got != tt.expected {
				t.Errorf("EnvVarClassifier.Classify(%q) = %v, want %v", tt.envVar, got, tt.expected)
			}
		})
	}
}

func TestDefaultEnvVarClassifier_ShouldWarn(t *testing.T) {
	tests := []struct {
		name       string
		envVar     string
		config     *Config
		shouldWarn bool
	}{
		{
			name:       "System variable PID never warns",
			envVar:     "PID",
			config:     &Config{},
			shouldWarn: false,
		},
		{
			name:   "GITHUB_TOKEN warns when auth_method is env",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "env"},
			},
			shouldWarn: true,
		},
		{
			name:   "GITHUB_TOKEN does not warn when auth_method is gh",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "gh"},
			},
			shouldWarn: false,
		},
		{
			name:   "GITHUB_TOKEN does not warn when auth_method is token",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "token"},
			},
			shouldWarn: false,
		},
		{
			name:   "SLACK_WEBHOOK_URL warns when notifications_enabled is true",
			envVar: "SLACK_WEBHOOK_URL",
			config: &Config{
				Slack: SlackConfig{NotificationsEnabled: true},
			},
			shouldWarn: true,
		},
		{
			name:   "SLACK_WEBHOOK_URL does not warn when notifications_enabled is false",
			envVar: "SLACK_WEBHOOK_URL",
			config: &Config{
				Slack: SlackConfig{NotificationsEnabled: false},
			},
			shouldWarn: false,
		},
		{
			name:       "User variable always warns",
			envVar:     "CUSTOM_VAR",
			config:     &Config{},
			shouldWarn: true,
		},
		{
			name:       "Unknown variable always warns",
			envVar:     "UNKNOWN_VAR",
			config:     &Config{},
			shouldWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewEnvVarClassifier()
			if got := classifier.ShouldWarn(tt.envVar, tt.config); got != tt.shouldWarn {
				t.Errorf("EnvVarClassifier.ShouldWarn(%q) = %v, want %v", tt.envVar, got, tt.shouldWarn)
			}
		})
	}
}

func TestIsSystemVariable(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		want   bool
	}{
		{"PID is system variable", "PID", true},
		{"pid lowercase is not system variable", "pid", false},
		{"GITHUB_TOKEN is not system variable", "GITHUB_TOKEN", false},
		{"SLACK_WEBHOOK_URL is not system variable", "SLACK_WEBHOOK_URL", false},
		{"CUSTOM_VAR is not system variable", "CUSTOM_VAR", false},
		{"Empty string is not system variable", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := &defaultEnvVarClassifier{}
			if got := classifier.isSystemVariable(tt.envVar); got != tt.want {
				t.Errorf("isSystemVariable(%q) = %v, want %v", tt.envVar, got, tt.want)
			}
		})
	}
}

func TestIsConditionalVariable(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		want   bool
	}{
		{"GITHUB_TOKEN is conditional", "GITHUB_TOKEN", true},
		{"SLACK_WEBHOOK_URL is conditional", "SLACK_WEBHOOK_URL", true},
		{"PID is not conditional", "PID", false},
		{"CUSTOM_VAR is not conditional", "CUSTOM_VAR", false},
		{"github_token lowercase is not conditional", "github_token", false},
		{"Empty string is not conditional", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := &defaultEnvVarClassifier{}
			if got := classifier.isConditionalVariable(tt.envVar); got != tt.want {
				t.Errorf("isConditionalVariable(%q) = %v, want %v", tt.envVar, got, tt.want)
			}
		})
	}
}

func TestShouldWarnForConditionalVariable(t *testing.T) {
	tests := []struct {
		name       string
		envVar     string
		config     *Config
		shouldWarn bool
	}{
		{
			name:   "GITHUB_TOKEN warns with auth_method=env",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "env"},
			},
			shouldWarn: true,
		},
		{
			name:   "GITHUB_TOKEN no warn with auth_method=gh",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "gh"},
			},
			shouldWarn: false,
		},
		{
			name:   "SLACK_WEBHOOK_URL warns with notifications_enabled=true",
			envVar: "SLACK_WEBHOOK_URL",
			config: &Config{
				Slack: SlackConfig{NotificationsEnabled: true},
			},
			shouldWarn: true,
		},
		{
			name:   "SLACK_WEBHOOK_URL no warn with notifications_enabled=false",
			envVar: "SLACK_WEBHOOK_URL",
			config: &Config{
				Slack: SlackConfig{NotificationsEnabled: false},
			},
			shouldWarn: false,
		},
		{
			name:       "Unknown conditional variable defaults to no warning",
			envVar:     "UNKNOWN_CONDITIONAL",
			config:     &Config{},
			shouldWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := &defaultEnvVarClassifier{}
			if got := classifier.shouldWarnForConditionalVariable(tt.envVar, tt.config); got != tt.shouldWarn {
				t.Errorf("shouldWarnForConditionalVariable(%q) = %v, want %v", tt.envVar, got, tt.shouldWarn)
			}
		})
	}
}

// TestShouldWarnForEnvVarWithClassifier tests the refactored shouldWarnForEnvVar function
func TestShouldWarnForEnvVarWithClassifier(t *testing.T) {
	tests := []struct {
		name       string
		envVar     string
		config     *Config
		shouldWarn bool
	}{
		// System variables
		{
			name:       "PID never warns",
			envVar:     "PID",
			config:     &Config{},
			shouldWarn: false,
		},
		// Conditional variables - GITHUB_TOKEN
		{
			name:   "GITHUB_TOKEN warns when auth_method is env",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "env"},
			},
			shouldWarn: true,
		},
		{
			name:   "GITHUB_TOKEN no warn when auth_method is gh",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "gh"},
			},
			shouldWarn: false,
		},
		{
			name:   "GITHUB_TOKEN no warn when auth_method is token",
			envVar: "GITHUB_TOKEN",
			config: &Config{
				GitHub: GitHubConfig{AuthMethod: "token"},
			},
			shouldWarn: false,
		},
		// Conditional variables - SLACK_WEBHOOK_URL
		{
			name:   "SLACK_WEBHOOK_URL warns when notifications_enabled is true",
			envVar: "SLACK_WEBHOOK_URL",
			config: &Config{
				Slack: SlackConfig{NotificationsEnabled: true},
			},
			shouldWarn: true,
		},
		{
			name:   "SLACK_WEBHOOK_URL no warn when notifications_enabled is false",
			envVar: "SLACK_WEBHOOK_URL",
			config: &Config{
				Slack: SlackConfig{NotificationsEnabled: false},
			},
			shouldWarn: false,
		},
		// User variables
		{
			name:       "User variable always warns",
			envVar:     "CUSTOM_VAR",
			config:     &Config{},
			shouldWarn: true,
		},
		{
			name:       "Another user variable always warns",
			envVar:     "MY_APP_CONFIG",
			config:     &Config{},
			shouldWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test using new classifier directly
			classifier := NewEnvVarClassifier()
			got := classifier.ShouldWarn(tt.envVar, tt.config)
			if got != tt.shouldWarn {
				t.Errorf("EnvVarClassifier.ShouldWarn(%q) = %v, want %v", tt.envVar, got, tt.shouldWarn)
			}

			// Also test the original shouldWarnForEnvVar function (for backwards compatibility)
			originalGot := shouldWarnForEnvVar(tt.envVar, tt.config)
			if originalGot != tt.shouldWarn {
				t.Errorf("shouldWarnForEnvVar(%q) = %v, want %v", tt.envVar, originalGot, tt.shouldWarn)
			}
		})
	}
}

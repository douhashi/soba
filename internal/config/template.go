package config

// TemplateOptions holds the options for generating a template
type TemplateOptions struct {
	// Repository in format "owner/repo"
	Repository string
}

// GenerateTemplate generates the default configuration template for soba
func GenerateTemplate() string {
	return GenerateTemplateWithOptions(nil)
}

// GenerateTemplateWithOptions generates a configuration template with custom options
func GenerateTemplateWithOptions(opts *TemplateOptions) string {
	// Default repository
	repository := "douhashi/soba-cli"

	if opts != nil && opts.Repository != "" {
		repository = opts.Repository
	}
	return `# GitHub settings
github:
  # Authentication method: 'gh', 'env', or omit for auto-detect
  # Use 'gh' to use GitHub CLI authentication (gh auth token)
  # Use 'env' to use environment variable
  auth_method: gh  # or 'env', or omit for auto-detect

  # Personal Access Token (required when auth_method is 'env' or omitted)
  # Can use environment variable
  # token: ${GITHUB_TOKEN}

  # Target repository (format: owner/repo)
  repository: ` + repository + `

# Workflow settings
workflow:
  # Issue polling interval in seconds (default: 20)
  interval: 20
  # Use tmux for Claude execution (default: true)
  use_tmux: true
  # Enable automatic PR merging (default: true)
  auto_merge_enabled: true
  # Clean up tmux windows for closed issues (default: true)
  closed_issue_cleanup_enabled: true
  # Cleanup interval in seconds (default: 300)
  closed_issue_cleanup_interval: 300
  # Command delay for tmux panes in seconds (default: 3)
  tmux_command_delay: 3

# Slack notifications
slack:
  # Webhook URL for Slack notifications
  # Get your webhook URL from: https://api.slack.com/messaging/webhooks
  webhook_url: ${SLACK_WEBHOOK_URL}
  # Enable notifications for phase starts (default: false)
  notifications_enabled: false

# Git settings
git:
  # Base path for git worktrees
  worktree_base_path: .git/soba/worktrees

# Logging settings
log:
  # Log file output path (default: .soba/logs/soba-{pid}.log)
  # ${PID} will be replaced with actual process ID at runtime
  output_path: .soba/logs/soba-${PID}.log
  # Number of log files to retain (default: 10)
  retention_count: 10
  # Log level: debug, info, warn, error (default: warn)
  level: warn

# Phase commands (optional - for custom Claude commands)
phase:
  plan:
    command: claude
    options:
      - --dangerously-skip-permissions
    parameter: '/soba:plan {{issue-number}}'
  implement:
    command: claude
    options:
      - --dangerously-skip-permissions
    parameter: '/soba:implement {{issue-number}}'
  review:
    command: claude
    options:
      - --dangerously-skip-permissions
    parameter: '/soba:review {{issue-number}}'
  revise:
    command: claude
    options:
      - --dangerously-skip-permissions
    parameter: '/soba:revise {{issue-number}}'
`
}

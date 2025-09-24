package config

import (
	"embed"
	_ "embed"
)

//go:embed config_template.yml
var configTemplateContent string

//go:embed templates/claude/commands/soba/*
var ClaudeCommandsFS embed.FS

//go:embed templates/slack/*.json
var SlackTemplatesFS embed.FS

// GetSlackTemplatesFS returns the embedded Slack templates filesystem
func GetSlackTemplatesFS() embed.FS {
	return SlackTemplatesFS
}

// GetClaudeCommandsManager returns a manager for Claude command templates
func GetClaudeCommandsManager() *ClaudeCommandsManager {
	return NewClaudeCommandsManager(ClaudeCommandsFS, "templates/claude/commands/soba")
}

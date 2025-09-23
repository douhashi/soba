package config

import (
	"embed"
	_ "embed"
)

//go:embed config_template.yml
var configTemplateContent string

//go:embed templates/claude/commands/soba/*
var ClaudeCommandsFS embed.FS

// GetClaudeCommandsManager returns a manager for Claude command templates
func GetClaudeCommandsManager() *ClaudeCommandsManager {
	return NewClaudeCommandsManager(ClaudeCommandsFS, "templates/claude/commands/soba")
}

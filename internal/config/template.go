package config

// TemplateOptions holds the options for generating a template
type TemplateOptions struct {
	// Repository in format "owner/repo"
	Repository string
	// LogLevel for logging configuration
	LogLevel string
}

// GenerateTemplate generates the default configuration template for soba
func GenerateTemplate() string {
	return GenerateTemplateWithOptions(nil)
}

// GenerateTemplateWithOptions generates a configuration template with custom options
func GenerateTemplateWithOptions(opts *TemplateOptions) string {
	manager := NewTemplateManager()
	result, err := manager.RenderTemplate(opts)
	if err != nil {
		// Fallback to basic template if rendering fails
		// This should not happen in practice as the template is embedded
		return generateFallbackTemplate()
	}
	return result
}

// generateFallbackTemplate returns a basic template in case of template rendering failure
func generateFallbackTemplate() string {
	return `# GitHub settings
github:
  repository: douhashi/soba-cli

workflow:
  interval: 20
  use_tmux: true

log:
  level: info
`
}

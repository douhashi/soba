package config

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplateManager handles configuration template rendering
type TemplateManager interface {
	RenderTemplate(opts *TemplateOptions) (string, error)
}

type templateManager struct {
	tmpl *template.Template
}

// NewTemplateManager creates a new TemplateManager instance
func NewTemplateManager() TemplateManager {
	tmpl := template.Must(template.New("config").Parse(configTemplateContent))
	return &templateManager{
		tmpl: tmpl,
	}
}

// RenderTemplate renders the configuration template with the given options
func (tm *templateManager) RenderTemplate(opts *TemplateOptions) (string, error) {
	// Set default values if options is nil or values are empty
	if opts == nil {
		opts = &TemplateOptions{}
	}
	// Repository should be set by the caller - no default value
	if opts.LogLevel == "" {
		opts.LogLevel = "info"
	}

	var buf bytes.Buffer
	if err := tm.tmpl.Execute(&buf, opts); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

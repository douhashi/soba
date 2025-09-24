package slack

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/douhashi/soba/pkg/logging"
)

type TemplateManager interface {
	RenderTemplate(templateName string, data interface{}) ([]byte, error)
	LoadTemplates() error
}

type templateManager struct {
	templates  map[string]*template.Template
	mutex      sync.RWMutex
	logger     logging.Logger
	filesystem embed.FS
	useEmbed   bool
}

type BlockMessage struct {
	Blocks []interface{} `json:"blocks"`
}

func NewTemplateManager(logger logging.Logger) TemplateManager {
	return &templateManager{
		templates: make(map[string]*template.Template),
		logger:    logger,
		useEmbed:  false,
	}
}

// NewTemplateManagerWithFS creates a new template manager with embedded filesystem
func NewTemplateManagerWithFS(logger logging.Logger, fs embed.FS) TemplateManager {
	return &templateManager{
		templates:  make(map[string]*template.Template),
		logger:     logger,
		filesystem: fs,
		useEmbed:   true,
	}
}

func (tm *templateManager) LoadTemplates() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	templateDir := "templates/slack"

	templateFiles := map[string]string{
		"notify":      "notify.json",
		"phase_start": "phase_start.json",
		"pr_merged":   "pr_merged.json",
		"error":       "error.json",
	}

	loadedCount := 0

	for name, filename := range templateFiles {
		filePath := filepath.Join(templateDir, filename)
		var content []byte
		var err error

		if tm.useEmbed {
			// Read from embedded filesystem
			content, err = tm.filesystem.ReadFile(filePath)
			if err != nil {
				tm.logger.Debug(context.Background(), "Template file not found in embed, skipping",
					logging.Field{Key: "template", Value: name},
					logging.Field{Key: "file", Value: filePath},
				)
				continue
			}
		} else {
			// Check if file exists on filesystem
			if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
				tm.logger.Debug(context.Background(), "Template file not found, skipping",
					logging.Field{Key: "template", Value: name},
					logging.Field{Key: "file", Value: filePath},
				)
				continue
			}

			content, err = os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read template file %s: %w", filePath, err)
			}
		}

		tmpl, err := template.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		tm.templates[name] = tmpl
		loadedCount++
		tm.logger.Debug(context.Background(), "Loaded Slack template",
			logging.Field{Key: "template", Value: name},
			logging.Field{Key: "file", Value: filePath},
		)
	}

	if loadedCount == 0 {
		return fmt.Errorf("no templates were loaded from %s", templateDir)
	}

	return nil
}

func (tm *templateManager) RenderTemplate(templateName string, data interface{}) ([]byte, error) {
	tm.mutex.RLock()
	tmpl, exists := tm.templates[templateName]
	tm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	var blockMsg BlockMessage
	if err := json.Unmarshal(buf.Bytes(), &blockMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rendered template %s: %w", templateName, err)
	}

	result, err := json.Marshal(blockMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block message for template %s: %w", templateName, err)
	}

	return result, nil
}

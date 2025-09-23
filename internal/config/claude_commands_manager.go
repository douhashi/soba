package config

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ClaudeCommandsManager manages Claude command templates using embedded file system
type ClaudeCommandsManager struct {
	embedFS  embed.FS
	rootPath string
}

// NewClaudeCommandsManager creates a new ClaudeCommandsManager
func NewClaudeCommandsManager(embedFS embed.FS, rootPath string) *ClaudeCommandsManager {
	return &ClaudeCommandsManager{
		embedFS:  embedFS,
		rootPath: rootPath,
	}
}

// ListTemplates returns a sorted list of template file names
func (m *ClaudeCommandsManager) ListTemplates() ([]string, error) {
	var templates []string

	err := fs.WalkDir(m.embedFS, m.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			// Get relative path from root
			relPath, err := filepath.Rel(m.rootPath, path)
			if err != nil {
				return err
			}
			templates = append(templates, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	sort.Strings(templates)
	return templates, nil
}

// GetTemplate returns an io.Reader for the specified template file
func (m *ClaudeCommandsManager) GetTemplate(filename string) (io.Reader, error) {
	return m.readFile(filename)
}

// CopyTemplates copies all template files to the target directory
func (m *ClaudeCommandsManager) CopyTemplates(targetDir string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// List all templates
	templates, err := m.ListTemplates()
	if err != nil {
		return err
	}

	// Copy each template
	for _, tmpl := range templates {
		targetPath := filepath.Join(targetDir, tmpl)

		// Check if target file already exists
		if _, err := os.Stat(targetPath); err == nil {
			// File exists, skip
			continue
		}

		// Read source file
		reader, err := m.readFile(tmpl)
		if err != nil {
			// Skip files that can't be read
			continue
		}

		// Create target file
		targetFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}
		defer targetFile.Close()

		// Copy content
		if _, err := io.Copy(targetFile, reader); err != nil {
			return fmt.Errorf("failed to copy content to %s: %w", targetPath, err)
		}

		// Set permissions
		if err := targetFile.Chmod(0644); err != nil {
			return fmt.Errorf("failed to set permissions for %s: %w", targetPath, err)
		}
	}

	return nil
}

// ValidateTemplates checks if the embedded templates are accessible
func (m *ClaudeCommandsManager) ValidateTemplates() error {
	templates, err := m.ListTemplates()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(templates) == 0 {
		return fmt.Errorf("no templates found in %s", m.rootPath)
	}

	return nil
}

// readFile reads a file from the embedded file system
func (m *ClaudeCommandsManager) readFile(filename string) (io.Reader, error) {
	fullPath := filepath.Join(m.rootPath, filename)
	file, err := m.embedFS.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	return file, nil
}

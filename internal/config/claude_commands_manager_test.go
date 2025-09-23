package config

import (
	"embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/claude/commands/soba/*
var testClaudeCommandsFS embed.FS

func TestClaudeCommandsManager_ListTemplates(t *testing.T) {
	manager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "testdata/claude/commands/soba",
	}

	templates, err := manager.ListTemplates()
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	expected := []string{"test1.md", "test2.md"}
	if len(templates) != len(expected) {
		t.Errorf("Expected %d templates, got %d", len(expected), len(templates))
	}

	for i, tmpl := range expected {
		if i >= len(templates) || templates[i] != tmpl {
			t.Errorf("Expected template[%d] to be %s, got %s", i, tmpl, templates[i])
		}
	}
}

func TestClaudeCommandsManager_GetTemplate(t *testing.T) {
	manager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "testdata/claude/commands/soba",
	}

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "existing file",
			filename: "test1.md",
			wantErr:  false,
		},
		{
			name:     "non-existing file",
			filename: "nonexistent.md",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := manager.GetTemplate(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && content == nil {
				t.Error("Expected content reader, got nil")
			}
		})
	}
}

func TestClaudeCommandsManager_CopyTemplates(t *testing.T) {
	manager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "testdata/claude/commands/soba",
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands", "soba")

	err := manager.CopyTemplates(targetDir)
	if err != nil {
		t.Fatalf("Failed to copy templates: %v", err)
	}

	// Check if files were copied
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("Failed to read target directory: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 files, got %d", len(entries))
	}

	// Test skipping existing files
	err = manager.CopyTemplates(targetDir)
	if err != nil {
		t.Errorf("Should not fail when files already exist: %v", err)
	}
}

func TestClaudeCommandsManager_CopyTemplatesWithSkip(t *testing.T) {
	manager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "testdata/claude/commands/soba",
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands", "soba")

	// Create target directory and one existing file
	os.MkdirAll(targetDir, 0755)
	existingFile := filepath.Join(targetDir, "test1.md")
	os.WriteFile(existingFile, []byte("existing content"), 0644)

	err := manager.CopyTemplates(targetDir)
	if err != nil {
		t.Fatalf("Failed to copy templates: %v", err)
	}

	// Check that existing file was not overwritten
	content, _ := os.ReadFile(existingFile)
	if string(content) != "existing content" {
		t.Error("Existing file was overwritten")
	}

	// Check that new file was created
	newFile := filepath.Join(targetDir, "test2.md")
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("New file was not created")
	}
}

func TestClaudeCommandsManager_ValidateTemplates(t *testing.T) {
	manager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "testdata/claude/commands/soba",
	}

	err := manager.ValidateTemplates()
	if err != nil {
		t.Errorf("ValidateTemplates() should not fail with valid templates: %v", err)
	}

	// Test with invalid embed path
	invalidManager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "invalid/path",
	}

	err = invalidManager.ValidateTemplates()
	if err == nil {
		t.Error("ValidateTemplates() should fail with invalid path")
	}
}

func TestClaudeCommandsManager_ReadFile(t *testing.T) {
	manager := &ClaudeCommandsManager{
		embedFS:  testClaudeCommandsFS,
		rootPath: "testdata/claude/commands/soba",
	}

	// Test reading existing file
	reader, err := manager.readFile("test1.md")
	if err != nil {
		t.Errorf("Failed to read existing file: %v", err)
	}
	if reader == nil {
		t.Error("Expected reader, got nil")
	}

	// Test reading non-existing file
	_, err = manager.readFile("nonexistent.md")
	if err == nil {
		t.Error("Expected error for non-existing file")
	}
}

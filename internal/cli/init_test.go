package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
)

func TestInitCommand(t *testing.T) {
	t.Run("should create config file in new directory", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()

		// Assert
		assert.NoError(t, err)

		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.FileExists(t, configPath)

		// Verify file content is not empty
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, string(content), "github:")
		assert.Contains(t, string(content), "workflow:")
	})

	t.Run("should not overwrite existing config file", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Create existing config file
		sobaDir := filepath.Join(tempDir, ".soba")
		require.NoError(t, os.MkdirAll(sobaDir, 0755))

		existingContent := []byte("existing: content\n")
		configPath := filepath.Join(sobaDir, "config.yml")
		require.NoError(t, os.WriteFile(configPath, existingContent, 0644))

		// Execute
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()

		// Assert - should return error and not overwrite
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")

		// Verify original content is preserved
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, existingContent, content)
	})

	t.Run("should handle permission errors gracefully", func(t *testing.T) {
		// Skip if running as root
		if os.Geteuid() == 0 {
			t.Skip("Test cannot run as root")
		}

		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Create directory with no write permission
		sobaDir := filepath.Join(tempDir, ".soba")
		require.NoError(t, os.MkdirAll(sobaDir, 0555))
		defer os.Chmod(sobaDir, 0755) // Restore permission for cleanup

		// Execute
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission")
	})

	t.Run("generated config should be loadable", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute init command
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		require.NoError(t, err)

		// Try to load the created config
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		loadedConfig, err := config.Load(configPath)

		// Assert
		assert.NoError(t, err, "Should be able to load generated config")
		assert.NotNil(t, loadedConfig)

		// Verify some basic fields
		assert.Equal(t, "gh", loadedConfig.GitHub.AuthMethod)
		assert.Equal(t, "douhashi/soba-cli", loadedConfig.GitHub.Repository)
		assert.Equal(t, 20, loadedConfig.Workflow.Interval)
		assert.True(t, loadedConfig.Workflow.UseTmux)
		assert.Equal(t, ".git/soba/worktrees", loadedConfig.Git.WorktreeBasePath)
		assert.True(t, loadedConfig.Git.SetupWorkspace)
	})
}

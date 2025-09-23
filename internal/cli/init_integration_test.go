package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
)

func TestInitWithGitRepository(t *testing.T) {
	t.Run("should detect repository from git remote", func(t *testing.T) {
		// Setup: Create a temporary directory with git repository
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		cmd := exec.Command("git", "init")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Configure git user for CI environment
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.email: %s", string(output))

		cmd = exec.Command("git", "config", "user.name", "Test User")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.name: %s", string(output))

		// Add remote
		cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to add remote: %s", string(output))

		// Execute init
		err = runInitWithClient(context.Background(), []string{}, nil)
		require.NoError(t, err)

		// Verify config file was created
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.FileExists(t, configPath)

		// Load and verify the config
		cfg, err := config.Load(configPath)
		require.NoError(t, err)

		// Verify repository was detected
		assert.Equal(t, "test-owner/test-repo", cfg.GitHub.Repository)
	})

	t.Run("should copy Claude command templates when templates directory exists", func(t *testing.T) {
		// Setup: Create a temporary directory with git repository
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		cmd := exec.Command("git", "init")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Configure git user for CI environment
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.email: %s", string(output))

		cmd = exec.Command("git", "config", "user.name", "Test User")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.name: %s", string(output))

		// Add remote
		cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to add remote: %s", string(output))

		// Create template files
		templateDir := filepath.Join(tempDir, "templates", "claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(templateDir, 0755))

		templateFiles := map[string]string{
			"plan.md":      "# Plan template content",
			"implement.md": "# Implement template content",
			"review.md":    "# Review template content",
			"revise.md":    "# Revise template content",
		}

		for filename, content := range templateFiles {
			filePath := filepath.Join(templateDir, filename)
			require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
		}

		// Execute init
		err = runInitWithClient(context.Background(), []string{}, nil)
		require.NoError(t, err)

		// Verify config file was created
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.FileExists(t, configPath)

		// Verify Claude command templates were copied
		claudeDir := filepath.Join(tempDir, ".claude", "commands", "soba")
		for filename, expectedContent := range templateFiles {
			targetPath := filepath.Join(claudeDir, filename)
			assert.FileExists(t, targetPath)

			content, err := os.ReadFile(targetPath)
			require.NoError(t, err)
			assert.Equal(t, expectedContent, string(content))
		}
	})

	t.Run("should not overwrite existing Claude command templates", func(t *testing.T) {
		// Setup: Create a temporary directory with git repository
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		cmd := exec.Command("git", "init")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Configure git user for CI environment
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.email: %s", string(output))

		cmd = exec.Command("git", "config", "user.name", "Test User")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.name: %s", string(output))

		// Add remote
		cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to add remote: %s", string(output))

		// Create template files
		templateDir := filepath.Join(tempDir, "templates", "claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(templateDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "plan.md"), []byte("# New template content"), 0644))

		// Create existing Claude command template
		claudeDir := filepath.Join(tempDir, ".claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(claudeDir, 0755))
		existingContent := []byte("# Existing template content")
		existingFile := filepath.Join(claudeDir, "plan.md")
		require.NoError(t, os.WriteFile(existingFile, existingContent, 0644))

		// Execute init
		err = runInitWithClient(context.Background(), []string{}, nil)
		require.NoError(t, err)

		// Verify existing file was not overwritten
		content, err := os.ReadFile(existingFile)
		require.NoError(t, err)
		assert.Equal(t, existingContent, content)
	})

	t.Run("should detect repository from SSH git remote", func(t *testing.T) {
		// Setup: Create a temporary directory with git repository
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		cmd := exec.Command("git", "init")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Configure git user for CI environment
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.email: %s", string(output))

		cmd = exec.Command("git", "config", "user.name", "Test User")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.name: %s", string(output))

		// Add SSH remote
		cmd = exec.Command("git", "remote", "add", "origin", "git@github.com:ssh-owner/ssh-repo.git")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to add remote: %s", string(output))

		// Execute init
		err = runInitWithClient(context.Background(), []string{}, nil)
		require.NoError(t, err)

		// Verify config file was created
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.FileExists(t, configPath)

		// Load and verify the config
		cfg, err := config.Load(configPath)
		require.NoError(t, err)

		// Verify repository was detected
		assert.Equal(t, "ssh-owner/ssh-repo", cfg.GitHub.Repository)
	})

	t.Run("should fail if no remote configured", func(t *testing.T) {
		// Setup: Create a temporary directory with git repository but no remote
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository without remote
		cmd := exec.Command("git", "init")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Configure git user for CI environment
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.email: %s", string(output))

		cmd = exec.Command("git", "config", "user.name", "Test User")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Failed to configure git user.name: %s", string(output))

		// Execute init - should fail without remote
		err = runInitWithClient(context.Background(), []string{}, nil)

		// Assert that it fails with proper error message
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git remote")

		// Verify config file was not created
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.NoFileExists(t, configPath)
	})

	t.Run("should fail if not a git repository", func(t *testing.T) {
		// Setup: Create a temporary directory without git repository
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute init
		err := runInitWithClient(context.Background(), []string{}, nil)

		// Should fail with validation error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")

		// Verify config file was NOT created
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.NoFileExists(t, configPath)
	})
}

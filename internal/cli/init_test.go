package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

		// Execute
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err = cmd.Execute()

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
		assert.Contains(t, string(content), "test-owner/test-repo")
	})

	t.Run("should not overwrite existing config file", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

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

		err = cmd.Execute()

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

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

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

		err = cmd.Execute()

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

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

		// Execute init command
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err = cmd.Execute()
		require.NoError(t, err)

		// Try to load the created config
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		loadedConfig, err := config.Load(configPath)

		// Assert
		assert.NoError(t, err, "Should be able to load generated config")
		assert.NotNil(t, loadedConfig)

		// Verify some basic fields
		assert.Equal(t, "gh", loadedConfig.GitHub.AuthMethod)
		assert.Equal(t, "test-owner/test-repo", loadedConfig.GitHub.Repository)
		assert.Equal(t, 20, loadedConfig.Workflow.Interval)
		assert.True(t, loadedConfig.Workflow.UseTmux)
		assert.Equal(t, ".git/soba/worktrees", loadedConfig.Git.WorktreeBasePath)
	})

	t.Run("should create GitHub labels when config has repository info", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

		// Mock gh command
		mockCmd := &MockGhCommand{
			available:     true,
			authenticated: true,
			created:       11,
			skipped:       0,
		}

		// Execute with mock command
		err = runInitWithGhCommand(context.Background(), []string{}, mockCmd)

		// Assert
		assert.NoError(t, err)
		assert.True(t, mockCmd.createCalled)
		assert.Equal(t, "test-owner", mockCmd.lastOwner)
		assert.Equal(t, "test-repo", mockCmd.lastRepo)
	})

	t.Run("should require git remote to be configured", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository without remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Execute init command
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err = cmd.Execute()

		// Assert - should fail without git remote
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git remote")
	})

	t.Run("should handle gh command not available", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

		// Mock gh command that is not available
		mockCmd := &MockGhCommand{
			available: false,
		}

		// Execute with mock command
		err = runInitWithGhCommand(context.Background(), []string{}, mockCmd)

		// Assert - should not fail completely, but log the error
		assert.NoError(t, err, "Init should not fail due to gh not being available")
	})

	t.Run("should handle gh command not authenticated", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

		// Mock gh command that is not authenticated
		mockCmd := &MockGhCommand{
			available:     true,
			authenticated: false,
		}

		// Execute with mock command
		err = runInitWithGhCommand(context.Background(), []string{}, mockCmd)

		// Assert - should not fail completely, but log the error
		assert.NoError(t, err, "Init should not fail due to gh not being authenticated")
	})

	t.Run("should fail when no git remote is configured", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository without remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Execute
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err = cmd.Execute()

		// Assert - should fail with clear error message
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git remote")
	})

	t.Run("should use detected repository from git remote", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository with remote
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Add remote origin
		gitCmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-owner/test-repo.git")
		output, err = gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to add git remote: %s", string(output))

		// Execute
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err = cmd.Execute()
		require.NoError(t, err)

		// Verify config file has correct repository
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test-owner/test-repo")
		assert.NotContains(t, string(content), "douhashi/soba-cli")
	})
}

// Mock gh command for testing
type MockGhCommand struct {
	available     bool
	authenticated bool
	authError     error
	created       int
	skipped       int
	createError   error
	createCalled  bool
	lastOwner     string
	lastRepo      string
}

func (m *MockGhCommand) IsAvailable() bool {
	return m.available
}

func (m *MockGhCommand) IsAuthenticated(ctx context.Context) (bool, error) {
	if m.authError != nil {
		return false, m.authError
	}
	return m.authenticated, nil
}

func (m *MockGhCommand) CreateSobaLabels(ctx context.Context, owner, repo string) (created int, skipped int, err error) {
	m.createCalled = true
	m.lastOwner = owner
	m.lastRepo = repo

	if m.createError != nil {
		return 0, 0, m.createError
	}
	return m.created, m.skipped, nil
}

func TestCopyClaudeCommandTemplates(t *testing.T) {
	t.Run("should copy embedded template files to target directory", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()

		// Set current directory to temp dir for relative path resolution
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute
		err := copyClaudeCommandTemplates()

		// Assert
		assert.NoError(t, err)

		// Verify files are copied to target location
		targetDir := filepath.Join(tempDir, ".claude", "commands", "soba")
		expectedFiles := []string{"plan.md", "implement.md", "review.md", "revise.md"}

		for _, filename := range expectedFiles {
			targetPath := filepath.Join(targetDir, filename)
			assert.FileExists(t, targetPath)

			// Verify file has content
			content, err := os.ReadFile(targetPath)
			require.NoError(t, err)
			assert.NotEmpty(t, content)
		}
	})

	t.Run("should not overwrite existing files", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()

		// Create target directory with existing file
		targetDir := filepath.Join(tempDir, ".claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(targetDir, 0755))

		existingContent := []byte("# Existing plan content")
		existingFile := filepath.Join(targetDir, "plan.md")
		require.NoError(t, os.WriteFile(existingFile, existingContent, 0644))

		// Set current directory to temp dir for relative path resolution
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute
		err := copyClaudeCommandTemplates()

		// Assert
		assert.NoError(t, err)

		// Verify existing file is not overwritten
		content, err := os.ReadFile(existingFile)
		require.NoError(t, err)
		assert.Equal(t, existingContent, content)

		// Verify other files are created
		otherFiles := []string{"implement.md", "review.md", "revise.md"}
		for _, filename := range otherFiles {
			targetPath := filepath.Join(targetDir, filename)
			assert.FileExists(t, targetPath)
		}
	})

	t.Run("should always create target directory with embedded files", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()

		// Set current directory to temp dir
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute
		err := copyClaudeCommandTemplates()

		// Assert
		assert.NoError(t, err)

		// Verify target directory is created with embedded files
		targetDir := filepath.Join(tempDir, ".claude", "commands", "soba")
		assert.DirExists(t, targetDir)

		// Verify all embedded files are created
		expectedFiles := []string{"plan.md", "implement.md", "review.md", "revise.md"}
		for _, filename := range expectedFiles {
			targetPath := filepath.Join(targetDir, filename)
			assert.FileExists(t, targetPath)
		}
	})

	t.Run("should handle file copy errors gracefully", func(t *testing.T) {
		// Skip if running as root
		if os.Geteuid() == 0 {
			t.Skip("Test cannot run as root")
		}

		// Setup
		tempDir := t.TempDir()

		// Create template source directory and files
		templateDir := filepath.Join(tempDir, "templates", "claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(templateDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "plan.md"), []byte("# Plan template"), 0644))

		// Create .claude directory with no write permission
		claudeDir := filepath.Join(tempDir, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0555))
		defer os.Chmod(claudeDir, 0755) // Restore permission for cleanup

		// Set current directory to temp dir for relative path resolution
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute
		err := copyClaudeCommandTemplates()

		// Assert - should return error but function should handle it gracefully
		assert.Error(t, err)
	})
}

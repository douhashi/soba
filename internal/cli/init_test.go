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
	"github.com/douhashi/soba/internal/infra/github"
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

		// Mock GitHub client
		mockClient := &MockGitHubClient{
			CreateLabelCalls: []CreateLabelCall{},
			ListLabelsCalls:  []ListLabelsCall{},
		}

		// Execute with mock client (this will create config first)
		err = runInitWithClient(context.Background(), []string{}, mockClient)

		// Assert
		assert.NoError(t, err)

		// Should have attempted to create labels for detected repository
		assert.GreaterOrEqual(t, len(mockClient.ListLabelsCalls), 1, "Should call ListLabels at least once")

		if len(mockClient.ListLabelsCalls) > 0 {
			// Verify first call is to list existing labels
			listCall := mockClient.ListLabelsCalls[0]
			assert.Equal(t, "test-owner", listCall.Owner)
			assert.Equal(t, "test-repo", listCall.Repo)
		}
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

	t.Run("should handle GitHub API errors gracefully", func(t *testing.T) {
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

		// Mock GitHub client that returns errors
		mockClient := &MockGitHubClient{
			ListLabelsError: assert.AnError,
		}

		// Execute with mock client
		err = runInitWithClient(context.Background(), []string{}, mockClient)

		// Assert - should not fail completely, but log the error
		assert.NoError(t, err, "Init should not fail due to GitHub API errors")
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

// Mock GitHub client for testing
type MockGitHubClient struct {
	CreateLabelCalls []CreateLabelCall
	ListLabelsCalls  []ListLabelsCall
	CreateLabelError error
	ListLabelsError  error
	ExistingLabels   []github.Label
}

type CreateLabelCall struct {
	Owner   string
	Repo    string
	Request github.CreateLabelRequest
}

type ListLabelsCall struct {
	Owner string
	Repo  string
}

func (m *MockGitHubClient) CreateLabel(ctx context.Context, owner, repo string, request github.CreateLabelRequest) (*github.Label, error) {
	m.CreateLabelCalls = append(m.CreateLabelCalls, CreateLabelCall{
		Owner:   owner,
		Repo:    repo,
		Request: request,
	})

	if m.CreateLabelError != nil {
		return nil, m.CreateLabelError
	}

	return &github.Label{
		ID:          int64(len(m.CreateLabelCalls)),
		Name:        request.Name,
		Color:       request.Color,
		Description: request.Description,
	}, nil
}

func (m *MockGitHubClient) ListLabels(ctx context.Context, owner, repo string) ([]github.Label, error) {
	m.ListLabelsCalls = append(m.ListLabelsCalls, ListLabelsCall{
		Owner: owner,
		Repo:  repo,
	})

	if m.ListLabelsError != nil {
		return nil, m.ListLabelsError
	}

	return m.ExistingLabels, nil
}

func TestCopyClaudeCommandTemplates(t *testing.T) {
	t.Run("should copy template files to target directory", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()

		// Create template source directory and files
		templateDir := filepath.Join(tempDir, "templates", "claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(templateDir, 0755))

		templateFiles := map[string]string{
			"plan.md":      "# Plan template",
			"implement.md": "# Implement template",
			"review.md":    "# Review template",
			"revise.md":    "# Revise template",
		}

		for filename, content := range templateFiles {
			filepath := filepath.Join(templateDir, filename)
			require.NoError(t, os.WriteFile(filepath, []byte(content), 0644))
		}

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
		for filename, expectedContent := range templateFiles {
			targetPath := filepath.Join(targetDir, filename)
			assert.FileExists(t, targetPath)

			content, err := os.ReadFile(targetPath)
			require.NoError(t, err)
			assert.Equal(t, expectedContent, string(content))
		}
	})

	t.Run("should not overwrite existing files", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()

		// Create template source directory and files
		templateDir := filepath.Join(tempDir, "templates", "claude", "commands", "soba")
		require.NoError(t, os.MkdirAll(templateDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "plan.md"), []byte("# New plan template"), 0644))

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
	})

	t.Run("should handle missing template directory gracefully", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()

		// Set current directory to temp dir (no templates directory exists)
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Execute
		err := copyClaudeCommandTemplates()

		// Assert - should not fail, just skip copying
		assert.NoError(t, err)

		// Verify no target directory is created
		targetDir := filepath.Join(tempDir, ".claude", "commands", "soba")
		_, err = os.Stat(targetDir)
		assert.True(t, os.IsNotExist(err))
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

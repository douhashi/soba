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

		// Initialize git repository
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

		// Initialize git repository
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

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

		// Initialize git repository
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

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

		// Initialize git repository
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
	})

	t.Run("should create GitHub labels when config has repository info", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Mock GitHub client
		mockClient := &MockGitHubClient{
			CreateLabelCalls: []CreateLabelCall{},
			ListLabelsCalls:  []ListLabelsCall{},
		}

		// Execute with mock client (this will create config first)
		err = runInitWithClient(context.Background(), []string{}, mockClient)

		// Assert
		assert.NoError(t, err)

		// Should have attempted to create labels for default repository
		assert.GreaterOrEqual(t, len(mockClient.ListLabelsCalls), 1, "Should call ListLabels at least once")

		if len(mockClient.ListLabelsCalls) > 0 {
			// Verify first call is to list existing labels
			listCall := mockClient.ListLabelsCalls[0]
			assert.Equal(t, "douhashi", listCall.Owner)
			assert.Equal(t, "soba-cli", listCall.Repo)
		}
	})

	t.Run("should skip label creation if no repository configured", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Execute with no config file (should create default config)
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err = cmd.Execute()

		// Assert - should succeed even without GitHub configuration
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Successfully created config file")
	})

	t.Run("should handle GitHub API errors gracefully", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		require.NoError(t, os.Chdir(tempDir))

		// Initialize git repository
		gitCmd := exec.Command("git", "init")
		output, err := gitCmd.CombinedOutput()
		require.NoError(t, err, "Failed to init git repository: %s", string(output))

		// Mock GitHub client that returns errors
		mockClient := &MockGitHubClient{
			ListLabelsError: assert.AnError,
		}

		// Execute with mock client (this will create default config)
		err = runInitWithClient(context.Background(), []string{}, mockClient)

		// Assert - should not fail completely, but log the error
		assert.NoError(t, err, "Init should not fail due to GitHub API errors")
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

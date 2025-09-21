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

	t.Run("should use default repository if no remote configured", func(t *testing.T) {
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

		// Execute init
		err = runInitWithClient(context.Background(), []string{}, nil)
		require.NoError(t, err)

		// Verify config file was created
		configPath := filepath.Join(tempDir, ".soba", "config.yml")
		assert.FileExists(t, configPath)

		// Load and verify the config
		cfg, err := config.Load(configPath)
		require.NoError(t, err)

		// Verify default repository is used
		assert.Equal(t, "douhashi/soba-cli", cfg.GitHub.Repository)
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

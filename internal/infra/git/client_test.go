package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		wantErr bool
	}{
		{
			name:    "Valid work directory",
			workDir: ".",
			wantErr: false,
		},
		{
			name:    "Empty work directory",
			workDir: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.workDir)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_CreateWorktree(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	// Setup test repository
	setupTestRepo(t, repoDir)

	client, err := NewClient(repoDir)
	require.NoError(t, err)

	tests := []struct {
		name       string
		worktreePath string
		branchName   string
		baseBranch   string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "Create worktree successfully",
			worktreePath: filepath.Join(tempDir, "worktree1"),
			branchName:   "soba/1",
			baseBranch:   "main",
			wantErr:      false,
		},
		{
			name:         "Empty worktree path",
			worktreePath: "",
			branchName:   "soba/2",
			baseBranch:   "main",
			wantErr:      true,
			errMessage:   "worktree path is required",
		},
		{
			name:         "Empty branch name",
			worktreePath: filepath.Join(tempDir, "worktree2"),
			branchName:   "",
			baseBranch:   "main",
			wantErr:      true,
			errMessage:   "branch name is required",
		},
		{
			name:         "Invalid base branch",
			worktreePath: filepath.Join(tempDir, "worktree3"),
			branchName:   "soba/3",
			baseBranch:   "nonexistent",
			wantErr:      true,
			errMessage:   "base branch not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CreateWorktree(tt.worktreePath, tt.branchName, tt.baseBranch)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				// Verify worktree was created
				assert.DirExists(t, tt.worktreePath)
				// Cleanup
				_ = client.RemoveWorktree(tt.worktreePath)
			}
		})
	}
}

func TestClient_RemoveWorktree(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	setupTestRepo(t, repoDir)

	client, err := NewClient(repoDir)
	require.NoError(t, err)

	// Create a worktree first
	worktreePath := filepath.Join(tempDir, "worktree-to-remove")
	err = client.CreateWorktree(worktreePath, "soba/99", "main")
	require.NoError(t, err)

	tests := []struct {
		name         string
		worktreePath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "Remove existing worktree",
			worktreePath: worktreePath,
			wantErr:      false,
		},
		{
			name:         "Remove non-existing worktree",
			worktreePath: filepath.Join(tempDir, "nonexistent"),
			wantErr:      true,
			errMessage:   "worktree not found",
		},
		{
			name:         "Empty worktree path",
			worktreePath: "",
			wantErr:      true,
			errMessage:   "worktree path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.RemoveWorktree(tt.worktreePath)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				// Verify worktree was removed
				assert.NoDirExists(t, tt.worktreePath)
			}
		})
	}
}

func TestClient_UpdateBaseBranch(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	setupTestRepo(t, repoDir)

	client, err := NewClient(repoDir)
	require.NoError(t, err)

	tests := []struct {
		name       string
		branch     string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Update main branch",
			branch:  "main",
			wantErr: false,
		},
		{
			name:       "Update non-existent branch",
			branch:     "nonexistent",
			wantErr:    true,
			errMessage: "branch not found",
		},
		{
			name:       "Empty branch name",
			branch:     "",
			wantErr:    true,
			errMessage: "branch name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.UpdateBaseBranch(tt.branch)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_WorktreeExists(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	setupTestRepo(t, repoDir)

	client, err := NewClient(repoDir)
	require.NoError(t, err)

	// Create a worktree
	worktreePath := filepath.Join(tempDir, "existing-worktree")
	err = client.CreateWorktree(worktreePath, "soba/100", "main")
	require.NoError(t, err)

	tests := []struct {
		name         string
		worktreePath string
		want         bool
	}{
		{
			name:         "Existing worktree",
			worktreePath: worktreePath,
			want:         true,
		},
		{
			name:         "Non-existing worktree",
			worktreePath: filepath.Join(tempDir, "nonexistent"),
			want:         false,
		},
		{
			name:         "Empty path",
			worktreePath: "",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := client.WorktreeExists(tt.worktreePath)
			assert.Equal(t, tt.want, exists)
		})
	}

	// Cleanup
	_ = client.RemoveWorktree(worktreePath)
}

// setupTestRepo creates a minimal git repository for testing
func setupTestRepo(t *testing.T, repoDir string) {
	t.Helper()

	// Create directory
	err := os.MkdirAll(repoDir, 0755)
	require.NoError(t, err)

	// Initialize git repo
	cmd := execCommand("git", "init", repoDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to init repo: %s", string(output))

	// Configure git
	cmd = execCommand("git", "-C", repoDir, "config", "user.email", "test@example.com")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = execCommand("git", "-C", repoDir, "config", "user.name", "Test User")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository\n"), 0644)
	require.NoError(t, err)

	cmd = execCommand("git", "-C", repoDir, "add", ".")
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = execCommand("git", "-C", repoDir, "commit", "-m", "Initial commit")
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to commit: %s", string(output))
}

// execCommand is a wrapper for testing
var execCommand = func(name string, arg ...string) commandRunner {
	return &osExecCommand{name: name, args: arg}
}

type commandRunner interface {
	CombinedOutput() ([]byte, error)
}

type osExecCommand struct {
	name string
	args []string
}

func (c *osExecCommand) CombinedOutput() ([]byte, error) {
	cmd := exec.Command(c.name, c.args...)
	return cmd.CombinedOutput()
}
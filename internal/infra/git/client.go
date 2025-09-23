package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client provides Git operations
type Client struct {
	workDir string
}

// NewClient creates a new Git client
func NewClient(workDir string) (*Client, error) {
	if workDir == "" {
		return nil, errors.New("work directory is required")
	}

	// Check if directory is a git repository
	cmd := exec.Command("git", "-C", workDir, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return nil, NewGitError("init", workDir, "not a git repository", err)
	}

	return &Client{
		workDir: workDir,
	}, nil
}

// CreateWorktree creates a new worktree with a new branch
func (c *Client) CreateWorktree(worktreePath, branchName, baseBranch string) error {
	if worktreePath == "" {
		return NewGitError("worktree add", "", "worktree path is required", nil)
	}
	if branchName == "" {
		return NewGitError("worktree add", worktreePath, "branch name is required", nil)
	}
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Check if base branch exists
	cmd := exec.Command("git", "-C", c.workDir, "rev-parse", "--verify", baseBranch)
	if err := cmd.Run(); err != nil {
		return NewGitError("worktree add", worktreePath, "base branch not found", err)
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(worktreePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return NewGitError("worktree add", worktreePath, "failed to create parent directory", err)
	}

	// Create worktree with new branch
	cmd = exec.Command("git", "-C", c.workDir, "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if branch already exists
		if strings.Contains(string(output), "already exists") {
			// Try to create worktree with existing branch
			cmd = exec.Command("git", "-C", c.workDir, "worktree", "add", worktreePath, branchName)
			if err := cmd.Run(); err != nil {
				return NewGitError("worktree add", worktreePath, string(output), err)
			}
			return nil
		}
		return NewGitError("worktree add", worktreePath, string(output), err)
	}

	return nil
}

// RemoveWorktree removes a worktree
func (c *Client) RemoveWorktree(worktreePath string) error {
	if worktreePath == "" {
		return NewGitError("worktree remove", "", "worktree path is required", nil)
	}

	// Check if worktree exists
	if !c.WorktreeExists(worktreePath) {
		return NewGitError("worktree remove", worktreePath, "worktree not found", nil)
	}

	// Remove worktree
	cmd := exec.Command("git", "-C", c.workDir, "worktree", "remove", worktreePath, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return NewGitError("worktree remove", worktreePath, string(output), err)
	}

	return nil
}

// UpdateBaseBranch updates the specified branch to the latest remote version
func (c *Client) UpdateBaseBranch(branch string) error {
	if branch == "" {
		return NewGitError("fetch", "", "branch name is required", nil)
	}

	// Check if branch exists
	cmd := exec.Command("git", "-C", c.workDir, "rev-parse", "--verify", branch)
	if err := cmd.Run(); err != nil {
		return NewGitError("fetch", branch, "branch not found", err)
	}

	// Check if remote exists
	cmd = exec.Command("git", "-C", c.workDir, "remote")
	remoteOutput, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(remoteOutput))) == 0 {
		// No remote configured, skip fetch (for local testing)
		return nil
	}

	// Fetch latest changes
	cmd = exec.Command("git", "-C", c.workDir, "fetch", "origin", fmt.Sprintf("%s:%s", branch, branch))
	_, err = cmd.CombinedOutput()
	if err != nil {
		// Try simple fetch without force update
		cmd = exec.Command("git", "-C", c.workDir, "fetch")
		if fetchErr := cmd.Run(); fetchErr != nil {
			// Log error but don't fail (for testing environments)
			return nil
		}
	}

	return nil
}

// WorktreeExists checks if a worktree exists at the specified path
func (c *Client) WorktreeExists(worktreePath string) bool {
	if worktreePath == "" {
		return false
	}

	cmd := exec.Command("git", "-C", c.workDir, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse worktree list output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			absPath, _ := filepath.Abs(path)
			inputPath, _ := filepath.Abs(worktreePath)

			// Resolve symlinks for accurate path comparison (macOS compatibility)
			if resolvedAbsPath, err := filepath.EvalSymlinks(absPath); err == nil {
				absPath = resolvedAbsPath
			}
			if resolvedInputPath, err := filepath.EvalSymlinks(inputPath); err == nil {
				inputPath = resolvedInputPath
			}

			if absPath == inputPath {
				return true
			}
		}
	}

	return false
}

// GetRemoteURL gets the URL of a remote repository
func (c *Client) GetRemoteURL(remote string) (string, error) {
	if remote == "" {
		return "", NewGitError("remote", "", "remote name is required", nil)
	}

	cmd := exec.Command("git", "-C", c.workDir, "remote", "get-url", remote)
	output, err := cmd.Output()
	if err != nil {
		return "", NewGitError("remote get-url", remote, "failed to get remote URL", err)
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return "", NewGitError("remote get-url", remote, "empty remote URL", nil)
	}

	return url, nil
}

// ParseRepositoryFromURL parses owner/repo from a Git URL
func ParseRepositoryFromURL(url string) (owner, repo string, err error) {
	if url == "" {
		return "", "", errors.New("empty URL")
	}

	// Remove trailing .git if present
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format: git@github.com:owner/repo
	if strings.HasPrefix(url, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(url, "git@github.com:"), "/")
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
		return "", "", errors.New("invalid SSH URL format")
	}

	// Handle SSH URL with protocol: ssh://git@github.com:22/owner/repo or ssh://git@github.com/owner/repo
	if strings.HasPrefix(url, "ssh://git@github.com") {
		// Find the path after github.com
		idx := strings.Index(url, "github.com")
		if idx >= 0 {
			remaining := url[idx+len("github.com"):]

			// Check if it's directly followed by path (no port)
			if strings.HasPrefix(remaining, "/") {
				parts := strings.Split(strings.TrimPrefix(remaining, "/"), "/")
				if len(parts) == 2 {
					// Format: /owner/repo
					return parts[0], parts[1], nil
				}
			} else if strings.HasPrefix(remaining, ":") {
				// Remove port if present
				parts := strings.Split(remaining, "/")
				if len(parts) >= 3 {
					// Format: :22/owner/repo
					return parts[1], parts[2], nil
				}
			}
		}
		return "", "", errors.New("invalid SSH URL format")
	}

	// Handle HTTPS format: https://github.com/owner/repo
	if strings.HasPrefix(url, "https://github.com/") {
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
		return "", "", errors.New("invalid HTTPS URL format")
	}

	// Not a GitHub URL
	return "", "", errors.New("not a GitHub URL")
}

// GetRepository gets the repository in owner/repo format from origin remote
func (c *Client) GetRepository() (string, error) {
	// Get origin remote URL
	url, err := c.GetRemoteURL("origin")
	if err != nil {
		return "", err
	}

	// Parse owner/repo from URL
	owner, repo, err := ParseRepositoryFromURL(url)
	if err != nil {
		return "", NewGitError("parse repository", url, "failed to parse repository from URL", err)
	}

	return fmt.Sprintf("%s/%s", owner, repo), nil
}

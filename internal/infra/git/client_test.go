package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_GetRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tmpDir string)
		remote    string
		wantURL   string
		wantErr   bool
	}{
		{
			name: "HTTPS URL format",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
				runCommand(t, tmpDir, "git", "remote", "add", "origin", "https://github.com/owner/repo.git")
			},
			remote:  "origin",
			wantURL: "https://github.com/owner/repo.git",
			wantErr: false,
		},
		{
			name: "SSH URL format",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
				runCommand(t, tmpDir, "git", "remote", "add", "origin", "git@github.com:owner/repo.git")
			},
			remote:  "origin",
			wantURL: "git@github.com:owner/repo.git",
			wantErr: false,
		},
		{
			name: "Remote not found",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
			},
			remote:  "origin",
			wantURL: "",
			wantErr: true,
		},
		{
			name: "Non-existent remote name",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
				runCommand(t, tmpDir, "git", "remote", "add", "origin", "https://github.com/owner/repo.git")
			},
			remote:  "upstream",
			wantURL: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test environment
			tt.setupFunc(t, tmpDir)

			// Create client
			client, err := NewClient(tmpDir)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Test GetRemoteURL
			gotURL, err := client.GetRemoteURL(tt.remote)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotURL != tt.wantURL {
				t.Errorf("GetRemoteURL() = %v, want %v", gotURL, tt.wantURL)
			}
		})
	}
}

func TestClient_ParseRepositoryFromURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL with .git",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "HTTPS URL without .git",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL with .git",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL without .git",
			url:       "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL with custom port",
			url:       "ssh://git@github.com:22/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "Invalid URL format",
			url:       "not-a-valid-url",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "Non-GitHub URL",
			url:       "https://gitlab.com/owner/repo.git",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "Empty URL",
			url:       "",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, err := ParseRepositoryFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepositoryFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOwner != tt.wantOwner {
				t.Errorf("ParseRepositoryFromURL() owner = %v, want %v", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("ParseRepositoryFromURL() repo = %v, want %v", gotRepo, tt.wantRepo)
			}
		})
	}
}

func TestClient_GetRepository(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tmpDir string)
		wantRepo  string
		wantErr   bool
	}{
		{
			name: "Get repository from origin remote",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
				runCommand(t, tmpDir, "git", "remote", "add", "origin", "https://github.com/douhashi/soba.git")
			},
			wantRepo: "douhashi/soba",
			wantErr:  false,
		},
		{
			name: "Get repository from SSH URL",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
				runCommand(t, tmpDir, "git", "remote", "add", "origin", "git@github.com:douhashi/soba-cli.git")
			},
			wantRepo: "douhashi/soba-cli",
			wantErr:  false,
		},
		{
			name: "No remote configured",
			setupFunc: func(t *testing.T, tmpDir string) {
				runCommand(t, tmpDir, "git", "init")
			},
			wantRepo: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test environment
			tt.setupFunc(t, tmpDir)

			// Create client
			client, err := NewClient(tmpDir)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Test GetRepository
			gotRepo, err := client.GetRepository()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("GetRepository() = %v, want %v", gotRepo, tt.wantRepo)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "Valid git repository",
			setup: func(t *testing.T, dir string) {
				runCommand(t, dir, "git", "init")
			},
			wantErr: false,
		},
		{
			name:    "Not a git repository",
			setup:   func(t *testing.T, dir string) {},
			wantErr: true,
		},
		{
			name:    "Empty work directory",
			workDir: "",
			setup:   func(t *testing.T, dir string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workDir string
			if tt.workDir == "" && tt.name != "Empty work directory" {
				workDir = t.TempDir()
			} else {
				workDir = tt.workDir
			}

			if workDir != "" {
				tt.setup(t, workDir)
			}

			_, err := NewClient(workDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_CreateWorktree(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		branchName   string
		baseBranch   string
		setup        func(t *testing.T, dir string)
		wantErr      bool
	}{
		{
			name:         "Create worktree with new branch",
			worktreePath: "worktree1",
			branchName:   "feature/test1",
			baseBranch:   "main",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: false,
		},
		{
			name:         "Create worktree with default base branch",
			worktreePath: "worktree2",
			branchName:   "feature/test2",
			baseBranch:   "",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: false,
		},
		{
			name:         "Create worktree with non-existent base branch",
			worktreePath: "worktree3",
			branchName:   "feature/test3",
			baseBranch:   "non-existent",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: true,
		},
		{
			name:         "Empty worktree path",
			worktreePath: "",
			branchName:   "feature/test4",
			baseBranch:   "main",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: true,
		},
		{
			name:         "Empty branch name",
			worktreePath: "worktree5",
			branchName:   "",
			baseBranch:   "main",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test repository
			tt.setup(t, tmpDir)

			// Create client
			client, err := NewClient(tmpDir)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Prepare worktree path
			var fullWorktreePath string
			if tt.worktreePath != "" {
				fullWorktreePath = filepath.Join(tmpDir, tt.worktreePath)
			}

			// Test CreateWorktree
			err = client.CreateWorktree(fullWorktreePath, tt.branchName, tt.baseBranch)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWorktree() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If success, verify worktree was created
			if err == nil && fullWorktreePath != "" {
				if !client.WorktreeExists(fullWorktreePath) {
					t.Errorf("Worktree was not created at %s", fullWorktreePath)
				}
			}
		})
	}
}

func TestClient_RemoveWorktree(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		setup        func(t *testing.T, dir string, worktreePath string) bool
		wantErr      bool
	}{
		{
			name:         "Remove existing worktree",
			worktreePath: "worktree-to-remove",
			setup: func(t *testing.T, dir string, worktreePath string) bool {
				createTestRepository(t, dir)
				client, _ := NewClient(dir)
				fullPath := filepath.Join(dir, worktreePath)
				err := client.CreateWorktree(fullPath, "test-branch", "main")
				return err == nil
			},
			wantErr: false,
		},
		{
			name:         "Remove non-existent worktree",
			worktreePath: "non-existent",
			setup: func(t *testing.T, dir string, worktreePath string) bool {
				createTestRepository(t, dir)
				return true
			},
			wantErr: true,
		},
		{
			name:         "Empty worktree path",
			worktreePath: "",
			setup: func(t *testing.T, dir string, worktreePath string) bool {
				createTestRepository(t, dir)
				return true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test environment
			if !tt.setup(t, tmpDir, tt.worktreePath) {
				t.Skip("Setup failed, skipping test")
			}

			// Create client
			client, err := NewClient(tmpDir)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Prepare worktree path
			var fullWorktreePath string
			if tt.worktreePath != "" {
				fullWorktreePath = filepath.Join(tmpDir, tt.worktreePath)
			}

			// Test RemoveWorktree
			err = client.RemoveWorktree(fullWorktreePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveWorktree() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If success, verify worktree was removed
			if err == nil && fullWorktreePath != "" {
				if client.WorktreeExists(fullWorktreePath) {
					t.Errorf("Worktree still exists at %s", fullWorktreePath)
				}
			}
		})
	}
}

func TestClient_UpdateBaseBranch(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name:   "Update existing branch",
			branch: "main",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: false,
		},
		{
			name:   "Update non-existent branch",
			branch: "non-existent",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: true,
		},
		{
			name:   "Empty branch name",
			branch: "",
			setup: func(t *testing.T, dir string) {
				createTestRepository(t, dir)
			},
			wantErr: true,
		},
		{
			name:   "Repository without remote",
			branch: "main",
			setup: func(t *testing.T, dir string) {
				runCommand(t, dir, "git", "init")
				// Set Git configuration for CI environment
				runCommand(t, dir, "git", "config", "user.email", "test@example.com")
				runCommand(t, dir, "git", "config", "user.name", "Test User")
				runCommand(t, dir, "git", "checkout", "-b", "main")
				writeFile(t, filepath.Join(dir, "README.md"), "# Test")
				runCommand(t, dir, "git", "add", ".")
				runCommand(t, dir, "git", "commit", "-m", "Initial commit")
			},
			wantErr: false, // Should not fail if no remote
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test environment
			tt.setup(t, tmpDir)

			// Create client
			client, err := NewClient(tmpDir)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Test UpdateBaseBranch
			err = client.UpdateBaseBranch(tt.branch)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateBaseBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_WorktreeExists(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		setup        func(t *testing.T, dir string) string
		want         bool
	}{
		{
			name:         "Existing worktree",
			worktreePath: "existing-worktree",
			setup: func(t *testing.T, dir string) string {
				createTestRepository(t, dir)
				client, _ := NewClient(dir)
				fullPath := filepath.Join(dir, "existing-worktree")
				client.CreateWorktree(fullPath, "test-branch", "main")
				return fullPath
			},
			want: true,
		},
		{
			name:         "Non-existent worktree",
			worktreePath: "non-existent",
			setup: func(t *testing.T, dir string) string {
				createTestRepository(t, dir)
				return filepath.Join(dir, "non-existent")
			},
			want: false,
		},
		{
			name:         "Empty path",
			worktreePath: "",
			setup: func(t *testing.T, dir string) string {
				createTestRepository(t, dir)
				return ""
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup test environment and get the path to check
			pathToCheck := tt.setup(t, tmpDir)

			// Create client
			client, err := NewClient(tmpDir)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Test WorktreeExists
			got := client.WorktreeExists(pathToCheck)
			if got != tt.want {
				t.Errorf("WorktreeExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func createTestRepository(t *testing.T, dir string) {
	t.Helper()

	// Initialize git repository
	runCommand(t, dir, "git", "init")

	// Set Git configuration for CI environment
	runCommand(t, dir, "git", "config", "user.email", "test@example.com")
	runCommand(t, dir, "git", "config", "user.name", "Test User")

	runCommand(t, dir, "git", "checkout", "-b", "main")

	// Create initial commit
	writeFile(t, filepath.Join(dir, "README.md"), "# Test Repository")
	runCommand(t, dir, "git", "add", ".")
	runCommand(t, dir, "git", "commit", "-m", "Initial commit")
}

func runCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %s %s\n%s\n%v", name, strings.Join(args, " "), output, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

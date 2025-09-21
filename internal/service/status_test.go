package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
)

// StatusMockTmuxClient for testing
type StatusMockTmuxClient struct {
	mock.Mock
}

func (m *StatusMockTmuxClient) CreateSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) DeleteSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) SessionExists(sessionName string) bool {
	args := m.Called(sessionName)
	return args.Bool(0)
}

func (m *StatusMockTmuxClient) CreateWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) DeleteWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

func (m *StatusMockTmuxClient) CreatePane(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) DeletePane(sessionName, windowName string, paneIndex int) error {
	args := m.Called(sessionName, windowName, paneIndex)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) GetPaneCount(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *StatusMockTmuxClient) GetFirstPaneIndex(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *StatusMockTmuxClient) GetLastPaneIndex(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *StatusMockTmuxClient) ResizePanes(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *StatusMockTmuxClient) SendCommand(sessionName, windowName string, paneIndex int, command string) error {
	args := m.Called(sessionName, windowName, paneIndex, command)
	return args.Error(0)
}

// StatusMockGitHubClient for testing
type StatusMockGitHubClient struct {
	mock.Mock
}

func (m *StatusMockGitHubClient) ListOpenIssues(ctx context.Context, owner, repo string, options *github.ListIssuesOptions) ([]github.Issue, bool, error) {
	args := m.Called(ctx, owner, repo, options)
	return args.Get(0).([]github.Issue), args.Bool(1), args.Error(2)
}

func (m *StatusMockGitHubClient) AddLabelToIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *StatusMockGitHubClient) RemoveLabelFromIssue(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *StatusMockGitHubClient) ListPullRequests(ctx context.Context, owner, repo string, opts *github.ListPullRequestsOptions) ([]github.PullRequest, bool, error) {
	args := m.Called(ctx, owner, repo, opts)
	return args.Get(0).([]github.PullRequest), args.Bool(1), args.Error(2)
}

func (m *StatusMockGitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, bool, error) {
	args := m.Called(ctx, owner, repo, number)
	if pr := args.Get(0); pr != nil {
		return pr.(*github.PullRequest), args.Bool(1), args.Error(2)
	}
	return nil, args.Bool(1), args.Error(2)
}

func (m *StatusMockGitHubClient) MergePullRequest(ctx context.Context, owner, repo string, number int, req *github.MergeRequest) (*github.MergeResponse, error) {
	args := m.Called(ctx, owner, repo, number, req)
	if resp := args.Get(0); resp != nil {
		return resp.(*github.MergeResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestStatusService_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*StatusMockGitHubClient, *StatusMockTmuxClient)
		pidFileExists  bool
		expectedDaemon bool
		expectedIssues int
		expectedTmux   bool
	}{
		{
			name: "daemon running with issues and tmux session",
			setupMocks: func(gh *StatusMockGitHubClient, tm *StatusMockTmuxClient) {
				gh.On("ListOpenIssues", mock.Anything, "test-owner", "test-repo", mock.Anything).
					Return([]github.Issue{
						{Number: 1, Title: "Issue 1", Labels: []github.Label{{Name: "soba:ready"}}},
						{Number: 2, Title: "Issue 2", Labels: []github.Label{{Name: "soba:doing"}}},
					}, false, nil)
				tm.On("SessionExists", mock.Anything).Return(true)
			},
			pidFileExists:  true,
			expectedDaemon: true,
			expectedIssues: 2,
			expectedTmux:   true,
		},
		{
			name: "daemon not running",
			setupMocks: func(gh *StatusMockGitHubClient, tm *StatusMockTmuxClient) {
				gh.On("ListOpenIssues", mock.Anything, "test-owner", "test-repo", mock.Anything).
					Return([]github.Issue{}, false, nil)
				tm.On("SessionExists", mock.Anything).Return(false)
			},
			pidFileExists:  false,
			expectedDaemon: false,
			expectedIssues: 0,
			expectedTmux:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockGH := new(StatusMockGitHubClient)
			mockTmux := new(StatusMockTmuxClient)
			tt.setupMocks(mockGH, mockTmux)

			// Setup PID file if needed
			if tt.pidFileExists {
				err := os.MkdirAll(".soba", 0755)
				require.NoError(t, err)
				// Use current process PID for testing
				pid := os.Getpid()
				err = os.WriteFile(".soba/soba.pid", []byte(fmt.Sprintf("%d", pid)), 0644)
				require.NoError(t, err)
				defer os.RemoveAll(".soba")
			}

			// Create service
			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "test-owner/test-repo",
				},
			}
			svc := NewStatusService(cfg, mockGH, mockTmux)

			// Get status
			status, err := svc.GetStatus(context.Background())
			require.NoError(t, err)
			require.NotNil(t, status)

			// Verify daemon status
			assert.Equal(t, tt.expectedDaemon, status.Daemon.Running)

			// Verify issues
			assert.Len(t, status.Issues, tt.expectedIssues)

			// Verify tmux
			if tt.expectedTmux {
				assert.NotNil(t, status.Tmux)
				assert.NotEmpty(t, status.Tmux.SessionName)
			}

			mockGH.AssertExpectations(t)
			mockTmux.AssertExpectations(t)
		})
	}
}

func TestStatusService_GetDaemonStatus(t *testing.T) {
	tests := []struct {
		name          string
		pidFileExists bool
		pidContent    string
		expectedPID   int
		expectedRun   bool
	}{
		{
			name:          "daemon running",
			pidFileExists: true,
			pidContent:    "12345",
			expectedPID:   12345,
			expectedRun:   true,
		},
		{
			name:          "daemon not running",
			pidFileExists: false,
			expectedPID:   0,
			expectedRun:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.pidFileExists {
				err := os.MkdirAll(".soba", 0755)
				require.NoError(t, err)
				// Use current process PID for testing if "12345" is provided
				pidContent := tt.pidContent
				if pidContent == "12345" {
					pidContent = fmt.Sprintf("%d", os.Getpid())
				}
				err = os.WriteFile(".soba/soba.pid", []byte(pidContent), 0644)
				require.NoError(t, err)
				defer os.RemoveAll(".soba")
			}

			// Create service
			cfg := &config.Config{}
			svc := &statusService{
				cfg:          cfg,
				tmuxClient:   new(StatusMockTmuxClient),
				githubClient: new(StatusMockGitHubClient),
			}

			// Get daemon status
			status := svc.getDaemonStatus()
			require.NotNil(t, status)

			// Verify
			assert.Equal(t, tt.expectedRun, status.Running)
			if tt.expectedRun && tt.expectedPID == 12345 {
				// For the test case, we expect the current process PID
				assert.Equal(t, os.Getpid(), status.PID)
			} else if tt.expectedRun {
				assert.Equal(t, tt.expectedPID, status.PID)
			}
		})
	}
}

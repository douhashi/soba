package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/pkg/logger"
)

type MockTmuxClient struct {
	mock.Mock
}

func (m *MockTmuxClient) CreateSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockTmuxClient) DeleteSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockTmuxClient) KillSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockTmuxClient) SessionExists(sessionName string) bool {
	args := m.Called(sessionName)
	return args.Bool(0)
}

func (m *MockTmuxClient) CreateWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) DeleteWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

func (m *MockTmuxClient) CreatePane(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) DeletePane(sessionName, windowName string, paneIndex int) error {
	args := m.Called(sessionName, windowName, paneIndex)
	return args.Error(0)
}

func (m *MockTmuxClient) GetPaneCount(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *MockTmuxClient) GetFirstPaneIndex(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *MockTmuxClient) GetLastPaneIndex(sessionName, windowName string) (int, error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Error(1)
}

func (m *MockTmuxClient) ResizePanes(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockTmuxClient) SendCommand(sessionName, windowName string, paneIndex int, command string) error {
	args := m.Called(sessionName, windowName, paneIndex, command)
	return args.Error(0)
}

type MockWorkspaceManager struct {
	mock.Mock
}

func (m *MockWorkspaceManager) PrepareWorkspace(issueNumber int) error {
	args := m.Called(issueNumber)
	return args.Error(0)
}

func (m *MockWorkspaceManager) CleanupWorkspace(issueNumber int) error {
	args := m.Called(issueNumber)
	return args.Error(0)
}

type MockIssueProcessorUpdater struct {
	mock.Mock
}

func (m *MockIssueProcessorUpdater) UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	args := m.Called(ctx, issueNumber, removeLabel, addLabel)
	return args.Error(0)
}

func (m *MockIssueProcessorUpdater) Configure(cfg *config.Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func TestWorkflowExecutor_ExecutePhase(t *testing.T) {
	tests := []struct {
		name         string
		issueNumber  int
		phase        domain.Phase
		currentLabel string
		nextLabel    string
		setupMocks   func(*MockTmuxClient, *MockWorkspaceManager, *MockIssueProcessorUpdater)
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "Execute queue phase successfully",
			issueNumber:  123,
			phase:        domain.PhaseQueue,
			currentLabel: domain.LabelTodo,
			nextLabel:    domain.LabelQueued,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("Configure", mock.Anything).Return(nil)
				processor.On("UpdateLabels", mock.Anything, 123, domain.LabelTodo, domain.LabelQueued).Return(nil)
				// Queueフェーズは ExecutionTypeLabelOnly なので、tmux操作は行われない
			},
			wantErr: false,
		},
		{
			name:         "Execute plan phase with existing session",
			issueNumber:  456,
			phase:        domain.PhasePlan,
			currentLabel: domain.LabelQueued,
			nextLabel:    domain.LabelPlanning,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("Configure", mock.Anything).Return(nil)
				processor.On("UpdateLabels", mock.Anything, 456, domain.LabelQueued, domain.LabelPlanning).Return(nil)
				workspace.On("PrepareWorkspace", 456).Return(nil) // Planフェーズでworktree準備
				tmux.On("SessionExists", "soba-test-repo").Return(true)
				tmux.On("WindowExists", "soba-test-repo", "issue-456").Return(false, nil)
				tmux.On("CreateWindow", "soba-test-repo", "issue-456").Return(nil)
				// Window was created, so no pane management
				tmux.On("GetLastPaneIndex", "soba-test-repo", "issue-456").Return(0, nil)
				tmux.On("SendCommand", "soba-test-repo", "issue-456", 0, `cd .git/soba/worktrees/issue-456 && echo "Planning"`).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "Delete old pane when max panes reached",
			issueNumber:  789,
			phase:        domain.PhaseImplement,
			currentLabel: domain.LabelReady,
			nextLabel:    domain.LabelDoing,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("Configure", mock.Anything).Return(nil)
				processor.On("UpdateLabels", mock.Anything, 789, domain.LabelReady, domain.LabelDoing).Return(nil)
				workspace.On("PrepareWorkspace", 789).Return(nil) // Implementフェーズでworktree準備
				tmux.On("SessionExists", "soba-test-repo").Return(true)
				tmux.On("WindowExists", "soba-test-repo", "issue-789").Return(true, nil)
				tmux.On("GetPaneCount", "soba-test-repo", "issue-789").Return(3, nil)      // Max panes reached
				tmux.On("GetFirstPaneIndex", "soba-test-repo", "issue-789").Return(0, nil) // 削除用
				tmux.On("DeletePane", "soba-test-repo", "issue-789", 0).Return(nil)
				tmux.On("CreatePane", "soba-test-repo", "issue-789").Return(nil)
				tmux.On("ResizePanes", "soba-test-repo", "issue-789").Return(nil)
				tmux.On("GetLastPaneIndex", "soba-test-repo", "issue-789").Return(2, nil) // 送信用（新しいペイン）
				tmux.On("SendCommand", "soba-test-repo", "issue-789", 2, `cd .git/soba/worktrees/issue-789 && echo "Implementing"`).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "Error when updating labels",
			issueNumber:  999,
			phase:        domain.PhaseReview,
			currentLabel: domain.LabelReviewRequested,
			nextLabel:    domain.LabelReviewing,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("Configure", mock.Anything).Return(nil)
				processor.On("UpdateLabels", mock.Anything, 999, domain.LabelReviewRequested, domain.LabelReviewing).
					Return(errors.New("failed to update labels"))
			},
			wantErr:    true,
			errMessage: "failed to update labels",
		},
		{
			name:         "Error when creating tmux session",
			issueNumber:  111,
			phase:        domain.PhasePlan,
			currentLabel: domain.LabelQueued,
			nextLabel:    domain.LabelPlanning,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("Configure", mock.Anything).Return(nil)
				processor.On("UpdateLabels", mock.Anything, 111, domain.LabelQueued, domain.LabelPlanning).Return(nil)
				workspace.On("PrepareWorkspace", 111).Return(nil) // Planフェーズでworktree準備
				tmux.On("SessionExists", "soba-test-repo").Return(false)
				tmux.On("CreateSession", "soba-test-repo").Return(errors.New("tmux error"))
			},
			wantErr:    true,
			errMessage: "tmux error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTmux := new(MockTmuxClient)
			mockWorkspace := new(MockWorkspaceManager)
			mockProcessor := new(MockIssueProcessorUpdater)

			if tt.setupMocks != nil {
				tt.setupMocks(mockTmux, mockWorkspace, mockProcessor)
			}

			executor := NewWorkflowExecutorWithLogger(mockTmux, mockWorkspace, mockProcessor, logger.NewNopLogger())

			cfg := &config.Config{
				Git: config.GitConfig{
					WorktreeBasePath: ".git/soba/worktrees",
				},
				GitHub: config.GitHubConfig{
					Repository: "test/repo",
				},
				Phase: config.PhaseConfig{
					Plan:      config.PhaseCommand{Command: "echo", Options: []string{}, Parameter: "Planning"},
					Implement: config.PhaseCommand{Command: "echo", Options: []string{}, Parameter: "Implementing"},
					Review:    config.PhaseCommand{Command: "echo", Options: []string{}, Parameter: "Review"},
				},
			}

			ctx := context.Background()

			err := executor.ExecutePhase(ctx, cfg, tt.issueNumber, tt.phase)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockTmux.AssertExpectations(t)
			mockWorkspace.AssertExpectations(t)
			mockProcessor.AssertExpectations(t)
		})
	}
}

func TestWorkflowExecutor_managePane(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		maxPanes    int
		setupMocks  func(*MockTmuxClient)
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "Create new pane when under limit",
			sessionName: "soba",
			windowName:  "issue-123",
			maxPanes:    3,
			setupMocks: func(tmux *MockTmuxClient) {
				tmux.On("GetPaneCount", "soba", "issue-123").Return(2, nil)
				tmux.On("CreatePane", "soba", "issue-123").Return(nil)
				tmux.On("ResizePanes", "soba", "issue-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "Delete oldest pane when at limit",
			sessionName: "soba",
			windowName:  "issue-456",
			maxPanes:    3,
			setupMocks: func(tmux *MockTmuxClient) {
				tmux.On("GetPaneCount", "soba", "issue-456").Return(3, nil)
				tmux.On("GetFirstPaneIndex", "soba", "issue-456").Return(0, nil)
				tmux.On("DeletePane", "soba", "issue-456", 0).Return(nil)
				tmux.On("CreatePane", "soba", "issue-456").Return(nil)
				tmux.On("ResizePanes", "soba", "issue-456").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "Error getting pane count",
			sessionName: "soba",
			windowName:  "issue-789",
			maxPanes:    3,
			setupMocks: func(tmux *MockTmuxClient) {
				tmux.On("GetPaneCount", "soba", "issue-789").Return(0, errors.New("tmux error"))
			},
			wantErr:    true,
			errMessage: "tmux error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTmux := new(MockTmuxClient)
			mockWorkspace := new(MockWorkspaceManager)
			mockProcessor := new(MockIssueProcessorUpdater)

			if tt.setupMocks != nil {
				tt.setupMocks(mockTmux)
			}

			executor := &workflowExecutor{
				tmux:           mockTmux,
				workspace:      mockWorkspace,
				issueProcessor: mockProcessor,
				logger:         logger.NewNopLogger(),
				maxPanes:       tt.maxPanes,
			}

			err := executor.managePane(tt.sessionName, tt.windowName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockTmux.AssertExpectations(t)
		})
	}
}

func TestWorkflowExecutor_ExecutePhase_WithWorktreePreparation(t *testing.T) {
	// Planフェーズ開始時にworktreeが準備されることを確認
	mockTmux := new(MockTmuxClient)
	mockWorkspace := new(MockWorkspaceManager)
	mockProcessor := new(MockIssueProcessorUpdater)

	// Mock設定
	mockProcessor.On("Configure", mock.Anything).Return(nil) // Configure呼び出しを追加
	mockProcessor.On("UpdateLabels", mock.Anything, 1, "soba:queued", "soba:planning").Return(nil)
	mockWorkspace.On("PrepareWorkspace", 1).Return(nil) // worktree準備が呼ばれることを期待
	mockTmux.On("SessionExists", "soba-test-repo").Return(true)
	mockTmux.On("WindowExists", "soba-test-repo", "issue-1").Return(false, nil)
	mockTmux.On("CreateWindow", "soba-test-repo", "issue-1").Return(nil)
	// Window was created, so no pane management
	mockTmux.On("GetLastPaneIndex", "soba-test-repo", "issue-1").Return(0, nil)
	mockTmux.On("SendCommand", "soba-test-repo", "issue-1", 0, `cd .git/soba/worktrees/issue-1 && soba:plan "1"`).Return(nil)

	executor := NewWorkflowExecutorWithLogger(mockTmux, mockWorkspace, mockProcessor, logger.NewNopLogger())

	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
		},
		GitHub: config.GitHubConfig{
			Repository: "test/repo",
		},
		Phase: config.PhaseConfig{
			Plan: config.PhaseCommand{
				Command:   "soba:plan",
				Parameter: "{issue_number}",
			},
		},
	}

	err := executor.ExecutePhase(context.Background(), cfg, 1, domain.PhasePlan)

	assert.NoError(t, err)
	mockWorkspace.AssertCalled(t, "PrepareWorkspace", 1) // worktree準備が呼ばれたことを確認
	mockTmux.AssertExpectations(t)
	mockProcessor.AssertExpectations(t)
}

func TestWorkflowExecutor_executeCommandPhase_PaneSkip(t *testing.T) {
	tests := []struct {
		name             string
		windowExists     bool
		requiresPane     bool
		expectManagePane bool
		description      string
	}{
		{
			name:             "New window created - skip pane management even if requiresPane is true",
			windowExists:     false, // window will be created
			requiresPane:     true,
			expectManagePane: false, // should skip pane management
			description:      "When a new window is created, pane management should be skipped",
		},
		{
			name:             "Existing window - perform pane management if requiresPane is true",
			windowExists:     true, // window exists
			requiresPane:     true,
			expectManagePane: true, // should perform pane management
			description:      "When using existing window, pane management should be performed",
		},
		{
			name:             "New window created - no pane management if requiresPane is false",
			windowExists:     false, // window will be created
			requiresPane:     false,
			expectManagePane: false, // should skip pane management
			description:      "When requiresPane is false, pane management should be skipped",
		},
		{
			name:             "Existing window - no pane management if requiresPane is false",
			windowExists:     true, // window exists
			requiresPane:     false,
			expectManagePane: false, // should skip pane management
			description:      "When requiresPane is false, pane management should be skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTmux := new(MockTmuxClient)
			mockWorkspace := new(MockWorkspaceManager)

			// Setup tmux session mocks
			mockTmux.On("SessionExists", "soba-test-repo").Return(true)
			mockTmux.On("WindowExists", "soba-test-repo", "issue-42").Return(tt.windowExists, nil)

			if !tt.windowExists {
				// Window will be created
				mockTmux.On("CreateWindow", "soba-test-repo", "issue-42").Return(nil)
			}

			// Setup pane management mocks if expected
			if tt.expectManagePane {
				mockTmux.On("GetPaneCount", "soba-test-repo", "issue-42").Return(1, nil)
				mockTmux.On("CreatePane", "soba-test-repo", "issue-42").Return(nil)
				mockTmux.On("ResizePanes", "soba-test-repo", "issue-42").Return(nil)
			}

			// Setup command execution mocks
			mockTmux.On("GetLastPaneIndex", "soba-test-repo", "issue-42").Return(0, nil)
			mockTmux.On("SendCommand", "soba-test-repo", "issue-42", 0, mock.Anything).Return(nil)

			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "test/repo",
				},
				Workflow: config.WorkflowConfig{
					TmuxCommandDelay: 0,
				},
				Phase: config.PhaseConfig{
					Plan: config.PhaseCommand{
						Command: "echo",
						Options: []string{"test"},
					},
				},
			}

			executor := &workflowExecutor{
				tmux:      mockTmux,
				workspace: mockWorkspace,
				logger:    logger.NewLogger(logger.GetLogger()),
				maxPanes:  4,
			}

			// Create phase definition with requiresPane setting
			phaseDef := &domain.PhaseDefinition{
				Name:         "plan",
				RequiresPane: tt.requiresPane,
			}

			err := executor.executeCommandPhase(cfg, 42, domain.PhasePlan, phaseDef)
			assert.NoError(t, err)

			mockTmux.AssertExpectations(t)
			mockWorkspace.AssertExpectations(t)
		})
	}
}

func TestWorkflowExecutor_buildCommand(t *testing.T) {
	tests := []struct {
		name         string
		phaseCommand config.PhaseCommand
		issueNumber  int
		expected     string
	}{
		{
			name: "Build command with parameter",
			phaseCommand: config.PhaseCommand{
				Command:   "soba",
				Options:   []string{"plan"},
				Parameter: "123",
			},
			issueNumber: 123,
			expected:    `soba plan "123"`,
		},
		{
			name: "Build command without parameter",
			phaseCommand: config.PhaseCommand{
				Command:   "echo",
				Options:   []string{"Hello", "World"},
				Parameter: "",
			},
			issueNumber: 456,
			expected:    "echo Hello World",
		},
		{
			name: "Build command with issue number placeholder (old format)",
			phaseCommand: config.PhaseCommand{
				Command:   "gh",
				Options:   []string{"issue", "view"},
				Parameter: "{issue_number}",
			},
			issueNumber: 789,
			expected:    `gh issue view "789"`,
		},
		{
			name: "Build command with {{issue-number}} placeholder",
			phaseCommand: config.PhaseCommand{
				Command:   "claude",
				Options:   []string{"--dangerously-skip-permissions"},
				Parameter: "/soba:implement {{issue-number}}",
			},
			issueNumber: 44,
			expected:    `claude --dangerously-skip-permissions "/soba:implement 44"`,
		},
		{
			name: "Build command with multiple {{issue-number}} placeholders",
			phaseCommand: config.PhaseCommand{
				Command:   "echo",
				Options:   []string{},
				Parameter: "Issue {{issue-number}} and {{issue-number}} again",
			},
			issueNumber: 100,
			expected:    `echo "Issue 100 and 100 again"`,
		},
		{
			name: "Build command with parameter should be quoted",
			phaseCommand: config.PhaseCommand{
				Command:   "claude",
				Options:   []string{"--dangerously-skip-permissions"},
				Parameter: "/soba:plan",
			},
			issueNumber: 123,
			expected:    `claude --dangerously-skip-permissions "/soba:plan"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &workflowExecutor{}
			result := executor.buildCommand(tt.phaseCommand, tt.issueNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSessionName(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		want       string
	}{
		{
			name:       "Normal repository format",
			repository: "douhashi/soba",
			want:       "soba-douhashi-soba",
		},
		{
			name:       "Repository with special characters",
			repository: "user-name/repo.name",
			want:       "soba-user-name-repo.name",
		},
		{
			name:       "Empty repository",
			repository: "",
			want:       "soba",
		},
		{
			name:       "Repository without slash",
			repository: "invalid",
			want:       "soba",
		},
		{
			name:       "Repository with multiple slashes",
			repository: "org/sub/repo",
			want:       "soba-org-sub-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &workflowExecutor{}
			result := executor.generateSessionName(tt.repository)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestWorkflowExecutor_setupTmuxSession(t *testing.T) {
	tests := []struct {
		name          string
		sessionExists bool
		windowExists  bool
		windowCreated bool
		expectedError error
		setupMocks    func(*MockTmuxClient)
	}{
		{
			name:          "Create new session and window",
			sessionExists: false,
			windowExists:  false,
			windowCreated: true,
			expectedError: nil,
			setupMocks: func(m *MockTmuxClient) {
				m.On("SessionExists", "soba").Return(false)
				m.On("CreateSession", "soba").Return(nil)
				m.On("WindowExists", "soba", "issue-1").Return(false, nil)
				m.On("CreateWindow", "soba", "issue-1").Return(nil)
			},
		},
		{
			name:          "Session exists, create new window",
			sessionExists: true,
			windowExists:  false,
			windowCreated: true,
			expectedError: nil,
			setupMocks: func(m *MockTmuxClient) {
				m.On("SessionExists", "soba").Return(true)
				m.On("WindowExists", "soba", "issue-1").Return(false, nil)
				m.On("CreateWindow", "soba", "issue-1").Return(nil)
			},
		},
		{
			name:          "Both session and window exist",
			sessionExists: true,
			windowExists:  true,
			windowCreated: false,
			expectedError: nil,
			setupMocks: func(m *MockTmuxClient) {
				m.On("SessionExists", "soba").Return(true)
				m.On("WindowExists", "soba", "issue-1").Return(true, nil)
			},
		},
		{
			name:          "Failed to create session",
			sessionExists: false,
			windowExists:  false,
			windowCreated: false,
			expectedError: errors.New("create session failed"),
			setupMocks: func(m *MockTmuxClient) {
				m.On("SessionExists", "soba").Return(false)
				m.On("CreateSession", "soba").Return(errors.New("create session failed"))
			},
		},
		{
			name:          "Failed to check window existence",
			sessionExists: true,
			windowExists:  false,
			windowCreated: false,
			expectedError: errors.New("check window failed"),
			setupMocks: func(m *MockTmuxClient) {
				m.On("SessionExists", "soba").Return(true)
				m.On("WindowExists", "soba", "issue-1").Return(false, errors.New("check window failed"))
			},
		},
		{
			name:          "Failed to create window",
			sessionExists: true,
			windowExists:  false,
			windowCreated: false,
			expectedError: errors.New("create window failed"),
			setupMocks: func(m *MockTmuxClient) {
				m.On("SessionExists", "soba").Return(true)
				m.On("WindowExists", "soba", "issue-1").Return(false, nil)
				m.On("CreateWindow", "soba", "issue-1").Return(errors.New("create window failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTmux := new(MockTmuxClient)
			tt.setupMocks(mockTmux)

			executor := &workflowExecutor{
				tmux:     mockTmux,
				logger:   logger.NewLogger(logger.GetLogger()),
				maxPanes: 4,
			}

			windowCreated, err := executor.setupTmuxSession("soba", "issue-1")

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.windowCreated, windowCreated)
			}

			mockTmux.AssertExpectations(t)
		})
	}
}

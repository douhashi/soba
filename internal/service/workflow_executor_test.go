package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
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
				processor.On("UpdateLabels", mock.Anything, 123, domain.LabelTodo, domain.LabelQueued).Return(nil)
				tmux.On("SessionExists", "soba").Return(false)
				tmux.On("CreateSession", "soba").Return(nil)
				tmux.On("WindowExists", "soba", "issue-123").Return(false, nil)
				tmux.On("CreateWindow", "soba", "issue-123").Return(nil)
				tmux.On("GetPaneCount", "soba", "issue-123").Return(0, nil)
				tmux.On("CreatePane", "soba", "issue-123").Return(nil)
				tmux.On("ResizePanes", "soba", "issue-123").Return(nil)
				// Queueフェーズはコマンドなしなので、GetFirstPaneIndexとSendCommandは呼ばれない
			},
			wantErr: false,
		},
		{
			name:         "Execute plan phase with existing session",
			issueNumber:  456,
			phase:        domain.PhasePlan,
			currentLabel: domain.LabelQueued,
			nextLabel:    domain.LabelReady,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("UpdateLabels", mock.Anything, 456, domain.LabelQueued, domain.LabelReady).Return(nil)
				tmux.On("SessionExists", "soba").Return(true)
				tmux.On("WindowExists", "soba", "issue-456").Return(false, nil)
				tmux.On("CreateWindow", "soba", "issue-456").Return(nil)
				tmux.On("GetPaneCount", "soba", "issue-456").Return(0, nil)
				tmux.On("CreatePane", "soba", "issue-456").Return(nil)
				tmux.On("ResizePanes", "soba", "issue-456").Return(nil)
				tmux.On("GetFirstPaneIndex", "soba", "issue-456").Return(0, nil)
				tmux.On("SendCommand", "soba", "issue-456", 0, "echo Planning").Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "Delete old pane when max panes reached",
			issueNumber:  789,
			phase:        domain.PhaseImplement,
			currentLabel: domain.LabelReady,
			nextLabel:    domain.LabelReviewRequested,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("UpdateLabels", mock.Anything, 789, domain.LabelReady, domain.LabelReviewRequested).Return(nil)
				tmux.On("SessionExists", "soba").Return(true)
				tmux.On("WindowExists", "soba", "issue-789").Return(true, nil)
				tmux.On("GetPaneCount", "soba", "issue-789").Return(3, nil)               // Max panes reached
				tmux.On("GetFirstPaneIndex", "soba", "issue-789").Return(0, nil).Times(2) // 削除と送信で2回呼ばれる
				tmux.On("DeletePane", "soba", "issue-789", 0).Return(nil)
				tmux.On("CreatePane", "soba", "issue-789").Return(nil)
				tmux.On("ResizePanes", "soba", "issue-789").Return(nil)
				tmux.On("SendCommand", "soba", "issue-789", 0, "echo Implementing").Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "Error when updating labels",
			issueNumber:  999,
			phase:        domain.PhaseReview,
			currentLabel: domain.LabelReviewRequested,
			nextLabel:    domain.LabelDone,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("UpdateLabels", mock.Anything, 999, domain.LabelReviewRequested, domain.LabelDone).
					Return(errors.New("failed to update labels"))
			},
			wantErr:    true,
			errMessage: "failed to update labels",
		},
		{
			name:         "Error when creating tmux session",
			issueNumber:  111,
			phase:        domain.PhaseQueue,
			currentLabel: domain.LabelTodo,
			nextLabel:    domain.LabelQueued,
			setupMocks: func(tmux *MockTmuxClient, workspace *MockWorkspaceManager, processor *MockIssueProcessorUpdater) {
				processor.On("UpdateLabels", mock.Anything, 111, domain.LabelTodo, domain.LabelQueued).Return(nil)
				tmux.On("SessionExists", "soba").Return(false)
				tmux.On("CreateSession", "soba").Return(errors.New("tmux error"))
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

			executor := NewWorkflowExecutor(mockTmux, mockWorkspace, mockProcessor)

			cfg := &config.Config{
				Phase: config.PhaseConfig{
					Plan:      config.PhaseCommand{Command: "echo", Options: []string{}, Parameter: "Planning"},
					Implement: config.PhaseCommand{Command: "echo", Options: []string{}, Parameter: "Implementing"},
					Review:    config.PhaseCommand{Command: "echo", Options: []string{}, Parameter: "Review"},
				},
			}

			phaseStrategy := domain.NewDefaultPhaseStrategy()
			ctx := context.Background()

			err := executor.ExecutePhase(ctx, cfg, tt.issueNumber, tt.phase, phaseStrategy)

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
			expected:    "soba plan 123",
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
			name: "Build command with issue number placeholder",
			phaseCommand: config.PhaseCommand{
				Command:   "gh",
				Options:   []string{"issue", "view"},
				Parameter: "{issue_number}",
			},
			issueNumber: 789,
			expected:    "gh issue view 789",
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

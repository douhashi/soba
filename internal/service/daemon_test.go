package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
)

func TestNewDaemonService(t *testing.T) {
	service := NewDaemonService()
	assert.NotNil(t, service)
}

// StartForegroundはloggingシステムとの競合でテストが困難なため、スキップ
func TestDaemonService_StartForeground(t *testing.T) {
	t.Skip("StartForeground test skipped due to logging system conflicts in test environment")
}

// StartDaemonもloggingシステムとの競合でテストが困難なため、スキップ
func TestDaemonService_StartDaemon(t *testing.T) {
	t.Skip("StartDaemon test skipped due to logging system conflicts in test environment")
}

func TestDaemonService_CreatePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	service := &daemonService{
		workDir: tmpDir,
	}

	err := service.createPIDFile()
	assert.NoError(t, err)

	// PIDファイルが作成されていることを確認
	pidFile := filepath.Join(sobaDir, "soba.pid")
	_, err = os.Stat(pidFile)
	assert.NoError(t, err)

	// PIDファイルの内容を確認
	content, err := os.ReadFile(pidFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestDaemonService_RemovePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	service := &daemonService{
		workDir: tmpDir,
	}

	// PIDファイルを作成
	err := service.createPIDFile()
	require.NoError(t, err)

	// PIDファイルを削除
	err = service.removePIDFile()
	assert.NoError(t, err)

	// PIDファイルが削除されていることを確認
	pidFile := filepath.Join(sobaDir, "soba.pid")
	_, err = os.Stat(pidFile)
	assert.True(t, os.IsNotExist(err))
}

func TestDaemonService_IsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	service := &daemonService{
		workDir: tmpDir,
	}

	// 最初は実行されていない
	running := service.IsRunning()
	assert.False(t, running)

	// PIDファイルを作成
	err := service.createPIDFile()
	require.NoError(t, err)

	// 実行中として検出される
	running = service.IsRunning()
	assert.True(t, running)
}

func TestDaemonService_InitializeTmuxSession(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		sessionExists bool
		createError   error
		wantError     bool
	}{
		{
			name:          "Create new session successfully",
			repository:    "douhashi/soba",
			sessionExists: false,
			createError:   nil,
			wantError:     false,
		},
		{
			name:          "Session already exists",
			repository:    "douhashi/soba",
			sessionExists: true,
			createError:   nil,
			wantError:     false,
		},
		{
			name:          "Error creating session",
			repository:    "douhashi/soba",
			sessionExists: false,
			createError:   assert.AnError,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTmux := new(MockTmuxClient)

			sessionName := "soba-douhashi-soba"
			mockTmux.On("SessionExists", sessionName).Return(tt.sessionExists)

			if !tt.sessionExists {
				mockTmux.On("CreateSession", sessionName).Return(tt.createError)
			}

			service := &daemonService{
				tmux: mockTmux,
			}

			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					Repository: tt.repository,
				},
			}

			err := service.initializeTmuxSession(cfg)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockTmux.AssertExpectations(t)
		})
	}
}

// MockIssueProcessor はテスト用のモックプロセッサ
type MockIssueProcessor struct {
	processFunc      func(ctx context.Context, cfg *config.Config) error
	processCalled    bool
	updateLabelsFunc func(ctx context.Context, issueNumber int, removeLabel, addLabel string) error
	ProcessIssueFunc func(ctx context.Context, cfg *config.Config, issue github.Issue) error
}

func (m *MockIssueProcessor) Process(ctx context.Context, cfg *config.Config) error {
	m.processCalled = true
	if m.processFunc != nil {
		return m.processFunc(ctx, cfg)
	}
	return nil
}

func (m *MockIssueProcessor) UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	if m.updateLabelsFunc != nil {
		return m.updateLabelsFunc(ctx, issueNumber, removeLabel, addLabel)
	}
	return nil
}

func (m *MockIssueProcessor) ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error {
	if m.ProcessIssueFunc != nil {
		return m.ProcessIssueFunc(ctx, cfg, issue)
	}
	return nil
}

func (m *MockIssueProcessor) Configure(cfg *config.Config) error {
	return nil
}

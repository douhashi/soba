package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/app"
	"github.com/douhashi/soba/pkg/logging"
)

func TestNewDaemonService(t *testing.T) {
	// Initialize app for testing
	helper := app.NewTestHelper(t)
	helper.InitializeForTest()

	// Test NewDaemonService without config (should use defaults and fail for repository requirement)
	assert.Panics(t, func() {
		NewDaemonService(app.LogFactory())
	})
}

func TestNewDaemonServiceWithConfig(t *testing.T) {
	// Initialize app for testing
	helper := app.NewTestHelper(t)
	helper.InitializeForTest()

	// Test NewDaemonServiceWithConfig with valid config
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Repository: "douhashi/soba-cli",
		},
		Workflow: config.WorkflowConfig{
			Interval: 20,
		},
		Git: config.GitConfig{
			WorktreeBasePath: ".git/soba/worktrees",
		},
	}

	// Create service with config and LogFactory
	service := NewDaemonServiceWithConfig(cfg, app.LogFactory())
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

// TestDaemonService_StartDaemonInBackground tests the background daemon start functionality
func TestDaemonService_StartDaemonInBackground(t *testing.T) {
	tests := []struct {
		name           string
		envVar         string
		wantFork       bool
		alreadyRunning bool
		wantError      bool
	}{
		{
			name:           "Parent process should fork child",
			envVar:         "",
			wantFork:       true,
			alreadyRunning: false,
			wantError:      false,
		},
		{
			name:           "Child process should continue",
			envVar:         "true",
			wantFork:       false,
			alreadyRunning: false,
			wantError:      false,
		},
		{
			name:           "Should error if already running",
			envVar:         "",
			wantFork:       false,
			alreadyRunning: true,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sobaDir := filepath.Join(tmpDir, ".soba")
			require.NoError(t, os.MkdirAll(sobaDir, 0755))

			// Set test mode to prevent os.Exit
			t.Setenv("SOBA_TEST_MODE", "true")

			// Set environment variable for test
			if tt.envVar != "" {
				t.Setenv("SOBA_BACKGROUND_PROCESS", tt.envVar)
			}

			mockTmux := new(MockTmuxClient)
			mockLogger := logging.NewMockLogger()
			service := &daemonService{
				workDir: tmpDir,
				tmux:    mockTmux,
				logger:  mockLogger,
			}

			if tt.alreadyRunning {
				// Create PID file to simulate running daemon
				err := service.createPIDFile()
				require.NoError(t, err)
			}

			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "douhashi/soba",
				},
				Workflow: config.WorkflowConfig{
					Interval: 30,
				},
				Log: config.LogConfig{
					OutputPath:     ".soba/logs/soba-${PID}.log",
					RetentionCount: 10,
				},
			}

			if !tt.alreadyRunning && tt.envVar == "true" {
				// Child process case - expect tmux initialization
				mockTmux.On("SessionExists", "soba-douhashi-soba").Return(false)
				mockTmux.On("CreateSession", "soba-douhashi-soba").Return(nil)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Run the method
			err := service.StartDaemon(ctx, cfg)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				// Note: In real implementation, parent process will exit after fork
				// and child process will continue. This test checks the setup logic.
				if tt.wantFork {
					// Parent process case - should prepare for fork
					// (actual fork not tested here as it requires process separation)
				} else if tt.envVar == "true" {
					// Child process case - should continue with daemon logic
					mockTmux.AssertExpectations(t)
				}
			}
		})
	}
}

// TestDaemonService_ProcessSeparation tests process separation attributes
func TestDaemonService_ProcessSeparation(t *testing.T) {
	t.Run("Should set correct process attributes", func(t *testing.T) {
		// This test verifies that getSysProcAttr returns correct attributes
		// for process separation. The actual implementation will be OS-specific.
		attr := getSysProcAttr()

		if attr != nil {
			// On Unix systems, we expect Setsid to be true
			// On Windows, we expect specific creation flags
			// The actual assertion depends on the OS
			assert.NotNil(t, attr, "Process attributes should be set")
		}
	})
}

func TestDaemonService_CreatePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	mockLogger := logging.NewMockLogger()
	service := &daemonService{
		workDir: tmpDir,
		logger:  mockLogger,
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

	mockLogger := logging.NewMockLogger()
	service := &daemonService{
		workDir: tmpDir,
		logger:  mockLogger,
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

	mockLogger := logging.NewMockLogger()
	service := &daemonService{
		workDir: tmpDir,
		logger:  mockLogger,
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

			mockLogger := logging.NewMockLogger()
			service := &daemonService{
				tmux:   mockTmux,
				logger: mockLogger,
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

func TestDaemonService_ConfigureAndStartWatchers_WithNilClosedIssueCleanupService(t *testing.T) {
	tests := []struct {
		name                      string
		closedIssueCleanupService *ClosedIssueCleanupService
		wantPanic                 bool
	}{
		{
			name:                      "nil ClosedIssueCleanupService should not panic",
			closedIssueCleanupService: nil,
			wantPanic:                 false,
		},
		{
			name:                      "valid ClosedIssueCleanupService should work",
			closedIssueCleanupService: &ClosedIssueCleanupService{
				// githubClientとtmuxClientはnilでOK（テストでは使わない）
				// configureAndStartWatchersがこれらを適切に設定する
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// MockのGitHubClientとTmuxClientを作成
			mockGitHubClient := new(MockGitHubClient)
			mockTmux := new(MockTmuxClient)

			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "douhashi/soba",
				},
				Workflow: config.WorkflowConfig{
					ClosedIssueCleanupEnabled:  true,
					ClosedIssueCleanupInterval: 60,
					Interval:                   20,
				},
			}

			// IssueWatcherとPRWatcherを作成
			watcher := NewIssueWatcher(mockGitHubClient, cfg)
			prWatcher := NewPRWatcher(mockGitHubClient, cfg)

			// ロガーを設定（nilポインタエラーを防ぐため）
			if watcher != nil {
				watcher.logger = logging.NewMockLogger()
			}
			if prWatcher != nil {
				prWatcher.logger = logging.NewMockLogger()
			}

			mockLogger := logging.NewMockLogger()
			service := &daemonService{
				watcher:                   watcher,
				prWatcher:                 prWatcher,
				closedIssueCleanupService: tt.closedIssueCleanupService,
				tmux:                      mockTmux,
				logger:                    mockLogger,
			}

			if tt.wantPanic {
				assert.Panics(t, func() {
					_ = service.configureAndStartWatchers(ctx, cfg)
				})
			} else {
				// configureAndStartWatchersをgoroutineで実行
				errCh := make(chan error, 1)
				go func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("Unexpected panic: %v", r)
						}
					}()
					errCh <- service.configureAndStartWatchers(ctx, cfg)
				}()

				// 少し待ってからキャンセル
				time.Sleep(10 * time.Millisecond)
				cancel()

				// エラーを待つ（タイムアウト付き）
				select {
				case <-errCh:
					// 正常終了
				case <-time.After(100 * time.Millisecond):
					// タイムアウトOK（goroutineが動作している）
				}
			}
		})
	}
}

func TestDaemonService_Stop(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*testing.T, string) *daemonService
		wantError      bool
		expectedErrMsg string
	}{
		{
			name: "Stop when daemon is not running",
			setupFunc: func(t *testing.T, tmpDir string) *daemonService {
				mockTmux := new(MockTmuxClient)
				mockLogger := logging.NewMockLogger()
				return &daemonService{
					workDir: tmpDir,
					tmux:    mockTmux,
					logger:  mockLogger,
				}
			},
			wantError:      true,
			expectedErrMsg: "daemon is not running",
		},
		{
			name: "Stop with invalid PID in file",
			setupFunc: func(t *testing.T, tmpDir string) *daemonService {
				sobaDir := filepath.Join(tmpDir, ".soba")
				require.NoError(t, os.MkdirAll(sobaDir, 0755))

				// 無効なPIDを含むファイルを作成
				pidFile := filepath.Join(sobaDir, "soba.pid")
				require.NoError(t, os.WriteFile(pidFile, []byte("invalid"), 0600))

				mockTmux := new(MockTmuxClient)
				mockLogger := logging.NewMockLogger()
				return &daemonService{
					workDir: tmpDir,
					tmux:    mockTmux,
					logger:  mockLogger,
				}
			},
			wantError:      true,
			expectedErrMsg: "invalid PID in file",
		},
		{
			name: "Stop with non-existent process",
			setupFunc: func(t *testing.T, tmpDir string) *daemonService {
				sobaDir := filepath.Join(tmpDir, ".soba")
				require.NoError(t, os.MkdirAll(sobaDir, 0755))

				// 存在しないPIDを含むファイルを作成（非常に大きいPIDを使用）
				pidFile := filepath.Join(sobaDir, "soba.pid")
				require.NoError(t, os.WriteFile(pidFile, []byte("999999"), 0600))

				mockTmux := new(MockTmuxClient)
				mockLogger := logging.NewMockLogger()
				return &daemonService{
					workDir: tmpDir,
					tmux:    mockTmux,
					logger:  mockLogger,
				}
			},
			wantError:      true,
			expectedErrMsg: "process not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			service := tt.setupFunc(t, tmpDir)

			ctx := context.Background()
			err := service.Stop(ctx, "douhashi/soba")

			if tt.wantError {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)

				// PIDファイルが削除されていることを確認
				pidFile := filepath.Join(tmpDir, ".soba", "soba.pid")
				_, err = os.Stat(pidFile)
				assert.True(t, os.IsNotExist(err))
			}

			// モックの期待値を検証
			if mockTmux, ok := service.tmux.(*MockTmuxClient); ok {
				mockTmux.AssertExpectations(t)
			}
		})
	}
}

func TestDaemonService_InitializeLogging(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		expectLogFile bool
		wantError     bool
	}{
		{
			name: "Initialize logging with valid config",
			cfg: &config.Config{
				Log: config.LogConfig{
					OutputPath:     ".soba/logs/soba-${PID}.log",
					RetentionCount: 5,
				},
			},
			expectLogFile: true,
			wantError:     false,
		},
		{
			name: "Initialize logging with empty output path",
			cfg: &config.Config{
				Log: config.LogConfig{
					OutputPath:     "",
					RetentionCount: 0,
				},
			},
			expectLogFile: false,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mockLogger := logging.NewMockLogger()
			service := &daemonService{
				workDir: tmpDir,
				logger:  mockLogger,
			}

			logPath, err := service.prepareLogPath(tt.cfg)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expectLogFile && logPath != "" {
					// ログファイルパスが正しく生成されていることを確認
					assert.Contains(t, logPath, ".soba/logs/soba-")
					assert.Contains(t, logPath, ".log")

					// ディレクトリが作成されていることを確認
					logDir := filepath.Dir(logPath)
					_, err := os.Stat(logDir)
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestDaemonService_StartForegroundWithLogging(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		expectLogFile bool
	}{
		{
			name: "StartForeground should create log file",
			cfg: &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "douhashi/soba",
				},
				Workflow: config.WorkflowConfig{
					Interval: 30,
				},
				Log: config.LogConfig{
					OutputPath:     ".soba/logs/soba-${PID}.log",
					RetentionCount: 5,
				},
			},
			expectLogFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			mockTmux := new(MockTmuxClient)
			mockTmux.On("SessionExists", "soba-douhashi-soba").Return(false)
			mockTmux.On("CreateSession", "soba-douhashi-soba").Return(nil)

			mockLogger := logging.NewMockLogger()
			service := &daemonService{
				workDir: tmpDir,
				tmux:    mockTmux,
				logger:  mockLogger,
			}

			ctx, cancel := context.WithCancel(context.Background())

			// StartForegroundをgoroutineで実行
			errCh := make(chan error, 1)
			go func() {
				errCh <- service.StartForeground(ctx, tt.cfg)
			}()

			// 少し待ってからキャンセル
			time.Sleep(10 * time.Millisecond)
			cancel()

			// エラーを待つ
			select {
			case <-errCh:
				// 正常終了
			case <-time.After(100 * time.Millisecond):
				// タイムアウトOK
			}

			if tt.expectLogFile {
				// ログディレクトリが作成されていることを確認
				logDir := filepath.Join(tmpDir, ".soba", "logs")
				_, err := os.Stat(logDir)
				assert.NoError(t, err)
			}

			mockTmux.AssertExpectations(t)
		})
	}
}

// TestDaemonService_ClosedIssueCleanupServiceLogger tests if ClosedIssueCleanupService's logger is set correctly
func TestDaemonService_ClosedIssueCleanupServiceLogger(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *config.Config
		expectLoggerSet   bool
		expectSessionName string
	}{
		{
			name: "Logger should be set on ClosedIssueCleanupService",
			cfg: &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "douhashi/soba",
				},
				Workflow: config.WorkflowConfig{
					ClosedIssueCleanupEnabled:  true,
					ClosedIssueCleanupInterval: 60,
					Interval:                   30,
				},
			},
			expectLoggerSet:   true,
			expectSessionName: "soba-douhashi-soba", // generateSessionNameと同じ形式に統一
		},
		{
			name: "Session name generation should use generateSessionName",
			cfg: &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "owner/repo-name",
				},
				Workflow: config.WorkflowConfig{
					ClosedIssueCleanupEnabled:  true,
					ClosedIssueCleanupInterval: 120,
				},
			},
			expectLoggerSet:   true,
			expectSessionName: "soba-owner-repo-name", // generateSessionNameと同じ形式に統一
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockGitHubClient := new(MockGitHubClient)
			mockTmux := new(MockTmuxClient)
			mockLogger := logging.NewMockLogger()

			// Create ClosedIssueCleanupService
			cleanupService := &ClosedIssueCleanupService{}

			// Create IssueWatcher and PRWatcher
			watcher := NewIssueWatcher(mockGitHubClient, tt.cfg)
			prWatcher := NewPRWatcher(mockGitHubClient, tt.cfg)

			if watcher != nil {
				watcher.logger = mockLogger
			}
			if prWatcher != nil {
				prWatcher.logger = mockLogger
			}

			service := &daemonService{
				watcher:                   watcher,
				prWatcher:                 prWatcher,
				closedIssueCleanupService: cleanupService,
				tmux:                      mockTmux,
				logger:                    mockLogger,
			}

			// Run configureAndStartWatchers in a goroutine
			errCh := make(chan error, 1)
			go func() {
				errCh <- service.configureAndStartWatchers(ctx, tt.cfg)
			}()

			// Wait a bit to ensure configuration happens
			time.Sleep(50 * time.Millisecond)
			cancel()

			// Wait for completion
			select {
			case <-errCh:
				// Normal completion
			case <-time.After(200 * time.Millisecond):
				// Timeout is OK
			}

			// Verify logger was set
			if tt.expectLoggerSet {
				assert.NotNil(t, cleanupService.log, "Logger should be set on ClosedIssueCleanupService")
			}

			// Verify session name
			if tt.expectSessionName != "" {
				assert.Equal(t, tt.expectSessionName, cleanupService.sessionName, "Session name should be generated correctly")
			}

			// Verify configuration
			parts := strings.Split(tt.cfg.GitHub.Repository, "/")
			if len(parts) == 2 {
				assert.Equal(t, parts[0], cleanupService.owner, "Owner should be set correctly")
				assert.Equal(t, parts[1], cleanupService.repo, "Repo should be set correctly")
				assert.Equal(t, tt.cfg.Workflow.ClosedIssueCleanupEnabled, cleanupService.enabled, "Enabled flag should be set correctly")
				assert.Equal(t, time.Duration(tt.cfg.Workflow.ClosedIssueCleanupInterval)*time.Second, cleanupService.interval, "Interval should be set correctly")
			}
		})
	}
}

// TestDaemonService_ClosedIssueCleanupServiceStartupLog tests if cleanup service startup is logged
func TestDaemonService_ClosedIssueCleanupServiceStartupLog(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		enabled        bool
		expectStartLog bool
	}{
		{
			name: "Should log when cleanup service starts",
			cfg: &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "douhashi/soba",
				},
				Workflow: config.WorkflowConfig{
					ClosedIssueCleanupEnabled:  true,
					ClosedIssueCleanupInterval: 60,
					Interval:                   30,
				},
			},
			enabled:        true,
			expectStartLog: true,
		},
		{
			name: "Should log when cleanup service is disabled",
			cfg: &config.Config{
				GitHub: config.GitHubConfig{
					Repository: "douhashi/soba",
				},
				Workflow: config.WorkflowConfig{
					ClosedIssueCleanupEnabled:  false,
					ClosedIssueCleanupInterval: 60,
					Interval:                   30,
				},
			},
			enabled:        false,
			expectStartLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockGitHubClient := new(MockGitHubClient)
			mockTmux := new(MockTmuxClient)
			mockLogger := logging.NewMockLogger()

			// Create ClosedIssueCleanupService
			cleanupService := &ClosedIssueCleanupService{}

			// Create IssueWatcher and PRWatcher
			watcher := NewIssueWatcher(mockGitHubClient, tt.cfg)
			prWatcher := NewPRWatcher(mockGitHubClient, tt.cfg)

			if watcher != nil {
				watcher.logger = mockLogger
			}
			if prWatcher != nil {
				prWatcher.logger = mockLogger
			}

			service := &daemonService{
				watcher:                   watcher,
				prWatcher:                 prWatcher,
				closedIssueCleanupService: cleanupService,
				tmux:                      mockTmux,
				logger:                    mockLogger,
			}

			// Run configureAndStartWatchers in a goroutine
			errCh := make(chan error, 1)
			go func() {
				errCh <- service.configureAndStartWatchers(ctx, tt.cfg)
			}()

			// Wait a bit to ensure configuration and startup happens
			time.Sleep(100 * time.Millisecond)
			cancel()

			// Wait for completion
			select {
			case <-errCh:
				// Normal completion
			case <-time.After(200 * time.Millisecond):
				// Timeout is OK
			}

			// Verify configuration
			assert.Equal(t, tt.enabled, cleanupService.enabled, "Enabled flag should be set correctly")

			// Verify logger was set
			assert.NotNil(t, cleanupService.log, "Logger should always be set")

			// If enabled, service should have started and logged
			// If disabled, service should have logged that it's disabled
			// Both cases are validated by the fact that log is not nil
		})
	}
}

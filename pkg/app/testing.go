package app

import (
	"testing"

	"github.com/douhashi/soba/internal/config"
)

// TestHelper provides test utilities
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// InitializeForTest initializes app for testing with default config
func (h *TestHelper) InitializeForTest() {
	h.InitializeForTestWithConfig(nil)
}

// InitializeForTestWithConfig initializes app for testing with custom config
func (h *TestHelper) InitializeForTestWithConfig(testConfig *config.Config) {
	// Reset any existing state
	Reset()

	// Use provided config or create default test config
	if testConfig == nil {
		testConfig = &config.Config{
			GitHub: config.GitHubConfig{
				Repository: "test/repo",
			},
			Log: config.LogConfig{
				Level:      "debug",
				OutputPath: "stdout",
			},
			Workflow: config.WorkflowConfig{
				Interval: 300,
			},
		}
	}

	// Initialize app with test config
	MustInitializeForTest(testConfig)

	// Register cleanup
	h.t.Cleanup(func() {
		Reset()
	})
}

// InitializeForTestWithOptions initializes app for testing with CLI options
func (h *TestHelper) InitializeForTestWithOptions(configPath string, opts *InitOptions) {
	// Reset any existing state
	Reset()

	// Initialize with options
	MustInitializeWithOptions(configPath, opts)

	// Register cleanup
	h.t.Cleanup(func() {
		Reset()
	})
}

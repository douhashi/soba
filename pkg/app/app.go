package app

import (
	"sync"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/slack"
	"github.com/douhashi/soba/pkg/logging"
)

var (
	cfg        *config.Config
	logFactory *logging.Factory
	mu         sync.RWMutex

	initialized bool
)

// InitOptions holds initialization options
type InitOptions struct {
	LogLevel string
	Verbose  bool
}

// MustInitialize initializes the application (panics on failure)
func MustInitialize(configPath string) {
	MustInitializeWithOptions(configPath, nil)
}

// MustInitializeWithOptions initializes the application with CLI options
func MustInitializeWithOptions(configPath string, opts *InitOptions) {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		panic("app already initialized")
	}

	// Load config
	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Determine effective log level (CLI > verbose > config > default)
	logLevel := cfg.Log.Level
	if opts != nil {
		if opts.LogLevel != "" {
			logLevel = opts.LogLevel
		} else if opts.Verbose {
			logLevel = "debug"
		}
	}
	if logLevel == "" {
		logLevel = "warn" // Default
	}

	// Create Logger Factory
	logFactory, err = logging.NewFactory(logging.Config{
		Level:        logLevel,
		Format:       cfg.Log.Format,
		Output:       cfg.Log.OutputPath,
		AlsoToStdout: true,
	})
	if err != nil {
		panic("failed to create logger factory: " + err.Error())
	}

	// Initialize Slack Manager
	logger := logFactory.CreateComponentLogger("slack")
	slack.Initialize(cfg, logger)

	initialized = true
}

// MustInitializeForTest initializes the application for testing
func MustInitializeForTest(testConfig *config.Config) {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		panic("app already initialized")
	}

	cfg = testConfig

	// Create Logger Factory for test
	var err error
	logFactory, err = logging.NewFactory(logging.Config{
		Level:        cfg.Log.Level,
		Format:       cfg.Log.Format,
		Output:       "stdout",
		AlsoToStdout: false,
	})
	if err != nil {
		panic("failed to create logger factory: " + err.Error())
	}

	// Initialize Slack Manager for test
	logger := logFactory.CreateComponentLogger("slack")
	slack.Initialize(cfg, logger)

	initialized = true
}

// Config returns the global Config
func Config() *config.Config {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized {
		panic("app not initialized")
	}
	return cfg
}

// LogFactory returns the global Logger Factory
func LogFactory() *logging.Factory {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized {
		panic("app not initialized")
	}
	return logFactory
}

// UpdateLogLevel updates the log level at runtime
func UpdateLogLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	if !initialized {
		panic("app not initialized")
	}

	// Create new Factory with updated level
	newFactory, err := logging.NewFactory(logging.Config{
		Level:        level,
		Format:       cfg.Log.Format,
		Output:       cfg.Log.OutputPath,
		AlsoToStdout: true,
	})
	if err != nil {
		panic("failed to update logger: " + err.Error())
	}

	logFactory = newFactory
}

// Reset resets the application state (for testing)
func Reset() {
	mu.Lock()
	defer mu.Unlock()

	cfg = nil
	logFactory = nil
	initialized = false
	// Reset Slack Manager singleton
	slack.Reset()
}

// IsInitialized returns whether the app is initialized
func IsInitialized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return initialized
}

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

func TestLogLevelPriority(t *testing.T) {
	tests := []struct {
		name           string
		cliLogLevel    string
		configLogLevel string
		envLogLevel    string
		verbose        bool
		expectedLevel  string
	}{
		{
			name:           "CLI flag takes highest priority",
			cliLogLevel:    "debug",
			configLogLevel: "info",
			envLogLevel:    "warn",
			verbose:        false,
			expectedLevel:  "debug",
		},
		{
			name:           "Verbose flag used when no CLI log level",
			cliLogLevel:    "",
			configLogLevel: "info",
			envLogLevel:    "warn",
			verbose:        true,
			expectedLevel:  "debug",
		},
		{
			name:           "Config file used when no CLI flags",
			cliLogLevel:    "",
			configLogLevel: "info",
			envLogLevel:    "warn",
			verbose:        false,
			expectedLevel:  "info",
		},
		{
			name:           "Environment variable used when no config",
			cliLogLevel:    "",
			configLogLevel: "",
			envLogLevel:    "error",
			verbose:        false,
			expectedLevel:  "error",
		},
		{
			name:           "Default to warn when nothing specified",
			cliLogLevel:    "",
			configLogLevel: "",
			envLogLevel:    "",
			verbose:        false,
			expectedLevel:  "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Set environment variable
			if tt.envLogLevel != "" {
				t.Setenv("LOG_LEVEL", tt.envLogLevel)
			}

			// Create daemon service with mock dependencies
			logFactory, err := logging.NewFactory(logging.Config{
				Level:  tt.expectedLevel,
				Format: "text",
			})
			require.NoError(t, err)

			service := &daemonService{
				workDir: tmpDir,
				logger:  logFactory.CreateComponentLogger("test"),
			}

			// Test initializeLogging with config
			cfg := &config.Config{
				Log: config.LogConfig{
					Level:      tt.configLogLevel,
					OutputPath: filepath.Join(tmpDir, "test.log"),
				},
			}

			// Initialize logging
			_, err = service.initializeLoggingWithCLILevel(cfg, false, tt.cliLogLevel, tt.verbose)
			assert.NoError(t, err)

			// Verify the effective log level
			effectiveLevel := service.getEffectiveLogLevel(cfg, tt.cliLogLevel, tt.verbose)
			assert.Equal(t, tt.expectedLevel, effectiveLevel)
		})
	}
}

func TestDaemonModeLogLevelPropagation(t *testing.T) {
	tmpDir := t.TempDir()
	sobaDir := filepath.Join(tmpDir, ".soba")
	require.NoError(t, os.MkdirAll(sobaDir, 0755))

	tests := []struct {
		name        string
		cliLogLevel string
		verbose     bool
		expectedLog string
	}{
		{
			name:        "CLI log level propagated to daemon",
			cliLogLevel: "debug",
			verbose:     false,
			expectedLog: "debug",
		},
		{
			name:        "Verbose flag propagated to daemon",
			cliLogLevel: "",
			verbose:     true,
			expectedLog: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create log factory with expected level
			logFactory, err := logging.NewFactory(logging.Config{
				Level:  tt.expectedLog,
				Format: "text",
			})
			require.NoError(t, err)

			// Create service with CLI level
			service := &daemonService{
				workDir:     tmpDir,
				logger:      logFactory.CreateComponentLogger("test"),
				cliLogLevel: tt.cliLogLevel,
				verbose:     tt.verbose,
			}

			cfg := &config.Config{
				Log: config.LogConfig{
					OutputPath: filepath.Join(tmpDir, "daemon.log"),
				},
			}

			// Test that daemon mode uses CLI level
			ctx := context.Background()
			effectiveLevel := service.getEffectiveLogLevel(cfg, tt.cliLogLevel, tt.verbose)
			service.logger.Debug(ctx, "Testing log level",
				logging.Field{Key: "effective", Value: effectiveLevel})

			assert.Equal(t, tt.expectedLog, effectiveLevel)
		})
	}
}

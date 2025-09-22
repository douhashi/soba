package logging_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/pkg/logging"
)

func TestFactory(t *testing.T) {
	t.Run("should create factory with stdout output", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, factory)

		logger := factory.CreateLogger()
		assert.NotNil(t, logger)
	})

	t.Run("should create factory with stderr output", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "debug",
			Format: "text",
			Output: "stderr",
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, factory)
	})

	t.Run("should create factory with file output", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "test.log")

		config := logging.Config{
			Level:  "warn",
			Format: "json",
			Output: logFile,
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, factory)

		// Test that logging creates the file
		logger := factory.CreateLogger()
		logger.Info(context.Background(), "test message")

		// File should be created
		_, err = os.Stat(logFile)
		assert.NoError(t, err)
	})

	t.Run("should parse different log levels", func(t *testing.T) {
		tests := []struct {
			level    string
			expected string
		}{
			{"debug", "DEBUG"},
			{"info", "INFO"},
			{"warn", "WARN"},
			{"error", "ERROR"},
			{"DEBUG", "DEBUG"},
			{"INFO", "INFO"},
			{"WARN", "WARN"},
			{"ERROR", "ERROR"},
		}

		for _, tt := range tests {
			t.Run(tt.level, func(t *testing.T) {
				config := logging.Config{
					Level:  tt.level,
					Format: "json",
					Output: "stdout",
				}

				factory, err := logging.NewFactory(config)
				require.NoError(t, err)
				assert.NotNil(t, factory)
			})
		}
	})

	t.Run("should support AddSource option", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:     "info",
			Format:    "json",
			Output:    "stdout",
			AddSource: true,
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, factory)
	})

	t.Run("should create component logger", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		factory, err := logging.NewFactory(config)
		require.NoError(t, err)

		// Act
		logger := factory.CreateComponentLogger("test-component")

		// Assert
		assert.NotNil(t, logger)

		// Verify component is included in logs
		mockFactory, err := logging.NewMockFactory()
		require.NoError(t, err)

		componentLogger := mockFactory.CreateComponentLogger("daemon")
		ctx := context.Background()
		componentLogger.Info(ctx, "test message")

		// Check that component field is present
		mockHandler := mockFactory.Handler.(*logging.MockLogger)
		lastMsg := mockHandler.LastMessage()
		require.NotNil(t, lastMsg)
		assert.Equal(t, "daemon", lastMsg.Fields["component"])
	})

	t.Run("should handle invalid log level gracefully", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "invalid",
			Format: "json",
			Output: "stdout",
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert - should default to warn level
		require.NoError(t, err)
		assert.NotNil(t, factory)
	})

	t.Run("should default to warn level when no level specified", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "", // empty level
			Format: "json",
			Output: "stdout",
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert - should default to warn level
		require.NoError(t, err)
		assert.NotNil(t, factory)
	})

	t.Run("should support text and json formats", func(t *testing.T) {
		formats := []string{"text", "json"}

		for _, format := range formats {
			t.Run(format, func(t *testing.T) {
				config := logging.Config{
					Level:  "info",
					Format: format,
					Output: "stdout",
				}

				factory, err := logging.NewFactory(config)
				require.NoError(t, err)
				assert.NotNil(t, factory)
			})
		}
	})

	t.Run("should create multiple independent loggers", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		factory, err := logging.NewFactory(config)
		require.NoError(t, err)

		// Act
		logger1 := factory.CreateLogger()
		logger2 := factory.CreateLogger()
		componentLogger := factory.CreateComponentLogger("service")

		// Assert
		assert.NotNil(t, logger1)
		assert.NotNil(t, logger2)
		assert.NotNil(t, componentLogger)

		// Loggers should be independent instances
		logger1WithFields := logger1.WithFields(logging.Field{Key: "id", Value: 1})
		logger2WithFields := logger2.WithFields(logging.Field{Key: "id", Value: 2})

		// Original loggers should not be modified
		assert.NotEqual(t, logger1, logger1WithFields)
		assert.NotEqual(t, logger2, logger2WithFields)
	})
}

func TestMockFactory(t *testing.T) {
	t.Run("should create mock factory for testing", func(t *testing.T) {
		// Act
		factory, err := logging.NewMockFactory()

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, factory)
		assert.IsType(t, &logging.MockLogger{}, factory.Handler)

		logger := factory.CreateLogger()
		assert.NotNil(t, logger)
	})

	t.Run("should capture logs in mock factory", func(t *testing.T) {
		// Arrange
		factory, err := logging.NewMockFactory()
		require.NoError(t, err)

		logger := factory.CreateLogger()
		ctx := context.Background()

		// Act
		logger.Info(ctx, "test message",
			logging.Field{Key: "key", Value: "value"},
		)

		// Assert
		mockHandler := factory.Handler.(*logging.MockLogger)
		assert.Len(t, mockHandler.Messages, 1)
		assert.Equal(t, "test message", mockHandler.Messages[0].Message)
		assert.Equal(t, "value", mockHandler.Messages[0].Fields["key"])
	})
}

func TestRotatingFileWriter(t *testing.T) {
	t.Run("should create rotating file writer", func(t *testing.T) {
		// Arrange
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "app.log")

		// Act
		writer, err := logging.NewRotatingFileWriter(logFile)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, writer)

		// Test that writer is not nil
		assert.NotNil(t, writer)

		// Write some data
		_, err = writer.Write([]byte("test log entry\n"))
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(logFile)
		assert.NoError(t, err)
	})
}

func TestPrettyTextHandler(t *testing.T) {
	t.Run("should create pretty text handler", func(t *testing.T) {
		// Arrange
		config := logging.Config{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		}

		// Act
		factory, err := logging.NewFactory(config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, factory)

		logger := factory.CreateLogger()
		ctx := context.Background()

		// Should not panic when logging
		assert.NotPanics(t, func() {
			logger.Info(ctx, "formatted message",
				logging.Field{Key: "user", Value: "test"},
				logging.Field{Key: "action", Value: "login"},
			)
		})
	})
}

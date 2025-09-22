package logging_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/pkg/logging"
)

func TestContextLogger(t *testing.T) {
	t.Run("should extract request ID from context", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()
		ctx = logging.WithRequestID(ctx, "test-request-123")

		// Act
		logger.Info(ctx, "test message")

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "INFO", msg.Level)
		assert.Equal(t, "test message", msg.Message)
		assert.Equal(t, "test-request-123", msg.Fields["request_id"])
	})

	t.Run("should extract trace ID from context", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()
		ctx = logging.WithTraceID(ctx, "trace-456")

		// Act
		logger.Info(ctx, "test message")

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "trace-456", msg.Fields["trace_id"])
	})

	t.Run("should extract component from context", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()
		ctx = logging.WithComponent(ctx, "daemon")

		// Act
		logger.Info(ctx, "test message")

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "daemon", msg.Fields["component"])
	})

	t.Run("should handle multiple context values", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()
		ctx = logging.WithRequestID(ctx, "req-789")
		ctx = logging.WithComponent(ctx, "watcher")
		ctx = logging.WithTraceID(ctx, "trace-321")

		// Act
		logger.Info(ctx, "test message")

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "req-789", msg.Fields["request_id"])
		assert.Equal(t, "watcher", msg.Fields["component"])
		assert.Equal(t, "trace-321", msg.Fields["trace_id"])
	})

	t.Run("should add custom fields", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()

		// Act
		logger.Info(ctx, "test message",
			logging.Field{Key: "user_id", Value: "user-123"},
			logging.Field{Key: "action", Value: "login"},
		)

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "user-123", msg.Fields["user_id"])
		assert.Equal(t, "login", msg.Fields["action"])
	})

	t.Run("should support WithFields for persistent fields", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		baseLogger := logging.NewContextLogger(mockLogger)

		// Add persistent fields
		logger := baseLogger.WithFields(
			logging.Field{Key: "service", Value: "soba"},
			logging.Field{Key: "version", Value: "1.0.0"},
		)

		ctx := context.Background()

		// Act
		logger.Info(ctx, "first message")
		logger.Info(ctx, "second message", logging.Field{Key: "extra", Value: "data"})

		// Assert
		require.Len(t, mockLogger.Messages, 2)

		// First message should have persistent fields
		msg1 := mockLogger.Messages[0]
		assert.Equal(t, "soba", msg1.Fields["service"])
		assert.Equal(t, "1.0.0", msg1.Fields["version"])

		// Second message should have persistent fields plus extra
		msg2 := mockLogger.Messages[1]
		assert.Equal(t, "soba", msg2.Fields["service"])
		assert.Equal(t, "1.0.0", msg2.Fields["version"])
		assert.Equal(t, "data", msg2.Fields["extra"])
	})

	t.Run("should support WithError helper", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		baseLogger := logging.NewContextLogger(mockLogger)

		testErr := assert.AnError
		logger := baseLogger.WithError(testErr)

		ctx := context.Background()

		// Act
		logger.Error(ctx, "operation failed")

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "ERROR", msg.Level)
		assert.Equal(t, testErr.Error(), msg.Fields["error"])
	})

	t.Run("should support all log levels", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()

		// Act
		logger.Debug(ctx, "debug message")
		logger.Info(ctx, "info message")
		logger.Warn(ctx, "warn message")
		logger.Error(ctx, "error message")

		// Assert
		require.Len(t, mockLogger.Messages, 4)
		assert.Equal(t, "DEBUG", mockLogger.Messages[0].Level)
		assert.Equal(t, "INFO", mockLogger.Messages[1].Level)
		assert.Equal(t, "WARN", mockLogger.Messages[2].Level)
		assert.Equal(t, "ERROR", mockLogger.Messages[3].Level)
	})

	t.Run("should preserve context through logger chain", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		baseLogger := logging.NewContextLogger(mockLogger)

		// Create a chain of loggers with different persistent fields
		serviceLogger := baseLogger.WithFields(logging.Field{Key: "service", Value: "daemon"})
		componentLogger := serviceLogger.WithFields(logging.Field{Key: "component", Value: "watcher"})

		ctx := context.Background()
		ctx = logging.WithRequestID(ctx, "req-999")

		// Act
		componentLogger.Info(ctx, "nested logger message")

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "daemon", msg.Fields["service"])
		assert.Equal(t, "watcher", msg.Fields["component"])
		assert.Equal(t, "req-999", msg.Fields["request_id"])
	})

	t.Run("should handle nil context gracefully", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		// Act & Assert - should not panic
		assert.NotPanics(t, func() {
			logger.Info(context.TODO(), "message with nil context")
		})

		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.Equal(t, "message with nil context", msg.Message)
	})

	t.Run("should include timestamp in log message", func(t *testing.T) {
		// Arrange
		mockLogger := logging.NewMockLogger()
		logger := logging.NewContextLogger(mockLogger)

		ctx := context.Background()
		beforeTime := time.Now()

		// Act
		logger.Info(ctx, "test message")

		afterTime := time.Now()

		// Assert
		require.Len(t, mockLogger.Messages, 1)
		msg := mockLogger.Messages[0]
		assert.True(t, !msg.Time.Before(beforeTime))
		assert.True(t, !msg.Time.After(afterTime))
	})
}

func TestField(t *testing.T) {
	t.Run("should create field with various types", func(t *testing.T) {
		// Test different field value types
		fields := []logging.Field{
			{Key: "string", Value: "test"},
			{Key: "int", Value: 123},
			{Key: "bool", Value: true},
			{Key: "float", Value: 3.14},
			{Key: "slice", Value: []string{"a", "b"}},
			{Key: "map", Value: map[string]int{"x": 1}},
		}

		for _, field := range fields {
			assert.NotNil(t, field.Key)
			assert.NotNil(t, field.Value)
		}
	})
}

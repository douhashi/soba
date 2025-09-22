// Demo application for new logging system
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/douhashi/soba/pkg/logging"
)

func main() {
	ctx := context.Background()
	ctx = logging.WithComponent(ctx, "demo")
	ctx = logging.WithRequestID(ctx, "demo-123")

	// Create logging factory
	logConfig := logging.Config{
		Level:     "debug",
		Format:    "text",
		Output:    "stderr",
		AddSource: false,
	}

	logFactory, err := logging.NewFactory(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create main logger
	logger := logFactory.CreateLogger()

	// Test various log levels
	logger.Debug(ctx, "Debug message",
		logging.Field{Key: "debug_value", Value: 123},
	)

	logger.Info(ctx, "Info message",
		logging.Field{Key: "user", Value: "test_user"},
		logging.Field{Key: "action", Value: "login"},
	)

	logger.Warn(ctx, "Warning message",
		logging.Field{Key: "warning_type", Value: "deprecation"},
	)

	logger.Error(ctx, "Error message",
		logging.Field{Key: "error_code", Value: "E1001"},
	)

	// Test component logger
	serviceLogger := logFactory.CreateComponentLogger("service")
	serviceLogger.Info(ctx, "Service initialized")

	// Test WithFields
	dbLogger := serviceLogger.WithFields(
		logging.Field{Key: "database", Value: "postgres"},
		logging.Field{Key: "pool_size", Value: 10},
	)
	dbLogger.Info(ctx, "Database connection established")

	// Test WithError
	testErr := fmt.Errorf("connection timeout")
	errorLogger := logger.WithError(testErr)
	errorLogger.Error(ctx, "Operation failed")

	// Test JSON output
	jsonConfig := logging.Config{
		Level:  "info",
		Format: "json",
		Output: "stderr",
	}

	jsonFactory, _ := logging.NewFactory(jsonConfig)
	jsonLogger := jsonFactory.CreateLogger()

	fmt.Println("\n--- JSON Output ---")
	jsonLogger.Info(ctx, "JSON formatted message",
		logging.Field{Key: "format", Value: "json"},
		logging.Field{Key: "structured", Value: true},
	)

	fmt.Println("\nDemo completed successfully!")
}

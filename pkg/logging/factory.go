package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config represents logger configuration
type Config struct {
	Level        string
	Format       string // "json" or "text"
	Output       string // "stdout", "stderr", or file path
	AddSource    bool
	TimeFormat   string
	AlsoToStdout bool // Also output to stdout when Output is a file path
}

// Factory creates logger instances with consistent configuration
type Factory struct {
	config  Config
	Handler slog.Handler // Exposed for testing
}

// NewFactory creates a new logger factory
func NewFactory(cfg Config) (*Factory, error) {
	var writer io.Writer

	switch cfg.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// Create file writer with rotation
		fw, err := NewRotatingFileWriter(cfg.Output)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %w", err)
		}

		// If AlsoToStdout is true, create a MultiWriter that writes to both file and stdout
		if cfg.AlsoToStdout {
			writer = MultiWriter(fw, os.Stdout)
		} else {
			writer = fw
		}
	}

	level := parseLevel(cfg.Level)

	// Create a LevelVar to properly set the minimum log level
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	opts := &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	default:
		// Use pretty text handler for development
		handler = NewPrettyTextHandler(writer, opts)
	}

	return &Factory{
		config:  cfg,
		Handler: handler,
	}, nil
}

// CreateLogger creates a new logger instance
func (f *Factory) CreateLogger() Logger {
	return NewContextLogger(f.Handler)
}

// CreateComponentLogger creates a logger with a component field
func (f *Factory) CreateComponentLogger(component string) Logger {
	logger := NewContextLogger(f.Handler)
	return logger.WithFields(Field{Key: "component", Value: component})
}

// parseLevel parses log level string to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn // Default to warn
	}
}

// NewMockFactory creates a factory with a mock handler for testing
func NewMockFactory() (*Factory, error) {
	mockHandler := NewMockLogger()
	return &Factory{
		config: Config{
			Level:  "debug",
			Format: "json",
			Output: "mock",
		},
		Handler: mockHandler,
	}, nil
}

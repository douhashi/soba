package logging

import (
	"context"
	"log/slog"
	"time"
)

// Logger is our main logging interface
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)

	// Structured logging helpers
	WithFields(fields ...Field) Logger
	WithError(err error) Logger
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value any
}

// contextLogger implements Logger interface with context awareness
type contextLogger struct {
	handler slog.Handler
	fields  []Field
}

// NewContextLogger creates a new context-aware logger
func NewContextLogger(handler slog.Handler) Logger {
	return &contextLogger{
		handler: handler,
		fields:  nil,
	}
}

// Debug logs a debug message
func (l *contextLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *contextLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *contextLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *contextLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelError, msg, fields...)
}

// WithFields returns a new logger with additional persistent fields
func (l *contextLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &contextLogger{
		handler: l.handler,
		fields:  newFields,
	}
}

// WithError returns a new logger with an error field
func (l *contextLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.WithFields(Field{Key: "error", Value: err.Error()})
}

// log performs the actual logging
func (l *contextLogger) log(ctx context.Context, level slog.Level, msg string, fields ...Field) {
	// Ensure context is not nil
	if ctx == nil {
		ctx = context.Background()
	}

	// Check if this level is enabled before processing
	if !l.handler.Enabled(ctx, level) {
		return
	}

	// Extract context values
	attrs := l.extractContextAttributes(ctx)

	// Add persistent fields
	for _, f := range l.fields {
		attrs = append(attrs, slog.Any(f.Key, f.Value))
	}

	// Add call-site fields
	for _, f := range fields {
		attrs = append(attrs, slog.Any(f.Key, f.Value))
	}

	// Create and handle the log record
	record := slog.NewRecord(time.Now(), level, msg, 0)
	record.AddAttrs(attrs...)

	// Handle the record
	_ = l.handler.Handle(ctx, record)
}

// extractContextAttributes extracts standard context values
func (l *contextLogger) extractContextAttributes(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	if requestID, ok := extractRequestID(ctx); ok && requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	if traceID, ok := extractTraceID(ctx); ok && traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}

	if component, ok := extractComponent(ctx); ok && component != "" {
		attrs = append(attrs, slog.String("component", component))
	}

	return attrs
}

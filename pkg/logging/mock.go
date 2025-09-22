package logging

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// MockLogger is a logger implementation for testing
type MockLogger struct {
	mu       sync.Mutex
	Messages []LogMessage
}

// LogMessage represents a captured log message
type LogMessage struct {
	Time    time.Time
	Level   string
	Message string
	Fields  map[string]any
	Context context.Context
}

// NewMockLogger creates a new mock logger for testing
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: make([]LogMessage, 0),
	}
}

// Handle implements slog.Handler interface for MockLogger
func (m *MockLogger) Handle(ctx context.Context, record slog.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fields := make(map[string]any)

	// Extract attributes from the record
	record.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = attr.Value.Any()
		return true
	})

	// Extract context values if they're not already in fields
	if requestID, ok := extractRequestID(ctx); ok && requestID != "" {
		if _, exists := fields["request_id"]; !exists {
			fields["request_id"] = requestID
		}
	}

	if traceID, ok := extractTraceID(ctx); ok && traceID != "" {
		if _, exists := fields["trace_id"]; !exists {
			fields["trace_id"] = traceID
		}
	}

	if component, ok := extractComponent(ctx); ok && component != "" {
		if _, exists := fields["component"]; !exists {
			fields["component"] = component
		}
	}

	levelStr := "INFO"
	switch record.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	}

	m.Messages = append(m.Messages, LogMessage{
		Time:    record.Time,
		Level:   levelStr,
		Message: record.Message,
		Fields:  fields,
		Context: ctx,
	})

	return nil
}

// Enabled implements slog.Handler interface
func (m *MockLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// WithAttrs implements slog.Handler interface
func (m *MockLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, return the same handler
	// In a real implementation, we would store attrs for later use
	return m
}

// WithGroup implements slog.Handler interface
func (m *MockLogger) WithGroup(name string) slog.Handler {
	// For simplicity, return the same handler
	// In a real implementation, we would handle groups
	return m
}

// Clear clears all captured messages
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = make([]LogMessage, 0)
}

// LastMessage returns the last logged message, or nil if none
func (m *MockLogger) LastMessage() *LogMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Messages) == 0 {
		return nil
	}
	msg := m.Messages[len(m.Messages)-1]
	return &msg
}

// HasMessage checks if a message with the given text was logged
func (m *MockLogger) HasMessage(text string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.Messages {
		if msg.Message == text {
			return true
		}
	}
	return false
}

// CountLevel returns the number of messages logged at the given level
func (m *MockLogger) CountLevel(level string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, msg := range m.Messages {
		if msg.Level == level {
			count++
		}
	}
	return count
}

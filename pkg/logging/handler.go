package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// PrettyTextHandler implements a human-readable text handler
type PrettyTextHandler struct {
	writer io.Writer
	opts   *slog.HandlerOptions
	attrs  []slog.Attr
	groups []string
}

// NewPrettyTextHandler creates a new pretty text handler
func NewPrettyTextHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &PrettyTextHandler{
		writer: w,
		opts:   opts,
	}
}

// Enabled implements slog.Handler
func (h *PrettyTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle implements slog.Handler
func (h *PrettyTextHandler) Handle(ctx context.Context, record slog.Record) error {
	// Format timestamp
	timestamp := record.Time.Format("2006-01-02 15:04:05.000")

	// Format level with color
	levelStr := h.formatLevel(record.Level)

	// Build message
	var sb strings.Builder
	sb.WriteString(timestamp)
	sb.WriteString(" ")
	sb.WriteString(levelStr)
	sb.WriteString(" ")
	sb.WriteString(record.Message)

	// Add stored attributes
	for _, attr := range h.attrs {
		sb.WriteString(" ")
		sb.WriteString(h.formatAttr(attr))
	}

	// Add record attributes
	record.Attrs(func(attr slog.Attr) bool {
		sb.WriteString(" ")
		sb.WriteString(h.formatAttr(attr))
		return true
	})

	// Add source location if configured
	if h.opts.AddSource && record.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		if f.File != "" {
			sb.WriteString(fmt.Sprintf(" [%s:%d]", f.File, f.Line))
		}
	}

	sb.WriteString("\n")

	_, err := h.writer.Write([]byte(sb.String()))
	return err
}

// WithAttrs implements slog.Handler
func (h *PrettyTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &PrettyTextHandler{
		writer: h.writer,
		opts:   h.opts,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup implements slog.Handler
func (h *PrettyTextHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &PrettyTextHandler{
		writer: h.writer,
		opts:   h.opts,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// formatLevel formats the log level with color for terminal output
func (h *PrettyTextHandler) formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "[DEBUG]"
	case slog.LevelInfo:
		return "[INFO ]"
	case slog.LevelWarn:
		return "[WARN ]"
	case slog.LevelError:
		return "[ERROR]"
	default:
		return fmt.Sprintf("[%5s]", level.String())
	}
}

// formatAttr formats an attribute for display
func (h *PrettyTextHandler) formatAttr(attr slog.Attr) string {
	// Handle special formatting for certain keys
	switch attr.Key {
	case "error":
		return fmt.Sprintf("error=\"%v\"", attr.Value.Any())
	case "time", "duration":
		if d, ok := attr.Value.Any().(time.Duration); ok {
			return fmt.Sprintf("%s=%v", attr.Key, d)
		}
	}

	// Default formatting
	value := attr.Value.Any()
	switch v := value.(type) {
	case string:
		// Quote strings if they contain spaces
		if strings.Contains(v, " ") {
			return fmt.Sprintf("%s=%q", attr.Key, v)
		}
		return fmt.Sprintf("%s=%s", attr.Key, v)
	case nil:
		return fmt.Sprintf("%s=<nil>", attr.Key)
	default:
		return fmt.Sprintf("%s=%v", attr.Key, v)
	}
}

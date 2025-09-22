package logging

import (
	"context"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
	componentKey contextKey = "component"
)

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithComponent adds component name to context
func WithComponent(ctx context.Context, component string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, componentKey, component)
}

// extractRequestID extracts request ID from context
func extractRequestID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

// extractTraceID extracts trace ID from context
func extractTraceID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok
}

// extractComponent extracts component from context
func extractComponent(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	component, ok := ctx.Value(componentKey).(string)
	return component, ok
}

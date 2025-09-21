package logger

import "log/slog"

// Logger はロギングインターフェース
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// slogLogger はslog.Loggerをラップする実装
type slogLogger struct {
	logger *slog.Logger
}

// NewLogger は新しいLoggerを作成する
func NewLogger(l *slog.Logger) Logger {
	if l == nil {
		l = GetLogger()
	}
	return &slogLogger{logger: l}
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{logger: l.logger.With(args...)}
}

// nopLogger は何もしないロガー
type nopLogger struct{}

// NewNopLogger は何もしないロガーを作成する
func NewNopLogger() Logger {
	return &nopLogger{}
}

func (n *nopLogger) Debug(msg string, args ...any) {}
func (n *nopLogger) Info(msg string, args ...any)  {}
func (n *nopLogger) Warn(msg string, args ...any)  {}
func (n *nopLogger) Error(msg string, args ...any) {}
func (n *nopLogger) With(args ...any) Logger       { return n }

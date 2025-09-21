package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	globalLogger  *slog.Logger
	mu            sync.RWMutex
	currentEnv    string
	currentOutput io.Writer
)

type Config struct {
	Environment string
	Level       slog.Level
	Output      io.Writer
}

type Fields map[string]interface{}

type contextKey int

const loggerKey contextKey = iota

func Init(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	initLogger(cfg)
}

// initLogger は内部用で、ロックが既に取得されていることを前提とする
func initLogger(cfg Config) {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: true,
	}

	currentEnv = cfg.Environment
	currentOutput = cfg.Output

	switch cfg.Environment {
	case "production":
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	globalLogger = slog.New(handler)
}

func InitFromEnv(output io.Writer) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	levelStr := os.Getenv("LOG_LEVEL")
	level := ParseLevel(levelStr)

	Init(Config{
		Environment: env,
		Level:       level,
		Output:      output,
	})
}

func GetLogger() *slog.Logger {
	mu.RLock()
	if globalLogger != nil {
		defer mu.RUnlock()
		return globalLogger
	}
	mu.RUnlock()

	// 読み取りロックを解放してから書き込みロックを取得
	mu.Lock()
	defer mu.Unlock()

	// ダブルチェック：他のgoroutineが初期化済みかもしれない
	if globalLogger == nil {
		initLogger(Config{})
	}
	return globalLogger
}

func SetLevel(level slog.Level) {
	mu.Lock()
	defer mu.Unlock()

	if globalLogger == nil {
		initLogger(Config{Level: level})
		return
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	if currentOutput == nil {
		currentOutput = os.Stdout
	}

	switch currentEnv {
	case "production":
		handler = slog.NewJSONHandler(currentOutput, opts)
	default:
		handler = slog.NewTextHandler(currentOutput, opts)
	}

	globalLogger = slog.New(handler)
}

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return GetLogger()
}

func WithFields(logger *slog.Logger, fields Fields) *slog.Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return logger.With(args...)
}

func WithError(logger *slog.Logger, err error) *slog.Logger {
	if err == nil {
		return logger
	}
	return logger.With("error", err.Error())
}

func ParseLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

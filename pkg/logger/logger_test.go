package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("development mode", func(t *testing.T) {
		buf := &bytes.Buffer{}

		Init(Config{
			Environment: "development",
			Level:       slog.LevelDebug,
			Output:      buf,
		})

		logger := GetLogger()
		assert.NotNil(t, logger)

		logger.Debug("test message", "key", "value")
		output := buf.String()
		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "key=value")
		assert.Contains(t, output, "DEBUG")
	})

	t.Run("production mode", func(t *testing.T) {
		buf := &bytes.Buffer{}

		Init(Config{
			Environment: "production",
			Level:       slog.LevelInfo,
			Output:      buf,
		})

		logger := GetLogger()
		assert.NotNil(t, logger)

		logger.Info("test message", "key", "value")
		output := buf.String()

		var jsonOutput map[string]interface{}
		err := json.Unmarshal([]byte(output), &jsonOutput)
		require.NoError(t, err)
		assert.Equal(t, "test message", jsonOutput["msg"])
		assert.Equal(t, "value", jsonOutput["key"])
	})

	t.Run("default configuration", func(t *testing.T) {
		buf := &bytes.Buffer{}

		Init(Config{
			Output: buf,
		})

		logger := GetLogger()
		assert.NotNil(t, logger)

		logger.Info("test")
		output := buf.String()
		assert.Contains(t, output, "test")
	})
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		logFunc   func(*slog.Logger)
		shouldLog bool
	}{
		{
			name:  "debug level allows debug",
			level: slog.LevelDebug,
			logFunc: func(l *slog.Logger) {
				l.Debug("debug message")
			},
			shouldLog: true,
		},
		{
			name:  "info level blocks debug",
			level: slog.LevelInfo,
			logFunc: func(l *slog.Logger) {
				l.Debug("debug message")
			},
			shouldLog: false,
		},
		{
			name:  "info level allows info",
			level: slog.LevelInfo,
			logFunc: func(l *slog.Logger) {
				l.Info("info message")
			},
			shouldLog: true,
		},
		{
			name:  "warn level allows warn",
			level: slog.LevelWarn,
			logFunc: func(l *slog.Logger) {
				l.Warn("warn message")
			},
			shouldLog: true,
		},
		{
			name:  "error level allows error",
			level: slog.LevelError,
			logFunc: func(l *slog.Logger) {
				l.Error("error message")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}

			Init(Config{
				Environment: "development",
				Level:       tt.level,
				Output:      buf,
			})

			logger := GetLogger()
			tt.logFunc(logger)

			if tt.shouldLog {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	t.Run("add logger to context", func(t *testing.T) {
		buf := &bytes.Buffer{}
		Init(Config{
			Environment: "development",
			Output:      buf,
		})

		ctx := context.Background()
		logger := GetLogger().With("request_id", "123")

		ctx = WithContext(ctx, logger)

		retrievedLogger := FromContext(ctx)
		assert.NotNil(t, retrievedLogger)

		retrievedLogger.Info("test message")
		output := buf.String()
		assert.Contains(t, output, "request_id=123")
		assert.Contains(t, output, "test message")
	})

	t.Run("fallback to default logger", func(t *testing.T) {
		buf := &bytes.Buffer{}
		Init(Config{
			Environment: "development",
			Output:      buf,
		})

		ctx := context.Background()
		logger := FromContext(ctx)

		assert.NotNil(t, logger)
		logger.Info("fallback test")
		assert.Contains(t, buf.String(), "fallback test")
	})
}

func TestWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Config{
		Environment: "production",
		Output:      buf,
	})

	logger := GetLogger()
	fieldLogger := WithFields(logger, Fields{
		"user_id":    "456",
		"session_id": "789",
		"action":     "login",
	})

	fieldLogger.Info("user logged in")

	var jsonOutput map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &jsonOutput)
	require.NoError(t, err)

	assert.Equal(t, "456", jsonOutput["user_id"])
	assert.Equal(t, "789", jsonOutput["session_id"])
	assert.Equal(t, "login", jsonOutput["action"])
	assert.Equal(t, "user logged in", jsonOutput["msg"])
}

func TestWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Config{
		Environment: "production",
		Output:      buf,
	})

	logger := GetLogger()
	err := errors.New("something went wrong")

	errorLogger := WithError(logger, err)
	errorLogger.Error("operation failed")

	var jsonOutput map[string]interface{}
	jsonErr := json.Unmarshal(buf.Bytes(), &jsonOutput)
	require.NoError(t, jsonErr)

	assert.Equal(t, "something went wrong", jsonOutput["error"])
	assert.Equal(t, "operation failed", jsonOutput["msg"])
}

func TestSetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Config{
		Environment: "development",
		Level:       slog.LevelInfo,
		Output:      buf,
	})

	logger := GetLogger()

	logger.Debug("should not appear")
	assert.Empty(t, buf.String())

	SetLevel(slog.LevelDebug)
	buf.Reset()

	logger = GetLogger() // Get updated logger after SetLevel
	logger.Debug("should appear")
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "should appear")
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("read from env", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		os.Setenv("APP_ENV", "production")
		defer os.Unsetenv("LOG_LEVEL")
		defer os.Unsetenv("APP_ENV")

		buf := &bytes.Buffer{}
		InitFromEnv(buf)

		logger := GetLogger()
		logger.Debug("debug in production")

		output := buf.String()
		assert.NotEmpty(t, output)

		var jsonOutput map[string]interface{}
		err := json.Unmarshal([]byte(output), &jsonOutput)
		assert.NoError(t, err)
	})
}

func TestConcurrency(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Config{
		Environment: "development",
		Output:      buf,
	})

	logger := GetLogger()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("concurrent log", "goroutine", id)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 10, len(lines))
}

func TestLoggerParseLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"warn", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"error", slog.LevelError},
		{"invalid", slog.LevelInfo}, // default
		{"", slog.LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInitWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		Environment: "development",
		Level:       slog.LevelInfo,
		FilePath:    logPath,
	}

	err := InitWithFile(cfg)
	require.NoError(t, err)

	logger := GetLogger()
	logger.Info("test message", "key", "value")

	// ファイルが作成されているか確認
	_, err = os.Stat(logPath)
	assert.NoError(t, err)

	// ファイルにログが出力されているか確認
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message")
	assert.Contains(t, string(content), "key=value")

	// クローズ
	CloseFileWriter()
}

func TestInitWithFileAndStdout(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	buf := &bytes.Buffer{}

	// 標準出力をキャプチャするためにバッファに変更
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	cfg := Config{
		Environment: "development",
		Level:       slog.LevelInfo,
		FilePath:    logPath,
		Output:      buf, // MultiWriterで両方に出力
	}

	err := InitWithFile(cfg)
	require.NoError(t, err)

	logger := GetLogger()
	logger.Info("dual output test")

	w.Close()
	os.Stdout = oldStdout

	// ファイルに出力されているか確認
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "dual output test")

	CloseFileWriter()
}

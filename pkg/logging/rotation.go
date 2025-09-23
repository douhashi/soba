package logging

import (
	"io"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewRotatingFileWriter creates a writer that rotates log files
func NewRotatingFileWriter(filename string) (io.Writer, error) {
	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// ログファイルが存在しない場合は空ファイルを作成
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    100, // megabytes
		MaxBackups: 7,
		MaxAge:     30, // days
		Compress:   true,
		LocalTime:  true,
	}, nil
}

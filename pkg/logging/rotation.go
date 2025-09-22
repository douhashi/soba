package logging

import (
	"io"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewRotatingFileWriter creates a writer that rotates log files
func NewRotatingFileWriter(filename string) (io.Writer, error) {
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    100, // megabytes
		MaxBackups: 7,
		MaxAge:     30, // days
		Compress:   true,
		LocalTime:  true,
	}, nil
}

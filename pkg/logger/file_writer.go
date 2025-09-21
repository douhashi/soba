package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileWriter struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// NewFileWriter creates a new FileWriter with the specified path
func NewFileWriter(path string) (*FileWriter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileWriter{
		path: path,
		file: file,
	}, nil
}

// Write implements io.Writer interface
func (fw *FileWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return 0, fmt.Errorf("file writer is closed")
	}

	return fw.file.Write(p)
}

// Close closes the underlying file
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return nil
	}

	err := fw.file.Close()
	fw.file = nil
	return err
}

// Path returns the file path
func (fw *FileWriter) Path() string {
	return fw.path
}

// CreateMultiWriter creates an io.MultiWriter that writes to both stdout and file
func CreateMultiWriter(filePath string) (io.Writer, *FileWriter, error) {
	fw, err := NewFileWriter(filePath)
	if err != nil {
		return nil, nil, err
	}

	return io.MultiWriter(os.Stdout, fw), fw, nil
}

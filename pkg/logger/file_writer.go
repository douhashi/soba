package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileWriter struct {
	path      string
	file      *os.File
	autoFlush bool
	mu        sync.Mutex
}

// FileWriterOption defines options for FileWriter
type FileWriterOption func(*FileWriter)

// WithAutoFlush sets the autoFlush option
func WithAutoFlush(autoFlush bool) FileWriterOption {
	return func(fw *FileWriter) {
		fw.autoFlush = autoFlush
	}
}

// NewFileWriter creates a new FileWriter with the specified path
func NewFileWriter(path string, opts ...FileWriterOption) (*FileWriter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	fw := &FileWriter{
		path:      path,
		file:      file,
		autoFlush: true, // Default to true
	}

	// Apply options
	for _, opt := range opts {
		opt(fw)
	}

	return fw, nil
}

// Write implements io.Writer interface
func (fw *FileWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return 0, fmt.Errorf("file writer is closed")
	}

	n, err = fw.file.Write(p)
	if err != nil {
		return n, err
	}

	if fw.autoFlush {
		if syncErr := fw.file.Sync(); syncErr != nil {
			return n, syncErr
		}
	}

	return n, nil
}

// Sync flushes the file buffer to disk
func (fw *FileWriter) Sync() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return fmt.Errorf("file writer is closed")
	}

	return fw.file.Sync()
}

// Close closes the underlying file
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return nil
	}

	// Sync before closing to ensure all data is written
	if syncErr := fw.file.Sync(); syncErr != nil {
		// Log sync error but still try to close
		fmt.Fprintf(os.Stderr, "Warning: failed to sync before close: %v\n", syncErr)
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

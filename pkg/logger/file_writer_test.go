package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath)
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer fw.Close()

	if fw.file == nil {
		t.Errorf("FileWriter.file should not be nil")
	}

	// ファイルが作成されたか確認
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}
}

func TestNewFileWriterCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nested", "dirs", "test.log")

	fw, err := NewFileWriter(logPath)
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer fw.Close()

	// ディレクトリとファイルが作成されたか確認
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}
}

func TestFileWriterWrite(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath)
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer fw.Close()

	testData := []byte("test log message\n")
	n, err := fw.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}

	if n != len(testData) {
		t.Errorf("Wrote %d bytes, expected %d", n, len(testData))
	}

	// ファイルの内容を確認
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("File content = %q, want %q", string(content), string(testData))
	}
}

func TestFileWriterPermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// 書き込み権限のないディレクトリを作成
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.Mkdir(readOnlyDir, 0555)
	logPath := filepath.Join(readOnlyDir, "test.log")

	_, err := NewFileWriter(logPath)
	if err == nil {
		t.Errorf("Expected permission error, got nil")
	}
}

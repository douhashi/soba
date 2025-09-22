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

func TestFileWriterSync(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath)
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer fw.Close()

	testData := []byte("test sync message\n")
	_, err = fw.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}

	// Sync()を呼び出し
	err = fw.Sync()
	if err != nil {
		t.Fatalf("Failed to sync: %v", err)
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

func TestFileWriterAutoFlush(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// AutoFlushを有効にして作成
	fw, err := NewFileWriter(logPath, WithAutoFlush(true))
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer fw.Close()

	testData := []byte("auto flush message\n")
	_, err = fw.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}

	// AutoFlushが有効なので、即座にファイルに書き込まれているはず
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("File content = %q, want %q", string(content), string(testData))
	}
}

func TestFileWriterAutoFlushDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// AutoFlushを無効にして作成
	fw, err := NewFileWriter(logPath, WithAutoFlush(false))
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}
	defer fw.Close()

	testData := []byte("no auto flush message\n")
	_, err = fw.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}

	// 明示的にSync()を呼ぶまで、バッファリングされている可能性がある
	// ただし、OSのバッファリング動作に依存するため、
	// ここでは明示的なSync()後に確実に書き込まれることを確認
	err = fw.Sync()
	if err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("File content = %q, want %q", string(content), string(testData))
	}
}

func TestFileWriterCloseWithSync(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fw, err := NewFileWriter(logPath, WithAutoFlush(false))
	if err != nil {
		t.Fatalf("Failed to create FileWriter: %v", err)
	}

	testData := []byte("close with sync message\n")
	_, err = fw.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}

	// Close時にSync()が呼ばれることを確認
	err = fw.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("File content = %q, want %q", string(content), string(testData))
	}
}

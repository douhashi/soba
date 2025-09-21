package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindOldLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, ".soba", "logs")
	os.MkdirAll(logsDir, 0755)

	// テスト用のログファイルを作成
	testFiles := []string{
		"soba-1000.log",
		"soba-1001.log",
		"soba-1002.log",
		"soba-1003.log",
		"other-file.log",
		"soba-invalid.log",
		"soba-1004.log",
	}

	for _, f := range testFiles {
		file, err := os.Create(filepath.Join(logsDir, f))
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
		file.Close()
		time.Sleep(10 * time.Millisecond) // ファイル作成時間に差をつける
	}

	files, err := FindLogFiles(logsDir, "soba-*.log")
	if err != nil {
		t.Fatalf("Failed to find log files: %v", err)
	}

	// soba-*.log パターンにマッチするファイルは6個（soba-invalid.logも含む）
	expectedCount := 6
	if len(files) != expectedCount {
		t.Errorf("Found %d files, expected %d", len(files), expectedCount)
	}

	// ファイルが時間順（古い順）にソートされているか確認
	for i := 1; i < len(files); i++ {
		prev, _ := os.Stat(files[i-1])
		curr, _ := os.Stat(files[i])
		if prev.ModTime().After(curr.ModTime()) {
			t.Errorf("Files are not sorted by modification time")
		}
	}
}

func TestCleanupOldLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, ".soba", "logs")
	os.MkdirAll(logsDir, 0755)

	// 15個のテストファイルを作成
	for i := 1; i <= 15; i++ {
		filename := fmt.Sprintf("soba-%d.log", 1000+i)
		file, err := os.Create(filepath.Join(logsDir, filename))
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		file.Close()
		time.Sleep(10 * time.Millisecond)
	}

	// 保持世代数10でクリーンアップ
	err := CleanupOldLogFiles(logsDir, "soba-*.log", 10)
	if err != nil {
		t.Fatalf("Failed to cleanup old log files: %v", err)
	}

	// 残っているファイルを確認
	files, err := FindLogFiles(logsDir, "soba-*.log")
	if err != nil {
		t.Fatalf("Failed to find remaining files: %v", err)
	}

	if len(files) != 10 {
		t.Errorf("Remaining files = %d, want 10", len(files))
	}

	// 新しい10個のファイルが残っているか確認
	for _, f := range files {
		base := filepath.Base(f)
		// soba-1006.log から soba-1015.log が残っているはず
		if base < "soba-1006.log" {
			t.Errorf("Old file %s should have been deleted", base)
		}
	}
}

func TestCleanupOldLogFilesNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, ".soba", "logs")
	os.MkdirAll(logsDir, 0755)

	// ファイルがない場合のテスト
	err := CleanupOldLogFiles(logsDir, "soba-*.log", 10)
	if err != nil {
		t.Errorf("CleanupOldLogFiles should not fail when no files exist: %v", err)
	}
}

func TestCleanupOldLogFilesLessThanRetention(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, ".soba", "logs")
	os.MkdirAll(logsDir, 0755)

	// 5個のファイルを作成（保持世代数10より少ない）
	for i := 1; i <= 5; i++ {
		filename := fmt.Sprintf("soba-%d.log", 1000+i)
		file, err := os.Create(filepath.Join(logsDir, filename))
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		file.Close()
	}

	err := CleanupOldLogFiles(logsDir, "soba-*.log", 10)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// 全てのファイルが残っているか確認
	files, err := FindLogFiles(logsDir, "soba-*.log")
	if err != nil {
		t.Fatalf("Failed to find files: %v", err)
	}

	if len(files) != 5 {
		t.Errorf("All files should remain. Found %d files", len(files))
	}
}

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FindLogFiles finds all log files matching the pattern in the specified directory
func FindLogFiles(dir, pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to glob log files: %w", err)
	}

	// ファイル情報を取得してソート用の構造体を作成
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue // エラーがあったファイルはスキップ
		}
		if !info.IsDir() {
			files = append(files, fileInfo{
				path:    path,
				modTime: info.ModTime(),
			})
		}
	}

	// 更新時間でソート（古い順）
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// パスのみを返す
	result := make([]string, len(files))
	for i, f := range files {
		result[i] = f.path
	}

	return result, nil
}

// CleanupOldLogFiles removes log files exceeding the retention count
func CleanupOldLogFiles(dir, pattern string, retentionCount int) error {
	if retentionCount <= 0 {
		return fmt.Errorf("retention count must be positive")
	}

	files, err := FindLogFiles(dir, pattern)
	if err != nil {
		return fmt.Errorf("failed to find log files: %w", err)
	}

	// 保持数を超えるファイルがない場合は何もしない
	if len(files) <= retentionCount {
		return nil
	}

	// 古いファイルから削除（最新のretentionCount個を残す）
	filesToDelete := files[:len(files)-retentionCount]

	for _, file := range filesToDelete {
		if err := os.Remove(file); err != nil {
			// エラーが発生してもログファイルの削除は継続
			fmt.Fprintf(os.Stderr, "Warning: failed to remove old log file %s: %v\n", file, err)
		}
	}

	return nil
}

// GetLogDirectory returns the directory path from a log file path
func GetLogDirectory(logPath string) string {
	return filepath.Dir(logPath)
}

// GetLogPattern returns the pattern for finding log files
func GetLogPattern(pid int) string {
	return "soba-*.log"
}

package infra

import (
	"fmt"

	"github.com/douhashi/soba/pkg/errors"
)

// NewGitHubAPIError はGitHub APIエラーを作成
func NewGitHubAPIError(statusCode int, endpoint, message string) error {
	msg := fmt.Sprintf("GitHub API error (%d) at %s: %s", statusCode, endpoint, message)
	var err error = errors.NewExternalError(msg)
	err = errors.WithContext(err, "status_code", statusCode)
	err = errors.WithContext(err, "endpoint", endpoint)
	return err
}

// NewTmuxExecutionError はTmux実行エラーを作成
func NewTmuxExecutionError(command string, exitCode int, stderr string) error {
	msg := fmt.Sprintf("tmux command failed: %s (exit code: %d): %s", command, exitCode, stderr)
	var err error = errors.NewExternalError(msg)
	err = errors.WithContext(err, "command", command)
	err = errors.WithContext(err, "exit_code", exitCode)
	err = errors.WithContext(err, "stderr", stderr)
	return err
}

// NewConfigLoadError は設定ファイル読み込みエラーを作成
func NewConfigLoadError(filePath, reason string) error {
	msg := fmt.Sprintf("failed to load config from %s: %s", filePath, reason)
	var err error = errors.NewValidationError(msg)
	err = errors.WithContext(err, "file", filePath)
	err = errors.WithContext(err, "reason", reason)
	return err
}

// WrapInfraError はインフラ層のエラーをラップ
func WrapInfraError(err error, message string) error {
	return errors.WrapExternal(err, message)
}

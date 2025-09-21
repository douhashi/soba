package service

import (
	"fmt"

	"github.com/douhashi/soba/pkg/errors"
)

// NewWorkflowExecutionError はワークフロー実行エラーを作成
func NewWorkflowExecutionError(workflow, phase, reason string) error {
	msg := fmt.Sprintf("workflow '%s' failed at phase '%s': %s", workflow, phase, reason)
	var err error = errors.NewInternalError(msg)
	err = errors.WithContext(err, "workflow", workflow)
	err = errors.WithContext(err, "phase", phase)
	return err
}

// NewIssueProcessingError はIssue処理エラーを作成
func NewIssueProcessingError(issueNum int, operation, reason string) error {
	msg := fmt.Sprintf("failed to process issue #%d during '%s': %s", issueNum, operation, reason)
	var err error = errors.NewInternalError(msg)
	err = errors.WithContext(err, "issue_number", issueNum)
	err = errors.WithContext(err, "operation", operation)
	return err
}

// NewDaemonError はデーモンエラーを作成
func NewDaemonError(component, reason string) error {
	msg := fmt.Sprintf("daemon component '%s' failed: %s", component, reason)
	var err error = errors.NewInternalError(msg)
	err = errors.WithContext(err, "component", component)
	return err
}

// WrapServiceError はサービス層のエラーをラップ
func WrapServiceError(err error, message string) error {
	return errors.WrapInternal(err, message)
}

// NewTmuxManagementError はtmux管理エラーを作成
func NewTmuxManagementError(operation, target, reason string) error {
	msg := fmt.Sprintf("tmux %s failed for %s: %s", operation, target, reason)
	var err error = errors.NewInternalError(msg)
	err = errors.WithContext(err, "operation", operation)
	err = errors.WithContext(err, "target", target)
	return err
}

// NewCommandExecutionError はコマンド実行エラーを作成
func NewCommandExecutionError(command, phase string, issueNum int, reason string) error {
	msg := fmt.Sprintf("command execution failed for phase '%s' on issue #%d: %s", phase, issueNum, reason)
	var err error = errors.NewInternalError(msg)
	err = errors.WithContext(err, "command", command)
	err = errors.WithContext(err, "phase", phase)
	err = errors.WithContext(err, "issue_number", issueNum)
	return err
}

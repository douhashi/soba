package tmux

import "fmt"

// TmuxError はtmux操作時に発生するエラーを表すカスタムエラー型
// 操作種別、メッセージ、元のエラーを含む
type TmuxError struct {
	Operation string
	Message   string
	Err       error
}

// Error はTmuxErrorを文字列として表現する（errorインターフェースの実装）
func (e *TmuxError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("tmux %s error: %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("tmux %s error: %s", e.Operation, e.Message)
}

// Unwrap はラップされた元のエラーを返す（Go 1.13+のerrors.Unwrap対応）
func (e *TmuxError) Unwrap() error {
	return e.Err
}

// NewTmuxError は指定されたパラメータで新しいTmuxErrorインスタンスを作成する
func NewTmuxError(operation, message string, err error) *TmuxError {
	return &TmuxError{
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}

// ErrSessionProtected は保護されたセッションの操作エラー
var ErrSessionProtected = &TmuxError{
	Operation: "session_operation",
	Message:   "cannot delete protected session",
}

// ErrSessionNotFound はセッションが見つからないエラー
var ErrSessionNotFound = &TmuxError{
	Operation: "session_operation",
	Message:   "session not found",
}

// ErrWindowNotFound はウィンドウが見つからないエラー
var ErrWindowNotFound = &TmuxError{
	Operation: "window_operation",
	Message:   "window not found",
}

// ErrPaneNotFound はペインが見つからないエラー
var ErrPaneNotFound = &TmuxError{
	Operation: "pane_operation",
	Message:   "pane not found",
}

// ErrInvalidRepository は無効なリポジトリ名エラー
var ErrInvalidRepository = &TmuxError{
	Operation: "session_name_generation",
	Message:   "invalid repository name",
}

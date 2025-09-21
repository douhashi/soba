package errors

import (
	"errors"
	"fmt"
)

// ErrorCode はエラーの種類を表す
type ErrorCode string

const (
	// CodeUnknown は不明なエラーコード
	CodeUnknown ErrorCode = "UNKNOWN"
	// CodeValidation は検証エラーコード
	CodeValidation ErrorCode = "VALIDATION"
	// CodeNotFound はリソースが見つからないエラーコード
	CodeNotFound ErrorCode = "NOT_FOUND"
	// CodeInternal は内部エラーコード
	CodeInternal ErrorCode = "INTERNAL"
	// CodeConflict は競合エラーコード
	CodeConflict ErrorCode = "CONFLICT"
	// CodeTimeout はタイムアウトエラーコード
	CodeTimeout ErrorCode = "TIMEOUT"
	// CodeExternal は外部システムエラーコード
	CodeExternal ErrorCode = "EXTERNAL"
)

// BaseError は共通エラー構造体
type BaseError struct {
	Code    ErrorCode              // エラーコード
	Message string                 // エラーメッセージ
	Cause   error                  // 原因となるエラー
	Context map[string]interface{} // 追加のコンテキスト情報
}

// Error はerrorインターフェースの実装
func (e *BaseError) Error() string {
	prefix := ""
	switch e.Code {
	case CodeValidation:
		prefix = "validation error"
	case CodeNotFound:
		prefix = "not found"
	case CodeInternal:
		prefix = "internal error"
	case CodeConflict:
		prefix = "conflict"
	case CodeTimeout:
		prefix = "timeout"
	case CodeExternal:
		prefix = "external error"
	default:
		prefix = "error"
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", prefix, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", prefix, e.Message)
}

// Unwrap は原因となるエラーを返す
func (e *BaseError) Unwrap() error {
	return e.Cause
}

// NewBaseError は基本エラーを作成
func NewBaseError(code ErrorCode, message string) *BaseError {
	return &BaseError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// NewValidationError は検証エラーを作成
func NewValidationError(message string) *BaseError {
	return NewBaseError(CodeValidation, message)
}

// NewNotFoundError はリソースが見つからないエラーを作成
func NewNotFoundError(message string) *BaseError {
	return NewBaseError(CodeNotFound, message)
}

// NewInternalError は内部エラーを作成
func NewInternalError(message string) *BaseError {
	return NewBaseError(CodeInternal, message)
}

// NewConflictError は競合エラーを作成
func NewConflictError(message string) *BaseError {
	return NewBaseError(CodeConflict, message)
}

// NewTimeoutError はタイムアウトエラーを作成
func NewTimeoutError(message string) *BaseError {
	return NewBaseError(CodeTimeout, message)
}

// NewExternalError は外部システムエラーを作成
func NewExternalError(message string) *BaseError {
	return NewBaseError(CodeExternal, message)
}

// Wrap はエラーをラップする
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		// BaseErrorの場合はコンテキストを保持してラップ
		newErr := &BaseError{
			Code:    baseErr.Code,
			Message: message,
			Cause:   err,
			Context: baseErr.Context,
		}
		return newErr
	}

	// 通常のエラーの場合はfmt.Errorfと同じ形式にする
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf はフォーマット付きでエラーをラップする
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return Wrap(err, fmt.Sprintf(format, args...))
}

// WrapValidation は検証エラーとしてラップする
func WrapValidation(err error, message string) error {
	if err == nil {
		return nil
	}

	return &BaseError{
		Code:    CodeValidation,
		Message: message,
		Cause:   err,
		Context: make(map[string]interface{}),
	}
}

// WrapNotFound はリソースが見つからないエラーとしてラップする
func WrapNotFound(err error, message string) error {
	if err == nil {
		return nil
	}

	return &BaseError{
		Code:    CodeNotFound,
		Message: message,
		Cause:   err,
		Context: make(map[string]interface{}),
	}
}

// WrapInternal は内部エラーとしてラップする
func WrapInternal(err error, message string) error {
	if err == nil {
		return nil
	}

	return &BaseError{
		Code:    CodeInternal,
		Message: message,
		Cause:   err,
		Context: make(map[string]interface{}),
	}
}

// WrapExternal は外部システムエラーとしてラップする
func WrapExternal(err error, message string) error {
	if err == nil {
		return nil
	}

	return &BaseError{
		Code:    CodeExternal,
		Message: message,
		Cause:   err,
		Context: make(map[string]interface{}),
	}
}

// WithContext はエラーにコンテキスト情報を追加する
func WithContext(err error, key string, value interface{}) error {
	if err == nil {
		return nil
	}

	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		// コンテキストマップが未初期化の場合は初期化
		if baseErr.Context == nil {
			baseErr.Context = make(map[string]interface{})
		}
		baseErr.Context[key] = value
		return baseErr
	}

	// BaseErrorでない場合は新しいBaseErrorでラップ
	newErr := &BaseError{
		Code:    CodeUnknown,
		Message: err.Error(),
		Cause:   err,
		Context: map[string]interface{}{
			key: value,
		},
	}
	return newErr
}

// GetCode はエラーコードを取得する
func GetCode(err error) ErrorCode {
	if err == nil {
		return CodeUnknown
	}

	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		return baseErr.Code
	}

	return CodeUnknown
}

// IsCode はエラーが特定のコードかを判定する
func IsCode(err error, code ErrorCode) bool {
	return GetCode(err) == code
}

// IsValidationError は検証エラーかを判定する
func IsValidationError(err error) bool {
	return IsCode(err, CodeValidation)
}

// IsNotFoundError はリソースが見つからないエラーかを判定する
func IsNotFoundError(err error) bool {
	return IsCode(err, CodeNotFound)
}

// IsInternalError は内部エラーかを判定する
func IsInternalError(err error) bool {
	return IsCode(err, CodeInternal)
}

// IsConflictError は競合エラーかを判定する
func IsConflictError(err error) bool {
	return IsCode(err, CodeConflict)
}

// IsTimeoutError はタイムアウトエラーかを判定する
func IsTimeoutError(err error) bool {
	return IsCode(err, CodeTimeout)
}

// IsExternalError は外部システムエラーかを判定する
func IsExternalError(err error) bool {
	return IsCode(err, CodeExternal)
}

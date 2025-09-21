package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestBaseError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
		wantMsg  string
	}{
		{
			name:     "validation error",
			err:      NewValidationError("invalid input"),
			wantCode: CodeValidation,
			wantMsg:  "validation error: invalid input",
		},
		{
			name:     "not found error",
			err:      NewNotFoundError("resource not found"),
			wantCode: CodeNotFound,
			wantMsg:  "not found: resource not found",
		},
		{
			name:     "internal error",
			err:      NewInternalError("system error"),
			wantCode: CodeInternal,
			wantMsg:  "internal error: system error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", got, tt.wantMsg)
			}

			var baseErr *BaseError
			if !errors.As(tt.err, &baseErr) {
				t.Fatalf("expected BaseError type")
			}

			if baseErr.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", baseErr.Code, tt.wantCode)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name         string
		wrapFunc     func() error
		wantMsg      string
		wantOriginal bool
	}{
		{
			name: "wrap with context",
			wrapFunc: func() error {
				return Wrap(originalErr, "failed to process")
			},
			wantMsg:      "failed to process: original error",
			wantOriginal: true,
		},
		{
			name: "wrap validation error",
			wrapFunc: func() error {
				return WrapValidation(originalErr, "validation failed")
			},
			wantMsg:      "validation error: validation failed: original error",
			wantOriginal: true,
		},
		{
			name: "wrap not found error",
			wrapFunc: func() error {
				return WrapNotFound(originalErr, "resource missing")
			},
			wantMsg:      "not found: resource missing: original error",
			wantOriginal: true,
		},
		{
			name: "wrap internal error",
			wrapFunc: func() error {
				return WrapInternal(originalErr, "internal failure")
			},
			wantMsg:      "internal error: internal failure: original error",
			wantOriginal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wrapFunc()

			if got := err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", got, tt.wantMsg)
			}

			if tt.wantOriginal && !errors.Is(err, originalErr) {
				t.Errorf("errors.Is() = false, want true")
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("original error")

	err := Wrapf(originalErr, "failed to process %s", "data")
	want := "failed to process data: original error"

	if got := err.Error(); got != want {
		t.Errorf("Wrapf() error = %v, want %v", got, want)
	}

	if !errors.Is(err, originalErr) {
		t.Errorf("errors.Is() = false, want true")
	}
}

func TestIs(t *testing.T) {
	baseErr := NewValidationError("test")
	wrappedErr := Wrap(baseErr, "wrapped")

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error",
			err:    baseErr,
			target: baseErr,
			want:   true,
		},
		{
			name:   "wrapped error",
			err:    wrappedErr,
			target: baseErr,
			want:   true,
		},
		{
			name:   "different error",
			err:    NewNotFoundError("other"),
			target: baseErr,
			want:   false,
		},
		{
			name:   "nil error",
			err:    nil,
			target: baseErr,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAs(t *testing.T) {
	validationErr := NewValidationError("test")
	wrappedErr := Wrap(validationErr, "wrapped")

	tests := []struct {
		name     string
		err      error
		wantErr  bool
		wantCode ErrorCode
	}{
		{
			name:     "direct BaseError",
			err:      validationErr,
			wantErr:  true,
			wantCode: CodeValidation,
		},
		{
			name:     "wrapped BaseError",
			err:      wrappedErr,
			wantErr:  true,
			wantCode: CodeValidation,
		},
		{
			name:    "non-BaseError",
			err:     errors.New("standard error"),
			wantErr: false,
		},
		{
			name:    "nil error",
			err:     nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var baseErr *BaseError
			if got := errors.As(tt.err, &baseErr); got != tt.wantErr {
				t.Errorf("errors.As() = %v, want %v", got, tt.wantErr)
			}

			if tt.wantErr && baseErr.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", baseErr.Code, tt.wantCode)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "validation error",
			err:  NewValidationError("test"),
			want: CodeValidation,
		},
		{
			name: "wrapped validation error",
			err:  Wrap(NewValidationError("test"), "context"),
			want: CodeValidation,
		},
		{
			name: "standard error",
			err:  errors.New("standard"),
			want: CodeUnknown,
		},
		{
			name: "nil error",
			err:  nil,
			want: CodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCode(tt.err); got != tt.want {
				t.Errorf("GetCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	err := NewValidationError("test")

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{
			name:  "string context",
			key:   "field",
			value: "username",
		},
		{
			name:  "int context",
			key:   "line",
			value: 42,
		},
		{
			name:  "bool context",
			key:   "required",
			value: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextErr := WithContext(err, tt.key, tt.value)

			var baseErr *BaseError
			if !errors.As(contextErr, &baseErr) {
				t.Fatalf("expected BaseError type")
			}

			if baseErr.Context == nil {
				t.Fatalf("Context is nil")
			}

			if got := baseErr.Context[tt.key]; got != tt.value {
				t.Errorf("Context[%s] = %v, want %v", tt.key, got, tt.value)
			}
		})
	}
}

func TestMultipleContext(t *testing.T) {
	var err error = NewValidationError("test")
	err = WithContext(err, "field", "username")
	err = WithContext(err, "required", true)
	err = WithContext(err, "line", 10)

	var baseErr *BaseError
	if !errors.As(err, &baseErr) {
		t.Fatalf("expected BaseError type")
	}

	if len(baseErr.Context) != 3 {
		t.Errorf("Context length = %d, want 3", len(baseErr.Context))
	}

	expectedContext := map[string]interface{}{
		"field":    "username",
		"required": true,
		"line":     10,
	}

	for k, v := range expectedContext {
		if got := baseErr.Context[k]; got != v {
			t.Errorf("Context[%s] = %v, want %v", k, got, v)
		}
	}
}

func ExampleWrap() {
	originalErr := errors.New("database connection failed")
	wrappedErr := Wrap(originalErr, "failed to fetch user")
	fmt.Println(wrappedErr)
	// Output: failed to fetch user: database connection failed
}

func ExampleNewValidationError() {
	err := NewValidationError("email format is invalid")
	fmt.Println(err)
	// Output: validation error: email format is invalid
}

func ExampleWithContext() {
	var err error = NewValidationError("invalid input")
	err = WithContext(err, "field", "email")
	err = WithContext(err, "value", "not-an-email")

	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		fmt.Printf("Error: %s\n", baseErr.Message)
		fmt.Printf("Field: %v\n", baseErr.Context["field"])
		fmt.Printf("Value: %v\n", baseErr.Context["value"])
	}
	// Output:
	// Error: invalid input
	// Field: email
	// Value: not-an-email
}

package domain

import (
	"errors"
	"testing"

	sobaErrors "github.com/douhashi/soba/pkg/errors"
)

func TestIssueNotFoundError(t *testing.T) {
	tests := []struct {
		name   string
		number int
		want   string
	}{
		{
			name:   "issue not found",
			number: 42,
			want:   "not found: issue #42 not found",
		},
		{
			name:   "issue zero",
			number: 0,
			want:   "not found: issue #0 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewIssueNotFoundError(tt.number)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsNotFoundError(err) {
				t.Errorf("IsNotFoundError() = false, want true")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
		want    string
	}{
		{
			name:    "field validation",
			field:   "title",
			message: "must not be empty",
			want:    "validation error: field 'title' is invalid: must not be empty",
		},
		{
			name:    "format validation",
			field:   "email",
			message: "invalid format",
			want:    "validation error: field 'email' is invalid: invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.message)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsValidationError(err) {
				t.Errorf("IsValidationError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["field"] != tt.field {
					t.Errorf("Context[field] = %v, want %v", baseErr.Context["field"], tt.field)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestPhaseTransitionError(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		issueNum int
		want     string
	}{
		{
			name:     "invalid transition",
			from:     "doing",
			to:       "todo",
			issueNum: 10,
			want:     "conflict: cannot transition issue #10 from phase 'doing' to 'todo'",
		},
		{
			name:     "another invalid transition",
			from:     "done",
			to:       "in_review",
			issueNum: 5,
			want:     "conflict: cannot transition issue #5 from phase 'done' to 'in_review'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewPhaseTransitionError(tt.from, tt.to, tt.issueNum)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsConflictError(err) {
				t.Errorf("IsConflictError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["from"] != tt.from {
					t.Errorf("Context[from] = %v, want %v", baseErr.Context["from"], tt.from)
				}
				if baseErr.Context["to"] != tt.to {
					t.Errorf("Context[to] = %v, want %v", baseErr.Context["to"], tt.to)
				}
				if baseErr.Context["issue"] != tt.issueNum {
					t.Errorf("Context[issue] = %v, want %v", baseErr.Context["issue"], tt.issueNum)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestWrapDomainError(t *testing.T) {
	originalErr := errors.New("database error")

	err := WrapDomainError(originalErr, "failed to save issue")
	want := "internal error: failed to save issue: database error"

	if got := err.Error(); got != want {
		t.Errorf("Error() = %v, want %v", got, want)
	}

	if !sobaErrors.IsInternalError(err) {
		t.Errorf("IsInternalError() = false, want true")
	}

	if !errors.Is(err, originalErr) {
		t.Errorf("errors.Is() = false, want true")
	}
}

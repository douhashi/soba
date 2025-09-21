package service

import (
	"errors"
	"testing"

	sobaErrors "github.com/douhashi/soba/pkg/errors"
)

func TestWorkflowExecutionError(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		phase    string
		reason   string
		want     string
	}{
		{
			name:     "workflow failed",
			workflow: "issue-processor",
			phase:    "analysis",
			reason:   "failed to parse issue body",
			want:     "internal error: workflow 'issue-processor' failed at phase 'analysis': failed to parse issue body",
		},
		{
			name:     "workflow timeout",
			workflow: "daemon",
			phase:    "initialization",
			reason:   "timeout waiting for resources",
			want:     "internal error: workflow 'daemon' failed at phase 'initialization': timeout waiting for resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWorkflowExecutionError(tt.workflow, tt.phase, tt.reason)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsInternalError(err) {
				t.Errorf("IsInternalError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["workflow"] != tt.workflow {
					t.Errorf("Context[workflow] = %v, want %v", baseErr.Context["workflow"], tt.workflow)
				}
				if baseErr.Context["phase"] != tt.phase {
					t.Errorf("Context[phase] = %v, want %v", baseErr.Context["phase"], tt.phase)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestIssueProcessingError(t *testing.T) {
	tests := []struct {
		name      string
		issueNum  int
		operation string
		reason    string
		want      string
	}{
		{
			name:      "label update failed",
			issueNum:  10,
			operation: "update-labels",
			reason:    "permission denied",
			want:      "internal error: failed to process issue #10 during 'update-labels': permission denied",
		},
		{
			name:      "comment creation failed",
			issueNum:  25,
			operation: "add-comment",
			reason:    "API rate limit exceeded",
			want:      "internal error: failed to process issue #25 during 'add-comment': API rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewIssueProcessingError(tt.issueNum, tt.operation, tt.reason)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsInternalError(err) {
				t.Errorf("IsInternalError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["issue_number"] != tt.issueNum {
					t.Errorf("Context[issue_number] = %v, want %v", baseErr.Context["issue_number"], tt.issueNum)
				}
				if baseErr.Context["operation"] != tt.operation {
					t.Errorf("Context[operation] = %v, want %v", baseErr.Context["operation"], tt.operation)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestDaemonError(t *testing.T) {
	tests := []struct {
		name      string
		component string
		reason    string
		want      string
	}{
		{
			name:      "watcher failed",
			component: "issue-watcher",
			reason:    "failed to connect to GitHub",
			want:      "internal error: daemon component 'issue-watcher' failed: failed to connect to GitHub",
		},
		{
			name:      "processor failed",
			component: "event-processor",
			reason:    "queue overflow",
			want:      "internal error: daemon component 'event-processor' failed: queue overflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDaemonError(tt.component, tt.reason)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsInternalError(err) {
				t.Errorf("IsInternalError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["component"] != tt.component {
					t.Errorf("Context[component] = %v, want %v", baseErr.Context["component"], tt.component)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestWrapServiceError(t *testing.T) {
	originalErr := errors.New("database connection lost")

	err := WrapServiceError(originalErr, "failed to save workflow state")
	want := "internal error: failed to save workflow state: database connection lost"

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

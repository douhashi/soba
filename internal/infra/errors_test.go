package infra

import (
	"errors"
	"testing"

	sobaErrors "github.com/douhashi/soba/pkg/errors"
)

func TestGitHubAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		endpoint   string
		message    string
		want       string
	}{
		{
			name:       "not found",
			statusCode: 404,
			endpoint:   "/repos/owner/repo",
			message:    "repository not found",
			want:       "external error: GitHub API error (404) at /repos/owner/repo: repository not found",
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			endpoint:   "/user",
			message:    "authentication required",
			want:       "external error: GitHub API error (401) at /user: authentication required",
		},
		{
			name:       "rate limit",
			statusCode: 429,
			endpoint:   "/issues",
			message:    "rate limit exceeded",
			want:       "external error: GitHub API error (429) at /issues: rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewGitHubAPIError(tt.statusCode, tt.endpoint, tt.message)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsExternalError(err) {
				t.Errorf("IsExternalError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["status_code"] != tt.statusCode {
					t.Errorf("Context[status_code] = %v, want %v", baseErr.Context["status_code"], tt.statusCode)
				}
				if baseErr.Context["endpoint"] != tt.endpoint {
					t.Errorf("Context[endpoint] = %v, want %v", baseErr.Context["endpoint"], tt.endpoint)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestTmuxExecutionError(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		exitCode int
		stderr   string
		want     string
	}{
		{
			name:     "command failed",
			command:  "tmux new-session",
			exitCode: 1,
			stderr:   "no server running",
			want:     "external error: tmux command failed: tmux new-session (exit code: 1): no server running",
		},
		{
			name:     "session exists",
			command:  "tmux new -s test",
			exitCode: 1,
			stderr:   "duplicate session: test",
			want:     "external error: tmux command failed: tmux new -s test (exit code: 1): duplicate session: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTmuxExecutionError(tt.command, tt.exitCode, tt.stderr)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsExternalError(err) {
				t.Errorf("IsExternalError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["command"] != tt.command {
					t.Errorf("Context[command] = %v, want %v", baseErr.Context["command"], tt.command)
				}
				if baseErr.Context["exit_code"] != tt.exitCode {
					t.Errorf("Context[exit_code] = %v, want %v", baseErr.Context["exit_code"], tt.exitCode)
				}
				if baseErr.Context["stderr"] != tt.stderr {
					t.Errorf("Context[stderr] = %v, want %v", baseErr.Context["stderr"], tt.stderr)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestConfigLoadError(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		reason   string
		want     string
	}{
		{
			name:     "file not found",
			filePath: "/etc/soba/config.yaml",
			reason:   "file not found",
			want:     "validation error: failed to load config from /etc/soba/config.yaml: file not found",
		},
		{
			name:     "invalid format",
			filePath: "~/.soba/config.toml",
			reason:   "invalid TOML syntax",
			want:     "validation error: failed to load config from ~/.soba/config.toml: invalid TOML syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewConfigLoadError(tt.filePath, tt.reason)

			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}

			if !sobaErrors.IsValidationError(err) {
				t.Errorf("IsValidationError() = false, want true")
			}

			// コンテキスト情報の確認
			var baseErr *sobaErrors.BaseError
			if errors.As(err, &baseErr) {
				if baseErr.Context["file"] != tt.filePath {
					t.Errorf("Context[file] = %v, want %v", baseErr.Context["file"], tt.filePath)
				}
				if baseErr.Context["reason"] != tt.reason {
					t.Errorf("Context[reason] = %v, want %v", baseErr.Context["reason"], tt.reason)
				}
			} else {
				t.Errorf("expected BaseError type")
			}
		})
	}
}

func TestWrapInfraError(t *testing.T) {
	originalErr := errors.New("network timeout")

	err := WrapInfraError(originalErr, "failed to connect to GitHub")
	want := "external error: failed to connect to GitHub: network timeout"

	if got := err.Error(); got != want {
		t.Errorf("Error() = %v, want %v", got, want)
	}

	if !sobaErrors.IsExternalError(err) {
		t.Errorf("IsExternalError() = false, want true")
	}

	if !errors.Is(err, originalErr) {
		t.Errorf("errors.Is() = false, want true")
	}
}

package git

import (
	"fmt"
)

// GitError represents a Git operation error
type GitError struct {
	Op      string
	Path    string
	Message string
	Err     error
}

func (e *GitError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("git %s %s: %s", e.Op, e.Path, e.Message)
	}
	return fmt.Sprintf("git %s: %s", e.Op, e.Message)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// NewGitError creates a new GitError
func NewGitError(op, path, message string, err error) error {
	return &GitError{
		Op:      op,
		Path:    path,
		Message: message,
		Err:     err,
	}
}
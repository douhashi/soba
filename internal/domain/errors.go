package domain

import (
	"fmt"

	"github.com/douhashi/soba/pkg/errors"
)

// NewIssueNotFoundError はIssueが見つからないエラーを作成
func NewIssueNotFoundError(number int) error {
	err := errors.NewNotFoundError(fmt.Sprintf("issue #%d not found", number))
	return errors.WithContext(err, "issue_number", number)
}

// NewValidationError はドメイン検証エラーを作成
func NewValidationError(field, message string) error {
	err := errors.NewValidationError(fmt.Sprintf("field '%s' is invalid: %s", field, message))
	return errors.WithContext(err, "field", field)
}

// NewPhaseTransitionError はフェーズ遷移エラーを作成
func NewPhaseTransitionError(from, to string, issueNum int) error {
	msg := fmt.Sprintf("cannot transition issue #%d from phase '%s' to '%s'", issueNum, from, to)
	var err error = errors.NewConflictError(msg)
	err = errors.WithContext(err, "from", from)
	err = errors.WithContext(err, "to", to)
	err = errors.WithContext(err, "issue", issueNum)
	return err
}

// WrapDomainError はドメイン層のエラーをラップ
func WrapDomainError(err error, message string) error {
	return errors.WrapInternal(err, message)
}

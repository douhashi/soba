package slack

// Manager defines the interface for Slack notification management
type Manager interface {
	// NotifyPhaseStart sends a notification when a phase starts
	NotifyPhaseStart(phase string, issueNumber int)

	// NotifyPRMerged sends a notification when a PR is merged
	NotifyPRMerged(prNumber, issueNumber int)

	// NotifyError sends an error notification
	NotifyError(title, errorMessage string)

	// IsEnabled returns whether Slack notifications are enabled
	IsEnabled() bool
}

// NoOpManager is a no-operation implementation of Manager
// Used when Slack notifications are disabled
type NoOpManager struct{}

func (n *NoOpManager) NotifyPhaseStart(phase string, issueNumber int) {}
func (n *NoOpManager) NotifyPRMerged(prNumber, issueNumber int)       {}
func (n *NoOpManager) NotifyError(title, errorMessage string)         {}
func (n *NoOpManager) IsEnabled() bool                                { return false }

// SlackClient interface for sending messages (moved from notifier.go)
type SlackClient interface {
	SendMessage(message string) error
}

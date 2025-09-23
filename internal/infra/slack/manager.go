package slack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

var (
	instance Manager
	once     sync.Once
)

// SlackManager implements Manager interface
type SlackManager struct {
	client          SlackClient
	config          config.SlackConfig
	githubConfig    config.GitHubConfig
	logger          logging.Logger
	templateManager TemplateManager
}

// Initialize initializes the global Slack manager based on config
func Initialize(cfg *config.Config, logger logging.Logger) {
	once.Do(func() {
		if !cfg.Slack.NotificationsEnabled || cfg.Slack.WebhookURL == "" {
			// Use NoOpManager when Slack is disabled
			instance = &NoOpManager{}
			logger.Info(context.Background(), "Slack notifications disabled")
			return
		}

		// Check if GitHub repository is configured
		if cfg.GitHub.Repository == "" {
			logger.Warn(context.Background(), "GitHub repository not configured, falling back to NoOp manager")
			instance = &NoOpManager{}
			return
		}

		// Use default timeout of 30 seconds
		timeout := 30 * time.Second
		client := NewClient(cfg.Slack.WebhookURL, timeout)

		// Initialize template manager
		templateManager := NewTemplateManager(logger)
		if err := templateManager.LoadTemplates(); err != nil {
			logger.Warn(context.Background(), "Failed to load Slack templates, falling back to NoOp manager",
				logging.Field{Key: "error", Value: err.Error()},
			)
			instance = &NoOpManager{}
			return
		}

		instance = &SlackManager{
			client:          client,
			config:          cfg.Slack,
			githubConfig:    cfg.GitHub,
			logger:          logger,
			templateManager: templateManager,
		}
		logger.Info(context.Background(), "Slack notifications enabled with block templates",
			logging.Field{Key: "repository", Value: cfg.GitHub.Repository},
		)
	})
}

// GetManager returns the global Slack manager instance
func GetManager() Manager {
	if instance == nil {
		// Return NoOpManager if not initialized
		return &NoOpManager{}
	}
	return instance
}

// Reset resets the singleton (for testing)
func Reset() {
	instance = nil
	once = sync.Once{}
}

// Template data structures
type NotifyData struct {
	Text string
}

type PhaseStartData struct {
	Phase       string
	IssueNumber int
	IssueURL    string
	Repository  string
}

type PRMergedData struct {
	PRNumber    int
	IssueNumber int
	PRURL       string
	IssueURL    string
}

type ErrorData struct {
	Title        string
	ErrorMessage string
}

// Helper methods for URL building
func (s *SlackManager) buildIssueURL(issueNumber int) string {
	return fmt.Sprintf("https://github.com/%s/issues/%d", s.githubConfig.Repository, issueNumber)
}

func (s *SlackManager) buildPRURL(prNumber int) string {
	return fmt.Sprintf("https://github.com/%s/pull/%d", s.githubConfig.Repository, prNumber)
}

func (s *SlackManager) sendBlockMessage(templateName string, data interface{}) {
	go func() {
		blockData, err := s.templateManager.RenderTemplate(templateName, data)
		if err != nil {
			s.logger.Warn(context.Background(), "Failed to render Slack template",
				logging.Field{Key: "template", Value: templateName},
				logging.Field{Key: "error", Value: err.Error()},
			)
			return
		}

		if err := s.client.SendBlockMessage(blockData); err != nil {
			s.logger.Warn(context.Background(), "Failed to send Slack block notification",
				logging.Field{Key: "template", Value: templateName},
				logging.Field{Key: "error", Value: err.Error()},
			)
		} else {
			s.logger.Info(context.Background(), "Slack notification sent",
				logging.Field{Key: "template", Value: templateName},
			)
		}
	}()
}

// Implementation methods
func (s *SlackManager) NotifyPhaseStart(phase string, issueNumber int) {
	s.logger.Debug(context.Background(), "Sending phase start notification",
		logging.Field{Key: "phase", Value: phase},
		logging.Field{Key: "issueNumber", Value: issueNumber},
	)
	// Ensure repository is set, use a fallback if empty
	repository := s.githubConfig.Repository
	if repository == "" {
		s.logger.Warn(context.Background(), "Repository not configured for NotifyPhaseStart",
			logging.Field{Key: "phase", Value: phase},
			logging.Field{Key: "issueNumber", Value: issueNumber},
		)
		repository = "unknown/repository"
	}

	data := PhaseStartData{
		Phase:       phase,
		IssueNumber: issueNumber,
		IssueURL:    s.buildIssueURL(issueNumber),
		Repository:  repository,
	}
	s.sendBlockMessage("phase_start", data)
}

func (s *SlackManager) NotifyPRMerged(prNumber, issueNumber int) {
	s.logger.Debug(context.Background(), "Sending PR merged notification",
		logging.Field{Key: "prNumber", Value: prNumber},
		logging.Field{Key: "issueNumber", Value: issueNumber},
	)
	data := PRMergedData{
		PRNumber:    prNumber,
		IssueNumber: issueNumber,
		PRURL:       s.buildPRURL(prNumber),
		IssueURL:    s.buildIssueURL(issueNumber),
	}
	s.sendBlockMessage("pr_merged", data)
}

func (s *SlackManager) NotifyError(title, errorMessage string) {
	s.logger.Debug(context.Background(), "Sending error notification",
		logging.Field{Key: "title", Value: title},
		logging.Field{Key: "error", Value: errorMessage},
	)
	data := ErrorData{
		Title:        title,
		ErrorMessage: errorMessage,
	}
	s.sendBlockMessage("error", data)
}

func (s *SlackManager) Notify(text string) {
	s.logger.Debug(context.Background(), "Sending general notification",
		logging.Field{Key: "text", Value: text},
	)
	data := NotifyData{
		Text: text,
	}
	s.sendBlockMessage("notify", data)
}

func (s *SlackManager) IsEnabled() bool {
	return true
}

// Package-level convenience functions
func NotifyPhaseStart(phase string, issueNumber int) {
	GetManager().NotifyPhaseStart(phase, issueNumber)
}

func NotifyPRMerged(prNumber, issueNumber int) {
	GetManager().NotifyPRMerged(prNumber, issueNumber)
}

func NotifyError(title, errorMessage string) {
	GetManager().NotifyError(title, errorMessage)
}

func Notify(text string) {
	GetManager().Notify(text)
}

func IsEnabled() bool {
	return GetManager().IsEnabled()
}

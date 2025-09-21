package service

import (
	"context"
	"fmt"
	"time"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/domain"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/service/builder"
	"github.com/douhashi/soba/pkg/logger"
)

// GitWorkspaceManagerAdapter adapts GitWorkspaceManager to builder interface
type GitWorkspaceManagerAdapter struct {
	GitWorkspaceManager
}

func (a *GitWorkspaceManagerAdapter) PrepareWorkspace(issueNumber int) error {
	return a.GitWorkspaceManager.PrepareWorkspace(issueNumber)
}

func (a *GitWorkspaceManagerAdapter) CleanupWorkspace(issueNumber int) error {
	return a.GitWorkspaceManager.CleanupWorkspace(issueNumber)
}

// IssueProcessorAdapter adapts IssueProcessorInterface to builder interface
type IssueProcessorAdapter struct {
	IssueProcessorInterface
}

func (a *IssueProcessorAdapter) Process(ctx context.Context, cfg *config.Config) error {
	return a.IssueProcessorInterface.Process(ctx, cfg)
}

func (a *IssueProcessorAdapter) ProcessIssue(ctx context.Context, cfg *config.Config, issue github.Issue) error {
	return a.IssueProcessorInterface.ProcessIssue(ctx, cfg, issue)
}

func (a *IssueProcessorAdapter) UpdateLabels(ctx context.Context, issueNumber int, removeLabel, addLabel string) error {
	return a.IssueProcessorInterface.UpdateLabels(ctx, issueNumber, removeLabel, addLabel)
}

func (a *IssueProcessorAdapter) Configure(cfg *config.Config) error {
	return a.IssueProcessorInterface.Configure(cfg)
}

// WorkflowExecutorAdapter adapts WorkflowExecutor to builder interface
type WorkflowExecutorAdapter struct {
	WorkflowExecutor
}

func (a *WorkflowExecutorAdapter) ExecutePhase(ctx context.Context, cfg *config.Config, issueNumber int, phase interface{}) error {
	// Convert interface{} to domain.Phase if needed
	if p, ok := phase.(domain.Phase); ok {
		return a.WorkflowExecutor.ExecutePhase(ctx, cfg, issueNumber, p)
	}
	return fmt.Errorf("invalid phase type: %T", phase)
}

// IssueWatcherAdapter adapts IssueWatcher to builder interface
type IssueWatcherAdapter struct {
	*IssueWatcher
}

func (a *IssueWatcherAdapter) Start(ctx context.Context) error {
	return a.IssueWatcher.Start(ctx)
}

func (a *IssueWatcherAdapter) SetProcessor(processor builder.IssueProcessorInterface) {
	a.IssueWatcher.SetProcessor(&IssueProcessorAdapter{processor})
}

func (a *IssueWatcherAdapter) SetQueueManager(manager interface{}) {
	if qm, ok := manager.(*QueueManager); ok {
		a.IssueWatcher.SetQueueManager(qm)
	}
}

func (a *IssueWatcherAdapter) SetLogger(log interface{}) {
	if l, ok := log.(logger.Logger); ok {
		a.IssueWatcher.SetLogger(l)
	}
}

// PRWatcherAdapter adapts PRWatcher to builder interface
type PRWatcherAdapter struct {
	*PRWatcher
}

func (a *PRWatcherAdapter) Start(ctx context.Context) error {
	return a.PRWatcher.Start(ctx)
}

func (a *PRWatcherAdapter) SetLogger(log interface{}) {
	if l, ok := log.(logger.Logger); ok {
		a.PRWatcher.SetLogger(l)
	}
}

// ClosedIssueCleanupServiceAdapter adapts ClosedIssueCleanupService to builder interface
type ClosedIssueCleanupServiceAdapter struct {
	*ClosedIssueCleanupService
}

func (a *ClosedIssueCleanupServiceAdapter) Start(ctx context.Context) error {
	return a.ClosedIssueCleanupService.Start(ctx)
}

func (a *ClosedIssueCleanupServiceAdapter) Configure(owner, repo, sessionName string, enabled bool, interval interface{}) {
	if dur, ok := interval.(time.Duration); ok {
		a.ClosedIssueCleanupService.Configure(owner, repo, sessionName, enabled, dur)
	}
}

// DaemonServiceAdapter adapts daemonService to builder interface
type DaemonServiceAdapter struct {
	*daemonService
}

func (a *DaemonServiceAdapter) StartForeground(ctx context.Context, cfg *config.Config) error {
	return a.daemonService.StartForeground(ctx, cfg)
}

func (a *DaemonServiceAdapter) StartDaemon(ctx context.Context, cfg *config.Config) error {
	return a.daemonService.StartDaemon(ctx, cfg)
}

func (a *DaemonServiceAdapter) IsRunning() bool {
	return a.daemonService.IsRunning()
}
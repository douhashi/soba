package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/internal/service/builder"
	"github.com/douhashi/soba/pkg/logging"
)

// statusService implements builder.StatusService
type statusService struct {
	cfg          *config.Config
	tmuxClient   tmux.TmuxClient
	githubClient builder.GitHubClientInterface
}

// NewStatusService creates a new status service
func NewStatusService(cfg *config.Config, githubClient builder.GitHubClientInterface, tmuxClient tmux.TmuxClient) builder.StatusService {
	return &statusService{
		cfg:          cfg,
		githubClient: githubClient,
		tmuxClient:   tmuxClient,
	}
}

// GetStatus returns the current status of soba
func (s *statusService) GetStatus(ctx context.Context) (*builder.Status, error) {
	log := logging.NewMockLogger()
	log.Debug(ctx, "Getting soba status")

	status := &builder.Status{}

	// Get daemon status
	status.Daemon = s.getDaemonStatus()

	// Get tmux status
	status.Tmux = s.getTmuxStatus()

	// Get issues status
	issues, err := s.getIssuesStatus(ctx)
	if err != nil {
		log.Error(ctx, "Failed to get issues status", logging.Field{Key: "error", Value: err.Error()})
		// Continue even if we can't get issues
	} else {
		status.Issues = issues
	}

	return status, nil
}

// getDaemonStatus checks the daemon process status
func (s *statusService) getDaemonStatus() *builder.DaemonStatus {
	log := logging.NewMockLogger()
	status := &builder.DaemonStatus{
		Running: false,
	}

	// Check PID file
	pidFile := ".soba/soba.pid"
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		log.Debug(context.Background(), "PID file not found", logging.Field{Key: "path", Value: pidFile})
		return status
	}

	// Parse PID
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		log.Error(context.Background(), "Invalid PID in file", logging.Field{Key: "content", Value: string(pidData)})
		return status
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Debug(context.Background(), "Process not found", logging.Field{Key: "pid", Value: pid})
		return status
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		log.Debug(context.Background(), "Process not running",
			logging.Field{Key: "pid", Value: pid},
			logging.Field{Key: "error", Value: err.Error()},
		)
		return status
	}

	status.Running = true
	status.PID = pid

	// Try to get uptime (approximate)
	status.Uptime = s.getProcessUptime(pid)

	return status
}

// getProcessUptime tries to calculate process uptime
func (s *statusService) getProcessUptime(pid int) string {
	// Try to get process start time using ps command
	// #nosec G204 - PID is from our own PID file, not user input
	cmd := exec.Command("ps", "-o", "etime=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	etimeStr := strings.TrimSpace(string(output))
	if etimeStr == "" {
		return ""
	}

	// Parse elapsed time format (DD-HH:MM:SS or HH:MM:SS or MM:SS)
	return formatElapsedTime(etimeStr)
}

// formatElapsedTime formats the elapsed time string
func formatElapsedTime(etime string) string {
	// Remove any whitespace
	etime = strings.TrimSpace(etime)

	// Handle different formats
	if strings.Contains(etime, "-") {
		// DD-HH:MM:SS format
		parts := strings.Split(etime, "-")
		if len(parts) == 2 {
			days := parts[0]
			time := parts[1]
			return fmt.Sprintf("%sd %s", days, time)
		}
	}

	// Just return as is for HH:MM:SS or MM:SS
	return etime
}

// getTmuxStatus gets the tmux session status
func (s *statusService) getTmuxStatus() *builder.TmuxStatus {
	log := logging.NewMockLogger()

	// Generate session name using existing logic
	sessionName := s.generateSessionName()

	status := &builder.TmuxStatus{
		SessionName: sessionName,
		Windows:     []builder.TmuxWindow{},
	}

	// Check if session exists
	exists := s.tmuxClient.SessionExists(sessionName)
	if !exists {
		log.Debug(context.Background(), "Tmux session not found", logging.Field{Key: "name", Value: sessionName})
		return status
	}

	// For now, we cannot list windows because the interface doesn't have ListWindows
	// Return empty windows list for now

	return status
}

// generateSessionName generates the tmux session name
func (s *statusService) generateSessionName() string {
	// Use the same logic as daemon service
	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}

	// Extract repo from repository string (format: owner/repo)
	repo := "soba"
	if s.cfg.GitHub.Repository != "" {
		parts := strings.Split(s.cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			repo = parts[1]
		}
	}

	return fmt.Sprintf("soba-%s-%s", user, repo)
}

// getOwnerAndRepo extracts owner and repo from repository configuration
func (s *statusService) getOwnerAndRepo() (string, string) {
	if s.cfg.GitHub.Repository != "" {
		parts := strings.Split(s.cfg.GitHub.Repository, "/")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return "", ""
}

// getIssuesStatus gets the status of issues with soba labels
func (s *statusService) getIssuesStatus(ctx context.Context) ([]builder.IssueStatus, error) {
	log := logging.NewMockLogger()
	log.Debug(ctx, "Getting issues status")

	var statuses []builder.IssueStatus

	// Get all issues with soba labels
	options := &github.ListIssuesOptions{
		Labels: []string{}, // We'll filter manually
		State:  "open",
	}

	// Extract owner and repo from repository string
	owner, repo := s.getOwnerAndRepo()
	issues, _, err := s.githubClient.ListOpenIssues(ctx, owner, repo, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Filter issues with soba labels
	for _, issue := range issues {
		hasSobaLabel := false
		sobaState := ""

		for _, label := range issue.Labels {
			if strings.HasPrefix(label.Name, "soba:") {
				hasSobaLabel = true
				sobaState = label.Name
				break
			}
		}

		if hasSobaLabel {
			// Convert Labels to string array
			labelNames := make([]string, len(issue.Labels))
			for i, label := range issue.Labels {
				labelNames[i] = label.Name
			}

			status := builder.IssueStatus{
				Number: issue.Number,
				Title:  issue.Title,
				Labels: labelNames,
				State:  sobaState,
			}
			statuses = append(statuses, status)
		}
	}

	log.Debug(ctx, "Found issues with soba labels", logging.Field{Key: "count", Value: len(statuses)})
	return statuses, nil
}

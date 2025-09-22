package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/service"
	"github.com/douhashi/soba/internal/service/builder"
	"github.com/douhashi/soba/pkg/logging"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display the current status of soba",
		Long: `Display the current status of soba including:
- Daemon process status
- Tmux session information
- Issue processing state`,
		RunE: runStatus,
	}

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	log := logging.NewMockLogger()
	log.Debug(context.Background(), "Running status command")

	// Get the global log factory
	logFactory := GetLogFactory()

	// Create service builder
	sb := builder.NewServiceBuilder(logFactory)
	if buildErr := sb.BuildDefault(context.Background()); buildErr != nil {
		return fmt.Errorf("failed to build default services: %w", buildErr)
	}

	// Get the factory and set dependencies
	factory := sb.GetServiceFactory()
	if defaultFactory, ok := factory.(*service.DefaultServiceFactory); ok {
		clients := sb.GetClients()
		if clients != nil {
			defaultFactory.SetDependencies(sb.GetConfig(), clients.GitHubClient, clients.TmuxClient)
		}
	}

	// Get status service
	statusService := factory.CreateStatusService()
	if statusService == nil {
		return fmt.Errorf("failed to create status service")
	}

	// Get status information
	status, err := statusService.GetStatus(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Display status
	fmt.Fprint(cmd.OutOrStdout(), formatStatus(status))

	return nil
}

func formatStatus(status *builder.Status) string {
	var output strings.Builder

	// Daemon status
	if status.Daemon != nil {
		if status.Daemon.Running {
			output.WriteString(fmt.Sprintf("Daemon Status: Running (PID: %d", status.Daemon.PID))
			if status.Daemon.Uptime != "" {
				output.WriteString(fmt.Sprintf(", Uptime: %s", status.Daemon.Uptime))
			}
			output.WriteString(")\n")
		} else {
			output.WriteString("Daemon Status: Not Running\n")
		}
	}

	// Tmux session status
	if status.Tmux != nil {
		output.WriteString(fmt.Sprintf("Tmux Session: %s\n", status.Tmux.SessionName))
		if len(status.Tmux.Windows) > 0 {
			output.WriteString("\nTmux Windows:\n")
			for _, window := range status.Tmux.Windows {
				if window.IssueNumber > 0 {
					output.WriteString(fmt.Sprintf("  - Issue #%d: %s\n", window.IssueNumber, window.Name))
				} else {
					output.WriteString(fmt.Sprintf("  - %s\n", window.Name))
				}
			}
		}
	}

	// Issues status
	if len(status.Issues) > 0 {
		output.WriteString("\nActive Issues:\n")
		for _, issue := range status.Issues {
			output.WriteString(fmt.Sprintf("  #%d [%s] %s\n", issue.Number, issue.State, issue.Title))
		}
	} else {
		output.WriteString("\nNo active issues with soba labels\n")
	}

	return output.String()
}

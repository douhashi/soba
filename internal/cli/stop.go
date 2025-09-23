package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/service"
	"github.com/douhashi/soba/pkg/app"
	"github.com/douhashi/soba/pkg/logging"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon process",
		Long:  `Stop the running soba daemon process and clean up associated tmux sessions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd, args)
		},
	}

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	// Initialize app with minimal setup for stop command
	// This avoids the need for config file
	app.MustInitializeMinimal()

	daemonService := service.NewDaemonServiceForStop(app.LogFactory())
	return runStopWithService(cmd, args, daemonService)
}

// StopServiceInterface はストップサービスのインターフェース（テスト用）
type StopServiceInterface interface {
	Stop(ctx context.Context, repository string) error
}

// runStopWithService allows dependency injection for testing
func runStopWithService(cmd *cobra.Command, _ []string, daemonService StopServiceInterface) error {
	log := app.LogFactory().CreateComponentLogger("cli")

	// Try to get config from global app if available
	// But don't fail if config is not available
	var repository string
	if app.IsInitialized() {
		cfg := app.Config()
		if cfg != nil {
			repository = cfg.GitHub.Repository
		}
	}

	ctx := context.Background()

	log.Info(context.Background(), "Stopping daemon process")
	err := daemonService.Stop(ctx, repository)
	if err != nil {
		log.Error(context.Background(), "Failed to stop daemon",
			logging.Field{Key: "error", Value: err.Error()},
		)
		return err
	}

	cmd.Printf("Daemon stopped successfully\n")
	return nil
}

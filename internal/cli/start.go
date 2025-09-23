package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra/slack"
	"github.com/douhashi/soba/internal/service"
	"github.com/douhashi/soba/pkg/app"
)

func newStartCmd() *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Issue monitoring in foreground or daemon mode",
		Long: `Start Issue monitoring process. By default, runs in foreground mode.
Use -d/--daemon flag to run in daemon mode (background).
Use -v/--verbose flag to enable debug logging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, args, daemon)
		},
	}

	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run in daemon mode (background)")

	return cmd
}

func runStart(cmd *cobra.Command, args []string, daemon bool) error {
	// Create service using global LogFactory and Config
	// (app is already initialized with proper log level)
	cfg := app.Config()
	daemonService := service.NewDaemonServiceWithConfig(cfg, app.LogFactory())
	return runStartWithService(cmd, args, daemon, daemonService)
}

// DaemonServiceInterface はデーモンサービスのインターフェース（テスト用）
type DaemonServiceInterface interface {
	StartForeground(ctx context.Context, cfg *config.Config) error
	StartDaemon(ctx context.Context, cfg *config.Config) error
}

// runStartWithService allows dependency injection for testing
func runStartWithService(cmd *cobra.Command, _ []string, daemon bool, daemonService DaemonServiceInterface) error {
	log := app.LogFactory().CreateComponentLogger("cli")

	// Get config from global app
	cfg := app.Config()

	ctx := context.Background()

	if daemon {
		log.Info(ctx, "Starting Issue monitoring in daemon mode")

		// Slack通知: デーモン開始
		slack.NotifyPhaseStart("daemon-start", 0)

		err := daemonService.StartDaemon(ctx, cfg)
		if err == nil {
			cmd.Printf("Successfully started daemon mode\n")
		}
		return err
	} else {
		log.Info(ctx, "Starting Issue monitoring in foreground mode")

		// Slack通知: フォアグラウンド開始
		slack.NotifyPhaseStart("foreground-start", 0)

		err := daemonService.StartForeground(ctx, cfg)
		if err == nil {
			cmd.Printf("Issue monitoring stopped\n")
		}
		return err
	}
}

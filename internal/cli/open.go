package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/douhashi/soba/internal/infra/tmux"
)

const defaultSessionName = "soba"

type openCmd struct {
	tmuxClient      tmux.TmuxClient
	attachToSession func(sessionName string) error
}

func newOpenCmd() *cobra.Command {
	o := &openCmd{
		tmuxClient: tmux.NewClient(),
	}
	// デフォルトの実装を設定
	o.attachToSession = o.defaultAttachToSession

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open tmux session",
		Long: `Opens a tmux session with the session name calculated from the configuration.

If an existing session exists, it attaches to it.
If no session exists, it creates a new one and attaches to it.

The session name is automatically calculated from the github.repository setting in the configuration file.`,
		RunE: o.runOpen,
	}

	return cmd
}

func (o *openCmd) runOpen(cmd *cobra.Command, args []string) error {
	repository := viper.GetString("github.repository")
	sessionName := o.generateSessionName(repository)

	if o.tmuxClient.SessionExists(sessionName) {
		fmt.Printf("Attaching to session '%s'\n", sessionName)
		return o.attachToSession(sessionName)
	}

	fmt.Printf("Creating session '%s'\n", sessionName)
	if err := o.tmuxClient.CreateSession(sessionName); err != nil {
		return fmt.Errorf("Failed to create session: %w", err)
	}

	return o.attachToSession(sessionName)
}

func (o *openCmd) generateSessionName(repository string) string {
	if repository == "" {
		return defaultSessionName
	}

	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return defaultSessionName
	}

	// 空文字列の部分を除外
	validParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}

	if len(validParts) < 2 {
		return defaultSessionName
	}

	return defaultSessionName + "-" + strings.Join(validParts, "-")
}

func (o *openCmd) defaultAttachToSession(sessionName string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "soba version %s\n", Version)
			fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", Commit)
			fmt.Fprintf(cmd.OutOrStdout(), "  date:   %s\n", Date)
		},
	}
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the devpulse version and build metadata",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "devpulse %s\n", version)
		_, _ = fmt.Fprintf(w, "commit: %s\n", commit)
		_, _ = fmt.Fprintf(w, "built: %s\n", buildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

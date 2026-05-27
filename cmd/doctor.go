package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run DevPulse diagnostics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Log("INFO", "doctor", "diagnostics_stub")
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "diagnostics will go here")
		return err
	},
}

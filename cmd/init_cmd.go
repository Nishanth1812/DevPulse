package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize DevPulse workspace files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		created, path, hasAPIKey, err := manager.InitializeGoalsFile()
		if err != nil {
			logger.LogError("init", err)
			return err
		}

		if created {
			logger.Log("INFO", "init", fmt.Sprintf("goals_created path=%s", path))
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path); err != nil {
				return err
			}
		} else {
			logger.Log("INFO", "init", fmt.Sprintf("goals_exists path=%s", path))
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Skipped existing %s\n", path); err != nil {
				return err
			}
		}

		if !hasAPIKey {
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "No API key stored. Run: devpulse auth")
			return err
		}

		return nil
	},
}

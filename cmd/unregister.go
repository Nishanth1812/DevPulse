package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var unregisterCmd = &cobra.Command{
	Use:   "unregister <repo>",
	Short: "Unregister a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.UnregisterRepository(name); err != nil {
			logger.Log("WARN", "unregister", fmt.Sprintf("repo_unregister_failed name=%s error=%s", name, err.Error()))
			return err
		}

		logger.Log("INFO", "unregister", fmt.Sprintf("repo_unregistered name=%s", name))
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Unregistered %s\n", name)
		return err
	},
}

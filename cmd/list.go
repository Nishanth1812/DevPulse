package cmd

import (
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos := manager.ListRepositories()
		logger.Log("INFO", "list", "repo_list_rendered")
		return output.RenderRepoTable(cmd.OutOrStdout(), repos)
	},
}

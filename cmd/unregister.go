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
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		repos := manager.ListRepositories()
		var names []string
		for _, r := range repos {
			names = append(names, r.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		result, err := fuzzyMatch(name)
		if err != nil {
			return fmt.Errorf("unregister: %w", err)
		}
		if len(result.Candidates) > 0 {
			printCandidates(cmd.OutOrStdout(), name, result.Candidates)
			return nil
		}
		name = result.Matched

		if err := manager.UnregisterRepository(name); err != nil {
			logger.Log("WARN", "unregister", fmt.Sprintf("repo_unregister_failed name=%s error=%s", name, err.Error()))
			return err
		}

		logger.Log("INFO", "unregister", fmt.Sprintf("repo_unregistered name=%s", name))
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Unregistered %s\n", name)
		return err
	},
}

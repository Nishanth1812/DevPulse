package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register <path>",
	Short: "Register a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := manager.RegisterRepository(args[0])
		if err != nil {
			logger.LogError("register", err)
			return err
		}

		logger.Log("INFO", "register", fmt.Sprintf("repo_registered name=%s path=%s", repo.Name, repo.Path))
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Registered %s\n", repo.Name)
		return err
	},
}

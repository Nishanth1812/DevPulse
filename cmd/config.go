package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read and update DevPulse configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Read a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		value, err := manager.Get(args[0])
		if err != nil {
			logger.LogError("config.get", err)
			return err
		}

		logger.Log("INFO", "config.get", fmt.Sprintf("config_get key=%s", args[0]))
		_, err = fmt.Fprintln(cmd.OutOrStdout(), value)
		return err
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := manager.Set(args[0], args[1]); err != nil {
			logger.LogError("config.set", err)
			return err
		}

		logger.Log("INFO", "config.set", fmt.Sprintf("config_set key=%s", args[0]))
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated %s\n", args[0])
		return err
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

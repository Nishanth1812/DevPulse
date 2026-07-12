package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/services"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use: "debug",
}

var debugCollectCmd = &cobra.Command{
	Use: "collect <repo-path>",
	RunE: func(cmd *cobra.Command, args []string) error {

		service := services.CollectService{}

		data, err := service.Run(args[0])
		if err != nil {
			return err
		}

		output, err := json.MarshalIndent(
			data,
			"",
			"  ",
		)

		if err != nil {
			return err
		}

		fmt.Println(string(output))

		return nil
	},
}

func init() {
	debugCmd.AddCommand(debugCollectCmd)

	rootCmd.AddCommand(debugCmd)
}

package cmd

import (
	"fmt"
	"os"

	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var authAdd bool

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Store Gemini API key securely",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.PromptAndStoreGeminiAPIKey(os.Stdin, cmd.ErrOrStderr()); err != nil {
			logger.LogError("auth", err)
			return err
		}

		logger.Log("INFO", "auth", "gemini_api_key_stored account=gemini-api-key-1")
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "API key stored securely")
		return err
	},
}

func init() {
	authCmd.Flags().BoolVar(&authAdd, "add", false, "store or replace the Gemini API key")
}

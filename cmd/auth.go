package cmd

import (
	"fmt"
	"os"

	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Store an API key securely in the OS keychain",
	Long: `Store an API key for an AI provider in the OS keychain.

Use --provider (or -p) to specify which provider to store the key for.
Defaults to groq. Examples:

  devpulse auth
  devpulse auth -p gemini`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.PromptAndStoreAPIKey(provider, os.Stdin, cmd.ErrOrStderr()); err != nil {
			logger.LogError("auth", err)
			return err
		}

		logger.Log("INFO", "auth", fmt.Sprintf("api_key_stored provider=%s", provider))
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s API key stored securely\n", provider)
		return err
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}

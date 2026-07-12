package cmd

import (
	"fmt"
	"os"

	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/spf13/cobra"
)

var (
	version    = "dev"
	verbose    bool
	noColor    bool
	dryRun     bool
	redactDiff bool
	provider   string
	manager    *config.Manager
)

var rootCmd = &cobra.Command{
	Use:           "devpulse",
	Short:         "DevPulse tracks repositories and prepares development briefings",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		loaded, err := config.Load()
		if err != nil {
			return err
		}
		manager = loaded

		if err := logger.Init(manager.LogsDir()); err != nil {
			return err
		}

		if verbose {
			logger.Log("DEBUG", commandName(cmd), "verbose=true")
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("devpulse {{.Version}}\n")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color and interactive terminal effects")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print the prompt and estimated token count without making an API call")
	rootCmd.PersistentFlags().BoolVar(&redactDiff, "redact-diff", false, "strip diff content from prompts (only commit messages and plan summaries are sent)")
	rootCmd.PersistentFlags().StringVarP(&provider, "provider", "p", "groq", "AI provider (groq or gemini)")

	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(unregisterCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(configCmd)
}

func commandName(cmd *cobra.Command) string {
	if cmd == nil {
		return "devpulse"
	}
	return cmd.CommandPath()
}

package cmd

import (
	"fmt"
	"os"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/output"
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

// resolveModel returns the model to use for a command: the configured
// model.fast / model.deep value when set and compatible with the active
// provider, otherwise the provider's built-in default. Deep models are used by
// commands that need deeper reasoning (resume).
func resolveModel(command string, deep bool) string {
	key := "model.fast"
	if deep {
		key = "model.deep"
	}

	configured, err := manager.Get(key)
	if err != nil {
		configured = ""
	}

	model, ignored := ai.ResolveModel(provider, configured, deep)
	if ignored {
		logger.Log("WARN", command, fmt.Sprintf("ignoring %s=%s: not compatible with provider %s", key, configured, provider))
	}
	return model
}

// newClientFactory builds the lazy AI client factory used by every command.
// The client is only constructed on a cache miss, so cached runs never touch
// the API key or network.
func newClientFactory(command string, deep bool) ai.ClientFactory {
	return func() (ai.Client, error) {
		apiKey, err := config.GetAPIKey(provider)
		if err != nil {
			return nil, err
		}
		return ai.NewClient(provider, apiKey, resolveModel(command, deep))
	}
}

// spinnerFactory returns the spinner wrapper for the shared pipeline.
func spinnerFactory() func(message string) func() {
	return func(msg string) func() {
		s := output.NewSpinner(noColor)
		s.Start(msg)
		return s.Stop
	}
}

// goalsLoader returns a loader that reads the project's goals file, returning
// empty data when it does not exist.
func goalsLoader() func() models.GoalsData {
	return func() models.GoalsData {
		goals, err := collector.ParseGoals()
		if err != nil {
			logger.Log("DEBUG", "goals", "goals not found: "+err.Error())
			return models.GoalsData{}
		}
		return goals
	}
}

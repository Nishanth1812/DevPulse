package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a Conventional Commit message from staged changes",
	Args:  cobra.NoArgs,
	RunE:  runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
	if redactDiff {
		return fmt.Errorf("commit: --redact-diff cannot be used with the commit command (diff is required to generate a message)")
	}

	diffBytes, err := exec.Command("git", "diff", "--staged").Output()
	if err != nil {
		return fmt.Errorf("commit: run git diff --staged: %w", err)
	}
	diff := strings.TrimSpace(string(diffBytes))
	if diff == "" {
		return fmt.Errorf("commit: no staged changes found; run git add first")
	}

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "commit",
		Provider:    provider,
		NewClient:   newClientFactory("commit", false),
		Cache:       nil,
		CacheMaxAge: 0,
		DryRun:      dryRun,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   func() models.GoalsData { return models.GoalsData{} },
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildCommitPrompt(diff) },
		Parse: func(raw string) (any, error) {
			return ai.ParseCommitResponse(raw)
		},
		DryRunInfo: func(prompt string, goals models.GoalsData) string {
			return fmt.Sprintf("Estimated tokens: ~%d", ai.EstimateTokens(prompt))
		},
	})
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	return renderCommit(cmd.OutOrStdout(), data.(ai.CommitResponse))
}

func renderCommit(w io.Writer, result ai.CommitResponse) error {
	if _, err := fmt.Fprintln(w, result.Subject); err != nil {
		return err
	}
	if strings.TrimSpace(result.Body) != "" {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w, result.Body)
		return err
	}
	return nil
}

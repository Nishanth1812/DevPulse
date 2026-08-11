package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/Nishanth1812/devpulse/internal/security"
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

	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		logger.LogError("commit", err)
		return err
	}

	client, err := ai.NewClient(provider, apiKey, resolveModel("commit", false))
	if err != nil {
		return fmt.Errorf("commit: initialize AI client: %w", err)
	}

	prompt := ai.BuildCommitPrompt(diff)

	scanResult := security.ScanPrompt(prompt)
	if scanResult.ContainsSecrets {
		logger.Log("WARN", "commit", fmt.Sprintf("sensitive_content_redacted count=%d", len(scanResult.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected and redacted before sending")
		prompt = scanResult.RedactedPrompt
	}

	if dryRun {
		w := cmd.OutOrStdout()
		estTokens := ai.EstimateTokens(prompt)
		_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
		_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
		_, _ = fmt.Fprintf(w, "Estimated tokens: ~%d\n\n", estTokens)
		_, _ = fmt.Fprintf(w, "%s\n\n", prompt)
		_, _ = fmt.Fprintf(w, "=== END DRY RUN ===\n")
		return nil
	}

	spinner := output.NewSpinner(noColor)
	spinner.Start("Generating commit message…")
	raw, err := client.Generate(cmd.Context(), prompt)
	spinner.Stop()
	if err != nil {
		logger.LogError("commit", err)
		return fmt.Errorf("commit: AI call failed: %w", err)
	}

	responseScan := security.ScanPrompt(raw)
	if responseScan.ContainsSecrets {
		logger.Log("WARN", "commit", fmt.Sprintf("sensitive_content_in_response count=%d", len(responseScan.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected in AI response and redacted")
		raw = responseScan.RedactedPrompt
	}

	result, err := ai.ParseCommitResponse(raw)
	if err != nil {
		logger.LogError("commit", err)
		return err
	}

	w := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(w, result.Subject); err != nil {
		return err
	}
	if strings.TrimSpace(result.Body) != "" {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, result.Body)
		return err
	}

	return nil
}

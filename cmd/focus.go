package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/cache"
	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/Nishanth1812/devpulse/internal/security"
	"github.com/spf13/cobra"
)

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Cross-repo triage: rank all repos by completion proximity",
	Long: `Makes one prompt for all registered repos combined.
Returns a ranked list with one-line justifications and urgency markers
where a deadline within 14 days is found in the goals file.`,
	Args: cobra.NoArgs,
	RunE: runFocus,
}

func init() {
	rootCmd.AddCommand(focusCmd)
}

func runFocus(cmd *cobra.Command, args []string) error {
	repos := manager.ListRepositories()
	if len(repos) == 0 {
		return fmt.Errorf("focus: no repositories registered; run: devpulse register <path>")
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting data from all repos...")
	var repoDataList []models.RepoData
	var collectErrors []string
	for _, r := range repos {
		data, err := collector.CollectRepo(r.Path, models.CollectOptions{
			MaxCommits:  10,
			IncludeDiff: !redactDiff,
		})
		if err != nil {
			collectErrors = append(collectErrors, fmt.Sprintf("%s: %s", r.Name, err.Error()))
			continue
		}
		repoDataList = append(repoDataList, data)
	}
	collectSpinner.Stop()

	if len(collectErrors) > 0 {
		for _, e := range collectErrors {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", e)
		}
	}

	if len(repoDataList) == 0 {
		return fmt.Errorf("focus: failed to collect data from any registered repository")
	}

	focusCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "focus", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour

	// Use a synthetic key for focus (hash of all repo names + HEAD SHAs)
	focusKey := "focus"
	if focusCache != nil {
		if rawJSON, ok := focusCache.GetRaw(focusKey, focusKey, provider, "focus", cacheMaxAge); ok {
			logger.LogCacheEvent("focus", focusKey, "hit")
			var cached ai.FocusResponse
			if err := json.Unmarshal(rawJSON, &cached); err == nil {
				return renderFocus(cmd.OutOrStdout(), cached)
			}
		}
		logger.LogCacheEvent("focus", focusKey, "miss")
	}

	goalsSpinner := output.NewSpinner(noColor)
	goalsSpinner.Start("Loading goals...")
	goals, err := collector.ParseGoals()
	goalsSpinner.Stop()
	if err != nil {
		logger.Log("DEBUG", "focus", "goals not found: "+err.Error())
		goals = models.GoalsData{}
	}

	prompt := ai.BuildFocusPrompt(repoDataList, goals)

	scanResult := security.ScanPrompt(prompt)
	if scanResult.ContainsSecrets {
		logger.Log("WARN", "focus", fmt.Sprintf("sensitive_content_redacted count=%d", len(scanResult.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected and redacted before sending")
		prompt = scanResult.RedactedPrompt
	}

	if dryRun {
		w := cmd.OutOrStdout()
		estTokens := len(prompt) / 4
		_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
		_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
		_, _ = fmt.Fprintf(w, "Repos: %d\n", len(repoDataList))
		_, _ = fmt.Fprintf(w, "Estimated tokens: ~%d\n\n", estTokens)
		_, _ = fmt.Fprintf(w, "%s\n\n", prompt)
		_, _ = fmt.Fprintf(w, "=== END DRY RUN ===\n")
		return nil
	}

	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		logger.LogError("focus", err)
		return err
	}

	client, err := ai.NewClient(provider, apiKey, "")
	if err != nil {
		return fmt.Errorf("focus: initialize AI client: %w", err)
	}

	aiSpinner := output.NewSpinner(noColor)
	aiSpinner.Start("Generating focus ranking...")
	raw, err := client.Generate(cmd.Context(), prompt)
	aiSpinner.Stop()
	if err != nil {
		logger.LogError("focus", err)
		return fmt.Errorf("focus: AI call failed: %w", err)
	}

	responseScan := security.ScanPrompt(raw)
	if responseScan.ContainsSecrets {
		logger.Log("WARN", "focus", fmt.Sprintf("sensitive_content_in_response count=%d", len(responseScan.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected in AI response and redacted")
		raw = responseScan.RedactedPrompt
	}

	focus, err := ai.ParseFocusResponse(raw)
	if err != nil {
		logger.LogError("focus", err)
		return err
	}

	if focusCache != nil {
		if data, err := json.Marshal(focus); err == nil {
			if storeErr := focusCache.PutRaw(focusKey, focusKey, provider, "focus", data); storeErr != nil {
				logger.Log("WARN", "focus", "cache_store_failed: "+storeErr.Error())
			}
		}
	}

	logger.Log("INFO", "focus", fmt.Sprintf("repos=%d provider=%s", len(repoDataList), provider))
	return renderFocus(cmd.OutOrStdout(), focus)
}

func renderFocus(w io.Writer, f ai.FocusResponse) error {
	if _, err := fmt.Fprintf(w, "\n=== Focus: Repo Ranking ===\n\n"); err != nil {
		return err
	}

	for i, item := range f.Ranked {
		urgency := ""
		if item.Urgency {
			urgency = " [URGENT]"
		}
		if _, err := fmt.Fprintf(w, "%d. %s (score: %d/5)%s\n", i+1, item.RepoName, item.ProximityScore, urgency); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "   %s\n", item.RankReason); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

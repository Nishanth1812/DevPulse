package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/cache"
	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/Nishanth1812/devpulse/internal/security"
	"github.com/spf13/cobra"
	"path/filepath"
	"time"
)

var briefCmd = &cobra.Command{
	Use:   "brief <repo-name>",
	Short: "Generate an AI-powered development brief for a registered repository",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		repos := manager.ListRepositories()
		var names []string
		for _, r := range repos {
			names = append(names, r.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runBrief,
}

func init() {
	rootCmd.AddCommand(briefCmd)
}

func runBrief(cmd *cobra.Command, args []string) error {
	repoName := args[0]

	repoPath, ok := manager.RepositoryPath(repoName)
	if !ok {
		return fmt.Errorf("repository %q is not registered; run: devpulse register <path>", repoName)
	}

	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		logger.LogError("brief", err)
		return err
	}

	client, err := ai.NewClient(provider, apiKey, "")
	if err != nil {
		return fmt.Errorf("brief: initialize AI client: %w", err)
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting repository data…")
	repoData, err := collector.CollectRepo(repoPath, models.CollectOptions{
		MaxCommits:  20,
		IncludeDiff: !redactDiff,
	})
	collectSpinner.Stop()
	if err != nil {
		logger.LogError("brief", err)
		return fmt.Errorf("brief: collect repository: %w", err)
	}

	briefCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "brief", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour
	if briefCache != nil {
		if rawJSON, ok := briefCache.GetRaw(repoName, repoData.HeadSHA, provider, "brief", cacheMaxAge); ok {
			logger.LogCacheEvent("brief", repoName, "hit")
			var cached ai.BriefResponse
			if err := json.Unmarshal(rawJSON, &cached); err == nil {
				return renderBrief(cmd.OutOrStdout(), repoData, cached)
			}
		}
		logger.LogCacheEvent("brief", repoName, "miss")
	}

	goalsSpinner := output.NewSpinner(noColor)
	goalsSpinner.Start("Loading goals…")
	goals, err := collector.ParseGoals()
	goalsSpinner.Stop()
	if err != nil {
		logger.Log("DEBUG", "brief", "goals not found: "+err.Error())
		goals = models.GoalsData{}
	}

	prompt := ai.BuildBriefPrompt(repoData, goals)

	scanResult := security.ScanPrompt(prompt)
	if scanResult.ContainsSecrets {
		logger.Log("WARN", "brief", fmt.Sprintf("sensitive_content_redacted count=%d", len(scanResult.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected and redacted before sending")
		prompt = scanResult.RedactedPrompt
	}

	if dryRun {
		w := cmd.OutOrStdout()
		estTokens := len(prompt) / 4
		_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
		_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
		_, _ = fmt.Fprintf(w, "Estimated tokens: ~%d\n\n", estTokens)
		_, _ = fmt.Fprintf(w, "%s\n\n", prompt)
		_, _ = fmt.Fprintf(w, "=== END DRY RUN ===\n")
		return nil
	}

	aiSpinner := output.NewSpinner(noColor)
	aiSpinner.Start("Generating brief…")
	raw, err := client.Generate(cmd.Context(), prompt)
	aiSpinner.Stop()
	if err != nil {
		logger.LogError("brief", err)
		return fmt.Errorf("brief: AI call failed: %w", err)
	}

	responseScan := security.ScanPrompt(raw)
	if responseScan.ContainsSecrets {
		logger.Log("WARN", "brief", fmt.Sprintf("sensitive_content_in_response count=%d", len(responseScan.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected in AI response and redacted")
		raw = responseScan.RedactedPrompt
	}

	brief, err := ai.ParseBriefResponse(raw)
	if err != nil {
		logger.LogError("brief", err)
		return err
	}

	if briefCache != nil {
		if data, err := json.Marshal(brief); err == nil {
			if storeErr := briefCache.PutRaw(repoName, repoData.HeadSHA, provider, "brief", data); storeErr != nil {
				logger.Log("WARN", "brief", "cache_store_failed: "+storeErr.Error())
			}
		}
	}

	logger.Log("INFO", "brief", fmt.Sprintf("repo=%s branch=%s provider=%s", repoData.Name, repoData.Branch, provider))
	return renderBrief(cmd.OutOrStdout(), repoData, brief)
}

func renderBrief(w io.Writer, repo models.RepoData, b ai.BriefResponse) error {
	if _, err := fmt.Fprintf(w, "\n=== Brief: %s (%s) ===\n\n", repo.Name, repo.Branch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", b.Summary); err != nil {
		return err
	}

	if len(b.KeyChanges) > 0 {
		if _, err := fmt.Fprintln(w, "\nKey Changes:"); err != nil {
			return err
		}
		for _, c := range b.KeyChanges {
			if _, err := fmt.Fprintf(w, "  • %s\n", c); err != nil {
				return err
			}
		}
	}

	if b.CurrentFocus != "" {
		if _, err := fmt.Fprintf(w, "\nCurrent Focus: %s\n", b.CurrentFocus); err != nil {
			return err
		}
	}

	if len(b.Blockers) > 0 {
		if _, err := fmt.Fprintln(w, "\nBlockers:"); err != nil {
			return err
		}
		for _, bl := range b.Blockers {
			if _, err := fmt.Fprintf(w, "  ! %s\n", bl); err != nil {
				return err
			}
		}
	}

	if len(b.NextSteps) > 0 {
		if _, err := fmt.Fprintln(w, "\nNext Steps:"); err != nil {
			return err
		}
		for _, ns := range b.NextSteps {
			if _, err := fmt.Fprintf(w, "  → %s\n", ns); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/cache"
	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/fuzzy"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/Nishanth1812/devpulse/internal/security"
	"github.com/spf13/cobra"
	"path/filepath"
)

var since string

var resumeCmd = &cobra.Command{
	Use:   "resume <partial-name>",
	Short: "Deep context recovery for a single repository",
	Long: `Reads the last two to three weeks of diffs and produces a narrative:
what you built, what you started but did not finish, and what the natural next step is.
Accepts partial names — typing "acm" matches "ACM-APP-BACKEND".`,
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		repos := manager.ListRepositories()
		var names []string
		for _, r := range repos {
			names = append(names, r.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runResume,
}

func init() {
	rootCmd.AddCommand(resumeCmd)
	resumeCmd.Flags().StringVar(&since, "since", "", "only include commits after this date (YYYY-MM-DD)")
}

func runResume(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Resolve repo name via fuzzy matching
	repos := manager.ListRepositories()
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name
	}

	threshold := 50
	if t, err := manager.Get("fuzzy.threshold"); err == nil {
		fmt.Sscanf(t, "%d", &threshold)
	}

	result, err := fuzzy.Match(query, names, threshold)
	if err != nil {
		return fmt.Errorf("resume: %w", err)
	}

	// If multiple candidates, show numbered list and exit
	if len(result.Candidates) > 0 {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Multiple repositories match %q:\n\n", query)
		for i, name := range result.Candidates {
			_, _ = fmt.Fprintf(w, "  %d. %s\n", i+1, name)
		}
		_, _ = fmt.Fprintln(w, "\nPlease use a more specific name.")
		return nil
	}

	repoName := result.Matched

	repoPath, ok := manager.RepositoryPath(repoName)
	if !ok {
		return fmt.Errorf("resume: repository %q is not registered; run: devpulse register <path>", repoName)
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting repository history...")
	opts := models.CollectOptions{
		MaxCommits:      50,
		FullDiffCommits: 15,
		IncludeDiff:     !redactDiff,
	}
	if since != "" {
		t, err := time.Parse("2006-01-02", since)
		if err != nil {
			return fmt.Errorf("resume: invalid --since date %q: use YYYY-MM-DD format", since)
		}
		opts.Since = &t
	}
	repoData, err := collector.CollectRepo(repoPath, opts)
	collectSpinner.Stop()
	if err != nil {
		logger.LogError("resume", err)
		return fmt.Errorf("resume: collect repository: %w", err)
	}

	resumeCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "resume", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour
	cacheKey := cache.Hash(repoData.HeadSHA, repoData.PlanSummary, since, fmt.Sprintf("%t", redactDiff))
	if !dryRun && resumeCache != nil {
		if rawJSON, ok := resumeCache.GetRaw(repoName, cacheKey, provider, "resume", cacheMaxAge); ok {
			logger.LogCacheEvent("resume", repoName, "hit")
			var cached ai.ResumeResponse
			if err := json.Unmarshal(rawJSON, &cached); err == nil {
				return renderResume(cmd.OutOrStdout(), repoData, cached)
			}
		}
		logger.LogCacheEvent("resume", repoName, "miss")
	}

	goalsSpinner := output.NewSpinner(noColor)
	goalsSpinner.Start("Loading goals...")
	goals, err := collector.ParseGoals()
	goalsSpinner.Stop()
	if err != nil {
		logger.Log("DEBUG", "resume", "goals not found: "+err.Error())
		goals = models.GoalsData{}
	}

	prompt := ai.BuildResumePrompt(repoData, goals)

	scanResult := security.ScanPrompt(prompt)
	if scanResult.ContainsSecrets {
		logger.Log("WARN", "resume", fmt.Sprintf("sensitive_content_redacted count=%d", len(scanResult.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected and redacted before sending")
		prompt = scanResult.RedactedPrompt
	}

	if dryRun {
		w := cmd.OutOrStdout()
		estTokens := ai.EstimateTokens(prompt)
		bk := ai.BreakdownTokens([]models.RepoData{repoData}, goals)
		_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
		_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
		_, _ = fmt.Fprintf(w, "Estimated tokens: ~%d (diffs ~%d, plan ~%d, notes ~%d, goals ~%d)\n\n",
			estTokens, bk.Diffs, bk.Plan, bk.Notes, bk.Goals)
		_, _ = fmt.Fprintf(w, "%s\n\n", prompt)
		_, _ = fmt.Fprintf(w, "=== END DRY RUN ===\n")
		return nil
	}

	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		logger.LogError("resume", err)
		return err
	}

	client, err := ai.NewClient(provider, apiKey, resolveModel("resume", true))
	if err != nil {
		return fmt.Errorf("resume: initialize AI client: %w", err)
	}

	aiSpinner := output.NewSpinner(noColor)
	aiSpinner.Start("Generating resume...")
	raw, err := client.Generate(cmd.Context(), prompt)
	aiSpinner.Stop()
	if err != nil {
		logger.LogError("resume", err)
		return fmt.Errorf("resume: AI call failed: %w", err)
	}

	responseScan := security.ScanPrompt(raw)
	if responseScan.ContainsSecrets {
		logger.Log("WARN", "resume", fmt.Sprintf("sensitive_content_in_response count=%d", len(responseScan.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected in AI response and redacted")
		raw = responseScan.RedactedPrompt
	}

	summary, err := ai.ParseResumeResponse(raw)
	if err != nil {
		logger.LogError("resume", err)
		return err
	}

	if resumeCache != nil {
		if data, err := json.Marshal(summary); err == nil {
			if storeErr := resumeCache.PutRaw(repoName, cacheKey, provider, "resume", data); storeErr != nil {
				logger.Log("WARN", "resume", "cache_store_failed: "+storeErr.Error())
			}
		}
	}

	logger.Log("INFO", "resume", fmt.Sprintf("repo=%s branch=%s provider=%s", repoData.Name, repoData.Branch, provider))
	return renderResume(cmd.OutOrStdout(), repoData, summary)
}

func renderResume(w io.Writer, repo models.RepoData, r ai.ResumeResponse) error {
	if _, err := fmt.Fprintf(w, "\n=== Resume: %s (%s) ===\n\n", repo.Name, repo.Branch); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "What Was Built:\n%s\n\n", r.WhatWasBuilt); err != nil {
		return err
	}

	if r.WhatIsIncomplete != "" {
		if _, err := fmt.Fprintf(w, "Incomplete Work:\n%s\n\n", r.WhatIsIncomplete); err != nil {
			return err
		}
	}

	if len(r.BlockersDetected) > 0 {
		if _, err := fmt.Fprintln(w, "Blockers:"); err != nil {
			return err
		}
		for _, bl := range r.BlockersDetected {
			if _, err := fmt.Fprintf(w, "  ! %s\n", bl); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	if r.NextStep != "" {
		if _, err := fmt.Fprintf(w, "Next Step: %s\n", r.NextStep); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

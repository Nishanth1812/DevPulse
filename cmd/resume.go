package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/cache"
	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/spf13/cobra"
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

	result, err := fuzzyMatch(query)
	if err != nil {
		return fmt.Errorf("resume: %w", err)
	}
	if len(result.Candidates) > 0 {
		printCandidates(cmd.OutOrStdout(), query, result.Candidates)
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
		// Parse in local time (matching git log --since) so the boundary day
		// aligns with the user's timezone rather than UTC midnight.
		t, err := time.ParseInLocation("2006-01-02", since, time.Local)
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

	if since != "" && len(repoData.Commits) == 0 {
		return fmt.Errorf("resume: no commits found in %s after %s", repoName, since)
	}

	resumeCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "resume", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour
	model := resolveModel("resume", true)
	cacheKey := cache.Hash(repoData.Name, repoData.Branch, repoData.HeadSHA, repoData.PlanSummary, repoData.Notes, since, fmt.Sprintf("%t", redactDiff), model)

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "resume",
		Provider:    provider,
		NewClient:   newClientFactory("resume", true),
		Cache:       resumeCache,
		RepoKey:     repoName,
		CacheKey:    cacheKey,
		CacheMaxAge: cacheMaxAge,
		DryRun:      dryRun,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   goalsLoader(),
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildResumePrompt(repoData, goals) },
		Parse: func(raw string) (any, error) {
			return ai.ParseResumeResponse(raw)
		},
		DryRunInfo: func(prompt string, goals models.GoalsData) string {
			estTokens := ai.EstimateTokens(prompt)
			bk := ai.BreakdownTokens([]models.RepoData{repoData}, goals)
			return fmt.Sprintf("Estimated tokens: ~%d (diffs ~%d, plan ~%d, notes ~%d, goals ~%d)",
				estTokens, bk.Diffs, bk.Plan, bk.Notes, bk.Goals)
		},
	})
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	logger.Log("INFO", "resume", fmt.Sprintf("repo=%s branch=%s provider=%s", repoData.Name, repoData.Branch, provider))
	return renderResume(cmd.OutOrStdout(), repoData, data.(ai.ResumeResponse))
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

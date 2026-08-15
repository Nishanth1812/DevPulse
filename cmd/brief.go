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

var briefCmd = &cobra.Command{
	Use:   "brief <partial-name>",
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
	query := args[0]

	result, err := fuzzyMatch(query)
	if err != nil {
		return fmt.Errorf("brief: %w", err)
	}
	if len(result.Candidates) > 0 {
		printCandidates(cmd.OutOrStdout(), query, result.Candidates)
		return nil
	}

	repoName := result.Matched
	repoPath, ok := manager.RepositoryPath(repoName)
	if !ok {
		return fmt.Errorf("brief: repository %q is not registered; run: devpulse register <path>", repoName)
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting repository data…")
	repoData, err := collector.CollectRepo(repoPath, models.CollectOptions{
		MaxCommits:      20,
		FullDiffCommits: 10,
		IncludeDiff:     !redactDiff,
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
	cacheKey := cache.Hash(repoData.HeadSHA, repoData.PlanSummary, fmt.Sprintf("%t", redactDiff))

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "brief",
		Provider:    provider,
		NewClient:   newClientFactory("brief", false),
		Cache:       briefCache,
		RepoKey:     repoName,
		CacheKey:    cacheKey,
		CacheMaxAge: cacheMaxAge,
		DryRun:      dryRun,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   goalsLoader(),
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildBriefPrompt(repoData, goals) },
		Parse: func(raw string) (any, error) {
			return ai.ParseBriefResponse(raw)
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

	logger.Log("INFO", "brief", fmt.Sprintf("repo=%s branch=%s provider=%s", repoData.Name, repoData.Branch, provider))
	return renderBrief(cmd.OutOrStdout(), repoData, data.(ai.BriefResponse))
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

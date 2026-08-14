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
	Use:   "brief [partial-name]",
	Short: "Generate an AI-powered cross-repository or focused development brief",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
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
	if len(args) == 0 {
		return runPortfolioBrief(cmd)
	}
	return runRepositoryBrief(cmd, args[0])
}

func runRepositoryBrief(cmd *cobra.Command, query string) error {
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
	model := resolveModel("brief", false)
	cacheKey := cache.Hash(repoData.Name, repoData.Branch, repoData.HeadSHA, repoData.PlanSummary, repoData.Notes, fmt.Sprintf("%t", redactDiff), model)

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

func runPortfolioBrief(cmd *cobra.Command) error {
	repos := manager.ListRepositories()
	if len(repos) == 0 {
		return fmt.Errorf("brief: no repositories registered; run: devpulse register <path>")
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting data from all repos...")
	var repoDataList []models.RepoData
	var collectErrors []string
	for _, repo := range repos {
		data, err := collector.CollectRepo(repo.Path, models.CollectOptions{
			MaxCommits:  10,
			IncludeDiff: false,
		})
		if err != nil {
			collectErrors = append(collectErrors, fmt.Sprintf("%s: %s", repo.Name, err.Error()))
			continue
		}
		repoDataList = append(repoDataList, data)
	}
	collectSpinner.Stop()

	for _, errMessage := range collectErrors {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", errMessage)
	}
	if len(repoDataList) == 0 {
		return fmt.Errorf("brief: failed to collect data from any registered repository")
	}

	briefCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "brief", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour
	model := resolveModel("brief", false)
	keyParts := []string{"portfolio"}
	var expectedRepos []string
	for _, repo := range repoDataList {
		keyParts = append(keyParts, repo.Name, repo.Path, repo.HeadSHA, repo.PlanSummary, repo.Notes)
		expectedRepos = append(expectedRepos, repo.Name)
	}
	keyParts = append(keyParts, fmt.Sprintf("%t", redactDiff), model)
	cacheKey := cache.Hash(keyParts...)

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "brief",
		Provider:    provider,
		NewClient:   newClientFactory("brief", false),
		Cache:       briefCache,
		RepoKey:     "portfolio",
		CacheKey:    cacheKey,
		CacheMaxAge: cacheMaxAge,
		DryRun:      dryRun,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   goalsLoader(),
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildPortfolioBriefPrompt(repoDataList, goals) },
		Parse: func(raw string) (any, error) {
			return ai.ParsePortfolioBriefResponse(raw, expectedRepos)
		},
		DryRunInfo: func(prompt string, goals models.GoalsData) string {
			estTokens := ai.EstimateTokens(prompt)
			bk := ai.BreakdownTokens(repoDataList, goals)
			return fmt.Sprintf("Repos: %d\nEstimated tokens: ~%d (diffs ~%d, plan ~%d, notes ~%d, goals ~%d)",
				len(repoDataList), estTokens, bk.Diffs, bk.Plan, bk.Notes, bk.Goals)
		},
	})
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	logger.Log("INFO", "brief", fmt.Sprintf("repos=%d provider=%s", len(repoDataList), provider))
	return renderPortfolioBrief(cmd.OutOrStdout(), data.(ai.PortfolioBriefResponse))
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

func renderPortfolioBrief(w io.Writer, b ai.PortfolioBriefResponse) error {
	if _, err := fmt.Fprint(w, "\n=== Brief: Portfolio ===\n\n"); err != nil {
		return err
	}

	for _, repo := range b.Repos {
		if _, err := fmt.Fprintf(w, "## %s\n\n%s\n", repo.RepoName, repo.Summary); err != nil {
			return err
		}
		if repo.CurrentFocus != "" {
			if _, err := fmt.Fprintf(w, "Current Focus: %s\n", repo.CurrentFocus); err != nil {
				return err
			}
		}
		if len(repo.KeyChanges) > 0 {
			if _, err := fmt.Fprintln(w, "Key Changes:"); err != nil {
				return err
			}
			for _, change := range repo.KeyChanges {
				if _, err := fmt.Fprintf(w, "  • %s\n", change); err != nil {
					return err
				}
			}
		}
		if len(repo.Blockers) > 0 {
			if _, err := fmt.Fprintln(w, "Blockers:"); err != nil {
				return err
			}
			for _, blocker := range repo.Blockers {
				if _, err := fmt.Fprintf(w, "  ! %s\n", blocker); err != nil {
					return err
				}
			}
		}
		if len(repo.NextSteps) > 0 {
			if _, err := fmt.Fprintln(w, "Next Steps:"); err != nil {
				return err
			}
			for _, nextStep := range repo.NextSteps {
				if _, err := fmt.Fprintf(w, "  → %s\n", nextStep); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	return nil
}

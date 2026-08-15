package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
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
	Short: "Generate an AI-powered development brief for a registered repository",
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
	repoDataList, err := collectBriefRepos(repoName)
	if err != nil {
		return err
	}
	return runFocusedBrief(cmd, repoName, repoDataList[0])
}

func collectBriefRepos(repoQuery string) ([]models.RepoData, error) {
	repos := manager.ListRepositories()
	if strings.TrimSpace(repoQuery) != "" {
		repoPath, ok := manager.RepositoryPath(repoQuery)
		if !ok {
			return nil, fmt.Errorf("brief: repository %q is not registered; run: devpulse register <path>", repoQuery)
		}
		repos = []models.RegisteredRepo{{Name: repoQuery, Path: repoPath}}
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("brief: no repositories registered; run: devpulse register <path>")
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting repository data…")
	defer collectSpinner.Stop()

	repoDataList := make([]models.RepoData, 0, len(repos))
	for _, repo := range repos {
		repoData, err := collector.CollectRepo(repo.Path, models.CollectOptions{
			MaxCommits:      20,
			FullDiffCommits: 10,
			IncludeDiff:     !redactDiff,
		})
		if err != nil {
			return nil, fmt.Errorf("brief: collect repository %q: %w", repo.Name, err)
		}
		repoDataList = append(repoDataList, repoData)
	}
	return repoDataList, nil
}

func runFocusedBrief(cmd *cobra.Command, repoName string, repoData models.RepoData) error {

	briefCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "brief", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour
	model := resolveModel("brief", false)

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "brief",
		Provider:    provider,
		Model:       model,
		NewClient:   newClientFactory("brief", false),
		Cache:       briefCache,
		RepoKey:     repoName,
		CacheKey:    "focused:" + repoName,
		CacheInputs: []any{repoData, briefCacheOptions{RedactDiff: redactDiff, Since: "", MaxCommits: 20, FullDiffCommits: 10, IncludeDiff: !redactDiff}},
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
		Validate: func(data any, _ models.GoalsData) (any, error) {
			response, ok := data.(ai.BriefResponse)
			if !ok {
				return nil, fmt.Errorf("brief: unexpected response type %T", data)
			}
			return response, ai.ValidateBriefResponse(response)
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
	repoDataList, err := collectBriefRepos("")
	if err != nil {
		return err
	}

	portfolioCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "brief", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour
	model := resolveModel("brief", false)
	allowedRepos := make(map[string]struct{}, len(repoDataList))
	for _, repo := range repoDataList {
		allowedRepos[repo.Name] = struct{}{}
	}

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "brief",
		Provider:    provider,
		Model:       model,
		NewClient:   newClientFactory("brief", false),
		Cache:       portfolioCache,
		RepoKey:     "portfolio",
		CacheKey:    "portfolio",
		CacheInputs: []any{repoDataList, briefCacheOptions{RedactDiff: redactDiff, Since: "", MaxCommits: 20, FullDiffCommits: 10, IncludeDiff: !redactDiff}},
		CacheMaxAge: cacheMaxAge,
		DryRun:      dryRun,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   goalsLoader(),
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildPortfolioBriefPrompt(repoDataList, goals) },
		Parse: func(raw string) (any, error) {
			return ai.ParsePortfolioBriefResponse(raw)
		},
		Validate: func(data any, _ models.GoalsData) (any, error) {
			response, ok := data.(ai.PortfolioBriefResponse)
			if !ok {
				return nil, fmt.Errorf("brief: unexpected portfolio response type %T", data)
			}
			if err := ai.ValidatePortfolioBriefResponse(response, allowedRepos); err != nil {
				return nil, err
			}
			return orderPortfolioBriefResponse(response, repoDataList)
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

	response, err := orderPortfolioBriefResponse(data.(ai.PortfolioBriefResponse), repoDataList)
	if err != nil {
		return err
	}
	logger.Log("INFO", "brief", fmt.Sprintf("repos=%d provider=%s", len(repoDataList), provider))
	return renderPortfolioBrief(cmd.OutOrStdout(), response)
}

type briefCacheOptions struct {
	RedactDiff      bool
	Since           string
	MaxCommits      int
	FullDiffCommits int
	IncludeDiff     bool
}

func orderPortfolioBriefResponse(response ai.PortfolioBriefResponse, repos []models.RepoData) (ai.PortfolioBriefResponse, error) {
	items := make(map[string]ai.PortfolioBriefItem, len(response.Repos))
	for _, item := range response.Repos {
		items[item.RepoName] = item
	}

	ordered := make([]ai.PortfolioBriefItem, 0, len(repos))
	for _, repo := range repos {
		item, ok := items[repo.Name]
		if !ok {
			return ai.PortfolioBriefResponse{}, fmt.Errorf("brief: portfolio response missing repository %q", repo.Name)
		}
		ordered = append(ordered, item)
		delete(items, repo.Name)
	}
	if len(items) > 0 {
		for name := range items {
			return ai.PortfolioBriefResponse{}, fmt.Errorf("brief: portfolio response contains unexpected repository %q", name)
		}
	}
	return ai.PortfolioBriefResponse{Repos: ordered}, nil
}

func renderPortfolioBrief(w io.Writer, response ai.PortfolioBriefResponse) error {
	if _, err := fmt.Fprintln(w, "\n=== Portfolio Brief ==="); err != nil {
		return err
	}

	for _, item := range response.Repos {
		if _, err := fmt.Fprintf(w, "\n--- %s ---\n\n%s\n", item.RepoName, item.Summary); err != nil {
			return err
		}
		if item.CurrentFocus != "" {
			if _, err := fmt.Fprintf(w, "Current Focus: %s\n", item.CurrentFocus); err != nil {
				return err
			}
		}
		if len(item.Blockers) > 0 {
			if _, err := fmt.Fprintln(w, "\nBlockers:"); err != nil {
				return err
			}
			for _, blocker := range item.Blockers {
				if _, err := fmt.Fprintf(w, "  ! %s\n", blocker); err != nil {
					return err
				}
			}
		}
		if len(item.NextSteps) > 0 {
			if _, err := fmt.Fprintln(w, "\nNext Steps:"); err != nil {
				return err
			}
			for _, nextStep := range item.NextSteps {
				if _, err := fmt.Fprintf(w, "  → %s\n", nextStep); err != nil {
					return err
				}
			}
		}
	}

	_, err := fmt.Fprintln(w)
	return err
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

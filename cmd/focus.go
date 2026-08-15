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
			MaxCommits:      10,
			FullDiffCommits: 10,
			IncludeDiff:     false,
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
	model := resolveModel("focus", false)
	allowedRepos := make(map[string]struct{}, len(repoDataList))
	for _, repo := range repoDataList {
		allowedRepos[repo.Name] = struct{}{}
	}

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:   "focus",
		Provider:  provider,
		Model:     model,
		NewClient: newClientFactory("focus", false),
		Cache:     focusCache,
		RepoKey:   "focus",
		CacheKey:  "focus",
		CacheInputs: []any{repoDataList, struct {
			RedactDiff bool
			MaxCommits int
		}{redactDiff, 10}},
		CacheMaxAge: cacheMaxAge,
		DryRun:      dryRun,
		RedactDiff:  redactDiff,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   goalsLoader(),
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildFocusPrompt(repoDataList, goals) },
		Parse: func(raw string) (any, error) {
			return ai.ParseFocusResponse(raw)
		},
		Validate: func(data any, goals models.GoalsData) (any, error) {
			response, ok := data.(ai.FocusResponse)
			if !ok {
				return nil, fmt.Errorf("focus: unexpected response type %T", data)
			}
			if err := ai.ValidateFocusResponse(response, allowedRepos); err != nil {
				return nil, err
			}
			response.Ranked = ai.ApplyDeadlineUrgency(response.Ranked, goals, 14)
			return response, nil
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

	logger.Log("INFO", "focus", fmt.Sprintf("repos=%d provider=%s", len(repoDataList), provider))
	return renderFocus(cmd.OutOrStdout(), data.(ai.FocusResponse))
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

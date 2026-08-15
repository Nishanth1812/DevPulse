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

var whyCmd = &cobra.Command{
	Use:   "why <partial-name> <file>",
	Short: "File-level commit archaeology: narrate every significant decision in a file",
	Long: `Walks the full commit history for a given file and produces a narrative
of every significant decision made in it. Accepts a partial repo name the same way
resume does.`,
	Args: cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Completing repo name
			repos := manager.ListRepositories()
			var names []string
			for _, r := range repos {
				names = append(names, r.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			// Completing file path — allow normal file completion
			return nil, cobra.ShellCompDirectiveFilterDirs
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runWhy,
}

func init() {
	rootCmd.AddCommand(whyCmd)
}

func runWhy(cmd *cobra.Command, args []string) error {
	query := args[0]
	filePath := args[1]

	result, err := fuzzyMatch(query)
	if err != nil {
		return fmt.Errorf("why: %w", err)
	}
	if len(result.Candidates) > 0 {
		printCandidates(cmd.OutOrStdout(), query, result.Candidates)
		return nil
	}

	repoName := result.Matched
	repoPath, ok := manager.RepositoryPath(repoName)
	if !ok {
		return fmt.Errorf("why: repository %q is not registered", repoName)
	}

	normalizedFilePath, err := collector.NormalizeRepoRelativePath(repoPath, filePath)
	if err != nil {
		return fmt.Errorf("why: %w", err)
	}
	filePath = normalizedFilePath

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting commit history for file...")
	commits, err := collector.CollectFileCommits(repoPath, filePath, 50, 15, !redactDiff)
	collectSpinner.Stop()
	if err != nil {
		logger.LogError("why", err)
		return fmt.Errorf("why: collect file commits: %w", err)
	}

	if len(commits) == 0 {
		return fmt.Errorf("why: no commits found for file %q in %s", filePath, repoName)
	}

	whyCache, cacheErr := cache.New(filepath.Join(manager.BaseDir(), "cache"))
	if cacheErr != nil {
		logger.Log("WARN", "why", "cache_unavailable: "+cacheErr.Error())
	}
	cacheMaxAge := time.Duration(manager.CacheDurationHours()) * time.Hour

	// Cache key includes the file path, the newest commit touching it, and the
	// redact-diff setting so the archaeology invalidates when the file changes
	// or the prompt content changes.
	newestSHA := ""
	if len(commits) > 0 {
		newestSHA = commits[len(commits)-1].SHA
	}
	cacheKey := cache.Hash(repoName, filePath, newestSHA, fmt.Sprintf("%t", redactDiff))

	data, err := ai.Run(cmd.Context(), ai.RunOptions{
		Command:     "why",
		Provider:    provider,
		NewClient:   newClientFactory("why", false),
		Cache:       whyCache,
		RepoKey:     cacheKey,
		CacheKey:    cacheKey,
		CacheMaxAge: cacheMaxAge,
		DryRun:      dryRun,
		Out:         cmd.OutOrStdout(),
		ErrOut:      cmd.ErrOrStderr(),
		Spinner:     spinnerFactory(),
		LoadGoals:   goalsLoader(),
		BuildPrompt: func(goals models.GoalsData) string { return ai.BuildWhyPrompt(repoName, filePath, commits) },
		Parse: func(raw string) (any, error) {
			return ai.ParseWhyResponse(raw)
		},
		DryRunInfo: func(prompt string, goals models.GoalsData) string {
			estTokens := ai.EstimateTokens(prompt)
			diffTokens := 0
			for _, c := range commits {
				diffTokens += ai.EstimateTokens(c.DiffSnippet)
			}
			return fmt.Sprintf("File: %s\nCommits: %d\nEstimated tokens: ~%d (file diffs ~%d)",
				filePath, len(commits), estTokens, diffTokens)
		},
	})
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}

	logger.Log("INFO", "why", fmt.Sprintf("repo=%s file=%s provider=%s commits=%d", repoName, filePath, provider, len(commits)))
	return renderWhy(cmd.OutOrStdout(), repoName, filePath, data.(ai.WhyResponse))
}

func renderWhy(w io.Writer, repoName, filePath string, r ai.WhyResponse) error {
	if _, err := fmt.Fprintf(w, "\n=== Why: %s in %s ===\n\n", filePath, repoName); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Purpose: %s\n\n", r.FilePurpose); err != nil {
		return err
	}

	if len(r.MajorDecisions) > 0 {
		if _, err := fmt.Fprintln(w, "Major Decisions:"); err != nil {
			return err
		}
		for _, d := range r.MajorDecisions {
			if _, err := fmt.Fprintf(w, "  [%s] %s\n", d.Date, d.Description); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	if r.CurrentState != "" {
		if _, err := fmt.Fprintf(w, "Current State: %s\n", r.CurrentState); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

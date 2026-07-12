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
	"github.com/Nishanth1812/devpulse/internal/fuzzy"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/Nishanth1812/devpulse/internal/security"
	"github.com/spf13/cobra"
)

var whyCmd = &cobra.Command{
	Use:   "why <partial-name> <file>",
	Short: "File-level commit archaeology: narrate every significant decision in a file",
	Long: `Walks the full commit history for a given file and produces a narrative
of every significant decision made in it. Accepts a partial repo name the same way
resume does.`,
	Args: cobra.ExactArgs(2),
	RunE: runWhy,
}

func init() {
	rootCmd.AddCommand(whyCmd)
}

func runWhy(cmd *cobra.Command, args []string) error {
	query := args[0]
	filePath := args[1]

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
		return fmt.Errorf("why: %w", err)
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
		return fmt.Errorf("why: repository %q is not registered", repoName)
	}

	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		logger.LogError("why", err)
		return err
	}

	client, err := ai.NewClient(provider, apiKey, "")
	if err != nil {
		return fmt.Errorf("why: initialize AI client: %w", err)
	}

	collectSpinner := output.NewSpinner(noColor)
	collectSpinner.Start("Collecting commit history for file...")
	commits, err := collector.CollectFileCommits(repoPath, filePath, 50)
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

	// Cache key includes file path
	cacheKey := repoName + ":" + filePath
	if whyCache != nil {
		if rawJSON, ok := whyCache.GetRaw(cacheKey, cacheKey, provider, "why", cacheMaxAge); ok {
			logger.LogCacheEvent("why", cacheKey, "hit")
			var cached ai.WhyResponse
			if err := json.Unmarshal(rawJSON, &cached); err == nil {
				return renderWhy(cmd.OutOrStdout(), repoName, filePath, cached)
			}
		}
		logger.LogCacheEvent("why", cacheKey, "miss")
	}

	prompt := ai.BuildWhyPrompt(repoName, filePath, commits)

	scanResult := security.ScanPrompt(prompt)
	if scanResult.ContainsSecrets {
		logger.Log("WARN", "why", fmt.Sprintf("sensitive_content_redacted count=%d", len(scanResult.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected and redacted before sending")
		prompt = scanResult.RedactedPrompt
	}

	if dryRun {
		w := cmd.OutOrStdout()
		estTokens := len(prompt) / 4
		_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
		_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
		_, _ = fmt.Fprintf(w, "File: %s\n", filePath)
		_, _ = fmt.Fprintf(w, "Commits: %d\n", len(commits))
		_, _ = fmt.Fprintf(w, "Estimated tokens: ~%d\n\n", estTokens)
		_, _ = fmt.Fprintf(w, "%s\n\n", prompt)
		_, _ = fmt.Fprintf(w, "=== END DRY RUN ===\n")
		return nil
	}

	aiSpinner := output.NewSpinner(noColor)
	aiSpinner.Start("Generating file archaeology...")
	raw, err := client.Generate(cmd.Context(), prompt)
	aiSpinner.Stop()
	if err != nil {
		logger.LogError("why", err)
		return fmt.Errorf("why: AI call failed: %w", err)
	}

	responseScan := security.ScanPrompt(raw)
	if responseScan.ContainsSecrets {
		logger.Log("WARN", "why", fmt.Sprintf("sensitive_content_in_response count=%d", len(responseScan.Matches)))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: sensitive content detected in AI response and redacted")
		raw = responseScan.RedactedPrompt
	}

	why, err := ai.ParseWhyResponse(raw)
	if err != nil {
		logger.LogError("why", err)
		return err
	}

	if whyCache != nil {
		if data, err := json.Marshal(why); err == nil {
			if storeErr := whyCache.PutRaw(cacheKey, cacheKey, provider, "why", data); storeErr != nil {
				logger.Log("WARN", "why", "cache_store_failed: "+storeErr.Error())
			}
		}
	}

	logger.Log("INFO", "why", fmt.Sprintf("repo=%s file=%s provider=%s commits=%d", repoName, filePath, provider, len(commits)))
	return renderWhy(cmd.OutOrStdout(), repoName, filePath, why)
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

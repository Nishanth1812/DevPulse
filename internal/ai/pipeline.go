package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Nishanth1812/devpulse/internal/cache"
	"github.com/Nishanth1812/devpulse/internal/logger"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/security"
)

// ClientFactory lazily builds the AI client, so the API key is only resolved
// when a cache miss actually requires a call.
type ClientFactory func() (Client, error)

// RunOptions carries everything the shared LLM pipeline needs. Each command
// provides callbacks so the cache/prompt/parse/dry-run behaviour is identical
// everywhere while the command-specific parts stay with the command.
type RunOptions struct {
	// Command is the command name used in logs and cache keys.
	Command string
	// Provider is the active AI provider (groq or gemini).
	Provider string
	// NewClient builds the client; only called on a cache miss.
	NewClient ClientFactory
	// Cache is optional; nil disables caching.
	Cache *cache.Cache
	// RepoKey and CacheKey are the cache lookup components.
	RepoKey  string
	CacheKey string
	// CacheMaxAge bounds how long a cached entry is valid.
	CacheMaxAge time.Duration
	// DryRun prints the prompt instead of calling the API.
	DryRun bool
	// Out and ErrOut receive user-facing output and warnings.
	Out   io.Writer
	ErrOut io.Writer
	// Spinner, when set, wraps the API call; returns a stop function.
	Spinner func(message string) func()
	// LoadGoals returns the goals file content (empty on missing file).
	LoadGoals func() models.GoalsData
	// BuildPrompt builds the prompt from the loaded goals.
	BuildPrompt func(goals models.GoalsData) string
	// Parse converts the raw AI response into the typed result.
	Parse func(raw string) (any, error)
	// DryRunInfo returns extra lines for the dry-run header (e.g. token breakdown).
	DryRunInfo func(prompt string, goals models.GoalsData) string
}

// Run executes the full LLM pipeline: cache check, goals load, prompt build,
// secret redaction, dry-run, API call with retry, response redaction, parse,
// and cache store. It returns the parsed result, or nil on dry-run. On a cache
// hit the client is never constructed and the API is never called.
func Run(ctx context.Context, opts RunOptions) (any, error) {
	if !opts.DryRun && opts.Cache != nil {
		if raw, ok := opts.Cache.GetRaw(opts.RepoKey, opts.CacheKey, opts.Provider, opts.Command, opts.CacheMaxAge); ok {
			logger.LogCacheEvent(opts.Command, opts.RepoKey, "hit")
			if data, err := opts.Parse(string(raw)); err == nil {
				return data, nil
			}
		}
		logger.LogCacheEvent(opts.Command, opts.RepoKey, "miss")
	}

	goals := opts.LoadGoals()
	prompt := opts.BuildPrompt(goals)

	scan := security.ScanPrompt(prompt)
	if scan.ContainsSecrets {
		logger.Log("WARN", opts.Command, fmt.Sprintf("sensitive_content_redacted count=%d", len(scan.Matches)))
		_, _ = fmt.Fprintln(opts.ErrOut, "Warning: sensitive content detected and redacted before sending")
		prompt = scan.RedactedPrompt
	}

	if opts.DryRun {
		printDryRun(opts.Out, opts.Provider, opts.DryRunInfo(prompt, goals), prompt)
		return nil, nil
	}

	client, err := opts.NewClient()
	if err != nil {
		logger.LogError(opts.Command, err)
		return nil, err
	}

	stop := func() {}
	if opts.Spinner != nil {
		stop = opts.Spinner("Generating…")
	}
	raw, err := client.Generate(ctx, prompt)
	stop()
	if err != nil {
		logger.LogError(opts.Command, err)
		return nil, fmt.Errorf("%s: AI call failed: %w", opts.Command, err)
	}

	responseScan := security.ScanPrompt(raw)
	if responseScan.ContainsSecrets {
		logger.Log("WARN", opts.Command, fmt.Sprintf("sensitive_content_in_response count=%d", len(responseScan.Matches)))
		_, _ = fmt.Fprintln(opts.ErrOut, "Warning: sensitive content detected in AI response and redacted")
		raw = responseScan.RedactedPrompt
	}

	data, err := opts.Parse(raw)
	if err != nil {
		logger.LogError(opts.Command, err)
		return nil, err
	}

	if opts.Cache != nil {
		if payload, merr := json.Marshal(data); merr == nil {
			if serr := opts.Cache.PutRaw(opts.RepoKey, opts.CacheKey, opts.Provider, opts.Command, payload); serr != nil {
				logger.Log("WARN", opts.Command, "cache_store_failed: "+serr.Error())
			}
		}
	}

	return data, nil
}

func printDryRun(w io.Writer, provider, info, prompt string) {
	_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
	_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
	if info != "" {
		_, _ = fmt.Fprintf(w, "%s\n", info)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, prompt)
	_, _ = fmt.Fprintln(w, "=== END DRY RUN ===")
}

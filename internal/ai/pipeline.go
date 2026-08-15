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

// defaultMaxPromptTokens is the estimated-token ceiling enforced before any
// API call. It is conservative relative to modern context windows so that
// oversized prompts fail with a helpful message instead of a provider error.
const defaultMaxPromptTokens = 64_000

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
	// CacheInputs contains the complete structured prompt inputs and command
	// options. It is fingerprinted together with the command/provider/model and
	// goals before a cache lookup.
	CacheInputs []any
	// Model is the resolved model identity used for this request.
	Model string
	// CacheMaxAge bounds how long a cached entry is valid.
	CacheMaxAge time.Duration
	// MaxPromptTokens caps the estimated prompt size before the API call.
	// Zero uses the default guard. Prompts that exceed the cap fail fast
	// instead of overflowing the model's context window.
	MaxPromptTokens int
	// DryRun prints the prompt instead of calling the API.
	DryRun bool
	// RedactDiff marks that diff content was intentionally omitted from the
	// collected evidence and makes the reduced-context boundary visible.
	RedactDiff bool
	// Out and ErrOut receive user-facing output and warnings.
	Out    io.Writer
	ErrOut io.Writer
	// Spinner, when set, wraps the API call; returns a stop function.
	Spinner func(message string) func()
	// LoadGoals returns the goals file content (empty on missing file).
	LoadGoals func() models.GoalsData
	// BuildPrompt builds the prompt from the loaded goals.
	BuildPrompt func(goals models.GoalsData) string
	// Parse converts the raw AI response into the typed result.
	Parse func(raw string) (any, error)
	// Validate runs after parsing and before rendering or caching. It may return
	// a normalized value, such as focus urgency derived from parsed goals.
	Validate func(data any, goals models.GoalsData) (any, error)
	// DryRunInfo returns extra lines for the dry-run header (e.g. token breakdown).
	DryRunInfo func(prompt string, goals models.GoalsData) string
}

// Run executes the full LLM pipeline: cache check, goals load, prompt build,
// secret redaction, dry-run, API call with retry, response redaction, parse,
// and cache store. It returns the parsed result, or nil on dry-run. On a cache
// hit the client is never constructed and the API is never called.
func Run(ctx context.Context, opts RunOptions) (any, error) {
	// Goals are loaded before the cache lookup so their content can be folded
	// into the cache key: changing goals/notes must invalidate cached output.
	goals := opts.LoadGoals()

	cacheKey := opts.CacheKey
	parseAndValidate := func(raw string) (any, error) {
		data, err := opts.Parse(raw)
		if err != nil {
			return nil, err
		}
		if opts.Validate != nil {
			return opts.Validate(data, goals)
		}
		return data, nil
	}
	if !opts.DryRun && opts.Cache != nil {
		fingerprint, err := cache.Fingerprint(
			opts.Command,
			opts.Provider,
			opts.Model,
			opts.CacheKey,
			opts.CacheInputs,
			goals,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: build cache fingerprint: %w", opts.Command, err)
		}
		cacheKey = fingerprint
		if raw, ok := opts.Cache.GetRaw(opts.RepoKey, cacheKey, opts.Provider, opts.Command, opts.CacheMaxAge); ok {
			logger.LogCacheEvent(opts.Command, opts.RepoKey, "hit")
			if data, err := parseAndValidate(string(raw)); err == nil {
				return data, nil
			}
			// The cached entry is stale (unparseable). Drop it so the next run
			// refetches instead of failing on the same poisoned entry.
			logger.Log("WARN", opts.Command, "cache_parse_failed; invalidating entry")
			if derr := opts.Cache.Delete(opts.RepoKey, cacheKey, opts.Provider, opts.Command); derr != nil {
				logger.Log("WARN", opts.Command, "cache_delete_failed: "+derr.Error())
			}
		}
		logger.LogCacheEvent(opts.Command, opts.RepoKey, "miss")
	}

	prompt := opts.BuildPrompt(goals)

	// Fail fast if the prompt would overflow the model's context window.
	maxTokens := opts.MaxPromptTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxPromptTokens
	}
	if EstimateTokens(prompt) > maxTokens {
		err := fmt.Errorf("%s: prompt too large (~%d estimated tokens; cap %d). Reduce the commit window (--since / max commits) or run with --redact-diff", opts.Command, EstimateTokens(prompt), maxTokens)
		logger.LogError(opts.Command, err)
		return nil, err
	}

	scan := security.ScanPrompt(prompt)
	if scan.ContainsSecrets {
		logger.Log("WARN", opts.Command, fmt.Sprintf("sensitive_content_redacted count=%d", len(scan.Matches)))
		_, _ = fmt.Fprintln(opts.ErrOut, "Warning: sensitive content detected and redacted before sending")
		prompt = scan.RedactedPrompt
	}

	if opts.DryRun {
		printDryRun(opts.Out, opts.Provider, opts.DryRunInfo(prompt, goals), prompt, opts.RedactDiff)
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

	data, err := parseAndValidate(raw)
	if err != nil {
		logger.LogError(opts.Command, err)
		return nil, err
	}

	if opts.Cache != nil {
		if payload, merr := json.Marshal(data); merr == nil {
			if serr := opts.Cache.PutRaw(opts.RepoKey, cacheKey, opts.Provider, opts.Command, payload); serr != nil {
				logger.Log("WARN", opts.Command, "cache_store_failed: "+serr.Error())
			}
		}
	}

	return data, nil
}

// printDryRun writes a formatted dry-run dump to w.
func printDryRun(w io.Writer, provider, info, prompt string, redactDiff bool) {
	_, _ = fmt.Fprintf(w, "=== DRY RUN ===\n")
	_, _ = fmt.Fprintf(w, "Provider: %s\n", provider)
	if redactDiff {
		_, _ = fmt.Fprintln(w, "Privacy: diff content omitted (--redact-diff); context is reduced")
	}
	if info != "" {
		_, _ = fmt.Fprintf(w, "%s\n", info)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, prompt)
	_, _ = fmt.Fprintln(w, "=== END DRY RUN ===")
}

// encodeGoals produces a deterministic string for folding the goals file into
// cache keys. Deadlines include DaysUntil so a goal approaching its deadline
// naturally invalidates older cached output.
func encodeGoals(g models.GoalsData) string {
	data, err := json.Marshal(g)
	if err != nil {
		return fmt.Sprintf("%+v", g)
	}
	return string(data)
}

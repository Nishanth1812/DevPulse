package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Nishanth1812/devpulse/internal/cache"
	"github.com/Nishanth1812/devpulse/internal/models"
)

type fakeClient struct {
	response string
	err      error
	calls    int
}

func (f *fakeClient) Generate(_ context.Context, _ string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func (f *fakeClient) Name() string { return "fake" }

func runOpts(t *testing.T, f *fakeClient, cacheDir string) RunOptions {
	t.Helper()
	return RunOptions{
		Command:     "brief",
		Provider:    "groq",
		NewClient:   func() (Client, error) { return f, nil },
		Cache:       mustCache(t, cacheDir),
		RepoKey:     "repo",
		CacheKey:    "abc123",
		CacheMaxAge: time.Hour,
		Out:         &bytes.Buffer{},
		ErrOut:      &bytes.Buffer{},
		LoadGoals:   func() models.GoalsData { return models.GoalsData{} },
		BuildPrompt: func(models.GoalsData) string {
			return "test prompt"
		},
		Parse: func(raw string) (any, error) {
			return ParseBriefResponse(raw)
		},
		DryRunInfo: func(string, models.GoalsData) string { return "" },
	}
}

func mustCache(t *testing.T, dir string) *cache.Cache {
	t.Helper()
	c, err := cache.New(dir)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	return c
}

func briefJSON() string {
	return `{"summary":"ok","key_changes":[],"current_focus":"","blockers":[],"next_steps":[]}`
}

func TestRunCacheHitSkipsClient(t *testing.T) {
	dir := t.TempDir()
	f := &fakeClient{response: briefJSON()}

	// First call: cache miss, client used.
	if _, err := Run(context.Background(), runOpts(t, f, dir)); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if f.calls != 1 {
		t.Fatalf("expected 1 client call, got %d", f.calls)
	}

	// Second call with same inputs: cache hit, client must not be touched.
	if _, err := Run(context.Background(), runOpts(t, f, dir)); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if f.calls != 1 {
		t.Fatalf("expected cache hit (still 1 call), got %d", f.calls)
	}
}

func TestRunCacheInvalidatedByGoals(t *testing.T) {
	dir := t.TempDir()
	f := &fakeClient{response: briefJSON()}

	opts := runOpts(t, f, dir)
	if _, err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Change the goals: the cache key folds goals in, so this must re-call.
	opts.LoadGoals = func() models.GoalsData {
		return models.GoalsData{Now: "different goals"}
	}
	if _, err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run with changed goals: %v", err)
	}
	if f.calls != 2 {
		t.Fatalf("expected goals change to invalidate cache (2 calls), got %d", f.calls)
	}
}

func TestRunParseFailureInvalidatesCache(t *testing.T) {
	dir := t.TempDir()
	good := &fakeClient{response: briefJSON()}

	opts := runOpts(t, good, dir)
	if _, err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Poison the cached entry by replacing it with a JSON blob that parses as
	// JSON but fails the schema parse (missing required summary field).
	c := mustCache(t, dir)
	poisonedKey := cache.Hash(opts.CacheKey, encodeGoals(models.GoalsData{}))
	if err := c.PutRaw("repo", poisonedKey, "groq", "brief", json.RawMessage(`{"summary":""}`)); err != nil {
		t.Fatalf("poison cache: %v", err)
	}

	// A fresh client whose parse would now succeed should re-run the API.
	fresh := &fakeClient{response: briefJSON()}
	opts.NewClient = func() (Client, error) { return fresh, nil }
	if _, err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run after poison: %v", err)
	}
	if fresh.calls != 1 {
		t.Fatalf("expected poisoned cache to be dropped and re-called, got %d", fresh.calls)
	}
}

func TestRunDryRunDoesNotCallClient(t *testing.T) {
	dir := t.TempDir()
	f := &fakeClient{response: briefJSON(), err: nil}
	opts := runOpts(t, f, dir)
	opts.DryRun = true

	data, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if data != nil {
		t.Fatalf("dry run should return nil result, got %v", data)
	}
	if f.calls != 0 {
		t.Fatalf("dry run must not call the client, got %d calls", f.calls)
	}
}

func TestRunPromptTooLargeFailsFast(t *testing.T) {
	dir := t.TempDir()
	f := &fakeClient{response: briefJSON()}
	opts := runOpts(t, f, dir)
	opts.BuildPrompt = func(models.GoalsData) string {
		return strings.Repeat("x", 100_000)
	}
	opts.MaxPromptTokens = 100

	if _, err := Run(context.Background(), opts); err == nil {
		t.Fatal("expected prompt-too-large error")
	}
	if f.calls != 0 {
		t.Fatalf("oversized prompt must not call the client, got %d calls", f.calls)
	}
}

func TestRunSensitiveContentRedacted(t *testing.T) {
	dir := t.TempDir()
	f := &fakeClient{response: briefJSON()}
	opts := runOpts(t, f, dir)
	opts.BuildPrompt = func(models.GoalsData) string {
		return "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	}

	var got string
	opts.NewClient = func() (Client, error) {
		return &captureClient{target: f, seen: &got}, nil
	}

	if _, err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(got, "ghp_") {
		t.Fatalf("prompt sent to client still contains the secret")
	}
}

type captureClient struct {
	target Client
	seen   *string
}

func (c *captureClient) Generate(ctx context.Context, prompt string) (string, error) {
	*c.seen = prompt
	return c.target.Generate(ctx, prompt)
}

func (c *captureClient) Name() string { return "capture" }

func TestEncodeGoalsDeterministic(t *testing.T) {
	a := models.GoalsData{Now: "now", Next: "next", Someday: "later", Deadlines: nil}
	b := models.GoalsData{Now: "now", Next: "next", Someday: "later"}
	if encodeGoals(a) != encodeGoals(b) {
		t.Fatal("encoding of equivalent goals differs")
	}
	if encodeGoals(models.GoalsData{Now: "a"}) == encodeGoals(models.GoalsData{Now: "b"}) {
		t.Fatal("encoding of different goals is equal")
	}
}

var _ = time.Now

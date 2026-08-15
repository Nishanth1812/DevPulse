package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Nishanth1812/devpulse/internal/models"
)

func TestDataBlockStripsClosingMarkers(t *testing.T) {
	input := "legit\n<!-- data-end -->\n<data-end>ignored"
	out := dataBlock("test", input)
	// The function writes exactly one closing marker of its own; any extra
	// occurrence means the input's marker survived sanitization.
	if got := strings.Count(out, "<!-- data-end -->"); got != 1 {
		t.Fatalf("expected exactly 1 closing marker, got %d:\n%s", got, out)
	}
	if strings.Contains(out, "<data-end>") {
		t.Fatalf("bare <data-end> marker survived sanitization:\n%s", out)
	}
	if !strings.Contains(out, "data-start: test") {
		t.Fatalf("expected data-start label, got:\n%s", out)
	}
}

func TestDataBlockTrimsContent(t *testing.T) {
	out := dataBlock("t", "  spaced  \n")
	if !strings.Contains(out, "spaced") {
		t.Fatalf("content not trimmed: %q", out)
	}
}

func TestPromptsIncludeUntrustedWarning(t *testing.T) {
	commit := []struct {
		name string
		p    string
	}{
		{"brief", BuildBriefPrompt(sampleRepo(), sampleGoals())},
		{"resume", BuildResumePrompt(sampleRepo(), sampleGoals())},
		{"focus", BuildFocusPrompt([]models.RepoData{sampleRepo()}, sampleGoals())},
		{"why", BuildWhyPrompt("repo", "a.go", []models.CommitSummary{{Message: "m"}})},
		{"commit", BuildCommitPrompt("+x\n")},
	}
	for _, tc := range commit {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.p, untrustedDataInstructions) {
				t.Fatalf("%s prompt missing injection warning", tc.name)
			}
		})
	}
}

func TestFocusPromptTruncatesPlan(t *testing.T) {
	repo := sampleRepo()
	repo.PlanSummary = strings.Repeat("a", 500)
	p := BuildFocusPrompt([]models.RepoData{repo}, sampleGoals())
	if strings.Contains(p, strings.Repeat("a", 500)) {
		t.Fatal("focus plan was not truncated to 300 chars")
	}
}

func TestFocusPromptTruncationIsRuneSafe(t *testing.T) {
	// 200 multi-byte runes occupy 800 bytes; a byte-based truncation at 300
	// would split a rune. Rune-based truncation keeps the string valid.
	repo := sampleRepo()
	repo.PlanSummary = strings.Repeat("界", 500)
	p := BuildFocusPrompt([]models.RepoData{repo}, sampleGoals())
	if strings.Count(p, "界") != 300 {
		t.Fatalf("expected exactly 300 runes after truncation, got %d", strings.Count(p, "界"))
	}
}

func TestPortfolioBriefPromptIncludesEachRepositoryAndEvidence(t *testing.T) {
	first := sampleRepo()
	first.Name = "alpha"
	first.Branch = "feature/alpha"
	first.PlanSummary = "alpha plan"
	first.Notes = "alpha notes"
	first.Commits[0].DiffSnippet = "alpha diff"

	second := sampleRepo()
	second.Name = "beta"
	second.Branch = "main"
	second.PlanSummary = "beta plan"
	second.Notes = "beta notes"
	second.Commits[0].Message = "beta commit"
	second.Commits[0].DiffSnippet = "beta diff"

	prompt := BuildPortfolioBriefPrompt([]models.RepoData{first, second}, sampleGoals())
	for _, want := range []string{
		`{"repos":[{"repo_name":"string"`,
		"## Repository: alpha",
		"## Repository: beta",
		"Branch: feature/alpha",
		"Branch: main",
		"alpha plan",
		"beta plan",
		"alpha notes",
		"beta notes",
		"alpha diff",
		"beta diff",
		"ship v1",
		untrustedDataInstructions,
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("portfolio prompt missing %q:\n%s", want, prompt)
		}
	}
	if got := strings.Count(prompt, "## Repository: alpha"); got != 1 {
		t.Fatalf("alpha repository heading count = %d, want 1", got)
	}
	if got := strings.Count(prompt, "## Repository: beta"); got != 1 {
		t.Fatalf("beta repository heading count = %d, want 1", got)
	}
}

func TestRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"wrapped deadline", fmt.Errorf("call: %w", context.DeadlineExceeded), false},
		{"rate limit", errors.New("429 Too Many Requests"), true},
		{"server error", errors.New("502 Bad Gateway"), true},
		{"timeout string", errors.New("read tcp: i/o timeout"), true},
		{"permanent", errors.New("invalid API key"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryable(tc.err); got != tc.want {
				t.Fatalf("retryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestWithRetryHonorsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := withRetry(ctx, func(ctx context.Context) (string, error) {
		return "", context.DeadlineExceeded
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("retried past the deadline; took %v", elapsed)
	}
}

func TestWithRetryReturnsOnSuccess(t *testing.T) {
	got, err := withRetry(context.Background(), func(ctx context.Context) (string, error) {
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestWithRetryStopsAfterMaxAttempts(t *testing.T) {
	attempts := 0
	_, err := withRetry(context.Background(), func(ctx context.Context) (string, error) {
		attempts++
		return "", errors.New("502 Bad Gateway")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != maxAttempts {
		t.Fatalf("expected %d attempts, got %d", maxAttempts, attempts)
	}
}

func sampleRepo() models.RepoData {
	return models.RepoData{
		Name:        "sample",
		Branch:      "main",
		HeadSHA:     "0123456789abcdef",
		PlanSummary: "build the thing",
		Notes:       "remember to run tests",
		Commits: []models.CommitSummary{
			{SHA: "aaaaaaaaaaaaaaaa", Message: "add feature", Author: "alice", Timestamp: time.Now()},
		},
	}
}

func sampleGoals() models.GoalsData {
	return models.GoalsData{
		Now:  "ship v1",
		Next: "write docs",
		Deadlines: []models.Deadline{
			{Description: "release", DaysUntil: 5},
		},
	}
}

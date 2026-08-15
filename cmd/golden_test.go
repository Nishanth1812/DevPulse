package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/models"
)

func readGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want := normalizeGolden(readGolden(t, name))
	got = normalizeGolden(got)
	if got != want {
		t.Fatalf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func normalizeGolden(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	// A trailing blank line is renderer formatting noise; meaningful content is
	// still compared byte-for-byte.
	return strings.TrimRight(value, "\n") + "\n"
}

func TestPublicRenderersMatchGoldens(t *testing.T) {
	previousNoColor := noColor
	noColor = true
	t.Cleanup(func() { noColor = previousNoColor })
	repo := models.RepoData{Name: "alpha", Branch: "main"}
	tests := []struct {
		name string
		gold string
		run  func() string
	}{
		{
			name: "brief-focused",
			gold: "brief-focused.golden",
			run: func() string {
				var b strings.Builder
				_ = renderBrief(&b, repo, ai.BriefResponse{Summary: "Summary line", KeyChanges: []string{"Added parser", "Added tests"}, CurrentFocus: "Validation", Blockers: []string{"Provider key missing"}, NextSteps: []string{"Run tests"}})
				return b.String()
			},
		},
		{
			name: "brief-portfolio",
			gold: "brief-cross-repo.golden",
			run: func() string {
				var b strings.Builder
				_ = renderPortfolioBrief(&b, ai.PortfolioBriefResponse{Repos: []ai.PortfolioBriefItem{{RepoName: "alpha", Summary: "Alpha summary", CurrentFocus: "Alpha focus", Blockers: []string{"Alpha blocker"}, NextSteps: []string{"Alpha next"}}, {RepoName: "beta", Summary: "Beta summary", CurrentFocus: "Beta focus", NextSteps: []string{"Beta next"}}}})
				return b.String()
			},
		},
		{
			name: "resume",
			gold: "resume.golden",
			run: func() string {
				var b strings.Builder
				_ = renderResume(&b, repo, ai.ResumeResponse{WhatWasBuilt: "Built the parser", WhatIsIncomplete: "Golden coverage", BlockersDetected: []string{"Needs review"}, NextStep: "Add tests"})
				return b.String()
			},
		},
		{
			name: "focus",
			gold: "focus.golden",
			run: func() string {
				var b strings.Builder
				_ = renderFocus(&b, ai.FocusResponse{Ranked: []ai.FocusItem{{RepoName: "alpha", RankReason: "Nearly complete", ProximityScore: 4, Urgency: true}, {RepoName: "beta", RankReason: "Early work", ProximityScore: 2}}})
				return b.String()
			},
		},
		{
			name: "health",
			gold: "health.golden",
			run: func() string {
				var b strings.Builder
				_ = renderHealth(&b, []healthIssue{{Repo: "alpha", Kind: "BRANCH", Message: "1 merged branch(es) not deleted"}})
				return b.String()
			},
		},
		{
			name: "why",
			gold: "why.golden",
			run: func() string {
				var b strings.Builder
				_ = renderWhy(&b, "alpha", "internal/app.go", ai.WhyResponse{FilePurpose: "Application entry point", MajorDecisions: []ai.DecisionItem{{Date: "2026-01-01", Description: "Added command boundary"}}, CurrentState: "Stable"})
				return b.String()
			},
		},
		{
			name: "commit",
			gold: "commit.golden",
			run: func() string {
				var b strings.Builder
				_ = renderCommit(&b, ai.CommitResponse{Subject: "fix: validate output", Body: "Keep model data bounded."})
				return b.String()
			},
		},
		{
			name: "doctor",
			gold: "doctor.golden",
			run: func() string {
				var b strings.Builder
				_ = renderDoctor(&b, []checkResult{{status: "PASS", message: "Config file loaded"}, {status: "WARN", message: "No API key (optional)"}, {status: "FAIL", message: "Repo \"alpha\": path does not exist"}})
				return b.String()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { assertGolden(t, tc.gold, tc.run()) })
	}
}

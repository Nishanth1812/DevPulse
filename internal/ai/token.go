package ai

import (
	"github.com/Nishanth1812/devpulse/internal/compressor"
	"github.com/Nishanth1812/devpulse/internal/models"
)

// TokenBreakdown reports where a prompt's token estimate comes from, so the
// --dry-run output makes prompt bloat visible instead of a single total.
type TokenBreakdown struct {
	Diffs int
	Plan  int
	Notes int
	Goals int
}

// EstimateTokens returns a rough token estimate for a string.
func EstimateTokens(s string) int {
	return compressor.EstimateTokens(s)
}

// BreakdownTokens estimates the token contribution of each prompt section
// across one or more repositories plus the goals file.
func BreakdownTokens(repos []models.RepoData, goals models.GoalsData) TokenBreakdown {
	var b TokenBreakdown

	for _, repo := range repos {
		for _, c := range repo.Commits {
			b.Diffs += compressor.EstimateTokens(c.DiffSnippet)
		}
		b.Plan += compressor.EstimateTokens(repo.PlanSummary)
		b.Notes += compressor.EstimateTokens(repo.Notes)
	}

	b.Goals = compressor.EstimateTokens(goals.Now) +
		compressor.EstimateTokens(goals.Next) +
		compressor.EstimateTokens(goals.Someday)
	for _, d := range goals.Deadlines {
		b.Goals += compressor.EstimateTokens(d.Description)
	}

	return b
}

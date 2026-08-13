package fuzzy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// MatchResult describes the outcome of a fuzzy match attempt.
type MatchResult struct {
	// Exact is true when query matched a single name exactly (case-insensitive).
	Exact bool
	// Matched is the best single match when confidence is above threshold.
	Matched string
	// Candidates lists all names that scored above threshold when multiple matched.
	Candidates []string
}

// Match resolves a partial query against a list of registered repo names.
// threshold is a value between 1 and 100 (default 50).
//
// Returns:
//   - MatchResult{Exact: true, Matched: name} for an exact (case-insensitive) hit.
//   - MatchResult{Matched: name} when exactly one candidate is above threshold.
//   - MatchResult{Candidates: [...] } when multiple candidates are above threshold.
//   - error when no candidates are found.
func Match(query string, names []string, threshold int) (MatchResult, error) {
	if len(names) == 0 {
		return MatchResult{}, fmt.Errorf("no repositories registered")
	}

	query = trimSpaces(query)
	if len(query) == 0 {
		return MatchResult{}, fmt.Errorf("empty query")
	}

	// Exact match (case-insensitive)
	for _, name := range names {
		if equalFold(name, query) {
			return MatchResult{Exact: true, Matched: name}, nil
		}
	}

	// Fuzzy match
	matches := fuzzy.Find(query, names)
	if len(matches) == 0 {
		return MatchResult{}, fmt.Errorf("no repository matches %q", query)
	}

	// sahilm/fuzzy scores are 0-indexed higher = better.
	// We normalise to a 0-100 scale using the best score as ceiling.
	bestScore := matches[0].Score
	if bestScore == 0 {
		return MatchResult{}, fmt.Errorf("no repository matches %q", query)
	}

	type scored struct {
		name  string
		score float64
	}
	var above []scored
	for _, m := range matches {
		pct := float64(m.Score) / float64(bestScore) * 100
		if pct >= float64(threshold) {
			above = append(above, scored{name: names[m.Index], score: pct})
		}
	}

	if len(above) == 0 {
		return MatchResult{}, fmt.Errorf("no repository matches %q (best score below threshold %d)", query, threshold)
	}

	// Sort by score descending, then alphabetically for stability
	sort.Slice(above, func(i, j int) bool {
		if above[i].score != above[j].score {
			return above[i].score > above[j].score
		}
		return above[i].name < above[j].name
	})

	if len(above) == 1 {
		return MatchResult{Matched: above[0].name}, nil
	}

	candidates := make([]string, len(above))
	for i, a := range above {
		candidates[i] = a.name
	}
	return MatchResult{Candidates: candidates}, nil
}

// Resolve is a convenience wrapper that returns the matched name or an error.
// It prints a numbered list to stdout when multiple candidates are found.
func Resolve(query string, names []string, threshold int) (string, error) {
	r, err := Match(query, names, threshold)
	if err != nil {
		return "", err
	}
	if r.Exact || r.Matched != "" {
		name := r.Matched
		if r.Exact {
			name = r.Matched
		}
		return name, nil
	}
	return "", fmt.Errorf("multiple matches: use one of %v", r.Candidates)
}

func trimSpaces(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func equalFold(a, b string) bool {
	return strings.EqualFold(a, b)
}

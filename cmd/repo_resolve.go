package cmd

import (
	"fmt"
	"io"

	"github.com/Nishanth1812/devpulse/internal/fuzzy"
)

// fuzzyMatch resolves a partial repo name against registered repos using the
// configured fuzzy threshold. It returns the raw match result so callers can
// distinguish a single match from a candidate list.
func fuzzyMatch(query string) (fuzzy.MatchResult, error) {
	repos := manager.ListRepositories()
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name
	}

	threshold := 50
	if t, err := manager.Get("fuzzy.threshold"); err == nil {
		fmt.Sscanf(t, "%d", &threshold)
	}

	return fuzzy.Match(query, names, threshold)
}

// printCandidates lists ambiguous matches so the user can pick a more specific
// name. It returns true if candidates were printed (caller should stop).
func printCandidates(w io.Writer, query string, candidates []string) {
	_, _ = fmt.Fprintf(w, "Multiple repositories match %q:\n\n", query)
	for i, name := range candidates {
		_, _ = fmt.Fprintf(w, "  %d. %s\n", i+1, name)
	}
	_, _ = fmt.Fprintln(w, "\nPlease use a more specific name.")
}

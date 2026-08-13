package compressor

import (
	"strings"
)

const (
	// maxDiffLineLen truncates individual diff lines that are pathological
	// (minified bundles, lockfiles, generated JSON) and would otherwise consume
	// thousands of tokens in a single line.
	maxDiffLineLen = 200
	// maxDiffLinesPerFile caps the number of changed lines kept per file so a
	// single file cannot dominate a commit's diff.
	maxDiffLinesPerFile = 60
	// maxDiffLinesTotal caps the total changed lines kept across all files in a
	// commit, matching the original README's 80-line single-file cap but bounded
	// for the whole commit.
	maxDiffLinesTotal = 80
	// maxPlanLines caps plan file summaries at roughly 300 tokens, as the README
	// intended (300 lines would be thousands of tokens).
	maxPlanLines = 80
	// maxNoteLines caps per-repo notes embedded in prompts.
	maxNoteLines = 50
)

// CompressDiff reduces a raw diff to the changed lines that carry information,
// dropping blank/context lines, import-only churn, and diff headers. It applies
// a per-file line cap and a total line cap, and truncates over-long lines.
func CompressDiff(diff string) string {
	var cleaned []string

	for _, file := range splitDiffFiles(diff) {
		var fileLines []string

		for _, line := range file {
			trimmed := strings.TrimSpace(line)

			if trimmed == "" {
				continue
			}

			if !isChangeLine(trimmed) {
				continue
			}

			if strings.HasPrefix(trimmed, "import ") {
				continue
			}

			fileLines = append(fileLines, truncateLine(line, maxDiffLineLen))
			if len(fileLines) >= maxDiffLinesPerFile {
				break
			}
		}

		cleaned = append(cleaned, fileLines...)
		if len(cleaned) >= maxDiffLinesTotal {
			cleaned = cleaned[:maxDiffLinesTotal]
			break
		}
	}

	return strings.Join(cleaned, "\n")
}

// CompressPlan strips completed checkboxes, deduplicates lines, retains headings,
// truncates long lines, and caps the result at maxPlanLines.
func CompressPlan(content string) string {
	lines := strings.Split(content, "\n")

	seen := map[string]bool{}
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if strings.Contains(trimmed, "- [x]") {
			continue
		}

		if seen[trimmed] {
			continue
		}

		seen[trimmed] = true
		result = append(result, truncateLine(line, maxDiffLineLen))

		if len(result) >= maxPlanLines {
			break
		}
	}

	return strings.Join(result, "\n")
}

// CompressNotes caps notes content so an unbounded notes file cannot bloat a prompt.
func CompressNotes(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > maxNoteLines {
		lines = lines[:maxNoteLines]
	}
	for i, line := range lines {
		lines[i] = truncateLine(line, maxDiffLineLen)
	}
	return strings.Join(lines, "\n")
}

// EstimateTokens is a rough token estimate (characters / 4), the same heuristic
// the dry-run output has always used, centralised so it is consistent everywhere.
func EstimateTokens(s string) int {
	return (len(s) + 3) / 4
}

// isChangeLine reports whether a trimmed diff line carries change information
// (an added line, removed line, or hunk header) and is not a file marker such
// as "--- a/x" or "+++ b/x".
func isChangeLine(trimmed string) bool {
	if strings.HasPrefix(trimmed, "--- ") || strings.HasPrefix(trimmed, "+++ ") {
		return false
	}
	return strings.HasPrefix(trimmed, "+") ||
		strings.HasPrefix(trimmed, "-") ||
		strings.HasPrefix(trimmed, "@@")
}

// splitDiffFiles splits a raw diff into per-file sections on "diff --git" headers.
// A raw diff with no such headers (unusual) is treated as a single file.
func splitDiffFiles(diff string) [][]string {
	lines := strings.Split(diff, "\n")

	var files [][]string
	var current []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "diff --git") {
			if len(current) > 0 {
				files = append(files, current)
			}
			current = []string{line}
			continue
		}
		if current != nil {
			current = append(current, line)
		}
	}

	if len(current) > 0 {
		files = append(files, current)
	}

	if len(files) == 0 {
		files = [][]string{lines}
	}

	return files
}

// truncateLine truncates a line to max runes, never splitting a multi-byte
// UTF-8 rune in half (which would emit invalid bytes into a prompt).
func truncateLine(line string, max int) string {
	runes := []rune(line)
	if len(runes) <= max {
		return line
	}
	return string(runes[:max]) + "..."
}

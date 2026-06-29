package compressor

import (
	"strings"
)

func CompressDiff(diff string) string {
	lines := strings.Split(diff, "\n")

	var cleaned []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "import ") {
			continue
		}

		if strings.HasPrefix(trimmed, "+") ||
			strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "@@") {
			cleaned = append(cleaned, line)
		}
	}

	if len(cleaned) > 80 {
		cleaned = cleaned[:80]
	}

	return strings.Join(cleaned, "\n")
}

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
		result = append(result, line)
	}

	if len(result) > 300 {
		result = result[:300]
	}

	return strings.Join(result, "\n")
}

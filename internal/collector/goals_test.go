package collector

import (
	"testing"
)

func TestSplitSections(t *testing.T) {
	content := `# DevPulse Goals

## Now
Active items.

## Next
Up next.

## Deadlines
2026-12-31 — ship v2

## Someday
Maybe later.
`
	sections := splitSections(content)

	if got := sections["Now"]; !contains(got, "Active items.") {
		t.Fatalf("Now section = %q", got)
	}
	if got := sections["Next"]; !contains(got, "Up next.") {
		t.Fatalf("Next section = %q", got)
	}
	if got := sections["Someday"]; !contains(got, "Maybe later.") {
		t.Fatalf("Someday section = %q", got)
	}
	if !contains(sections["Deadlines"], "ship v2") {
		t.Fatalf("Deadlines section missing content: %q", sections["Deadlines"])
	}
}

func TestSplitSectionsIgnoresBareWord(t *testing.T) {
	// A line "Now" without a heading must not switch sections.
	content := "## Next\nSome note\nNow is not a heading\nStill next.\n"
	sections := splitSections(content)
	if got := sections["Now"]; got != "" {
		t.Fatalf("bare 'Now' leaked into Now section: %q", got)
	}
	if got := sections["Next"]; !contains(got, "Now is not a heading") {
		t.Fatalf("bare 'Now' should stay in Next: %q", got)
	}
}

func TestSplitSectionsUnknownHeadingKeepsContent(t *testing.T) {
	content := "## Now\n## Subheading\ncontent here\n"
	sections := splitSections(content)
	if got := sections["Now"]; !contains(got, "Subheading") || !contains(got, "content here") {
		t.Fatalf("unknown heading content lost: %q", got)
	}
}

func TestParseDeadlinesSeparators(t *testing.T) {
	content := "2026-12-31 — em dash\n2026-11-01 - hyphen\n2026-10-02: colon\nnot a deadline\n"
	deadlines := parseDeadlines(content)
	if len(deadlines) != 3 {
		t.Fatalf("expected 3 deadlines, got %d: %+v", len(deadlines), deadlines)
	}
	// Sorted oldest first.
	if !deadlines[0].Date.Before(deadlines[1].Date) || !deadlines[1].Date.Before(deadlines[2].Date) {
		t.Fatalf("deadlines not sorted: %+v", deadlines)
	}
	if deadlines[0].Description != "colon" {
		t.Fatalf("expected 'colon' first (oldest), got %q", deadlines[0].Description)
	}
}

func TestParseDeadlinesSorts(t *testing.T) {
	content := "2026-01-10 — later\n2026-01-01 — earlier\n"
	deadlines := parseDeadlines(content)
	if deadlines[0].Description != "earlier" {
		t.Fatalf("expected 'earlier' first, got %q", deadlines[0].Description)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

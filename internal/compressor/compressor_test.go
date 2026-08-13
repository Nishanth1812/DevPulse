package compressor

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCompressDiffBasic(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@
 import "fmt"
+func New() {}
 func Old() {}
 `
	out := CompressDiff(diff)
	if !strings.Contains(out, "+func New() {}") {
		t.Fatalf("expected added line preserved:\n%s", out)
	}
	if strings.Contains(out, "import \"fmt\"") {
		t.Fatalf("import churn should be dropped:\n%s", out)
	}
	if strings.Contains(out, "diff --git") {
		t.Fatalf("diff headers should be dropped:\n%s", out)
	}
}

func TestCompressDiffCapsPerFileAndTotal(t *testing.T) {
	var b strings.Builder
	b.WriteString("diff --git a/big.go b/big.go\n")
	b.WriteString("--- a/big.go\n+++ b/big.go\n")
	for i := 0; i < maxDiffLinesPerFile+10; i++ {
		b.WriteString("+line\n")
	}
	b.WriteString("diff --git a/other.go b/other.go\n")
	b.WriteString("--- a/other.go\n+++ b/other.go\n")
	for i := 0; i < maxDiffLinesPerFile; i++ {
		b.WriteString("+other\n")
	}

	out := CompressDiff(b.String())
	lines := strings.Count(out, "\n") + 1
	if lines > maxDiffLinesTotal {
		t.Fatalf("output exceeds total cap: %d lines", lines)
	}
}

func TestTruncateLineValidUTF8(t *testing.T) {
	// A line of multi-byte runes longer than the cap must not split a rune.
	line := strings.Repeat("é", 100)
	out := truncateLine(line, 10)
	if !utf8.ValidString(out) {
		t.Fatalf("truncated line contains invalid UTF-8: %q", out)
	}
	// Exactly 10 runes plus the ellipsis.
	if got := len([]rune(out)); got != 13 {
		t.Fatalf("expected 10 runes + ..., got %d", got)
	}
}

func TestTruncateLineShortLineUntouched(t *testing.T) {
	if got := truncateLine("short", 10); got != "short" {
		t.Fatalf("short line should not be altered: %q", got)
	}
}

func TestCompressPlan(t *testing.T) {
	content := "## Now\n- [x] done\n- [ ] active\n- [ ] active\n\nNext\n- [ ] todo\n"
	out := CompressPlan(content)
	if strings.Contains(out, "- [x]") {
		t.Fatalf("completed checkbox should be dropped:\n%s", out)
	}
	if strings.Count(out, "- [ ] active") != 1 {
		t.Fatalf("duplicate lines should be deduplicated:\n%s", out)
	}
}

func TestCompressNotesCaps(t *testing.T) {
	content := strings.Repeat("line\n", maxNoteLines+20)
	out := CompressNotes(content)
	if strings.Count(out, "\n") >= maxNoteLines {
		t.Fatalf("notes not capped: %d lines", strings.Count(out, "\n"))
	}
}

func TestCompressNotesKeepsNewest(t *testing.T) {
	var b strings.Builder
	for i := 0; i < maxNoteLines+5; i++ {
		fmt.Fprintf(&b, "old-entry-%d\n", i)
	}
	b.WriteString("NEWEST-ENTRY\n")
	out := CompressNotes(b.String())
	if !strings.Contains(out, "NEWEST-ENTRY") {
		t.Fatalf("newest note was dropped: %q", out)
	}
	if strings.Contains(out, "old-entry-0") {
		t.Fatalf("oldest notes should be truncated, got: %q", out)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(strings.Repeat("a", 40)); got != 10 {
		t.Fatalf("EstimateTokens = %d, want 10", got)
	}
}

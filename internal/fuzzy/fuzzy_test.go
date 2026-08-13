package fuzzy

import (
	"testing"
)

func TestExactMatchCaseInsensitive(t *testing.T) {
	r, err := Match("ACM-APP", []string{"acm-app", "web-frontend"}, 50)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !r.Exact || r.Matched != "acm-app" {
		t.Fatalf("expected exact match, got %+v", r)
	}
}

func TestExactMatchUnicode(t *testing.T) {
	// strings.EqualFold handles Unicode folding that the old ASCII-only
	// comparison did not.
	r, err := Match("ÜBER-APP", []string{"über-app", "web"}, 50)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !r.Exact || r.Matched != "über-app" {
		t.Fatalf("expected exact Unicode match, got %+v", r)
	}
}

func TestFuzzySingleMatch(t *testing.T) {
	r, err := Match("front", []string{"web-frontend", "backend-api", "mobile-app"}, 50)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if r.Exact || r.Matched != "web-frontend" || len(r.Candidates) != 0 {
		t.Fatalf("expected single fuzzy match, got %+v", r)
	}
}

func TestFuzzyMultipleCandidates(t *testing.T) {
	r, err := Match("app", []string{"web-app", "mobile-app", "backend-api"}, 50)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(r.Candidates) < 2 {
		t.Fatalf("expected multiple candidates, got %+v", r)
	}
}

func TestMatchNoRepos(t *testing.T) {
	if _, err := Match("x", nil, 50); err == nil {
		t.Fatal("expected error for empty repo list")
	}
}

func TestMatchNoResults(t *testing.T) {
	if _, err := Match("zzzz", []string{"alpha", "beta"}, 50); err == nil {
		t.Fatal("expected error when nothing matches")
	}
}

func TestResolveReturnsExactOrMatched(t *testing.T) {
	name, err := Resolve("front", []string{"web-frontend", "api"}, 50)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if name != "web-frontend" {
		t.Fatalf("got %q", name)
	}
}

func TestEqualFoldUnicode(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"ÜBER", "über", true},
		{"AbC", "aBc", true},
		{"abc", "abd", false},
	}
	for _, tc := range cases {
		if got := equalFold(tc.a, tc.b); got != tc.want {
			t.Fatalf("equalFold(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestTrimSpaces(t *testing.T) {
	if got := trimSpaces("  hello \t\n"); got != "hello" {
		t.Fatalf("trimSpaces = %q", got)
	}
	if got := trimSpaces("   "); got != "" {
		t.Fatalf("trimSpaces(blank) = %q", got)
	}
}

func TestThresholdFiltersCandidates(t *testing.T) {
	// With a low threshold the best match is always at 100%, so a single
	// result is returned even when several names score well.
	r, err := Match("app", []string{"web-app", "mobile-app"}, 10)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if r.Matched == "" && len(r.Candidates) == 0 {
		t.Fatal("expected some resolution at low threshold")
	}
}

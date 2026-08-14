package cmd

import (
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/ai"
)

func TestBriefCommandAcceptsPortfolioAndFocusedModes(t *testing.T) {
	if err := briefCmd.Args(briefCmd, nil); err != nil {
		t.Fatalf("brief should accept no repository argument: %v", err)
	}
	if err := briefCmd.Args(briefCmd, []string{"repo"}); err != nil {
		t.Fatalf("brief should accept one repository argument: %v", err)
	}
	if err := briefCmd.Args(briefCmd, []string{"one", "two"}); err == nil {
		t.Fatal("brief should reject more than one repository argument")
	}
}

func TestRenderPortfolioBrief(t *testing.T) {
	var out strings.Builder
	err := renderPortfolioBrief(&out, ai.PortfolioBriefResponse{
		Repos: []ai.PortfolioBriefItem{{
			RepoName:     "demo",
			Summary:      "The repository is nearly ready.",
			CurrentFocus: "release checks",
			Blockers:     []string{"Windows CI is missing"},
			NextSteps:    []string{"Add the matrix job"},
		}},
	})
	if err != nil {
		t.Fatalf("renderPortfolioBrief: %v", err)
	}
	for _, want := range []string{"Brief: Portfolio", "demo", "Windows CI is missing", "Add the matrix job"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("rendered output missing %q: %s", want, out.String())
		}
	}
}

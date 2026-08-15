package ai

import (
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/models"
)

func TestValidateFocusResponseRejectsInvalidItems(t *testing.T) {
	allowed := map[string]struct{}{"alpha": {}}
	tests := []struct {
		name string
		item FocusItem
		want string
	}{
		{name: "score below one", item: FocusItem{RepoName: "alpha", RankReason: "reason", ProximityScore: 0}, want: "score"},
		{name: "score above five", item: FocusItem{RepoName: "alpha", RankReason: "reason", ProximityScore: 6}, want: "score"},
		{name: "empty name", item: FocusItem{RankReason: "reason", ProximityScore: 3}, want: "repository name"},
		{name: "empty reason", item: FocusItem{RepoName: "alpha", ProximityScore: 3}, want: "reason"},
		{name: "unknown repository", item: FocusItem{RepoName: "beta", RankReason: "reason", ProximityScore: 3}, want: "unknown repository"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFocusResponse(FocusResponse{Ranked: []FocusItem{tc.item}}, allowed)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestValidateFocusResponseRejectsDuplicatesAndTooManyItems(t *testing.T) {
	allowed := map[string]struct{}{"alpha": {}, "beta": {}}
	duplicate := FocusResponse{Ranked: []FocusItem{
		{RepoName: "alpha", RankReason: "one", ProximityScore: 3},
		{RepoName: "alpha", RankReason: "two", ProximityScore: 4},
	}}
	if err := ValidateFocusResponse(duplicate, allowed); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate error = %v", err)
	}

	tooMany := make([]FocusItem, 0, maxFocusItems+1)
	for i := 0; i < maxFocusItems+1; i++ {
		tooMany = append(tooMany, FocusItem{RepoName: "alpha", RankReason: "reason", ProximityScore: 3})
	}
	if err := ValidateFocusResponse(FocusResponse{Ranked: tooMany}, allowed); err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("too-many error = %v", err)
	}
}

func TestApplyDeadlineUrgencyUsesParsedGoalsAndIgnoresModelFlag(t *testing.T) {
	items := []FocusItem{
		{RepoName: "alpha", RankReason: "a", ProximityScore: 3, Urgency: false},
		{RepoName: "beta", RankReason: "b", ProximityScore: 3, Urgency: true},
	}
	goals := models.GoalsData{Deadlines: []models.Deadline{
		{Description: "alpha release", DaysUntil: 5},
		{Description: "beta release", DaysUntil: 30},
	}}

	got := ApplyDeadlineUrgency(items, goals, 14)
	if !got[0].Urgency {
		t.Fatal("deadline within the window did not mark alpha urgent")
	}
	if got[1].Urgency {
		t.Fatal("model-provided urgency was trusted for beta")
	}
	if items[0].Urgency || !items[1].Urgency {
		t.Fatal("ApplyDeadlineUrgency mutated its input")
	}
}

func TestValidatePortfolioBriefResponseRequiresAllowedRepositories(t *testing.T) {
	allowed := map[string]struct{}{"alpha": {}, "beta": {}}
	response := PortfolioBriefResponse{Repos: []PortfolioBriefItem{
		{RepoName: "alpha", Summary: "ok"},
		{RepoName: "unknown", Summary: "bad"},
	}}
	if err := ValidatePortfolioBriefResponse(response, allowed); err == nil || !strings.Contains(err.Error(), "unknown repository") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateBriefResponseBoundsFields(t *testing.T) {
	response := BriefResponse{Summary: strings.Repeat("x", maxBriefSummaryLength+1)}
	if err := ValidateBriefResponse(response); err == nil || !strings.Contains(err.Error(), "summary") {
		t.Fatalf("error = %v", err)
	}
}

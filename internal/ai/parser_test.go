package ai

import "testing"

func TestParseBriefResponse(t *testing.T) {
	raw := `{"summary":"Good state","key_changes":["a"],"current_focus":"x","blockers":[],"next_steps":["y"]}`
	r, err := ParseBriefResponse(raw)
	if err != nil {
		t.Fatalf("ParseBriefResponse: %v", err)
	}
	if r.Summary != "Good state" {
		t.Fatalf("summary = %q", r.Summary)
	}
}

func TestParseBriefResponseWithFences(t *testing.T) {
	raw := "```json\n{\"summary\":\"Fenced\",\"key_changes\":[],\"current_focus\":\"\",\"blockers\":[],\"next_steps\":[]}\n```"
	r, err := ParseBriefResponse(raw)
	if err != nil {
		t.Fatalf("ParseBriefResponse with fences: %v", err)
	}
	if r.Summary != "Fenced" {
		t.Fatalf("summary = %q", r.Summary)
	}
}

func TestParseBriefResponseMissingSummary(t *testing.T) {
	_, err := ParseBriefResponse(`{"key_changes":["a"]}`)
	if err == nil {
		t.Fatal("expected error for missing summary")
	}
}

func TestParsePortfolioBriefResponse(t *testing.T) {
	raw := `{"repos":[{"repo_name":"one","summary":"Good","key_changes":[],"current_focus":"x","blockers":[],"next_steps":[]},{"repo_name":"two","summary":"Needs work","key_changes":[],"current_focus":"y","blockers":[],"next_steps":[]}]}`
	r, err := ParsePortfolioBriefResponse(raw, []string{"one", "two"})
	if err != nil {
		t.Fatalf("ParsePortfolioBriefResponse: %v", err)
	}
	if len(r.Repos) != 2 || r.Repos[0].RepoName != "one" {
		t.Fatalf("parsed %+v", r)
	}
}

func TestParsePortfolioBriefResponseRejectsUnknownOrMissingRepo(t *testing.T) {
	unknown := `{"repos":[{"repo_name":"other","summary":"x"}]}`
	if _, err := ParsePortfolioBriefResponse(unknown, []string{"one"}); err == nil {
		t.Fatal("expected unknown repository error")
	}
	missing := `{"repos":[{"repo_name":"one","summary":"x"}]}`
	if _, err := ParsePortfolioBriefResponse(missing, []string{"one", "two"}); err == nil {
		t.Fatal("expected missing repository error")
	}
}

func TestParseBriefResponseInvalidJSON(t *testing.T) {
	if _, err := ParseBriefResponse("not json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseCommitResponse(t *testing.T) {
	r, err := ParseCommitResponse(`{"subject":"fix: thing","body":"details"}`)
	if err != nil {
		t.Fatalf("ParseCommitResponse: %v", err)
	}
	if r.Subject != "fix: thing" || r.Body != "details" {
		t.Fatalf("parsed %+v", r)
	}
}

func TestParseCommitResponseMissingSubject(t *testing.T) {
	if _, err := ParseCommitResponse(`{"body":"details"}`); err == nil {
		t.Fatal("expected error for missing subject")
	}
}

func TestParseResumeResponse(t *testing.T) {
	raw := "```\n{\"what_was_built\":\"A\",\"what_is_incomplete\":\"B\",\"blockers_detected\":[],\"next_step\":\"C\"}\n```"
	r, err := ParseResumeResponse(raw)
	if err != nil {
		t.Fatalf("ParseResumeResponse: %v", err)
	}
	if r.WhatWasBuilt != "A" {
		t.Fatalf("what_was_built = %q", r.WhatWasBuilt)
	}
}

func TestParseResumeResponseMissingField(t *testing.T) {
	if _, err := ParseResumeResponse(`{}`); err == nil {
		t.Fatal("expected error for missing what_was_built")
	}
}

func TestParseFocusResponse(t *testing.T) {
	r, err := ParseFocusResponse(`{"ranked":[{"repo_name":"r","rank_reason":"x","proximity_score":4,"urgency":false}]}`)
	if err != nil {
		t.Fatalf("ParseFocusResponse: %v", err)
	}
	if len(r.Ranked) != 1 || r.Ranked[0].ProximityScore != 4 {
		t.Fatalf("parsed %+v", r)
	}
}

func TestParseFocusResponseEmptyRanked(t *testing.T) {
	if _, err := ParseFocusResponse(`{"ranked":[]}`); err == nil {
		t.Fatal("expected error for empty ranked")
	}
}

func TestParseWhyResponse(t *testing.T) {
	r, err := ParseWhyResponse(`{"file_purpose":"purpose","major_decisions":[{"date":"2026-01-01","description":"d"}],"current_state":"state"}`)
	if err != nil {
		t.Fatalf("ParseWhyResponse: %v", err)
	}
	if r.FilePurpose != "purpose" || len(r.MajorDecisions) != 1 {
		t.Fatalf("parsed %+v", r)
	}
}

func TestParseWhyResponseMissingPurpose(t *testing.T) {
	if _, err := ParseWhyResponse(`{"current_state":"state"}`); err == nil {
		t.Fatal("expected error for missing file_purpose")
	}
}

func TestStripFences(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`{"a":1}`, `{"a":1}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"```\n{\"a\":1}\n```", `{"a":1}`},
		{"```json{\"a\":1}```", `{"a":1}`},
		{"```json\n[1,2]\n```", "[1,2]"},
		{"  {\"a\":1}  ", `{"a":1}`},
	}
	for _, tt := range tests {
		if got := stripFences(tt.in); got != tt.want {
			t.Errorf("stripFences(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

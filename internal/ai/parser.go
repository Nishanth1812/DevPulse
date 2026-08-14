package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseBriefResponse strips markdown fences if present, validates the JSON,
// and unmarshals it into a BriefResponse. Returns an error if summary is missing.
func ParseBriefResponse(raw string) (BriefResponse, error) {
	clean := stripFences(raw)
	var r BriefResponse
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return BriefResponse{}, fmt.Errorf("ai: parse brief response: %w", err)
	}
	if strings.TrimSpace(r.Summary) == "" {
		return BriefResponse{}, fmt.Errorf("ai: brief response missing required field: summary")
	}
	return r, nil
}

// ParsePortfolioBriefResponse parses and validates a cross-repository brief.
// expectedRepos is the set of repositories that were included in the prompt;
// requiring an exact match prevents a model from silently omitting a repo or
// inventing one in the rendered output.
func ParsePortfolioBriefResponse(raw string, expectedRepos []string) (PortfolioBriefResponse, error) {
	clean := stripFences(raw)
	var r PortfolioBriefResponse
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: parse portfolio brief response: %w", err)
	}
	if len(r.Repos) == 0 {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response missing required field: repos")
	}

	expected := make(map[string]struct{}, len(expectedRepos))
	for _, name := range expectedRepos {
		expected[name] = struct{}{}
	}
	seen := make(map[string]struct{}, len(r.Repos))
	for _, item := range r.Repos {
		if strings.TrimSpace(item.RepoName) == "" {
			return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief item missing required field: repo_name")
		}
		if strings.TrimSpace(item.Summary) == "" {
			return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief item %q missing required field: summary", item.RepoName)
		}
		if _, duplicate := seen[item.RepoName]; duplicate {
			return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response contains duplicate repository %q", item.RepoName)
		}
		seen[item.RepoName] = struct{}{}
		if len(expected) > 0 {
			if _, ok := expected[item.RepoName]; !ok {
				return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response contains unknown repository %q", item.RepoName)
			}
		}
	}
	if len(expected) > 0 && len(seen) != len(expected) {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response contains %d repositories, expected %d", len(seen), len(expected))
	}
	return r, nil
}

// ParseCommitResponse strips markdown fences if present, validates the JSON,
// and unmarshals it into a CommitResponse. Returns an error if subject is missing.
func ParseCommitResponse(raw string) (CommitResponse, error) {
	clean := stripFences(raw)
	var r CommitResponse
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return CommitResponse{}, fmt.Errorf("ai: parse commit response: %w", err)
	}
	if strings.TrimSpace(r.Subject) == "" {
		return CommitResponse{}, fmt.Errorf("ai: commit response missing required field: subject")
	}
	return r, nil
}

// ParseResumeResponse strips markdown fences if present, validates the JSON,
// and unmarshals it into a ResumeResponse. Returns an error if what_was_built is missing.
func ParseResumeResponse(raw string) (ResumeResponse, error) {
	clean := stripFences(raw)
	var r ResumeResponse
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return ResumeResponse{}, fmt.Errorf("ai: parse resume response: %w", err)
	}
	if strings.TrimSpace(r.WhatWasBuilt) == "" {
		return ResumeResponse{}, fmt.Errorf("ai: resume response missing required field: what_was_built")
	}
	return r, nil
}

// ParseFocusResponse strips markdown fences if present, validates the JSON,
// and unmarshals it into a FocusResponse. Returns an error if ranked is empty.
func ParseFocusResponse(raw string) (FocusResponse, error) {
	clean := stripFences(raw)
	var r FocusResponse
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return FocusResponse{}, fmt.Errorf("ai: parse focus response: %w", err)
	}
	if len(r.Ranked) == 0 {
		return FocusResponse{}, fmt.Errorf("ai: focus response missing required field: ranked")
	}
	return r, nil
}

// ParseWhyResponse strips markdown fences if present, validates the JSON,
// and unmarshals it into a WhyResponse. Returns an error if file_purpose is missing.
func ParseWhyResponse(raw string) (WhyResponse, error) {
	clean := stripFences(raw)
	var r WhyResponse
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return WhyResponse{}, fmt.Errorf("ai: parse why response: %w", err)
	}
	if strings.TrimSpace(r.FilePurpose) == "" {
		return WhyResponse{}, fmt.Errorf("ai: why response missing required field: file_purpose")
	}
	return r, nil
}

// stripFences removes markdown code fences that models sometimes wrap JSON in.
// Handles ```json\n{...}\n```, ```\n{...}\n```, ```json{...}```, and bare JSON.
// Rather than relying on a newline after the opening fence (which may be absent),
// it scans forward to the first JSON delimiter ({ or [) and strips backward from there.
func stripFences(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Find where the JSON actually starts — first { or [
	if i := strings.IndexAny(s, "{["); i != -1 {
		s = s[i:]
	}
	// Remove the trailing closing fence if present
	if i := strings.LastIndex(s, "```"); i != -1 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

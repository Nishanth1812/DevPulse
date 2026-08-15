package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
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

// ParsePortfolioBriefResponse strips optional markdown fences, validates the
// bounded portfolio response contract, and unmarshals it into a typed value.
func ParsePortfolioBriefResponse(raw string) (PortfolioBriefResponse, error) {
	clean := stripFences(raw)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(clean), &fields); err != nil {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response parse: %w", err)
	}
	reposRaw, ok := fields["repos"]
	if !ok || strings.TrimSpace(string(reposRaw)) == "null" {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response missing required field: repos")
	}

	var response PortfolioBriefResponse
	if err := json.Unmarshal(reposRaw, &response.Repos); err != nil {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response parse repos: %w", err)
	}
	if len(response.Repos) > maxPortfolioItems {
		return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response contains %d repositories; maximum is %d", len(response.Repos), maxPortfolioItems)
	}

	seen := make(map[string]struct{}, len(response.Repos))
	for i := range response.Repos {
		item := &response.Repos[i]
		item.RepoName = strings.TrimSpace(item.RepoName)
		if item.RepoName == "" {
			return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response repository name at index %d is empty", i)
		}
		if runeLen(item.RepoName) > maxPortfolioRepoNameLength {
			return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response repository name %q exceeds %d characters", item.RepoName, maxPortfolioRepoNameLength)
		}
		if _, exists := seen[item.RepoName]; exists {
			return PortfolioBriefResponse{}, fmt.Errorf("ai: portfolio brief response duplicate repository %q", item.RepoName)
		}
		seen[item.RepoName] = struct{}{}

		item.Summary = strings.TrimSpace(item.Summary)
		item.CurrentFocus = strings.TrimSpace(item.CurrentFocus)
		if err := validatePortfolioText("summary", item.RepoName, item.Summary, maxPortfolioSummaryLength); err != nil {
			return PortfolioBriefResponse{}, err
		}
		if err := validatePortfolioText("current_focus", item.RepoName, item.CurrentFocus, maxPortfolioFocusLength); err != nil {
			return PortfolioBriefResponse{}, err
		}
		if err := validatePortfolioList("blockers", item.RepoName, item.Blockers); err != nil {
			return PortfolioBriefResponse{}, err
		}
		if err := validatePortfolioList("next_steps", item.RepoName, item.NextSteps); err != nil {
			return PortfolioBriefResponse{}, err
		}
	}

	return response, nil
}

func validatePortfolioText(field, repoName, value string, limit int) error {
	if runeLen(value) > limit {
		return fmt.Errorf("ai: portfolio brief response %s for repository %q exceeds %d characters", field, repoName, limit)
	}
	return nil
}

func validatePortfolioList(field, repoName string, values []string) error {
	if len(values) > maxPortfolioListLength {
		return fmt.Errorf("ai: portfolio brief response %s for repository %q contains %d items; maximum is %d", field, repoName, len(values), maxPortfolioListLength)
	}
	for _, value := range values {
		if runeLen(strings.TrimSpace(value)) > maxPortfolioListItemLength {
			return fmt.Errorf("ai: portfolio brief response %s item for repository %q exceeds %d characters", field, repoName, maxPortfolioListItemLength)
		}
	}
	return nil
}

func runeLen(value string) int {
	return utf8.RuneCountInString(value)
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

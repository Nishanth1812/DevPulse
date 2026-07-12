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

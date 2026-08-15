package ai

import (
	"fmt"
	"strings"

	"github.com/Nishanth1812/devpulse/internal/models"
)

const (
	maxFocusItems           = 100
	maxFocusRepoNameLength  = 200
	maxFocusReasonLength    = 1000
	maxBriefSummaryLength   = 4000
	maxBriefFocusLength     = 1000
	maxBriefListLength      = 10
	maxBriefListItemLength  = 1000
	maxResumeTextLength     = 4000
	maxResumeListLength     = 10
	maxResumeListItemLength = 1000
	maxWhyPurposeLength     = 4000
	maxWhyStateLength       = 2000
	maxWhyDecisionLength    = 1000
	maxCommitSubjectLength  = 72
	maxCommitBodyLength     = 4000
)

// ValidateBriefResponse checks the focused brief contract after JSON parsing
// and before rendering or caching.
func ValidateBriefResponse(response BriefResponse) error {
	if strings.TrimSpace(response.Summary) == "" {
		return fmt.Errorf("ai: brief response missing required field: summary")
	}
	if err := validateText("summary", response.Summary, maxBriefSummaryLength); err != nil {
		return err
	}
	if err := validateText("current_focus", response.CurrentFocus, maxBriefFocusLength); err != nil {
		return err
	}
	if err := validateStringList("key_changes", response.KeyChanges, maxBriefListLength, maxBriefListItemLength); err != nil {
		return err
	}
	if err := validateStringList("blockers", response.Blockers, maxBriefListLength, maxBriefListItemLength); err != nil {
		return err
	}
	return validateStringList("next_steps", response.NextSteps, maxBriefListLength, maxBriefListItemLength)
}

// ValidatePortfolioBriefResponse validates repository membership and bounded
// fields. The caller remains responsible for ordering and requiring every
// allowed repository when that command's contract requires it.
func ValidatePortfolioBriefResponse(response PortfolioBriefResponse, allowedRepos map[string]struct{}) error {
	if len(response.Repos) == 0 {
		return fmt.Errorf("ai: portfolio brief response contains no repositories")
	}
	if len(response.Repos) > maxPortfolioItems {
		return fmt.Errorf("ai: portfolio brief response contains %d repositories; maximum is %d", len(response.Repos), maxPortfolioItems)
	}
	seen := make(map[string]struct{}, len(response.Repos))
	for _, item := range response.Repos {
		name := strings.TrimSpace(item.RepoName)
		if name == "" {
			return fmt.Errorf("ai: portfolio brief response repository name is empty")
		}
		if _, ok := allowedRepos[name]; !ok {
			return fmt.Errorf("ai: portfolio brief response contains unknown repository %q", name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("ai: portfolio brief response contains duplicate repository %q", name)
		}
		seen[name] = struct{}{}
		if err := validateText("summary", item.Summary, maxPortfolioSummaryLength); err != nil {
			return fmt.Errorf("repository %q: %w", name, err)
		}
		if err := validateText("current_focus", item.CurrentFocus, maxPortfolioFocusLength); err != nil {
			return fmt.Errorf("repository %q: %w", name, err)
		}
		if err := validateStringList("blockers", item.Blockers, maxPortfolioListLength, maxPortfolioListItemLength); err != nil {
			return fmt.Errorf("repository %q: %w", name, err)
		}
		if err := validateStringList("next_steps", item.NextSteps, maxPortfolioListLength, maxPortfolioListItemLength); err != nil {
			return fmt.Errorf("repository %q: %w", name, err)
		}
	}
	return nil
}

// ValidateFocusResponse validates model-created ranking data against the
// repositories actually collected by the command.
func ValidateFocusResponse(response FocusResponse, allowedRepos map[string]struct{}) error {
	if len(response.Ranked) == 0 {
		return fmt.Errorf("ai: focus response contains no ranked repositories")
	}
	if len(response.Ranked) > maxFocusItems {
		return fmt.Errorf("ai: focus response contains %d repositories; maximum is %d", len(response.Ranked), maxFocusItems)
	}
	seen := make(map[string]struct{}, len(response.Ranked))
	for _, item := range response.Ranked {
		name := strings.TrimSpace(item.RepoName)
		if name == "" {
			return fmt.Errorf("ai: focus response repository name is empty")
		}
		if _, ok := allowedRepos[name]; !ok {
			return fmt.Errorf("ai: focus response contains unknown repository %q", name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("ai: focus response contains duplicate repository %q", name)
		}
		seen[name] = struct{}{}
		if item.ProximityScore < 1 || item.ProximityScore > 5 {
			return fmt.Errorf("ai: focus response score for repository %q must be between 1 and 5", name)
		}
		if strings.TrimSpace(item.RankReason) == "" {
			return fmt.Errorf("ai: focus response rank reason for repository %q is empty", name)
		}
		if err := validateText("rank reason", item.RankReason, maxFocusReasonLength); err != nil {
			return fmt.Errorf("repository %q: %w", name, err)
		}
	}
	return nil
}

// ApplyDeadlineUrgency ignores model-provided urgency and derives it from
// parsed goals. It returns a copy so callers can safely retain parsed data.
func ApplyDeadlineUrgency(items []FocusItem, goals models.GoalsData, window int) []FocusItem {
	result := append([]FocusItem(nil), items...)
	for i := range result {
		result[i].Urgency = false
		name := strings.ToLower(strings.TrimSpace(result[i].RepoName))
		for _, deadline := range goals.Deadlines {
			if deadline.DaysUntil >= 0 && deadline.DaysUntil <= window && strings.Contains(strings.ToLower(deadline.Description), name) {
				result[i].Urgency = true
				break
			}
		}
	}
	return result
}

// The remaining validators protect the other typed model responses before
// they cross the rendering/cache boundary. They intentionally stay small and
// reuse the same bounded string/list checks as the primary R3 contracts.
func ValidateResumeResponse(response ResumeResponse) error {
	if strings.TrimSpace(response.WhatWasBuilt) == "" {
		return fmt.Errorf("ai: resume response missing required field: what_was_built")
	}
	if err := validateText("what_was_built", response.WhatWasBuilt, maxResumeTextLength); err != nil {
		return err
	}
	if err := validateText("what_is_incomplete", response.WhatIsIncomplete, maxResumeTextLength); err != nil {
		return err
	}
	if err := validateStringList("blockers_detected", response.BlockersDetected, maxResumeListLength, maxResumeListItemLength); err != nil {
		return err
	}
	return validateText("next_step", response.NextStep, maxResumeTextLength)
}

func ValidateCommitResponse(response CommitResponse) error {
	if strings.TrimSpace(response.Subject) == "" {
		return fmt.Errorf("ai: commit response missing required field: subject")
	}
	if err := validateText("subject", response.Subject, maxCommitSubjectLength); err != nil {
		return err
	}
	return validateText("body", response.Body, maxCommitBodyLength)
}

func ValidateWhyResponse(response WhyResponse) error {
	if strings.TrimSpace(response.FilePurpose) == "" {
		return fmt.Errorf("ai: why response missing required field: file_purpose")
	}
	if err := validateText("file_purpose", response.FilePurpose, maxWhyPurposeLength); err != nil {
		return err
	}
	if err := validateText("current_state", response.CurrentState, maxWhyStateLength); err != nil {
		return err
	}
	if len(response.MajorDecisions) > maxPortfolioItems {
		return fmt.Errorf("ai: why response contains too many decisions")
	}
	for _, decision := range response.MajorDecisions {
		if strings.TrimSpace(decision.Date) == "" || strings.TrimSpace(decision.Description) == "" {
			return fmt.Errorf("ai: why response contains an incomplete decision")
		}
		if err := validateText("decision description", decision.Description, maxWhyDecisionLength); err != nil {
			return err
		}
	}
	return nil
}

func validateText(field, value string, limit int) error {
	if runeLen(value) > limit {
		return fmt.Errorf("ai: %s exceeds %d characters", field, limit)
	}
	return nil
}

func validateStringList(field string, values []string, maxItems, maxItemLength int) error {
	if len(values) > maxItems {
		return fmt.Errorf("ai: %s contains %d items; maximum is %d", field, len(values), maxItems)
	}
	for _, value := range values {
		if err := validateText(field+" item", strings.TrimSpace(value), maxItemLength); err != nil {
			return err
		}
	}
	return nil
}

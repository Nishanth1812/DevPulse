package ai

// BriefResponse is the structured output produced by the AI for the brief command.
type BriefResponse struct {
	// Summary is a 2–3 sentence narrative of the repository's current state.
	Summary string `json:"summary"`
	// KeyChanges lists up to 5 notable recent changes.
	KeyChanges []string `json:"key_changes"`
	// CurrentFocus is a one-line statement of the active area of work.
	CurrentFocus string `json:"current_focus"`
	// Blockers lists identified risks or blockers; may be empty.
	Blockers []string `json:"blockers"`
	// NextSteps lists up to 3 recommended actions.
	NextSteps []string `json:"next_steps"`
}

// CommitResponse is the structured output produced by the AI for the commit command.
type CommitResponse struct {
	// Subject is the first line of the commit message (<=72 chars, imperative mood).
	Subject string `json:"subject"`
	// Body is the optional extended description. May be an empty string.
	Body string `json:"body"`
}

// ResumeResponse is the structured output produced by the AI for the resume command.
type ResumeResponse struct {
	// WhatWasBuilt summarises the work completed in recent commits.
	WhatWasBuilt string `json:"what_was_built"`
	// WhatIsIncomplete lists work that was started but not finished.
	WhatIsIncomplete string `json:"what_is_incomplete"`
	// BlockersDetected lists any blockers or risks identified in the diffs.
	BlockersDetected []string `json:"blockers_detected"`
	// NextStep is the recommended single next action.
	NextStep string `json:"next_step"`
}

// FocusItem is a single entry in the focus ranked list.
type FocusItem struct {
	// RepoName is the name of the repository.
	RepoName string `json:"repo_name"`
	// RankReason is a one-line justification for the ranking.
	RankReason string `json:"rank_reason"`
	// ProximityScore is 1 (far from working) to 5 (nearly complete).
	ProximityScore int `json:"proximity_score"`
	// Urgency is true when a deadline within 14 days is detected.
	Urgency bool `json:"urgency"`
}

// FocusResponse is the structured output produced by the AI for the focus command.
type FocusResponse struct {
	// Ranked is the list of repositories ordered by completion proximity.
	Ranked []FocusItem `json:"ranked"`
}

// DecisionItem is a single decision in the file's history.
type DecisionItem struct {
	// Date is the commit date (YYYY-MM-DD).
	Date string `json:"date"`
	// Description explains what changed and why.
	Description string `json:"description"`
}

// WhyResponse is the structured output produced by the AI for the why command.
type WhyResponse struct {
	// FilePurpose describes the overall purpose of the file.
	FilePurpose string `json:"file_purpose"`
	// MajorDecisions lists significant changes made to the file over time.
	MajorDecisions []DecisionItem `json:"major_decisions"`
	// CurrentState describes the file's current role and condition.
	CurrentState string `json:"current_state"`
}

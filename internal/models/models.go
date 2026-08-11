package models

import "time"

type CollectOptions struct {
	MaxCommits  int
	Since       *time.Time
	IncludeDiff bool
	// FullDiffCommits is how many of the newest commits keep full diffs.
	// Older commits are reduced to one-line summaries. 0 means all commits
	// keep diffs (previous behaviour).
	FullDiffCommits int
}

type CommitSummary struct {
	SHA          string    `json:"sha"`
	Message      string    `json:"message"`
	Author       string    `json:"author"`
	Timestamp    time.Time `json:"timestamp"`
	FilesChanged []string  `json:"files_changed"`
	DiffSnippet  string    `json:"diff_snippet"`
}

type RepoData struct {
	Name           string          `json:"name"`
	Path           string          `json:"path"`
	Branch         string          `json:"branch"`
	HeadSHA        string          `json:"head_sha"`
	Commits        []CommitSummary `json:"commits"`
	PlanSummary    string          `json:"plan_summary"`
	ActiveBranches []string        `json:"active_branches"`
	Notes          string          `json:"notes"`
}

type Deadline struct {
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	DaysUntil   int       `json:"days_until"`
}

type GoalsData struct {
	Now       string     `json:"now"`
	Next      string     `json:"next"`
	Deadlines []Deadline `json:"deadlines"`
	Someday   string     `json:"someday"`
}

// RegisteredRepo represents a named repository registered with DevPulse.
type RegisteredRepo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

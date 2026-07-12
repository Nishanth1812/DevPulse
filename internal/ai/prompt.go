package ai

import (
	"fmt"
	"strings"

	"github.com/Nishanth1812/devpulse/internal/compressor"
	"github.com/Nishanth1812/devpulse/internal/models"
)

// BuildBriefPrompt constructs the AI prompt for a repository brief.
// It embeds repo data and goals context, and explicitly requests JSON-only output.
func BuildBriefPrompt(repo models.RepoData, goals models.GoalsData) string {
	var b strings.Builder

	b.WriteString("You are a software development assistant.\n")
	b.WriteString("Analyse the repository context below and produce a concise development brief.\n")
	b.WriteString("Respond with ONLY a valid JSON object matching this exact schema — no markdown, no explanation:\n\n")
	b.WriteString(`{"summary":"string","key_changes":["string"],"current_focus":"string","blockers":["string"],"next_steps":["string"]}`)
	b.WriteString("\n\n")

	headSHA := repo.HeadSHA
	if len(headSHA) > 7 {
		headSHA = headSHA[:7]
	}
	b.WriteString(fmt.Sprintf("## Repository: %s\n", repo.Name))
	b.WriteString(fmt.Sprintf("Branch: %s  HEAD: %s\n\n", repo.Branch, headSHA))

	if repo.PlanSummary != "" {
		b.WriteString("## Plan / Roadmap\n")
		b.WriteString(repo.PlanSummary)
		b.WriteString("\n\n")
	}

	if repo.Notes != "" {
		b.WriteString("## Notes\n")
		b.WriteString(repo.Notes)
		b.WriteString("\n\n")
	}

	if len(repo.Commits) > 0 {
		b.WriteString("## Recent Commits\n")
		for _, c := range repo.Commits {
			sha := c.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			b.WriteString(fmt.Sprintf("- %s %s (%s)\n", sha, c.Message, c.Author))
			if c.DiffSnippet != "" {
				b.WriteString("  ```\n")
				b.WriteString(c.DiffSnippet)
				b.WriteString("\n  ```\n")
			}
		}
		b.WriteString("\n")
	}

	if goals.Now != "" || goals.Next != "" || len(goals.Deadlines) > 0 {
		b.WriteString("## Goals\n")
		if goals.Now != "" {
			b.WriteString("Now: " + strings.TrimSpace(goals.Now) + "\n")
		}
		if goals.Next != "" {
			b.WriteString("Next: " + strings.TrimSpace(goals.Next) + "\n")
		}
		for _, d := range goals.Deadlines {
			b.WriteString(fmt.Sprintf("Deadline: %s — %d days away\n", d.Description, d.DaysUntil))
		}
		b.WriteString("\n")
	}

	b.WriteString("Respond with ONLY the JSON object. No prose before or after.\n")
	return b.String()
}

// BuildCommitPrompt constructs the AI prompt for generating a Conventional Commit message.
// The diff is compressed before embedding to avoid token overflow on large changesets.
func BuildCommitPrompt(diff string) string {
	var b strings.Builder

	b.WriteString("You are a software development assistant.\n")
	b.WriteString("Write a Conventional Commit message for the staged diff below.\n")
	b.WriteString("Respond with ONLY a valid JSON object matching this exact schema — no markdown, no explanation:\n\n")
	b.WriteString(`{"subject":"string (≤72 chars, type(scope): description, imperative mood, no trailing period)","body":"string (optional extended description, empty string if not needed)"}`)
	b.WriteString("\n\n")
	b.WriteString("## Staged Diff\n")
	b.WriteString("```\n")
	b.WriteString(compressor.CompressDiff(diff))
	b.WriteString("\n```\n\n")
	b.WriteString("Respond with ONLY the JSON object. No prose before or after.\n")
	return b.String()
}

// BuildResumePrompt constructs the AI prompt for deep context recovery.
// It embeds a longer commit window with diffs, plan context, and goals.
func BuildResumePrompt(repo models.RepoData, goals models.GoalsData) string {
	var b strings.Builder

	b.WriteString("You are a software development assistant.\n")
	b.WriteString("Analyse the repository context below and reconstruct what was built recently.\n")
	b.WriteString("Respond with ONLY a valid JSON object matching this exact schema — no markdown, no explanation:\n\n")
	b.WriteString(`{"what_was_built":"string","what_is_incomplete":"string","blockers_detected":["string"],"next_step":"string"}`)
	b.WriteString("\n\n")

	headSHA := repo.HeadSHA
	if len(headSHA) > 7 {
		headSHA = headSHA[:7]
	}
	b.WriteString(fmt.Sprintf("## Repository: %s\n", repo.Name))
	b.WriteString(fmt.Sprintf("Branch: %s  HEAD: %s\n\n", repo.Branch, headSHA))

	if repo.PlanSummary != "" {
		b.WriteString("## Plan / Roadmap\n")
		b.WriteString(repo.PlanSummary)
		b.WriteString("\n\n")
	}

	if repo.Notes != "" {
		b.WriteString("## Notes\n")
		b.WriteString(repo.Notes)
		b.WriteString("\n\n")
	}

	if len(repo.Commits) > 0 {
		b.WriteString("## Recent Commits (newest first)\n")
		for _, c := range repo.Commits {
			sha := c.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			b.WriteString(fmt.Sprintf("- %s %s (%s, %s)\n", sha, c.Message, c.Author, c.Timestamp.Format("2006-01-02")))
			if c.DiffSnippet != "" {
				b.WriteString("  ```\n")
				b.WriteString(c.DiffSnippet)
				b.WriteString("\n  ```\n")
			}
		}
		b.WriteString("\n")
	}

	if goals.Now != "" || goals.Next != "" || len(goals.Deadlines) > 0 {
		b.WriteString("## Goals\n")
		if goals.Now != "" {
			b.WriteString("Now: " + strings.TrimSpace(goals.Now) + "\n")
		}
		if goals.Next != "" {
			b.WriteString("Next: " + strings.TrimSpace(goals.Next) + "\n")
		}
		for _, d := range goals.Deadlines {
			b.WriteString(fmt.Sprintf("Deadline: %s — %d days away\n", d.Description, d.DaysUntil))
		}
		b.WriteString("\n")
	}

	b.WriteString("Focus on reconstructing a narrative: what was accomplished, what remains, and what the natural next step is.\n")
	b.WriteString("Respond with ONLY the JSON object. No prose before or after.\n")
	return b.String()
}

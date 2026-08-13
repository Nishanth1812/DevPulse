package collector

import (
	"testing"
	"time"

	"github.com/Nishanth1812/devpulse/internal/models"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initTestRepo creates a repo in a temp dir with a linear history of commits
// whose committer times are fixed (newest last). It returns the repo path.
func initTestRepo(t *testing.T, times []time.Time, messages []string) string {
	t.Helper()
	if len(times) != len(messages) {
		t.Fatal("times and messages must be the same length")
	}

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	// A file whose content changes so each commit has a distinct tree.
	const file = "a.txt"
	for i := range times {
		f, err := w.Filesystem.Create(file)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := f.Write([]byte(messages[i])); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if _, err := w.Add(file); err != nil {
			t.Fatalf("Add: %v", err)
		}
		sig := &object.Signature{Name: "tester", Email: "t@example.com", When: times[i]}
		if _, err := w.Commit(messages[i], &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}

	return dir
}

func TestCollectCommitsSince(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	mid := now.Add(-24 * time.Hour)
	newest := now

	dir := initTestRepo(t,
		[]time.Time{old, mid, newest},
		[]string{"old commit", "middle commit", "newest commit"},
	)

	// All commits (newest first).
	commits, branch, headSHA, err := CollectCommits(dir, models.CollectOptions{})
	if err != nil {
		t.Fatalf("CollectCommits: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}
	if commits[0].Message != "newest commit" {
		t.Fatalf("expected newest first, got %q", commits[0].Message)
	}
	if branch == "" || headSHA == "" {
		t.Fatal("expected branch and headSHA")
	}

	// --since should exclude only the commit older than the cutoff.
	since := mid.Add(-time.Hour)
	commits, _, _, err = CollectCommits(dir, models.CollectOptions{Since: &since})
	if err != nil {
		t.Fatalf("CollectCommits with Since: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits after --since, got %d", len(commits))
	}
	for _, c := range commits {
		if c.Timestamp.Before(since) {
			t.Fatalf("commit %q timestamp %v is before --since %v", c.Message, c.Timestamp, since)
		}
	}

	// Commits must be filtered by committer time, so using the author time in
	// the walk would still yield the correct window here.
	sinceNewer := newest.Add(-time.Hour)
	commits, _, _, err = CollectCommits(dir, models.CollectOptions{Since: &sinceNewer})
	if err != nil {
		t.Fatalf("CollectCommits with newer Since: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
}

func TestCollectCommitsMaxAndDiff(t *testing.T) {
	now := time.Now()
	dir := initTestRepo(t,
		[]time.Time{now.Add(-3 * time.Hour), now.Add(-2 * time.Hour), now.Add(-time.Hour)},
		[]string{"one", "two", "three"},
	)

	commits, _, _, err := CollectCommits(dir, models.CollectOptions{
		MaxCommits:      2,
		IncludeDiff:     true,
		FullDiffCommits: 1,
	})
	if err != nil {
		t.Fatalf("CollectCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits (MaxCommits=2), got %d", len(commits))
	}
	if commits[0].DiffSnippet == "" {
		t.Fatal("expected newest commit to carry a diff (FullDiffCommits=1)")
	}
	if commits[1].DiffSnippet != "" {
		t.Fatal("expected second commit to have its diff reduced (older than FullDiffCommits)")
	}
}

func TestCollectFileCommits(t *testing.T) {
	now := time.Now()
	dir := initTestRepo(t,
		[]time.Time{now.Add(-3 * time.Hour), now.Add(-2 * time.Hour), now.Add(-time.Hour)},
		[]string{"one", "two", "three"},
	)

	commits, err := CollectFileCommits(dir, "a.txt", 50, 15, true)
	if err != nil {
		t.Fatalf("CollectFileCommits: %v", err)
	}
	// Oldest-first ordering for the "why" narrative.
	if len(commits) != 3 {
		t.Fatalf("expected 3 file commits, got %d", len(commits))
	}
	// The root commit has no parent, so it carries no diff.
	if commits[0].Message != "one" || commits[len(commits)-1].Message != "three" {
		t.Fatalf("expected oldest-first order, got %q..%q", commits[0].Message, commits[len(commits)-1].Message)
	}
	for _, c := range commits[1:] {
		if c.DiffSnippet == "" {
			t.Fatalf("expected diff for commit %q", c.Message)
		}
	}

	// includeDiff=false must drop all diff snippets.
	noDiff, err := CollectFileCommits(dir, "a.txt", 50, 15, false)
	if err != nil {
		t.Fatalf("CollectFileCommits(no diff): %v", err)
	}
	for _, c := range noDiff {
		if c.DiffSnippet != "" {
			t.Fatalf("expected no diff for commit %q when includeDiff=false", c.Message)
		}
	}
}

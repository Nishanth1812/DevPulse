package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func healthTestRepo(t *testing.T, times []time.Time, messages []string) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	for i := range times {
		f, err := w.Filesystem.Create("a.txt")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := f.Write([]byte(messages[i])); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if _, err := w.Add("a.txt"); err != nil {
			t.Fatalf("Add: %v", err)
		}
		sig := &object.Signature{Name: "t", Email: "t@example.com", When: times[i]}
		if _, err := w.Commit(messages[i], &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}
	return dir
}

func TestCheckStaleRepoFlagsPreviouslyActive(t *testing.T) {
	now := time.Now()
	dir := healthTestRepo(t,
		[]time.Time{now.Add(-60 * 24 * time.Hour), now.Add(-30 * 24 * time.Hour)},
		[]string{"old", "older"},
	)
	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	issues := checkStaleRepo(r, "repo-a")
	if len(issues) == 0 {
		t.Fatal("expected STALE issue for previously-active repo with no recent commits")
	}
	if issues[0].Kind != "STALE" {
		t.Fatalf("expected STALE kind, got %q", issues[0].Kind)
	}
}

func TestCheckStaleRepoIgnoresBrandNewRepo(t *testing.T) {
	now := time.Now()
	dir := healthTestRepo(t, []time.Time{now.Add(-30 * 24 * time.Hour)}, []string{"only commit"})
	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	issues := checkStaleRepo(r, "repo-b")
	// A single-commit repo that has been quiet has no prior activity to lose.
	if len(issues) != 0 {
		t.Fatalf("brand-new repo should not be flagged STALE, got %+v", issues)
	}
}

func TestCheckStaleRepoActiveRepoNoIssue(t *testing.T) {
	now := time.Now()
	dir := healthTestRepo(t,
		[]time.Time{now.Add(-30 * 24 * time.Hour), now.Add(-time.Hour)},
		[]string{"old", "recent"},
	)
	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	if issues := checkStaleRepo(r, "repo-c"); len(issues) != 0 {
		t.Fatalf("recently-active repo should have no STALE issue, got %+v", issues)
	}
}

func TestCheckTodoAccumulationCounts(t *testing.T) {
	dir := t.TempDir()
	// A file with TODO/FIXME comments.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// TODO fix this\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	issues := checkTodoAccumulation(dir, "repo-todo")
	if len(issues) == 0 {
		t.Fatal("expected TODO issue")
	}
	if issues[0].Kind != "TODO" {
		t.Fatalf("expected TODO kind, got %q", issues[0].Kind)
	}
}

func TestCheckTodoAccumulationIgnoresNonCode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# TODO\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// README.md is not in the code extension list.
	if issues := checkTodoAccumulation(dir, "repo-todo2"); len(issues) != 0 {
		t.Fatalf("markdown TODO should not be counted, got %+v", issues)
	}
}

func TestCheckMergedBranches(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	now := time.Now()
	sig := &object.Signature{Name: "t", Email: "t@example.com", When: now}

	write := func(name, content string) {
		f, err := w.Filesystem.Create(name)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if _, err := w.Add(name); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	// main with one commit.
	write("a.txt", "base")
	if _, err := w.Commit("base", &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("Commit base: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	mainBranch := head.Name().Short()

	// A feature branch fully merged back into main (fast-forwarded by moving
	// the main ref to the feature commit).
	if err := w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("feature"), Create: true}); err != nil {
		t.Fatalf("Checkout feature: %v", err)
	}
	write("a.txt", "feature change")
	featCommit, err := w.Commit("feature work", &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatalf("Commit feature: %v", err)
	}
	mainRef, err := repo.Reference(plumbing.NewBranchReferenceName(mainBranch), false)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(mainRef.Name(), featCommit)); err != nil {
		t.Fatalf("fast-forward main: %v", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(mainBranch)}); err != nil {
		t.Fatalf("Checkout main: %v", err)
	}

	// A second, unmerged branch.
	if err := w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("wip"), Create: true}); err != nil {
		t.Fatalf("Checkout wip: %v", err)
	}
	write("b.txt", "unmerged work")
	if _, err := w.Commit("wip work", &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("Commit wip: %v", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(mainBranch)}); err != nil {
		t.Fatalf("Checkout main: %v", err)
	}

	issues := checkMergedBranches(repo, "repo-merged")
	found := false
	for _, is := range issues {
		if is.Kind == "BRANCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected BRANCH issue for merged-but-not-deleted feature branch, got %+v", issues)
	}
	_ = featCommit
}

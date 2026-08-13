package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Nishanth1812/devpulse/internal/output"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Per-repo hygiene: detect stale branches, TODO accumulation, and silent repos",
	Long: `Fully rule-based (no LLM). Detects:
- Merged branches not deleted
- TODO/FIXME line counts per repo
- Repos whose last commit is more than 14 days old but that have earlier
  history (previously active, now silent)`,
	Args: cobra.NoArgs,
	RunE: runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

type healthIssue struct {
	Repo    string
	Kind    string
	Message string
}

func runHealth(cmd *cobra.Command, args []string) error {
	repos := manager.ListRepositories()
	if len(repos) == 0 {
		return fmt.Errorf("health: no repositories registered")
	}

	spinner := output.NewSpinner(noColor)
	spinner.Start("Scanning repositories...")

	var issues []healthIssue

	for _, repo := range repos {
		repoIssues := checkRepoHealth(repo.Path, repo.Name)
		issues = append(issues, repoIssues...)
	}

	spinner.Stop()

	return renderHealth(cmd.OutOrStdout(), issues)
}

func checkRepoHealth(repoPath, repoName string) []healthIssue {
	var issues []healthIssue

	// Check if it's a valid git repo
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		issues = append(issues, healthIssue{
			Repo:    repoName,
			Kind:    "ERROR",
			Message: fmt.Sprintf("not a valid git repository: %s", err),
		})
		return issues
	}

	// Check for stale branches (merged but not deleted)
	issues = append(issues, checkMergedBranches(r, repoName)...)

	// Check for TODO/FIXME accumulation
	issues = append(issues, checkTodoAccumulation(repoPath, repoName)...)

	// Check for stale repos (no commits in 14 days)
	issues = append(issues, checkStaleRepo(r, repoName)...)

	return issues
}

func checkMergedBranches(r *git.Repository, repoName string) []healthIssue {
	var issues []healthIssue

	ref, err := r.Head()
	if err != nil {
		return issues
	}

	branches, err := r.Branches()
	if err != nil {
		return issues
	}

	mainBranch := ref.Name().Short()
	mergedCount := 0

	branches.ForEach(func(b *plumbing.Reference) error {
		name := b.Name().Short()
		// Skip main/master and current branch
		if name == mainBranch || name == "main" || name == "master" || name == "develop" {
			return nil
		}

		// Check if branch is fully merged into main
		mainRef, err := r.Reference(plumbing.NewBranchReferenceName(mainBranch), false)
		if err != nil {
			return nil
		}

		commit, err := r.CommitObject(b.Hash())
		if err != nil {
			return nil
		}

		mainCommit, err := r.CommitObject(mainRef.Hash())
		if err != nil {
			return nil
		}

		isMerged, err := commit.IsAncestor(mainCommit)
		if err == nil && isMerged {
			mergedCount++
		}

		return nil
	})

	if mergedCount > 0 {
		issues = append(issues, healthIssue{
			Repo:    repoName,
			Kind:    "BRANCH",
			Message: fmt.Sprintf("%d merged branch(es) not deleted", mergedCount),
		})
	}

	return issues
}

func checkTodoAccumulation(repoPath, repoName string) []healthIssue {
	var issues []healthIssue

	todoCount := 0

	todoPattern := regexp.MustCompile(`(?i)\b(TODO|FIXME)\b`)

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip hidden dirs and vendor
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only check code files
		ext := filepath.Ext(path)
		switch ext {
		case ".go", ".js", ".ts", ".py", ".rs", ".java", ".rb", ".php", ".c", ".cpp", ".h":
			// continue
		default:
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if todoPattern.MatchString(line) {
				todoCount++
			}
		}

		return nil
	})

	if todoCount > 20 {
		issues = append(issues, healthIssue{
			Repo:    repoName,
			Kind:    "TODO",
			Message: fmt.Sprintf("%d TODO/FIXME comments found (high accumulation)", todoCount),
		})
	} else if todoCount > 0 {
		issues = append(issues, healthIssue{
			Repo:    repoName,
			Kind:    "TODO",
			Message: fmt.Sprintf("%d TODO/FIXME comment(s)", todoCount),
		})
	}

	return issues
}

func checkStaleRepo(r *git.Repository, repoName string) []healthIssue {
	var issues []healthIssue

	ref, err := r.Head()
	if err != nil {
		return issues
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return issues
	}

	daysSinceLastCommit := time.Since(commit.Committer.When).Hours() / 24

	if daysSinceLastCommit > 14 {
		// Only flag repos that were active before going quiet: a brand-new repo
		// with a single commit has no "regular activity" to have lost.
		hasHistory := false
		iter, err := r.Log(&git.LogOptions{From: commit.Hash, Order: git.LogOrderCommitterTime})
		if err == nil {
			iter.ForEach(func(c *object.Commit) error {
				if time.Since(c.Committer.When).Hours()/24 > 14 {
					hasHistory = true
					return storer.ErrStop
				}
				return nil
			})
			iter.Close()
		}

		if hasHistory {
			issues = append(issues, healthIssue{
				Repo:    repoName,
				Kind:    "STALE",
				Message: fmt.Sprintf("no commits in %d days (last: %s)", int(daysSinceLastCommit), commit.Committer.When.Format("2006-01-02")),
			})
		}
	}

	return issues
}

func renderHealth(w io.Writer, issues []healthIssue) error {
	if _, err := fmt.Fprintf(w, "\n=== Health Report ===\n\n"); err != nil {
		return err
	}

	if len(issues) == 0 {
		_, err := fmt.Fprintln(w, "All repositories look healthy!")
		return err
	}

	// Group by repo
	byRepo := make(map[string][]healthIssue)
	for _, issue := range issues {
		byRepo[issue.Repo] = append(byRepo[issue.Repo], issue)
	}

	for repo, repoIssues := range byRepo {
		if _, err := fmt.Fprintf(w, "%s:\n", repo); err != nil {
			return err
		}
		for _, issue := range repoIssues {
			icon := "  *"
			switch issue.Kind {
			case "ERROR":
				icon = "  !"
			case "STALE":
				icon = "  -"
			case "BRANCH":
				icon = "  ~"
			}
			if _, err := fmt.Fprintf(w, "%s [%s] %s\n", icon, issue.Kind, issue.Message); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	_, err := fmt.Fprintf(w, "Found %d issue(s) across %d repo(s)\n", len(issues), len(byRepo))
	return err
}

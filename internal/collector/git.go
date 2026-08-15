package collector

import (
	"strings"

	"github.com/Nishanth1812/devpulse/internal/compressor"
	"github.com/Nishanth1812/devpulse/internal/models"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

func CollectCommits(
	repoPath string,
	opts models.CollectOptions,
) ([]models.CommitSummary, string, string, error) {

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, "", "", err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, "", "", err
	}

	branch := head.Name().Short()
	headSHA := head.Hash().String()

	// LogOrderCommitterTime walks commits newest-first by committer time
	// (matching `git log --since` semantics). Without it, the default DFS
	// pre-order can visit an old commit early, making a `storer.ErrStop` on the
	// first out-of-window commit drop valid in-window commits.
	refIter, err := repo.Log(&git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	})

	if err != nil {
		return nil, "", "", err
	}

	var commits []models.CommitSummary

	count := 0

	err = refIter.ForEach(func(commit *object.Commit) error {
		if opts.MaxCommits > 0 && count >= opts.MaxCommits {
			return storer.ErrStop
		}

		// Skip commits before the --since date. Because the iterator is ordered
		// by committer time (newest first), the first out-of-window commit means
		// every later commit is also out of window, so stopping is safe.
		if opts.Since != nil && commit.Committer.When.Before(*opts.Since) {
			return storer.ErrStop
		}

		// Commits older than the full-diff window collapse to one-line summaries
		// (timestamp + message) to bound prompt size.
		includeFull := opts.IncludeDiff &&
			(opts.FullDiffCommits == 0 || count < opts.FullDiffCommits)

		var files []string
		var diffText string

		if includeFull {
			parent, err := commit.Parent(0)

			if err == nil {
				patch, err := parent.Patch(commit)

				if err == nil {
					diffText = compressor.CompressDiff(
						patch.String(),
					)

					stats := patch.Stats()

					for _, stat := range stats {
						files = append(files, stat.Name)
					}
				}
			}
		}

		commits = append(commits, models.CommitSummary{
			SHA:          commit.Hash.String(),
			Message:      strings.TrimSpace(commit.Message),
			Author:       commit.Author.Name,
			Timestamp:    commit.Committer.When,
			FilesChanged: files,
			DiffSnippet:  diffText,
		})

		count++

		return nil
	})

	if err != nil {
		return nil, "", "", err
	}

	return commits, branch, headSHA, nil
}

// CollectFileCommits returns the commit history for a specific file path.
// Commits are ordered oldest-first to support the "why" narrative.
// Only the newest fullDiffCommits commits keep their diffs; older ones are
// reduced to messages to bound prompt size (0 = keep all diffs).
func CollectFileCommits(repoPath, filePath string, maxCommits, fullDiffCommits int, includeDiff bool) ([]models.CommitSummary, error) {
	normalizedFilePath, err := NormalizeRepoRelativePath(repoPath, filePath)
	if err != nil {
		return nil, err
	}
	filePath = normalizedFilePath

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	// PathFilter walks the log using tree diffs between successive commits and
	// only yields commits that touched the target path, avoiding a full-history
	// commit.Stats() scan (which is O(total commits) and reads every tree).
	refIter, err := repo.Log(&git.LogOptions{
		From:       head.Hash(),
		PathFilter: func(path string) bool { return path == filePath },
	})
	if err != nil {
		return nil, err
	}

	var commits []models.CommitSummary
	matched := 0

	err = refIter.ForEach(func(commit *object.Commit) error {
		if maxCommits > 0 && len(commits) >= maxCommits {
			return storer.ErrStop
		}

		// Get the diff for this file (only for the newest fullDiffCommits commits)
		var diffText string
		if includeDiff && (fullDiffCommits == 0 || matched < fullDiffCommits) {
			parent, err := commit.Parent(0)
			if err == nil {
				patch, err := parent.Patch(commit)
				if err == nil {
					for _, fp := range patch.FilePatches() {
						if filePatchTouches(fp, filePath) {
							diffText = compressor.CompressDiff(filePatchDiffText(fp))
							break
						}
					}
				}
			}
		}
		matched++

		commits = append(commits, models.CommitSummary{
			SHA:         commit.Hash.String(),
			Message:     strings.TrimSpace(commit.Message),
			Author:      commit.Author.Name,
			Timestamp:   commit.Committer.When,
			DiffSnippet: diffText,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Reverse to oldest-first
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, nil
}

// filePatchTouches reports whether a file patch involves the target path,
// checking both the old (pre-rename) and new side.
func filePatchTouches(fp diff.FilePatch, filePath string) bool {
	from, to := fp.Files()
	if from != nil && from.Path() == filePath {
		return true
	}
	if to != nil && to.Path() == filePath {
		return true
	}
	return false
}

// filePatchDiffText renders a single file patch's changed lines (not the whole
// commit diff), prefixing each line with its + / - / context marker so it can be
// fed through the compressor like a normal diff.
func filePatchDiffText(fp diff.FilePatch) string {
	var b strings.Builder

	for _, chunk := range fp.Chunks() {
		marker := byte(' ')
		switch chunk.Type() {
		case diff.Add:
			marker = '+'
		case diff.Delete:
			marker = '-'
		}

		for _, line := range strings.Split(chunk.Content(), "\n") {
			if line == "" {
				continue
			}
			b.WriteByte(marker)
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	return b.String()
}

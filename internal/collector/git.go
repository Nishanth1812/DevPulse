package collector

import (
	"strings"

	"github.com/Nishanth1812/devpulse/internal/compressor"
	"github.com/Nishanth1812/devpulse/internal/models"
	git "github.com/go-git/go-git/v5"
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

	refIter, err := repo.Log(&git.LogOptions{
		From: head.Hash(),
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

		// Skip commits before the --since date
		if opts.Since != nil && commit.Author.When.Before(*opts.Since) {
			return storer.ErrStop
		}

		var files []string
		var diffText string

		if opts.IncludeDiff {
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
			Timestamp:    commit.Author.When,
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
func CollectFileCommits(repoPath, filePath string, maxCommits int) ([]models.CommitSummary, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	refIter, err := repo.Log(&git.LogOptions{
		From: head.Hash(),
	})
	if err != nil {
		return nil, err
	}

	var commits []models.CommitSummary

	err = refIter.ForEach(func(commit *object.Commit) error {
		if maxCommits > 0 && len(commits) >= maxCommits {
			return storer.ErrStop
		}

		// Check if this commit touched the file
		stats, err := commit.Stats()
		if err != nil {
			return nil
		}

		touched := false
		for _, stat := range stats {
			if stat.Name == filePath {
				touched = true
				break
			}
		}
		if !touched {
			return nil
		}

		// Get the diff for this file
		var diffText string
		parent, err := commit.Parent(0)
		if err == nil {
			patch, err := parent.Patch(commit)
			if err == nil {
				// Filter patch to only the target file
				for _, stat := range patch.Stats() {
					if stat.Name == filePath {
						diffText = compressor.CompressDiff(patch.String())
						break
					}
				}
			}
		}

		commits = append(commits, models.CommitSummary{
			SHA:         commit.Hash.String(),
			Message:     strings.TrimSpace(commit.Message),
			Author:      commit.Author.Name,
			Timestamp:   commit.Author.When,
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

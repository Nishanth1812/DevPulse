package collector

import (
	"devpulse/internal/compressor"
	"devpulse/internal/models"
	"strings"

	git "github.com/go-git/go-git/v5"
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

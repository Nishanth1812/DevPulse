package collector

import (
	"devpulse/internal/compressor"
	"devpulse/internal/models"
	"devpulse/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

var planFiles = []string{
	"PLAN.md",
	"ROADMAP.md",
	"TODO.md",
	"CHANGELOG.md",
	"NOTES.md",
	"docs/PLAN.md",
	"docs/ROADMAP.md",
}

func CollectRepo(
	path string,
	opts models.CollectOptions,
) (models.RepoData, error) {

	absPath, err := filepath.Abs(path)
	if err != nil {
		return models.RepoData{}, err
	}

	commits, branch, headSHA, err := CollectCommits(
		absPath,
		opts,
	)

	if err != nil {
		return models.RepoData{}, err
	}

	planSummary := loadPlanFiles(absPath)

	repoName := filepath.Base(absPath)

	notes := loadNotes(repoName)

	return models.RepoData{
		Name:           repoName,
		Path:           absPath,
		Branch:         branch,
		HeadSHA:        headSHA,
		Commits:        commits,
		PlanSummary:    planSummary,
		ActiveBranches: []string{branch},
		Notes:          notes,
	}, nil
}

func loadPlanFiles(repoPath string) string {
	var combined strings.Builder

	for _, file := range planFiles {
		path := filepath.Join(repoPath, file)

		if !utils.FileExists(path) {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		combined.WriteString(string(content))
		combined.WriteString("\n")
	}

	return compressor.CompressPlan(combined.String())
}

func loadNotes(repo string) string {
	path := utils.NotesPath(repo)

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(content)
}

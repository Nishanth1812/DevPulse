package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Nishanth1812/devpulse/internal/compressor"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/utils"
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
		Name:        repoName,
		Path:        absPath,
		Branch:      branch,
		HeadSHA:     headSHA,
		Commits:     commits,
		PlanSummary: planSummary,
		Notes:       notes,
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

	return compressor.CompressNotes(string(content))
}

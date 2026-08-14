package services

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/models"
	"github.com/Nishanth1812/devpulse/internal/utils"
)

type CollectService struct{}

func (s *CollectService) Run(
	path string,
) (models.RepoData, error) {

	return collector.CollectRepo(
		path,
		models.CollectOptions{
			MaxCommits:  20,
			IncludeDiff: true,
		},
	)
}

type NoteService struct{}

func (s *NoteService) Add(
	repo string,
	text string,
) error {

	safeRepo, err := utils.SanitizeRepoName(repo)
	if err != nil {
		return err
	}

	path := utils.NotesPath(safeRepo)

	file, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)

	if err != nil {
		return err
	}

	entry := fmt.Sprintf(
		"<!-- %s --> %s\n",
		time.Now().Format("2006-01-02 15:04"),
		text,
	)

	_, err = file.WriteString(entry)
	if err != nil {
		_ = file.Close()
		return err
	}
	// Close explicitly so a buffered write failure is surfaced instead of
	// silently dropping the note.
	return file.Close()
}

func (s *NoteService) List(repo string) (string, error) {
	safeRepo, err := utils.SanitizeRepoName(repo)
	if err != nil {
		return "", err
	}

	path := utils.NotesPath(safeRepo)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	return utils.ReadFile(path)
}

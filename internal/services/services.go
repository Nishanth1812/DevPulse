package services

import (
	"devpulse/internal/collector"
	"devpulse/internal/models"
	"devpulse/internal/utils"
	"fmt"
	"os"
	"time"
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

	path := utils.NotesPath(repo)

	file, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)

	if err != nil {
		return err
	}

	defer file.Close()

	entry := fmt.Sprintf(
		"<!-- %s --> %s\n",
		time.Now().Format("2006-01-02 15:04"),
		text,
	)

	_, err = file.WriteString(entry)

	return err
}

func (s *NoteService) List(repo string) (string, error) {
	return utils.ReadFile(utils.NotesPath(repo))
}

func (s *NoteService) Clear(repo string) error {
	return os.Remove(utils.NotesPath(repo))
}

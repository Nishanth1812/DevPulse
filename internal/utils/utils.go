package utils

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ExpandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return strings.Replace(path, "~", home, 1)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0700)
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func NotesPath(repo string) string {
	return filepath.Join(
		ExpandHome("~/.devpulse/notes"),
		repo+".md",
	)
}

func GoalsPath() string {
	return ExpandHome("~/.devpulse/goals.md")
}

func DaysUntil(t time.Time) int {
	return int(time.Until(t).Hours() / 24)
}

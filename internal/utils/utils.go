package utils

import (
	"fmt"
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

// SanitizeRepoName validates that a name is safe to use as a single filename
// segment for note storage. It rejects empty names, names containing path
// separators, or ".." segments, which would otherwise let a caller escape the
// notes directory via path traversal (e.g. "../../.bashrc").
func SanitizeRepoName(name string) (string, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "", fmt.Errorf("repo name cannot be empty")
	}
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("repo name %q must not contain '..'", clean)
	}
	if strings.ContainsRune(clean, filepath.Separator) || strings.ContainsRune(clean, '/') {
		return "", fmt.Errorf("repo name %q must not contain path separators", clean)
	}
	return clean, nil
}

package utils

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nishanth1812/devpulse/internal/config"
)

// ExpandHome expands a leading "~" to the user's home directory. It only
// treats "~" as home when it is the first character and is followed by a path
// separator or is the whole string, so paths like "~/a" expand but a literal
// "~" inside a path (e.g. "x~y") is left untouched.
func ExpandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
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

// WorkspaceBaseDir resolves the DevPulse data root, honouring DEVPULSE_CONFIG
// so notes/goals/cache/logs follow the config location.
func WorkspaceBaseDir() string {
	base, err := config.WorkspaceBaseDir()
	if err != nil {
		return ExpandHome("~/.devpulse")
	}
	return base
}

func NotesPath(repo string) string {
	return filepath.Join(
		filepath.Join(WorkspaceBaseDir(), "notes"),
		repo+".md",
	)
}

func GoalsPath() string {
	return filepath.Join(WorkspaceBaseDir(), "goals.md")
}

// DaysUntil returns the number of days until t, rounded half away from zero so
// a deadline a few hours from now reads 0, one a day out reads 1, and a
// deadline that passed reads negative — past and future are never conflated by
// naive truncation.
func DaysUntil(t time.Time) int {
	return int(math.Round(time.Until(t).Hours() / 24))
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

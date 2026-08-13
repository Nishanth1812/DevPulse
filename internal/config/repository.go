package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Nishanth1812/devpulse/internal/models"
	git "github.com/go-git/go-git/v5"
)

func (m *Manager) ListRepositories() []models.RegisteredRepo {
	repos := make([]models.RegisteredRepo, 0, len(m.config.RegisteredRepos))
	for name, path := range m.config.RegisteredRepos {
		repos = append(repos, models.RegisteredRepo{Name: name, Path: path})
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos
}

func (m *Manager) RegisterRepository(path string) (models.RegisteredRepo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return models.RegisteredRepo{}, fmt.Errorf("stat repository path %q: %w", path, err)
	}
	if !info.IsDir() {
		return models.RegisteredRepo{}, fmt.Errorf("repository path %q is not a directory", path)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return models.RegisteredRepo{}, fmt.Errorf("resolve absolute repository path: %w", err)
	}
	absolutePath = filepath.Clean(absolutePath)
	name := filepath.Base(absolutePath)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return models.RegisteredRepo{}, fmt.Errorf("could not derive repository name from %q", absolutePath)
	}

	// Only git repositories are usable; validate early so a mistaken path is
	// rejected here rather than surfacing confusing errors from every command.
	if _, err := git.PlainOpen(absolutePath); err != nil {
		return models.RegisteredRepo{}, fmt.Errorf("path %q is not a git repository: %w", absolutePath, err)
	}

	if _, exists := m.config.RegisteredRepos[name]; exists {
		return models.RegisteredRepo{}, fmt.Errorf("repository %q is already registered", name)
	}
	for existingName, existingPath := range m.config.RegisteredRepos {
		if samePath(existingPath, absolutePath) {
			return models.RegisteredRepo{}, fmt.Errorf("path %q is already registered as %q", absolutePath, existingName)
		}
	}

	repo := models.RegisteredRepo{Name: name, Path: absolutePath}
	m.config.RegisteredRepos[name] = absolutePath
	if err := m.Save(); err != nil {
		return models.RegisteredRepo{}, err
	}

	return repo, nil
}

func (m *Manager) UnregisterRepository(name string) error {
	if _, exists := m.config.RegisteredRepos[name]; !exists {
		return fmt.Errorf("repository %q is not registered", name)
	}

	delete(m.config.RegisteredRepos, name)
	if err := m.Save(); err != nil {
		return err
	}

	return nil
}

// RepositoryPath returns the absolute path for a registered repository name.
// Returns false if the name is not registered.
func (m *Manager) RepositoryPath(name string) (string, bool) {
	path, ok := m.config.RegisteredRepos[name]
	return path, ok
}

func samePath(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}

	return left == right
}

package collector

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NormalizeRepoRelativePath converts a user-supplied file path into the slash
// separated form expected by Git and rejects paths outside repoPath.
func NormalizeRepoRelativePath(repoPath, filePath string) (string, error) {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Accept Windows separators even when the caller is using a shell that
	// normally emits POSIX separators. A colon in the second position catches
	// a Windows volume path when tests run on a POSIX host.
	trimmed = strings.ReplaceAll(trimmed, "\\", string(filepath.Separator))
	if filepath.IsAbs(trimmed) || (len(trimmed) >= 2 && trimmed[1] == ':') {
		return "", fmt.Errorf("file path %q must be relative to the repository", filePath)
	}

	clean := filepath.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file path %q escapes the repository", filePath)
	}

	repoRoot, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	repoRoot = filepath.Clean(repoRoot)

	relative, err := filepath.Rel(repoRoot, filepath.Join(repoRoot, clean))
	if err != nil {
		return "", fmt.Errorf("resolve file path %q: %w", filePath, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file path %q escapes the repository", filePath)
	}

	return filepath.ToSlash(relative), nil
}

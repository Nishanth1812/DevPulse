package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) InitializeGoalsFile() (created bool, path string, hasAPIKey bool, err error) {
	path = filepath.Join(m.baseDir, "goals.md")
	if _, err := os.Stat(path); err == nil {
		hasAPIKey, keyErr := HasAPIKey("groq")
		if keyErr != nil {
			return false, path, false, keyErr
		}
		return false, path, hasAPIKey, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, path, false, fmt.Errorf("stat goals file: %w", err)
	}

	content := strings.TrimSpace(`# DevPulse Goals

## Current Focus
- Keep repository work visible and intentional.

## Short Term
- Register active repositories.
- Build useful daily development context.

## Long Term
- Prepare AI-powered repository briefings.

## Notes
- Add project-specific notes here.
`) + "\n"

	if err := os.WriteFile(path, []byte(content), filePermission); err != nil {
		return false, path, false, fmt.Errorf("write goals file: %w", err)
	}

	hasAPIKey, err = HasAPIKey("groq")
	if err != nil {
		return true, path, false, err
	}

	return true, path, hasAPIKey, nil
}

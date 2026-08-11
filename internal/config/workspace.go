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

## Now
Things you are actively trying to finish this sprint or this week.

## Next
Things that become active once the Now items are done.

## Deadlines
YYYY-MM-DD — description. DevPulse reads these and flags urgency.

## Someday
Projects or ideas you are not touching now but do not want to forget.
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

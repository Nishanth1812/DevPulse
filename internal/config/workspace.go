package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hasAnyAPIKey reports whether the user has configured a key for either
// provider. It treats a keyring/env check failure as "no key" so setup advice
// is not blocked by a transient keyring error.
func hasAnyAPIKey() (bool, error) {
	for _, p := range []string{"groq", "gemini"} {
		has, err := HasAPIKey(p)
		if err == nil && has {
			return true, nil
		}
	}
	// Distinguish "no key anywhere" from a genuine lookup error.
	_, err := HasAPIKey("groq")
	return false, err
}

func (m *Manager) InitializeGoalsFile() (created bool, path string, hasAPIKey bool, err error) {
	path = filepath.Join(m.baseDir, "goals.md")
	hasAPIKey, err = hasAnyAPIKey()
	if err != nil {
		return false, path, false, err
	}
	if _, err := os.Stat(path); err == nil {
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

	return true, path, hasAPIKey, nil
}

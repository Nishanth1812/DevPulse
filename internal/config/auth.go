package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

func HasGeminiAPIKey() (bool, error) {
	_, err := keyring.Get(KeyringService, GeminiAPIKeyAccount)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("read Gemini API key from keychain: %w", err)
}

func PromptAndStoreGeminiAPIKey(stdin *os.File, stderr io.Writer) error {
	if _, err := fmt.Fprint(stderr, "Enter Gemini API key: "); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	keyBytes, err := term.ReadPassword(int(stdin.Fd()))
	if err != nil {
		return fmt.Errorf("read Gemini API key: %w", err)
	}
	if _, err := fmt.Fprintln(stderr); err != nil {
		return fmt.Errorf("write prompt newline: %w", err)
	}

	key := strings.TrimSpace(string(keyBytes))
	if key == "" {
		return fmt.Errorf("Gemini API key cannot be empty")
	}

	if err := keyring.Set(KeyringService, GeminiAPIKeyAccount, key); err != nil {
		return fmt.Errorf("store Gemini API key in keychain: %w", err)
	}

	return nil
}

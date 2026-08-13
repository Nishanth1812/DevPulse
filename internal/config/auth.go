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

func apiKeyAccount(provider string) string {
	return provider + "-api-key"
}

// HasAPIKey reports whether an API key is available for the provider, matching
// the precedence of GetAPIKey: the *_API_KEY environment variable first, then
// the OS keychain.
func HasAPIKey(provider string) (bool, error) {
	envVar := strings.ToUpper(provider) + "_API_KEY"
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return true, nil
	}

	_, err := keyring.Get(KeyringService, apiKeyAccount(provider))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("read %s API key from keychain: %w", provider, err)
}

func GetAPIKey(provider string) (string, error) {
	envVar := strings.ToUpper(provider) + "_API_KEY"
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v, nil
	}

	key, err := keyring.Get(KeyringService, apiKeyAccount(provider))
	if err == nil {
		return strings.TrimSpace(key), nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("no %s API key found: set %s environment variable or run: devpulse auth --provider %s", provider, envVar, provider)
	}
	return "", fmt.Errorf("read %s API key from keychain: %w", provider, err)
}

func PromptAndStoreAPIKey(provider string, stdin *os.File, stderr io.Writer) error {
	if _, err := fmt.Fprintf(stderr, "Enter %s API key: ", provider); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	keyBytes, err := term.ReadPassword(int(stdin.Fd()))
	if err != nil {
		return fmt.Errorf("read %s API key: %w", provider, err)
	}
	if _, err := fmt.Fprintln(stderr); err != nil {
		return fmt.Errorf("write prompt newline: %w", err)
	}

	key := strings.TrimSpace(string(keyBytes))
	if key == "" {
		return fmt.Errorf("%s API key cannot be empty", provider)
	}

	if err := keyring.Set(KeyringService, apiKeyAccount(provider), key); err != nil {
		return fmt.Errorf("store %s API key in keychain: %w", provider, err)
	}

	return nil
}

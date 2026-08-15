package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	ConfigEnv             = "DEVPULSE_CONFIG"
	KeyringService        = "devpulse"
	defaultCacheHours     = 24
	defaultFuzzyThreshold = 50
	dirPermission         = 0o700
	filePermission        = 0o600
)

type Config struct {
	RegisteredRepos    map[string]string `toml:"registered_repos"`
	ModelFast          string            `toml:"model_fast"`
	ModelDeep          string            `toml:"model_deep"`
	CacheDurationHours int               `toml:"cache_duration_hours"`
	FuzzyThreshold     int               `toml:"fuzzy_threshold"`
}

type Manager struct {
	configPath string
	baseDir    string
	config     Config
}

func Load() (*Manager, error) {
	baseDir, err := WorkspaceBaseDir()
	if err != nil {
		return nil, err
	}

	configPath, err := resolveConfigPath(baseDir)
	if err != nil {
		return nil, err
	}

	if err := ensureWorkspace(baseDir, configPath); err != nil {
		return nil, err
	}

	manager := &Manager{
		configPath: configPath,
		baseDir:    baseDir,
		config:     defaultConfig(),
	}

	if _, err := os.Stat(configPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat config file: %w", err)
		}
		if err := manager.Save(); err != nil {
			return nil, err
		}
		return manager, nil
	}

	if _, err := toml.DecodeFile(configPath, &manager.config); err != nil {
		return nil, fmt.Errorf("decode config file %q: %w", configPath, err)
	}
	if manager.applyDefaults() {
		if err := manager.Save(); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func (m *Manager) Save() error {
	if err := os.MkdirAll(filepath.Dir(m.configPath), dirPermission); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Write via a temp file + rename so a crash mid-write cannot leave a
	// truncated config that fails to parse on the next load.
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(m.config); err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}

	tmp := m.configPath + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), filePermission); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	if err := enforceFilePermissions(tmp); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("set config permissions: %w", err)
	}
	if err := os.Rename(tmp, m.configPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("commit config file: %w", err)
	}

	return nil
}

func (m *Manager) ConfigPath() string {
	return m.configPath
}

func (m *Manager) BaseDir() string {
	return m.baseDir
}

// CacheDurationHours returns the configured cache lifetime in hours.
func (m *Manager) CacheDurationHours() int {
	return m.config.CacheDurationHours
}

func (m *Manager) LogsDir() string {
	return filepath.Join(m.baseDir, "logs")
}

func defaultConfig() Config {
	return Config{
		RegisteredRepos:    make(map[string]string),
		CacheDurationHours: defaultCacheHours,
		FuzzyThreshold:     defaultFuzzyThreshold,
	}
}

func (m *Manager) applyDefaults() bool {
	changed := false
	if m.config.RegisteredRepos == nil {
		m.config.RegisteredRepos = make(map[string]string)
		changed = true
	}
	if m.config.CacheDurationHours <= 0 {
		m.config.CacheDurationHours = defaultCacheHours
		changed = true
	}
	if m.config.FuzzyThreshold == 0 {
		m.config.FuzzyThreshold = defaultFuzzyThreshold
		changed = true
	}

	return changed
}

// WorkspaceBaseDir returns the root DevPulse data directory. When the
// DEVPULSE_CONFIG env var points at a config file, the workspace follows it so
// notes, goals, cache, and logs live alongside the config; otherwise it is
// ~/.devpulse.
func WorkspaceBaseDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv(ConfigEnv)); override != "" {
		absolutePath, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve %s path: %w", ConfigEnv, err)
		}
		return filepath.Dir(filepath.Clean(absolutePath)), nil
	}

	return defaultBaseDir()
}

func defaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	return filepath.Join(home, ".devpulse"), nil
}

func resolveConfigPath(baseDir string) (string, error) {
	if override := strings.TrimSpace(os.Getenv(ConfigEnv)); override != "" {
		absolutePath, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve %s path: %w", ConfigEnv, err)
		}
		return filepath.Clean(absolutePath), nil
	}

	return filepath.Join(baseDir, "config.toml"), nil
}

func ensureWorkspace(baseDir string, configPath string) error {
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "cache"),
		filepath.Join(baseDir, "logs"),
		filepath.Join(baseDir, "notes"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPermission); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
		if err := enforceDirectoryPermissions(dir); err != nil {
			return fmt.Errorf("set directory permissions for %q: %w", dir, err)
		}
	}

	if configDir := filepath.Dir(configPath); configDir != baseDir {
		if err := os.MkdirAll(configDir, dirPermission); err != nil {
			return fmt.Errorf("create override config directory %q: %w", configDir, err)
		}
		if err := enforceDirectoryPermissions(configDir); err != nil {
			return fmt.Errorf("set override config directory permissions for %q: %w", configDir, err)
		}
	}

	return nil
}

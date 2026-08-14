package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	git "github.com/go-git/go-git/v5"
)

// setConfigEnv sets DEVPULSE_CONFIG for the duration of the test.
func setConfigEnv(t *testing.T, path string) {
	t.Helper()
	t.Setenv(ConfigEnv, path)
}

func setTempConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	path := filepath.Join(home, "devpulse", "config.toml")
	setConfigEnv(t, path)
	return path
}

func TestWorkspaceBaseDirDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	setConfigEnv(t, "")
	base, err := WorkspaceBaseDir()
	if err != nil {
		t.Fatalf("WorkspaceBaseDir: %v", err)
	}
	want := filepath.Join(home, ".devpulse")
	if base != want {
		t.Fatalf("WorkspaceBaseDir = %q, want %q", base, want)
	}
}

func TestWorkspaceBaseDirFollowsConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, "custom", "devpulse")
	setConfigEnv(t, filepath.Join(cfgDir, "config.toml"))

	base, err := WorkspaceBaseDir()
	if err != nil {
		t.Fatalf("WorkspaceBaseDir: %v", err)
	}
	if base != cfgDir {
		t.Fatalf("WorkspaceBaseDir = %q, want %q (should follow DEVPULSE_CONFIG)", base, cfgDir)
	}
}

func TestRegisterRepositoryRejectsNonGitDir(t *testing.T) {
	setTempConfig(t)
	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	dir := t.TempDir() // plain dir, not a git repo
	if _, err := m.RegisterRepository(dir); err == nil {
		t.Fatal("expected error registering a non-git directory")
	}

	// A real git repo must register fine.
	repoDir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := git.PlainInit(repoDir, false); err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	registered, err := m.RegisterRepository(repoDir)
	if err != nil {
		t.Fatalf("RegisterRepository(git dir): %v", err)
	}
	if registered.Name == "" {
		t.Fatal("expected a repo name")
	}
}

func TestRegisterRepositoryRejectsMissingPath(t *testing.T) {
	setTempConfig(t)
	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := m.RegisterRepository(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestSaveConfigRoundTrip(t *testing.T) {
	setTempConfig(t)
	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	m.config.RegisteredRepos = map[string]string{"demo": "/tmp/demo"}
	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	m2, err := Load()
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if m2.config.RegisteredRepos["demo"] != "/tmp/demo" {
		t.Fatalf("registered repo not round-tripped: %v", m2.config.RegisteredRepos)
	}
}

func TestSaveIsAtomic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	setConfigEnv(t, filepath.Join(home, "devpulse", "config.toml"))

	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// A pre-existing config file must survive a save with no leftover temp file.
	m.config.RegisteredRepos = map[string]string{"a": "/repo/a"}
	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(m.ConfigPath()); err != nil {
		t.Fatalf("config file missing after save: %v", err)
	}
	if _, err := os.Stat(m.ConfigPath() + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file left behind after save: %v", err)
	}

	// Config must have restrictive permissions. Windows has no POSIX
	// permission bits (Go reports 0666), so enforce this only on POSIX.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(m.ConfigPath())
		if err != nil {
			t.Fatalf("stat config: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("config permissions = %o, want 600", perm)
		}
	}

	// The saved file must still parse.
	m2, err := Load()
	if err != nil {
		t.Fatalf("Load after atomic save: %v", err)
	}
	if m2.config.RegisteredRepos["a"] != "/repo/a" {
		t.Fatalf("registered repo lost after save: %v", m2.config.RegisteredRepos)
	}
}

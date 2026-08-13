package config

import (
	"os"
	"path/filepath"
	"testing"

	git "github.com/go-git/go-git/v5"
)

// setConfigEnv sets DEVPULSE_CONFIG for the duration of the test.
func setConfigEnv(t *testing.T, path string) {
	t.Helper()
	t.Setenv(ConfigEnv, path)
}

func TestWorkspaceBaseDirDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
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
	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := m.RegisterRepository(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestSaveConfigRoundTrip(t *testing.T) {
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

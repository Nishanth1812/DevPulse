package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/config"
)

func setupFixtureWorkspace(t *testing.T, options fixtureRepoOptions) *fixtureClient {
	t.Helper()
	t.Setenv(config.ConfigEnv, filepath.Join(t.TempDir(), "config.toml"))
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	manager = loaded
	provider = ai.ProviderGroq
	noColor = true
	dryRun = false
	redactDiff = false
	path := newFixtureRepository(t, options)
	if _, err := manager.RegisterRepository(path); err != nil {
		t.Fatalf("RegisterRepository: %v", err)
	}
	client := &fixtureClient{Reply: func(prompt string) string {
		if strings.Contains(prompt, `"repos"`) {
			return portfolioReplyForPrompt(prompt)
		}
		return defaultFixtureReply(prompt)
	}}
	installFixtureClient(t, client)
	t.Cleanup(func() {
		manager = nil
		provider = ai.ProviderGroq
		noColor = false
		dryRun = false
		redactDiff = false
	})
	return client
}

func TestFixtureBriefIsNetworkFreeAndCacheable(t *testing.T) {
	client := setupFixtureWorkspace(t, fixtureRepoOptions{Name: "alpha", WithPlan: true, Commits: 2})
	stdout, stderr, err := executeForTest("brief", "--no-color")
	if err != nil {
		t.Fatalf("first brief: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if client.Calls != 1 || !strings.Contains(stdout, "Portfolio Brief") {
		t.Fatalf("first brief calls/output = %d/%q", client.Calls, stdout)
	}
	stdout, stderr, err = executeForTest("brief", "--no-color")
	if err != nil {
		t.Fatalf("cached brief: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if client.Calls != 1 {
		t.Fatalf("cache miss constructed provider again: %d calls", client.Calls)
	}
}

func TestFixtureRedactDiffNeverSendsSecret(t *testing.T) {
	client := setupFixtureWorkspace(t, fixtureRepoOptions{Name: "alpha", Commits: 2, SecretDiff: true})
	stdout, stderr, err := executeForTest("brief", "--dry-run", "--redact-diff", "--no-color")
	if err != nil {
		t.Fatalf("redacted dry-run: %v\n%s", err, stderr)
	}
	if client.Calls != 0 {
		t.Fatalf("dry-run constructed provider: %d calls", client.Calls)
	}
	if strings.Contains(stdout, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("dry-run exposed the fixture secret:\n%s", stdout)
	}
	if !strings.Contains(stdout, "redact") && !strings.Contains(stderr, "redacted") {
		t.Fatalf("redacted dry-run did not communicate its boundary:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestFixtureCorpusSupportsEmptyRepository(t *testing.T) {
	path := newFixtureRepository(t, fixtureRepoOptions{Name: "empty", Empty: true})
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		t.Fatalf("fixture repository missing .git: %v", err)
	}
}

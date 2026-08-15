package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nishanth1812/devpulse/internal/ai"
)

func TestUnsupportedProviderFailsBeforeCollection(t *testing.T) {
	stdout, stderr, err := executeForTest("brief", "--provider", "unsupported", "--no-color")
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("error = %v, want unsupported provider\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func TestMalformedAIResponseReturnsActionableError(t *testing.T) {
	client := setupFixtureWorkspace(t, fixtureRepoOptions{Name: "alpha", Commits: 1})
	client.Reply = func(string) string { return "not json" }
	_, _, err := executeForTest("brief", "alpha", "--no-color")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "parse brief response") {
		t.Fatalf("error = %v, want malformed response error", err)
	}
}

func TestInvalidSinceFailsBeforeProvider(t *testing.T) {
	client := setupFixtureWorkspace(t, fixtureRepoOptions{Name: "alpha", Commits: 1})
	_, _, err := executeForTest("resume", "alpha", "--since", "not-a-date", "--no-color")
	if err == nil || !strings.Contains(err.Error(), "invalid --since date") {
		t.Fatalf("error = %v, want invalid date", err)
	}
	if client.Calls != 0 {
		t.Fatalf("invalid --since constructed provider: %d calls", client.Calls)
	}
}

func TestDeletedRegisteredRepositoryHasSafeError(t *testing.T) {
	writeDoctorConfig(t, filepath.Join(t.TempDir(), "deleted-repo"))
	_, _, err := executeForTest("brief", "--no-color")
	if err == nil || !strings.Contains(err.Error(), "collect repository") {
		t.Fatalf("error = %v, want collection error", err)
	}
}

func TestEmptyRepositoryHasSafeError(t *testing.T) {
	setupFixtureWorkspace(t, fixtureRepoOptions{Name: "empty", Empty: true})
	_, _, err := executeForTest("brief", "--no-color")
	if err == nil {
		t.Fatal("expected empty repository error")
	}
	if strings.Contains(strings.ToLower(err.Error()), "panic") {
		t.Fatalf("empty repository produced panic-like error: %v", err)
	}
}

func TestDefaultFixtureClientUsesSupportedProvider(t *testing.T) {
	if ai.DefaultModel(ai.ProviderGroq, false) == "" {
		t.Fatal("fixture test requires a configured provider default")
	}
}

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/config"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

type briefFakeClient struct {
	response string
	calls    int
	prompt   string
}

func (f *briefFakeClient) Generate(_ context.Context, prompt string) (string, error) {
	f.calls++
	f.prompt = prompt
	return f.response, nil
}

func (f *briefFakeClient) Name() string { return "brief-test-client" }

func setupBriefCommandTest(t *testing.T, response string, names ...string) *briefFakeClient {
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

	fake := &briefFakeClient{response: response}
	clientFactoryOverride = func(string, bool) ai.ClientFactory {
		return func() (ai.Client, error) { return fake, nil }
	}

	for _, name := range names {
		repoPath := filepath.Join(t.TempDir(), name)
		briefTestRepo(t, repoPath, name)
		if _, err := manager.RegisterRepository(repoPath); err != nil {
			t.Fatalf("RegisterRepository(%q): %v", name, err)
		}
	}

	t.Cleanup(func() {
		manager = nil
		clientFactoryOverride = nil
		provider = ai.ProviderGroq
		noColor = false
		dryRun = false
		redactDiff = false
		briefCmd.SetOut(nil)
		briefCmd.SetErr(nil)
	})
	return fake
}

func briefTestRepo(t *testing.T, path, name string) {
	t.Helper()
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatalf("PlainInit(%q): %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte(name), 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree(%q): %v", name, err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("Add(%q): %v", name, err)
	}
	sig := &object.Signature{Name: "brief-test", Email: "brief@example.com", When: time.Unix(1, 0)}
	if _, err := worktree.Commit("initial "+name, &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("Commit(%q): %v", name, err)
	}
}

func portfolioResponse(names ...string) string {
	var b strings.Builder
	b.WriteString(`{"repos":[`)
	for i, name := range names {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"repo_name":%q,"summary":"summary %s","current_focus":"focus %s","blockers":[],"next_steps":["next %s"]}`, name, name, name, name)
	}
	b.WriteString(`]}`)
	return b.String()
}

func TestBriefCommandAcceptsZeroOrOneArgument(t *testing.T) {
	setupBriefCommandTest(t, `{"summary":"focused"}`, "alpha")

	if err := briefCmd.Args(briefCmd, nil); err != nil {
		t.Fatalf("zero arguments rejected: %v", err)
	}
	if err := briefCmd.Args(briefCmd, []string{"alpha"}); err != nil {
		t.Fatalf("one argument rejected: %v", err)
	}
	if err := briefCmd.Args(briefCmd, []string{"alpha", "extra"}); err == nil {
		t.Fatal("expected more than one argument to be rejected")
	}

	completions, directive := briefCmd.ValidArgsFunction(briefCmd, nil, "a")
	if directive != cobra.ShellCompDirectiveNoFileComp || len(completions) != 1 || completions[0] != "alpha" {
		t.Fatalf("completion = %v, directive = %v", completions, directive)
	}
	if completions, _ := briefCmd.ValidArgsFunction(briefCmd, []string{"alpha"}, ""); len(completions) != 0 {
		t.Fatalf("unexpected completion after first argument: %v", completions)
	}
}

func TestRunBriefPortfolioUsesOneCallAndStableOrder(t *testing.T) {
	fake := setupBriefCommandTest(t, portfolioResponse("beta", "alpha"), "beta", "alpha")
	var output, errors bytes.Buffer
	briefCmd.SetOut(&output)
	briefCmd.SetErr(&errors)

	if err := runBrief(briefCmd, nil); err != nil {
		t.Fatalf("runBrief portfolio: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", fake.calls)
	}
	if !strings.Contains(fake.prompt, "## Repository: alpha") || !strings.Contains(fake.prompt, "## Repository: beta") {
		t.Fatalf("portfolio prompt omitted repository evidence:\n%s", fake.prompt)
	}
	alpha := strings.Index(output.String(), "--- alpha ---")
	beta := strings.Index(output.String(), "--- beta ---")
	if alpha == -1 || beta == -1 || alpha > beta {
		t.Fatalf("portfolio output order is not stable:\n%s", output.String())
	}
}

func TestRunBriefPortfolioRejectsNoRepositories(t *testing.T) {
	fake := setupBriefCommandTest(t, portfolioResponse("alpha"))
	var output bytes.Buffer
	briefCmd.SetOut(&output)

	err := runBrief(briefCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "brief: no repositories registered") {
		t.Fatalf("error = %v, want no-repositories error", err)
	}
	if fake.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", fake.calls)
	}
}

func TestRunBriefFocusedPartialNameRemainsFocused(t *testing.T) {
	fake := setupBriefCommandTest(t, `{"summary":"focused result","key_changes":[],"current_focus":"focus","blockers":[],"next_steps":[]}`, "alpha")
	var output bytes.Buffer
	briefCmd.SetOut(&output)

	if err := runBrief(briefCmd, []string{"alp"}); err != nil {
		t.Fatalf("runBrief focused: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", fake.calls)
	}
	if !strings.Contains(output.String(), "=== Brief: alpha") || strings.Contains(output.String(), "Portfolio Brief") {
		t.Fatalf("focused output = %q", output.String())
	}
}

func TestRunBriefAmbiguousNamePrintsCandidatesWithoutCallingProvider(t *testing.T) {
	fake := setupBriefCommandTest(t, portfolioResponse("alpha", "alpine"), "alpha", "alpine")
	var output bytes.Buffer
	briefCmd.SetOut(&output)

	if err := runBrief(briefCmd, []string{"alp"}); err != nil {
		t.Fatalf("runBrief ambiguous: %v", err)
	}
	if !strings.Contains(output.String(), "Multiple repositories match") || !strings.Contains(output.String(), "alpha") || !strings.Contains(output.String(), "alpine") {
		t.Fatalf("ambiguous output = %q", output.String())
	}
	if fake.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", fake.calls)
	}
}

func TestRunBriefInvalidNameReturnsError(t *testing.T) {
	fake := setupBriefCommandTest(t, `{"summary":"focused"}`, "alpha")
	var output bytes.Buffer
	briefCmd.SetOut(&output)

	err := runBrief(briefCmd, []string{"missing"})
	if err == nil || !strings.Contains(err.Error(), "brief:") {
		t.Fatalf("error = %v, want brief error", err)
	}
	if fake.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", fake.calls)
	}
}

func TestRunBriefPortfolioCollectionFailureNamesRepository(t *testing.T) {
	fake := setupBriefCommandTest(t, portfolioResponse("alpha"), "alpha")
	path, ok := manager.RepositoryPath("alpha")
	if !ok {
		t.Fatal("alpha was not registered")
	}
	if err := os.RemoveAll(filepath.Join(path, ".git")); err != nil {
		t.Fatalf("RemoveAll .git: %v", err)
	}

	var output bytes.Buffer
	briefCmd.SetOut(&output)
	err := runBrief(briefCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "collect repository") {
		t.Fatalf("error = %v, want named collection failure", err)
	}
	if fake.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", fake.calls)
	}
}

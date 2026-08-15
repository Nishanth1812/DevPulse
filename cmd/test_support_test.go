package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Nishanth1812/devpulse/internal/ai"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/logger"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

type fixtureRepoOptions struct {
	Name        string
	WithPlan    bool
	Plan        string
	Commits     int
	SecretDiff  bool
	Empty       bool
	CommitTimes []time.Time
}

func newFixtureRepository(t *testing.T, options fixtureRepoOptions) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), options.Name)
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	if options.Empty {
		return path
	}
	if options.Commits <= 0 {
		options.Commits = 1
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	if options.WithPlan {
		plan := options.Plan
		if plan == "" {
			plan = "# Plan\n\n## Now\nShip the fixture\n"
		}
		if err := os.WriteFile(filepath.Join(path, "PLAN.md"), []byte(plan), 0o600); err != nil {
			t.Fatalf("write plan: %v", err)
		}
		if _, err := worktree.Add("PLAN.md"); err != nil {
			t.Fatalf("add plan: %v", err)
		}
	}
	for i := 0; i < options.Commits; i++ {
		message := fmt.Sprintf("fixture commit %d", i+1)
		content := fmt.Sprintf("fixture content %d\n", i+1)
		if options.SecretDiff && i == options.Commits-1 {
			content = "TOKEN=sk-abcdefghijklmnopqrstuvwxyz123456\n"
		}
		if err := os.WriteFile(filepath.Join(path, "README.md"), []byte(content), 0o600); err != nil {
			t.Fatalf("write README: %v", err)
		}
		if _, err := worktree.Add("README.md"); err != nil {
			t.Fatalf("add README: %v", err)
		}
		when := time.Unix(int64(i+1), 0).UTC()
		if len(options.CommitTimes) > i {
			when = options.CommitTimes[i]
		}
		sig := &object.Signature{Name: "fixture", Email: "fixture@example.com", When: when}
		if _, err := worktree.Commit(message, &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
			t.Fatalf("commit fixture: %v", err)
		}
	}
	return path
}

type fixtureClient struct {
	Calls   int
	Prompts []string
	Reply   func(prompt string) string
}

func (f *fixtureClient) Generate(_ context.Context, prompt string) (string, error) {
	f.Calls++
	f.Prompts = append(f.Prompts, prompt)
	return f.Reply(prompt), nil
}

func (f *fixtureClient) Name() string { return "fixture-client" }

func defaultFixtureReply(prompt string) string {
	switch {
	case strings.Contains(prompt, `"ranked"`):
		return `{"ranked":[{"repo_name":"alpha","rank_reason":"fixture reason","proximity_score":4,"urgency":true}]}`
	case strings.Contains(prompt, `"repos"`):
		return `{"repos":[{"repo_name":"alpha","summary":"fixture summary","current_focus":"fixture focus","blockers":[],"next_steps":["fixture next"]}]}`
	case strings.Contains(prompt, `"what_was_built"`):
		return `{"what_was_built":"fixture built","what_is_incomplete":"","blockers_detected":[],"next_step":"fixture next"}`
	case strings.Contains(prompt, `"file_purpose"`):
		return `{"file_purpose":"fixture purpose","major_decisions":[],"current_state":"fixture state"}`
	case strings.Contains(prompt, `"subject"`):
		return `{"subject":"fix: fixture output","body":""}`
	default:
		return `{"summary":"fixture summary","key_changes":[],"current_focus":"fixture focus","blockers":[],"next_steps":["fixture next"]}`
	}
}

func executeForTest(args ...string) (stdout, stderr string, err error) {
	old := struct {
		manager    *config.Manager
		provider   string
		verbose    bool
		noColor    bool
		dryRun     bool
		redactDiff bool
		since      string
	}{manager, provider, verbose, noColor, dryRun, redactDiff, since}
	defer func() {
		_ = logger.Close()
		manager = old.manager
		provider = old.provider
		verbose = old.verbose
		noColor = old.noColor
		dryRun = old.dryRun
		redactDiff = old.redactDiff
		since = old.since
		rootCmd.SetArgs(nil)
		setCommandOutput(rootCmd, nil, nil)
	}()

	var out, errs bytes.Buffer
	setCommandOutput(rootCmd, &out, &errs)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	return out.String(), errs.String(), err
}

func setCommandOutput(command *cobra.Command, out, errs io.Writer) {
	command.SetOut(out)
	command.SetErr(errs)
	for _, child := range command.Commands() {
		setCommandOutput(child, out, errs)
	}
}

var repositoryHeadingRE = regexp.MustCompile(`(?m)^## Repository: ([^\n]+)$`)

func portfolioReplyForPrompt(prompt string) string {
	names := repositoryHeadingRE.FindAllStringSubmatch(prompt, -1)
	var b strings.Builder
	b.WriteString(`{"repos":[`)
	for i, match := range names {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"repo_name":%q,"summary":"fixture summary","current_focus":"fixture focus","blockers":[],"next_steps":["fixture next"]}`, strings.TrimSpace(match[1]))
	}
	b.WriteString(`]}`)
	return b.String()
}

func installFixtureClient(t *testing.T, client *fixtureClient) {
	t.Helper()
	clientFactoryOverride = func(string, bool) ai.ClientFactory {
		return func() (ai.Client, error) { return client, nil }
	}
	t.Cleanup(func() { clientFactoryOverride = nil })
}

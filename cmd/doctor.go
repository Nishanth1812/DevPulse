package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Nishanth1812/devpulse/internal/collector"
	"github.com/Nishanth1812/devpulse/internal/config"
	"github.com/Nishanth1812/devpulse/internal/security"
	"github.com/Nishanth1812/devpulse/internal/utils"
	git "github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

type checkResult struct {
	status  string // "PASS", "WARN", "FAIL"
	message string
}

type doctorFailure struct {
	FailCount int
}

func (e doctorFailure) Error() string {
	return fmt.Sprintf("doctor: %d check(s) failed", e.FailCount)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run DevPulse diagnostics",
	Long: `Checks API key presence, registered repo paths, git validity,
cache directory, goals file, plan files, and the sensitive content detector.
Useful for debugging after a new install or a machine migration.`,
	Args: cobra.NoArgs,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	var results []checkResult

	// 1. Config (already loaded by PersistentPreRunE)
	results = append(results, checkResult{"PASS", "Config file loaded"})

	// 2. API keys — check both providers
	for _, p := range []string{"groq", "gemini"} {
		has, err := config.HasAPIKey(p)
		if err != nil {
			results = append(results, checkResult{"WARN", fmt.Sprintf("API key check for %s: %s", p, err)})
		} else if has {
			results = append(results, checkResult{"PASS", fmt.Sprintf("API key found (%s)", p)})
		} else {
			results = append(results, checkResult{"WARN", fmt.Sprintf("No API key for %s (optional)", p)})
		}
	}

	// 3. Goals file
	goalsPath := utils.GoalsPath()
	if utils.FileExists(goalsPath) {
		results = append(results, checkResult{"PASS", fmt.Sprintf("Goals file: %s", goalsPath)})
	} else {
		results = append(results, checkResult{"WARN", fmt.Sprintf("Goals file not found: %s (run: devpulse init)", goalsPath)})
	}

	// 4. Registered repos
	repos := manager.ListRepositories()
	if len(repos) == 0 {
		results = append(results, checkResult{"WARN", "No repositories registered (run: devpulse register <path>)"})
	}

	for _, repo := range repos {
		// 4a. Path exists
		info, err := os.Stat(repo.Path)
		if err != nil {
			results = append(results, checkResult{"FAIL", fmt.Sprintf("Repo %q: path %s does not exist", repo.Name, repo.Path)})
			continue
		}
		if !info.IsDir() {
			results = append(results, checkResult{"FAIL", fmt.Sprintf("Repo %q: %s is not a directory", repo.Name, repo.Path)})
			continue
		}

		// 4b. Valid git repo
		r, err := git.PlainOpen(repo.Path)
		if err != nil {
			results = append(results, checkResult{"FAIL", fmt.Sprintf("Repo %q: not a valid git repository", repo.Name)})
			continue
		}

		// 4c. Check HEAD is accessible
		_, err = r.Head()
		if err != nil {
			results = append(results, checkResult{"WARN", fmt.Sprintf("Repo %q: HEAD not found (empty repo?)", repo.Name)})
		}

		// 4d. Count plan files
		planCount := countPlanFiles(repo.Path)
		if planCount > 0 {
			results = append(results, checkResult{"PASS", fmt.Sprintf("Repo %q: %d plan file(s) found", repo.Name, planCount)})
		} else {
			results = append(results, checkResult{"WARN", fmt.Sprintf("Repo %q: no plan files found (PLAN.md, ROADMAP.md, etc.)", repo.Name)})
		}
	}

	// 5. Cache directory writable
	cacheDir := filepath.Join(manager.BaseDir(), "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		results = append(results, checkResult{"FAIL", fmt.Sprintf("Cache directory not writable: %s", cacheDir)})
	} else {
		testFile := filepath.Join(cacheDir, ".doctor-test")
		if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
			results = append(results, checkResult{"FAIL", fmt.Sprintf("Cache directory not writable: %s", cacheDir)})
		} else {
			os.Remove(testFile)
			results = append(results, checkResult{"PASS", fmt.Sprintf("Cache directory writable: %s", cacheDir)})
		}
	}

	// 6. Sensitive content detector
	testPrompt := "Here is a secret: sk-test1234567890abcdef"
	scan := security.ScanPrompt(testPrompt)
	if scan.ContainsSecrets && len(scan.Matches) > 0 {
		results = append(results, checkResult{"PASS", "Sensitive content detector working"})
	} else {
		results = append(results, checkResult{"FAIL", "Sensitive content detector not working"})
	}

	return renderDoctor(w, results)
}

func renderDoctor(w io.Writer, results []checkResult) error {
	if _, err := fmt.Fprintf(w, "\n=== DevPulse Doctor ===\n\n"); err != nil {
		return err
	}

	passCount, warnCount, failCount := 0, 0, 0
	for _, r := range results {
		icon := "  "
		switch r.status {
		case "PASS":
			icon = colored("32", "[PASS]")
			passCount++
		case "WARN":
			icon = colored("33", "[WARN]")
			warnCount++
		case "FAIL":
			icon = colored("31", "[FAIL]")
			failCount++
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", icon, r.message); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	if failCount > 0 {
		if _, err := fmt.Fprintf(w, "%d issue(s) found: %d pass, %d warn, %d fail\n", failCount+warnCount, passCount, warnCount, failCount); err != nil {
			return err
		}
		return doctorFailure{FailCount: failCount}
	}
	if warnCount > 0 {
		_, err := fmt.Fprintf(w, "%d warning(s): %d pass, %d warn\n", warnCount, passCount, warnCount)
		return err
	}
	_, err := fmt.Fprintf(w, "All checks passed (%d)\n", passCount)
	return err
}

// colored wraps text in an ANSI color code unless color output is disabled.
func colored(code, text string) string {
	if noColor {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

// countPlanFiles returns the number of known plan files found in a repo.
// The file list comes from the collector so doctor and collection always agree.
func countPlanFiles(repoPath string) int {
	count := 0
	for _, name := range collector.PlanFiles() {
		if utils.FileExists(filepath.Join(repoPath, name)) {
			count++
		}
	}
	return count
}

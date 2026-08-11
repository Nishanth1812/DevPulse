> **Status:** Design document / historical plan. This preserves the original
> project-planning document from the initial README. For current user-facing
> documentation, see the [README](../README.md).

# DevPulse CLI — Project Plan v2

---

## What Is This

DevPulse is a command-line tool that gives you back the context you lose between coding sessions.

When you work across multiple repositories, the hardest part is not the code — it is remembering where you left off, which project needs your attention most, and what is quietly going wrong in the ones you have not touched in a while. No existing tool solves this because no existing tool reads both your Git history and understands the content of what changed.

DevPulse reads your commit diffs, your project plan files, and a personal goals file through a language model. Every suggestion it makes is grounded in your actual code history and your own stated intentions — not generic productivity advice.

You run it from anywhere on your machine. You type partial repo names, not full paths. It costs almost nothing to run because everything sent to the LLM is compressed to the minimum required.

**One-line pitch: a morning standup with yourself, powered by your own repositories and your own Gemini key.**

---

## Full Plan

### The Core Problem

You have multiple repos open at any given moment. You switch between them, you take breaks, you start features and get pulled away. Every time you come back to a repo after more than a day, you spend the first fifteen minutes re-establishing context — reading old commits, scanning diffs, trying to remember what "WIP" meant. On top of that, repos often have planning documents — a PLAN.md, a ROADMAP.md, a TODO.md — that capture intent but never get connected to what you actually did.

DevPulse makes context retrieval instant and connects your stated plans to your actual commit history.

---

### The Five Commands

#### `devpulse brief`

Your default command. Run it when you sit down to code. Scans all registered repos, reads recent commit history and any plan files found, and gives you a cross-repo summary: what is in-flight, what is blocked, what has gone stale, and what has an open PR sitting unreviewed. Names exact files and endpoints, not just repos.

#### `devpulse resume <partial-name>`

Deep context recovery for a single repository. Reads the last two to three weeks of diffs and produces a narrative: what you built, what you started but did not finish, and what the natural next step is. Accepts partial names — typing `acm` matches `ACM-APP-BACKEND`, typing `nut` matches `nutrition-tracker`. If multiple repos match, it shows a numbered list.

#### `devpulse focus`

Cross-repo triage. Ranks all active repos by completion proximity, weighting that ranking against your goals file and any deadline signals found in plan documents. Tells you which project is one or two sessions away from a working state and which one is deep in the hole.

#### `devpulse health`

Per-repo hygiene. Detects stale branches, TODO/FIXME accumulation, open PRs with no activity, and repos that were active two weeks ago and have gone silent. All rule-based — no LLM involved, instant output.

#### `devpulse why <partial-name> <file>`

File-level commit archaeology. Walks the full commit history for a given file and produces a narrative of every significant decision made in it. Accepts a partial repo name the same way `resume` does.

---

### The Goals File — Improved

The goals file lives at `~/.devpulse/goals.md` — one file for your entire portfolio of projects, not one per repo. It has four loose sections that DevPulse recognises by heading name:

```
## Now
Things you are actively trying to finish this sprint or this week.

## Next
Things that become active once the Now items are done.

## Deadlines
Hard dates. Format: YYYY-MM-DD — description. DevPulse reads these and flags urgency.

## Someday
Projects or ideas you are not touching now but do not want to forget.
```

You do not have to use all sections. You do not have to follow any format inside them. Plain sentences work fine. The heading names are the only convention DevPulse looks for.

When DevPulse makes a suggestion, it checks your Now section first, then cross-references your Deadlines section. A repo that appears in Now ranks higher in `focus` output regardless of commit frequency. A repo with a deadline within 14 days gets a visual urgency marker in `brief` output.

Running `devpulse init` on first setup creates this file with the section headers and example entries pre-filled.

---

### Project Plan File Scanning

Many repos contain planning documents: `PLAN.md`, `ROADMAP.md`, `TODO.md`, `CHANGELOG.md`, a `docs/` folder with spec files, or even a `.github/ISSUE_TEMPLATE`. DevPulse scans for these on registration and re-reads them on every invocation.

The scan looks for files matching a priority list: `PLAN.md`, `ROADMAP.md`, `TODO.md`, `CHANGELOG.md`, `NOTES.md`, `docs/PLAN.md`, `docs/ROADMAP.md`. If multiple are found, they are all collected and summarised together.

Before sending to the LLM, these files are compressed: headings are kept, completed checkbox items are stripped, duplicate lines are deduplicated, and the result is capped at 300 tokens. This compressed summary is injected into the prompt alongside commit data, giving the model context about what the project is supposed to become — not just what it currently is.

This is what makes the suggestions meaningful. The model can say "your PLAN.md lists a billing service as the next milestone and your commits show you are three endpoints away from finishing the auth service — finish auth first" instead of just counting recent commits.

---

### Token Efficiency Strategy

Everything that goes to Gemini is treated as expensive even though the free tier is generous. This discipline keeps costs zero for light users and predictable for heavy users.

**Diff compression.** Before a diff is included in any prompt, it goes through a compressor: blank lines removed, lines that are only whitespace changes skipped, import-only changes summarised as a single note ("5 import changes"), binary file changes replaced with a placeholder. A single file diff is capped at 80 lines.

**Commit selection.** Only the 10 most recent commits are included in full. Commits older than that are reduced to a one-line summary: timestamp, message, and a count of files changed. Commits older than 30 days are dropped entirely.

**Plan file compression.** As described above — headings retained, completed items stripped, capped at 300 tokens before injection.

**Response format.** Every prompt instructs the model to respond in a compact JSON schema with specific fields: `summary` (one paragraph), `in_progress` (one sentence), `next_step` (one sentence), `warnings` (array of short strings). No markdown, no prose padding, no explanatory text. The output layer formats this into the terminal display — the model just fills the fields.

**Batching.** The `brief` command sends all registered repos in a single API call, not one call per repo. The prompt is structured as a list of repo blocks, and the model returns a corresponding list of JSON objects. This cuts API calls from N-repos to one.

**Caching.** Per-repo cache is keyed on the HEAD commit SHA plus a hash of the plan file content. If neither has changed since the last run, the cached response is used and no API call is made. Cache entries expire after 24 hours regardless.

**Dry-run flag.** `devpulse brief --dry-run` prints the exact prompt that would be sent to Gemini, including token count estimate, without making any API call. Use this to verify what is being sent before trusting the tool with sensitive codebases.

---

### Stack

#### Language: Go

Detailed justification is at the bottom of this document if you want to compare with Python. The short version: single binary, instant startup, trivial cross-compilation, best-in-class CLI tooling. For a tool that is expected to work globally from any folder, compile to a `.exe`, and have shell autocomplete, Go has no serious competition.

#### CLI Framework: Cobra

Cobra is the standard Go CLI library. It handles subcommands, flags, help generation, and — critically — shell completion for bash, zsh, fish, and PowerShell with almost no extra work. The fuzzy name matching for repo names is layered on top of Cobra's argument handling.

#### Fuzzy Matching: sahilm/fuzzy

When you type `devpulse resume acm`, Cobra receives `acm` as a positional argument. Before doing anything with it, DevPulse passes it through a fuzzy matcher against the list of registered repo names. The `sahilm/fuzzy` library ranks candidates by match quality — substring, prefix, and character-spread matches all work. If exactly one repo matches above the confidence threshold, it proceeds. If multiple match, it prints a numbered list and asks you to pick.

Shell tab completion is registered separately via Cobra's `ValidArgsFunction` — when you press Tab after `devpulse resume`, it lists registered repo names filtered by what you have already typed.

#### Git Integration: go-git

Pure Go Git library. Reads commit logs, walks diffs, detects renames, inspects branches. No dependency on the system `git` binary — works on any machine where the binary is installed.

#### LLM: Gemini API (user-supplied key)

Uses `google.golang.org/genai`, the official Go SDK for Gemini. The API key is provided by the user and stored securely (see Security section). Default model is `gemini-2.0-flash` — fast, cheap, and more than capable for this use case. The model choice is configurable in the config file so users can switch to `gemini-1.5-pro` for `resume` if they want deeper reasoning.

#### Security: OS Keychain via go-keyring

The Gemini API key is never stored in a plaintext config file. It is stored in the OS keychain — Keychain on macOS, Secret Service on Linux, Windows Credential Manager on Windows — via the `zalando/go-keyring` library. The config file stores only non-sensitive settings (registered repo paths, model preference, cache duration). If the keychain is unavailable, the tool falls back to reading from a `DEVPULSE_API_KEY` environment variable with a warning printed to stderr.

#### Config Format: TOML

Simple, human-readable, and well-supported in Go via `BurntSushi/toml`. The config file lives at `~/.devpulse/config.toml`. It contains registered repo paths, model preferences, cache settings, and GitHub token (if used). The API key does not appear here.

#### Distribution: goreleaser + GitHub Actions

Goreleaser builds binaries for Windows (amd64, arm64), macOS (amd64, arm64), and Linux (amd64, arm64) on every tagged release. GitHub Actions triggers on a version tag push. Artifacts are uploaded to GitHub Releases automatically. Each release includes SHA-256 checksums for every binary so users can verify downloads. A Homebrew formula and a one-line install script are generated as part of the release process.

The Windows `.exe` is distributed both as a standalone file and as a zip archive. The install script on Windows adds `~\AppData\Local\devpulse\` to the user PATH automatically so the tool is available globally from any terminal session.

---

### Architecture

Three layers, clean separation.

The **collection layer** reads raw data: Git history via go-git, plan files from disk, the goals file from the fixed location, and optionally PR data from the GitHub API. Each source is a separate function returning a typed struct. No decisions are made here — only gathering.

The **analysis layer** takes collected data and builds the compressed prompt. It runs the diff compressor, selects which commits to include in full versus summarise, reads and compresses plan files, injects the goals file, and formats the final prompt string. It also makes the Gemini API call and deserialises the JSON response. Cache reads and writes happen here.

The **output layer** takes the parsed response structs and renders them to the terminal. It handles colour, alignment, urgency markers, and the fuzzy-match disambiguation flow. It never talks to any API or reads any files.

---

### Security Considerations

**API key storage.** Always in OS keychain, never in a file on disk. The `devpulse auth` command handles the initial storage. Key rotation is done by running `devpulse auth` again.

**What leaves your machine.** Only commit messages, compressed diff snippets, plan file summaries, and the goals file are sent to Gemini. File contents are never sent in full — only changed lines with minimal context. The `--dry-run` flag on any command shows exactly what would be sent before you commit to sending it.

**Sensitive content detection.** Before any prompt is sent, the content is scanned for patterns that look like secrets: strings matching common API key formats, strings matching JWT patterns, strings matching private key headers. If any are found, those lines are redacted in the prompt and a warning is printed. This is not foolproof but it catches the most common case of accidentally committing a key and then having DevPulse send it to an LLM.

**No telemetry.** DevPulse never phones home, checks for updates silently, or sends usage data anywhere. The only outbound network calls are to the Gemini API (when an LLM command is run) and optionally to the GitHub API (if configured).

**Config file permissions.** On creation, the config file is written with `0600` permissions (owner read/write only). The cache directory is created with `0700`.

---

## Phases

### Phase 1 — Foundation and Global Install

The goal of this phase is a working binary that installs globally and can register repos.

Set up the Go module and Cobra CLI scaffold with placeholder commands for all five main commands plus `register`, `unregister`, `list`, `auth`, `init`, and `doctor`. Implement the config file reader and writer using TOML. Implement `devpulse register <path>` and `devpulse unregister <partial-name>`. Implement `devpulse list` which prints all registered repos with their paths. Implement `devpulse init` which creates `~/.devpulse/goals.md` with the four-section template. Set up goreleaser config targeting all six platforms. Set up the GitHub Actions workflow that triggers on version tags and publishes to GitHub Releases. Write the Windows install script that adds the binary to user PATH.

Completion signal: you can install the binary globally, run `devpulse` from any folder, register a repo, and see it in `devpulse list`.

---

### Phase 2 — Git Reading and Plan File Scanning

The goal of this phase is all raw data collection working correctly.

Integrate `go-git`. Implement commit history walker: reads the last 50 commits, extracts timestamp, message, author, and list of changed files. Implement diff reader: for the 10 most recent commits, reads the actual line-level diff per file. Implement the diff compressor: strips blank lines, skips whitespace-only changes, summarises import-only changes, caps single-file diffs at 80 lines, caps per-commit file count at 10. Implement plan file scanner: given a repo path, searches for files in the priority list and reads them. Implement plan file compressor: strips completed checkboxes, deduplicates lines, retains headings, caps output at 300 tokens by line count. Implement the goals file parser: reads `~/.devpulse/goals.md` and extracts the four sections as separate strings. Parse deadline lines in the Deadlines section into structured date + description pairs.

Completion signal: given any registered repo, you can call the collection layer and get back a well-structured Go struct containing compressed commit data, plan file summary, and goals file content, all ready to be formatted into a prompt.

---

### Phase 3 — Gemini Integration and Token-Efficient Prompts

The goal of this phase is the first real LLM-powered output with full cost control.

Integrate the `google.golang.org/genai` SDK. Implement `devpulse auth` which prompts for the Gemini API key, stores it via `go-keyring`, and confirms storage. Implement the prompt builder for `brief`: takes a list of repo data structs and formats them into a single batched prompt with a tight JSON response schema. Implement the Gemini API caller with a 15-second timeout and graceful error handling. Implement the JSON response parser. Implement the cache layer: keyed on HEAD SHA plus plan file hash, stored in `~/.devpulse/cache/`, 24-hour expiry. Implement the `--dry-run` flag which prints the prompt and estimated token count without making the API call. Implement the sensitive content detector that scans prompt content before sending.

Wire up the full `brief` command using the real LLM path when a key is present, falling back to rule-based output when it is not.

Completion signal: `devpulse brief` makes a single Gemini API call for all registered repos, returns a formatted briefing, and uses the cache on subsequent runs when HEAD has not changed.

---

### Phase 4 — Fuzzy Matching and Shell Completion

The goal of this phase is the repo name ergonomics that make the tool pleasant to use.

Integrate `sahilm/fuzzy`. Implement the fuzzy resolver function: takes a partial string and the list of registered repo names, returns the best match if confidence is high enough, or a numbered candidate list if multiple match. Wire this into `resume`, `focus` (for single-repo deep-dive), `why`, and `unregister`. Implement the disambiguation prompt: when multiple repos match, print a numbered list and read a single keypress to select. Register Cobra `ValidArgsFunction` for all commands that accept repo names — this powers Tab completion. Generate and document the shell completion setup commands for bash, zsh, fish, and PowerShell.

Completion signal: typing `devpulse resume acm` correctly resolves to `ACM-APP-BACKEND`, and pressing Tab after `devpulse resume ` shows the list of registered repos filtered by what has been typed.

---

### Phase 5 — `resume` and `focus`

The goal of this phase is the two highest-value commands.

Implement `devpulse resume <partial-name>`. Uses the fuzzy resolver. Reads a longer commit window (3 weeks or 50 commits). Sends a single-repo prompt with a narrative-reconstruction schema: `what_was_built`, `what_is_incomplete`, `blockers_detected`, `next_step`. Includes plan file summary and relevant goals file section. Add `--since <YYYY-MM-DD>` flag for custom lookback window. Use `gemini-2.0-flash` by default but allow override to `gemini-1.5-pro` in config.

Implement `devpulse focus`. Makes one prompt for all registered repos combined. The schema asks for a ranked array of objects, each with `repo_name`, `rank_reason`, `proximity_score` (1-5), and `urgency` (boolean, set true if a deadline within 14 days is found in the goals file). Output is a ranked list with one-line justifications and urgency markers where applicable.

Completion signal: `devpulse resume` produces a paragraph-level narrative, and `devpulse focus` produces a ranked list with deadline urgency correctly reflected from the goals file.

---

### Phase 6 — `health`, `why`, and `doctor`

The goal of this phase is the remaining commands.

Implement `devpulse health`. Fully rule-based, no LLM. Detects: merged branches not deleted, TODO/FIXME line counts per repo with trend (more than last week = warning), repos with zero commits in 14 days that had regular activity before, plan file items that have been in the Now section for more than 30 days without appearing in any commit message. Output is a structured list of issues per repo with suggested actions.

Implement `devpulse why <partial-name> <file>`. Uses the fuzzy resolver for repo name. Walks full commit history for the specified file using go-git rename detection. Compresses the per-commit diffs the same way as other commands. Sends to Gemini with a schema requesting `file_purpose`, `major_decisions` (array of objects with `date` and `description`), `current_state`.

Implement `devpulse doctor`. No LLM involved. Checks and prints pass/fail for: API key present in keychain, all registered repo paths exist on disk and are valid Git repos, cache directory writable, goals file present, plan files found in each repo, sensitive content detector functioning. Useful for debugging after a new install or a machine migration.

Completion signal: all five main commands work end to end with real output.

---

### Phase 7 — Security Hardening and Release Polish

The goal of this phase is a tool you would trust with a sensitive codebase and would recommend to someone else.

Harden the sensitive content detector: add patterns for AWS keys, GCP service account JSON fragments, private key PEM headers, and high-entropy strings above a threshold length. Implement the `--redact-diff` flag that strips all diff content from prompts and sends only commit messages and plan files — a lower-trust mode for codebases with strict data policies. Audit all file permission settings on config and cache files. Write a `SECURITY.md` documenting exactly what data leaves the machine, what is cached locally, and how to revoke or rotate the API key.

Set up binary signing for macOS releases using GitHub Actions with an Apple Developer certificate (or document the Gatekeeper bypass for unsigned binaries). Write a proper README with install instructions for all platforms, a getting-started walkthrough, screenshots of each command's output, and a FAQ section. Write a `CONTRIBUTING.md`. Tag `v1.0.0` and publish the full release.

Completion signal: a new user on any supported platform can install, configure, and get a useful `devpulse brief` within five minutes, and can verify the binary checksum matches the published SHA-256 before running it.
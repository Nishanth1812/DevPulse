# DevPulse V1 Development Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current DevPulse MVP into a technically release-ready `v1.0.0` local-first CLI whose documented commands, cross-platform behavior, cache semantics, privacy controls, and release artifacts agree.

**Architecture:** Keep the existing Go CLI and package boundaries. `cmd/` remains the Cobra adapter; `internal/collector`, `compressor`, `config`, `cache`, `security`, `ai`, `output`, and `logger` remain independently testable. Add only the seams needed for portfolio brief generation, deterministic validation, cache fingerprinting, and command-level testing; do not introduce a hosted service, agent runtime, or new persistence system for V1.

**Tech Stack:** Go 1.25.12, Cobra, go-git, BurntSushi TOML, go-keyring, Groq and Gemini provider clients, GoReleaser, GitHub Actions.

## Global Constraints

- V1 is a read-only context-recovery CLI; it must not edit repository files, create commits, run destructive Git commands, or execute arbitrary model-generated shell commands.
- The public command contract is `brief` with zero or one repository argument, `resume`, `focus`, `health`, `why`, `commit`, and the existing setup, notes, registration, configuration, diagnostic, debug, version, and completion commands.
- `brief` with no repository argument is the cross-repository briefing; `brief <partial-name>` remains the focused single-repository briefing.
- Repository-derived text is untrusted data. Prompt instructions must not be allowed to override system behavior, and all provider-bound prompts must pass through secret scanning.
- API keys remain in the OS keychain or provider-specific environment variables; they must never be written to config, cache, history, logs, test fixtures, or release artifacts.
- All prompt inputs must be represented in the cache fingerprint: collected repository evidence, plans, notes, goals, provider, model, redaction mode, and command-specific options.
- V1 supports Groq and Gemini through the existing `internal/ai.Client` interface; no provider-specific product identity or new provider is required.
- V1 does not include standup, changelog, velocity, natural-language search, goals subcommands, MCP integrations, background watch mode, IDE/UI clients, hosted storage, team accounts, billing, telemetry, or autonomous code changes.
- Every implementation task ends with focused tests, `gofmt`, and a small conventional commit. Do not combine unrelated cleanup with a task.

---

## Current baseline and dependency order

The current checkout is branch `mvp` at `b34959b`, ahead of `origin/mvp` by ten commits. The intended fixes for cross-repository `brief`, cache invalidation, Windows workspace handling, and Windows CI exist in the local history but were reverted. Implement the behavior from this plan deliberately rather than relying on the reverted commits.

Known audit blockers to close:

- Windows workspace initialization can fail because `os.Chmod` is treated as mandatory.
- CI tests only Linux even though Windows archives are published.
- README/help promise cross-repository `brief`, while the command currently requires one argument.
- Cache keys omit notes, model/provider details, and other prompt inputs.
- Focus responses are not checked for repository membership, duplicates, or score bounds.
- `doctor` prints failures but returns success to the shell.
- README model/release claims are stale.
- Command-layer coverage is too low to protect public behavior.

Implementation order:

```text
cross-platform config + path normalization
        ↓
portfolio brief contract + shared test seam
        ↓
complete cache fingerprints + deterministic response validation
        ↓
doctor exit semantics + privacy/error-path hardening
        ↓
fixture repositories + golden CLI tests
        ↓
Windows CI + release artifact verification
        ↓
technical V1 readiness gate
```

## Release train

Each release is independently buildable, testable, and reviewable. A release is complete only when its tasks, focused verification, and exit gate pass. Later releases may depend on earlier release behavior, but no release should contain a mixture of unrelated V2 work.

| Release | Theme | Outcome | Tasks |
|---|---|---|---|
| `v0.3.0` | Platform Stability | Config/workspace behavior is portable and Git paths are normalized | `R1-T1`, `R1-T2` |
| `v0.4.0` | Portfolio Brief | `brief` works across all registered repositories and for one focused repository | `R2-T1`, `R2-T2` |
| `v0.5.0` | Trust and Correctness | Cache, model-output validation, urgency, diagnostics, and privacy boundaries are reliable | `R3-T1` through `R3-T4` |
| `v0.6.0` | Testable CLI | Error behavior, docs, fixtures, and public command output are protected by tests | `R4-T1` through `R4-T3` |
| `v0.7.0` | Release Candidate | Linux and Windows CI plus technical release verification are in place | `R5-T1` |
| `v1.0.0` | Technical V1 Release | The final technical gate passes and the maintainer can tag from `main` | `R6-T1` |

Sub-release tags are engineering checkpoints, not launch or marketing milestones. The release sequence is strictly sequential: finish and verify one release before starting the next.

## Definition of Done for V1 development

- `go test ./...`, `go vet ./...`, `go build ./...`, and `go mod verify` pass on Linux and Windows CI.
- `go test -race ./...` passes on the supported race-detector runner.
- The documented zero-argument and one-argument `brief` flows both work, including completion and deterministic fallback/error behavior.
- Config/workspace creation and atomic writes are hermetic and do not fail solely because Windows cannot apply POSIX mode bits.
- Cache mutation tests prove invalidation after HEAD, plan, notes, goals, provider, model, redaction, and command-option changes.
- Focus output rejects unknown or duplicate repositories and scores outside `1..5`; deadline urgency is computed from parsed goals in code.
- `doctor` produces human-readable output and a non-zero process status for any failed check.
- Secret scanning, prompt-injection handling, dry-run, redaction, logs, and AI-response handling have regression coverage.
- Public command output has golden or focused renderer tests for `brief`, `resume`, `focus`, `health`, `why`, `commit`, and `doctor`.
- README, SECURITY.md, help text, model defaults, provider behavior, CI, and release instructions describe the same implementation.
- GoReleaser produces Linux, Windows, and macOS archives with checksums, and a clean-install smoke check is documented and repeatable.

## Task List

### Release R1 — `v0.3.0` Platform Stability

**Release goal:** Remove the platform and path failures that can prevent a normal command from starting or make file-level Git history unreliable.

**Release deliverables:**
- Portable workspace/config initialization on Windows and POSIX systems.
- Absolute, cleaned repository paths.
- Safe repository-relative file paths for `why` and normalized Git path comparisons.

### R1-T1: Make workspace and config handling portable

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/permissions_unix.go`
- Create: `internal/config/permissions_windows.go`
- Modify: `internal/config/config_test.go`
- Create: `internal/config/workspace_test.go`

**Interfaces:**
- Produce `enforceDirectoryPermissions(path string) error` and `enforceFilePermissions(path string) error` as platform-specific helpers.
- `Load`, `Save`, and `ensureWorkspace` continue to expose their current public behavior; only their permission implementation changes.

**Steps:**

- [ ] Write tests that load a clean workspace through `DEVPULSE_CONFIG`, load an already-created workspace, save atomically, and verify no `.tmp` file remains.
- [ ] Add Unix helpers that apply `0700` to workspace directories and `0600` to private files.
- [ ] Add Windows helpers that create the directories/files with restrictive requested modes but do not fail solely when a redundant `Chmod` is unsupported or denied.
- [ ] Replace direct permission calls in `Save` and `ensureWorkspace` with the helpers. Preserve real create, write, rename, and path errors.
- [ ] Make tests assert POSIX permissions only when `runtime.GOOS != "windows"`; on Windows assert successful creation, read/write behavior, and absence of permission-related false failures.
- [ ] Run `go test ./internal/config -v` on the host and confirm the same package is covered by the Windows CI job.
- [ ] Commit as `fix: make workspace permissions portable`.

**Acceptance criteria:**
- [ ] A normal command works against both a clean and an existing workspace on Windows.
- [ ] Unix config/cache/log/notes directories remain `0700`; private files remain `0600` where the platform exposes those mode bits.
- [ ] Atomic config writes still remove temporary files after both success and rename failure.

**Dependencies:** None.

**Estimated scope:** Medium.

### R1-T2: Normalize repository and file paths before Git queries

**Files:**
- Modify: `internal/config/repository.go`
- Modify: `internal/collector/collector.go`
- Modify: `internal/collector/git.go`
- Create: `internal/collector/paths.go`
- Modify: `cmd/why.go`
- Modify: `internal/collector/collector_test.go`

**Interfaces:**
- Produce `NormalizeRepoPath(path string) (string, error)` for absolute, cleaned registration paths.
- Produce `NormalizeRepoRelativePath(repoRoot, filePath string) (string, error)` for `why` and file-history queries; reject absolute paths and paths that escape the repository root.
- `collector.CollectFileCommits(repoPath, filePath string, maxCommits, fullDiffCommits int, includeDiff bool)` continues to accept the existing signature but internally uses normalized slash-separated Git paths.

**Steps:**

- [ ] Add failing tests for relative registration paths, duplicate separators, Windows separators, missing paths, absolute file arguments, and `..` traversal outside a repository.
- [ ] Normalize and validate paths before storing them in `Config.RegisteredRepos`.
- [ ] Normalize the file path once in `runWhy`; pass the normalized repository-relative slash path to `CollectFileCommits` and the prompt.
- [ ] Update `CollectFileCommits` path filtering and `filePatchTouches` to compare Git paths using `/`, including rename old/new paths.
- [ ] Add tests covering renamed files and a Windows-style input such as `internal\api\handler.go`.
- [ ] Run `go test ./internal/config ./internal/collector ./cmd -run 'Path|File|Register|Why' -v`.
- [ ] Commit as `fix: normalize repository paths before git queries`.

**Acceptance criteria:**
- [ ] Registered repository paths are absolute and stable across repeated registration.
- [ ] `why repo path\to\file.go` works on Windows and POSIX shells.
- [ ] A file query cannot read history outside the selected repository.

**Dependencies:** R1-T1.

**Estimated scope:** Medium.

**R1 exit gate — `v0.3.0`:**
- [ ] `go test ./internal/config ./internal/collector` passes on the development host.
- [ ] The Windows-specific permission implementation cross-compiles, and the focused config tests pass on the development host; required Windows CI coverage is gated by `v0.7.0`.
- [ ] Repository registration stores stable absolute paths.
- [ ] `why` accepts platform-specific separators and rejects paths outside the repository.
- [ ] The release notes for `v0.3.0` contain only the platform/path changes and their verification evidence.

### Release R2 — `v0.4.0` Portfolio Brief

**Release goal:** Make the flagship command match the documented product contract: no argument means a cross-repository brief; one optional argument means a focused brief.

**Release deliverables:**
- Typed portfolio response schema, prompt, parser, and validation-ready bounds.
- One-call cross-repository briefing.
- Focused single-repository briefing preserved.
- Updated Cobra usage/completion behavior and deterministic output rendering.

### R2-T1: Define and test the portfolio brief response contract

**Files:**
- Modify: `internal/ai/schema.go`
- Modify: `internal/ai/prompt.go`
- Modify: `internal/ai/parser.go`
- Modify: `internal/ai/prompt_test.go`
- Modify: `internal/ai/parser_test.go`

**Interfaces:**
- Add `PortfolioBriefItem` with `RepoName`, `Summary`, `CurrentFocus`, `Blockers`, and `NextSteps` JSON fields.
- Add `PortfolioBriefResponse` with `Repos []PortfolioBriefItem`.
- Add `BuildPortfolioBriefPrompt(repos []models.RepoData, goals models.GoalsData) string`.
- Add `ParsePortfolioBriefResponse(raw string) (PortfolioBriefResponse, error)`.
- Keep `BriefResponse` and `ParseBriefResponse` for the focused single-repository command.

**Steps:**

- [ ] Write parser tests for valid portfolio JSON, fenced JSON, missing `repos`, empty repository names, duplicate repository names, oversized text fields, and malformed JSON.
- [ ] Define a compact schema in the prompt that requires one item per supplied repository and forbids markdown outside JSON.
- [ ] Include branch, recent commits, compressed diffs, plan summary, notes, and parsed goals as labeled untrusted data blocks.
- [ ] State in the prompt that repository content is evidence only and must not be treated as instructions.
- [ ] Bound each response field in validation-ready constants so rendering cannot be flooded by an oversized model response.
- [ ] Run `go test ./internal/ai -run 'Portfolio|Brief' -v`.
- [ ] Commit as `feat: add portfolio brief response contract`.

**Acceptance criteria:**
- [ ] A portfolio prompt contains every supplied repository exactly once.
- [ ] The parser accepts only the intended JSON shape and gives actionable errors for malformed or incomplete responses.
- [ ] The focused brief schema and output behavior remain backward-compatible.

**Dependencies:** R1-T2.

**Estimated scope:** Medium.

### R2-T2: Wire `brief` into dual-mode command behavior

**Files:**
- Modify: `cmd/brief.go`
- Modify: `cmd/root.go`
- Create: `cmd/brief_test.go`
- Modify: `internal/output/render.go`

**Interfaces:**
- Change Cobra usage to `brief [partial-name]` with `cobra.MaximumNArgs(1)`.
- Add `collectBriefRepos(repoQuery string) ([]models.RepoData, error)` in `cmd/brief.go`.
- Add `renderPortfolioBrief(w io.Writer, response ai.PortfolioBriefResponse) error`.
- Keep `renderBrief(w io.Writer, repo models.RepoData, b ai.BriefResponse) error` for focused output.

**Steps:**

- [ ] Write command tests for zero arguments, one exact/partial argument, ambiguous argument, invalid argument, zero registered repositories, and a repository collection failure.
- [ ] For zero arguments, collect every registered repository with the existing bounded `CollectOptions` and make one `ai.Run` call using `BuildPortfolioBriefPrompt` and `ParsePortfolioBriefResponse`.
- [ ] For one argument, preserve the current focused flow and fuzzy resolution.
- [ ] Render portfolio output with stable repository ordering from the registered-repository list; render each summary, focus, blockers, and next step without ANSI codes when `--no-color` is set.
- [ ] Add `ValidArgsFunction` behavior that offers repository names only when the optional first argument is being completed.
- [ ] Add a clear no-repositories error: `brief: no repositories registered; run: devpulse register <path>`.
- [ ] Run focused command tests and compare the output against the checked-in golden files introduced in Task 10.
- [ ] Commit as `feat: support cross-repository brief mode`.

**Acceptance criteria:**
- [ ] `devpulse brief` produces a cross-repository result with one provider call.
- [ ] `devpulse brief <partial-name>` still produces the focused result.
- [ ] README, Cobra help, completion, tests, and implementation all describe the same optional-argument contract.

**Dependencies:** R2-T1.

**Estimated scope:** Large; keep the implementation limited to this vertical slice.

**R2 exit gate — `v0.4.0`:**
- [ ] `devpulse brief` makes one provider call for all registered repositories.
- [ ] `devpulse brief <partial-name>` still produces a focused result.
- [ ] Zero, one, ambiguous, invalid, and no-repository argument cases have tests.
- [ ] `devpulse brief --help`, README examples, completion, and implementation agree.
- [ ] The portfolio output is stable enough to become a golden test fixture in the next release.

### Release R3 — `v0.5.0` Trust and Correctness

**Release goal:** Ensure DevPulse never silently serves stale context, accepts invalid ranking data, reports diagnostics inaccurately, or leaks tested secrets across the provider boundary.

**Release deliverables:**
- Complete cache fingerprints for all prompt inputs and command options.
- Strict typed model-output validation and deterministic deadline urgency.
- Non-zero `doctor` status for failed checks.
- Secret scanning, dry-run, redaction, and prompt-injection regression coverage.

### R3-T1: Create complete, deterministic cache fingerprints

**Files:**
- Create: `internal/cache/fingerprint.go`
- Modify: `internal/cache/cache.go`
- Modify: `internal/ai/pipeline.go`
- Modify: `internal/cache/cache_test.go`
- Modify: `internal/ai/pipeline_test.go`

**Interfaces:**
- Add `Fingerprint(parts ...any) (string, error)` that JSON-encodes each input deterministically and returns a SHA-256 digest.
- Keep `cache.Hash(parts ...string)` for existing callers, but use `Fingerprint` when a structured `RepoData` or options object is a cache input.
- `ai.Run` remains responsible for folding the loaded `GoalsData` into the final cache key.

**Steps:**

- [ ] Write tests proving two identical structured inputs produce the same fingerprint and any changed field produces a different fingerprint.
- [ ] Include the complete collected evidence in each command’s input fingerprint: `RepoData` contains commits, diffs, plan summary, notes, path, branch, and HEAD SHA.
- [ ] Include provider, resolved model, `redactDiff`, `since`, prompt-window options, and any command-specific flags in the command-level fingerprint.
- [ ] For portfolio `brief` and `focus`, fingerprint the ordered list of collected repository data rather than only HEAD SHAs.
- [ ] Preserve the current cache file atomic-write and TTL behavior; do not expand cache contents to include secrets or raw provider keys.
- [ ] Add mutation tests for HEAD, plan, notes, goals, provider, model, redaction, and `since` changes. Confirm cache hits do not construct the fake client.
- [ ] Run `go test ./internal/cache ./internal/ai -v`.
- [ ] Commit as `fix: fingerprint all prompt inputs for cache invalidation`.

**Acceptance criteria:**
- [ ] Editing a plan or note invalidates `brief`, `resume`, and `focus` without waiting for TTL expiry.
- [ ] Switching provider, model, redaction mode, or lookback options cannot reuse an incompatible response.
- [ ] Cache hits remain immediate and do not resolve API keys or construct network clients.

**Dependencies:** R2-T1 and R2-T2.

**Estimated scope:** Medium.

### R3-T2: Validate model output and compute urgency in code

**Files:**
- Create: `internal/ai/validate.go`
- Modify: `internal/ai/parser.go`
- Modify: `internal/ai/schema.go`
- Modify: `cmd/focus.go`
- Modify: `internal/ai/parser_test.go`
- Create: `internal/ai/validate_test.go`

**Interfaces:**
- Add `ValidateFocusResponse(response FocusResponse, allowedRepos map[string]struct{}) error`.
- Add `ValidateBriefResponse(response BriefResponse) error` and `ValidatePortfolioBriefResponse(response PortfolioBriefResponse, allowedRepos map[string]struct{}) error`.
- Add `ApplyDeadlineUrgency(items []FocusItem, goals models.GoalsData, window int) []FocusItem`.

**Steps:**

- [ ] Write failing tests for focus scores below 1 and above 5, empty names/reasons, unknown repositories, duplicate repositories, too many entries, and oversized strings.
- [ ] Validate required fields and bounded lengths immediately after parsing and before rendering or caching.
- [ ] Pass the collected repository-name set into focus validation; reject a model-created repository instead of rendering it.
- [ ] Make `ApplyDeadlineUrgency` ignore model-provided urgency and set it only when a parsed deadline is within the requested window and its description contains the repository name case-insensitively.
- [ ] Ensure focus cache entries cannot be written until the response passes validation and deterministic urgency has been applied.
- [ ] Add tests that use fixed `models.GoalsData` deadlines so urgency assertions do not depend on wall-clock time.
- [ ] Run `go test ./internal/ai ./cmd -run 'Focus|Validate|Urgency' -v`.
- [ ] Commit as `fix: validate and normalize focus responses`.

**Acceptance criteria:**
- [ ] Invalid ranking data fails with an actionable command error and is never cached.
- [ ] Every rendered focus repository belongs to the collected set and appears once.
- [ ] Deadline urgency is reproducible from parsed goals and the configured window, independent of model claims.

**Dependencies:** R2-T1 and R3-T1.

**Estimated scope:** Medium.

### R3-T3: Make `doctor` communicate failures through the process status

**Files:**
- Modify: `cmd/doctor.go`
- Create: `cmd/doctor_test.go`
- Modify: `cmd/root.go` only if the testable execution seam requires it

**Interfaces:**
- Add an internal `doctorFailure` error carrying `FailCount` and a stable `Error()` string.
- `runDoctor` continues printing the complete report but returns `doctorFailure` when `failCount > 0`.

**Steps:**

- [ ] Write a unit test with a missing registered repository that asserts the report contains `[FAIL]` and the returned error exposes the failure count.
- [ ] Write a passing test that asserts a healthy fixture returns `nil` while retaining the existing human-readable summary.
- [ ] Keep warnings non-fatal; only `FAIL` results produce a non-zero status.
- [ ] Verify `Execute` prints the error to stderr and exits with status 1 by running the compiled binary in a subprocess test or a small test helper process.
- [ ] Run `go test ./cmd -run Doctor -v` and `go build ./...`.
- [ ] Commit as `fix: return failure status from doctor`.

**Acceptance criteria:**
- [ ] Shell scripts can gate on `devpulse doctor` exit status.
- [ ] Human output still lists every pass, warning, and failure.
- [ ] A warning-only diagnostic run remains successful.

**Dependencies:** R1-T1.

**Estimated scope:** Small.

### R3-T4: Harden the privacy and provider-boundary behavior

**Files:**
- Modify: `internal/security/scanner.go`
- Modify: `internal/security/scanner_test.go`
- Modify: `internal/ai/pipeline.go`
- Modify: `internal/ai/pipeline_test.go`
- Modify: `cmd/root.go`
- Modify: `cmd/brief.go`
- Modify: `cmd/resume.go`
- Modify: `cmd/focus.go`
- Modify: `cmd/why.go`

**Interfaces:**
- Keep `security.ScanPrompt(prompt string) ScanResult` as the single pre-provider and post-provider redaction entry point.
- Preserve `--dry-run` as “show the post-redaction provider-bound prompt without making a provider call.”
- Preserve `--redact-diff` as “omit diff snippets before prompt construction,” not merely redact them after construction.

**Steps:**

- [ ] Add scanner tests for multiline PEM blocks, AWS/GCP credential fragments, JWTs, `.env` assignments, high-entropy values, and normal code with no false-positive redaction.
- [ ] Add pipeline tests proving commit messages, plan files, notes, and diffs containing instruction-like text are wrapped as untrusted data and cannot change the request schema.
- [ ] Add tests proving dry-run output contains `[REDACTED]` where needed, never contains the original secret, and never constructs the fake client.
- [ ] Add tests proving `--redact-diff` sends no diff content while retaining commit metadata, plans, notes, and goals; render an explicit low-trust banner.
- [ ] Verify provider responses are scanned before parsing and that redacted response text is not written to cache or logs in unredacted form.
- [ ] Audit log messages in the affected commands so they contain counts and event names, never prompt contents, API keys, or full model responses.
- [ ] Run `go test ./internal/security ./internal/ai ./cmd -run 'Secret|Redact|DryRun|Prompt' -v`.
- [ ] Commit as `fix: harden provider-boundary redaction`.

**Acceptance criteria:**
- [ ] No tested secret reaches the fake provider, cache, log, or dry-run output in original form.
- [ ] Prompt-injection text is treated as repository evidence and does not alter the requested JSON contract.
- [ ] Redaction and dry-run behavior are consistent for `brief`, `resume`, `focus`, and `why`.

**Dependencies:** R2-T1 through R3-T2.

**Estimated scope:** Large; split scanner and command changes into separate commits if review becomes difficult.

**R3 exit gate — `v0.5.0`:**
- [ ] Cache mutation tests pass for HEAD, plans, notes, goals, provider, model, redaction, and command options.
- [ ] Invalid focus output is rejected and never cached.
- [ ] Deadline urgency is computed from parsed goals, not trusted from model output.
- [ ] A failing doctor check returns a non-zero status; warning-only output remains successful.
- [ ] Dry-run and redaction tests prove original secrets do not reach the fake provider, cache, logs, or output.
- [ ] No unresolved high-severity audit issue remains in the R3 scope.

### Release R4 — `v0.6.0` Testable CLI

**Release goal:** Make the public CLI behavior understandable and protected by deterministic, network-free tests.

**Release deliverables:**
- Actionable error paths and current technical documentation.
- A temporary-workspace fixture corpus and fake provider seam.
- Golden output coverage for every public V1 command.

### R4-T1: Improve documented error paths and technical documentation

**Files:**
- Modify: `README.md`
- Modify: `SECURITY.md`
- Modify: `CONTRIBUTING.md`
- Modify: `cmd/root.go`
- Modify: `internal/ai/client.go`
- Modify: `internal/ai/models.go`
- Modify: `cmd/register.go`
- Modify: `cmd/brief.go`
- Modify: `cmd/resume.go`
- Modify: `cmd/why.go`

**Interfaces:**
- Keep provider names limited to `groq` and `gemini`; invalid providers fail before collection or network access.
- Document the actual defaults: Groq `openai/gpt-oss-20b`, Gemini fast `gemini-2.5-flash-lite`, Gemini deep `gemini-2.5-flash`, and 24-hour cache duration.
- Document exact behavior for missing keys, stale registrations, empty Git repositories, unreadable workspaces, malformed provider JSON, dry-run, and `--redact-diff`.

**Steps:**

- [ ] Add command tests for missing provider key, unsupported provider, empty repository, deleted registered path, invalid `--since`, and malformed AI response.
- [ ] Ensure errors name the failed operation and provide the next safe command where one exists; do not expose raw secrets or full provider payloads.
- [ ] Update README command help and examples for `brief` with zero or one argument, including shell completion and checksum verification.
- [ ] Update SECURITY.md data-flow, cache/history/log retention, key revocation, redaction guarantees, and known limitations so they match the code.
- [ ] Update CONTRIBUTING.md with the required focused test, formatting, vet, build, and cross-platform checks.
- [ ] Remove stale Gemini 2.0/1.5 claims and any unsupported statement that binaries are signed or that a specific installation script exists.
- [ ] Run `go test ./...`, then manually compare `devpulse --help`, `devpulse brief --help`, and `devpulse doctor --help` with the documentation.
- [ ] Commit as `docs: align v1 command and privacy contracts`.

**Acceptance criteria:**
- [ ] A clean user can follow the README to configure a provider, register a repository, and reach the first brief without undocumented prerequisites.
- [ ] Documentation makes clear exactly what leaves the machine and what remains local.
- [ ] Every documented model, flag, command, and release verification instruction exists in the implementation.

**Dependencies:** R2-T2, R3-T3, and R3-T4.

**Estimated scope:** Large; keep this task documentation-focused and avoid feature additions.

### R4-T2: Add deterministic fixture repositories and command test seams

**Files:**
- Create: `cmd/test_support_test.go`
- Create: `cmd/test_fixtures_test.go`
- Modify: `cmd/root.go`
- Modify: `cmd/brief.go`
- Modify: `cmd/resume.go`
- Modify: `cmd/focus.go`
- Modify: `cmd/why.go`
- Create: `testdata/golden/.gitkeep`

**Interfaces:**
- Add a test-only-safe override around client construction, for example `clientFactoryOverride ai.ClientFactory`, while production defaults continue to use `newClientFactory`.
- Add fixture helpers that create temporary go-git repositories with controlled commits, branches, plan files, notes, deadlines, renamed files, and secret-bearing diffs.
- Add `executeForTest(args ...string) (stdout, stderr string, err error)` that resets mutable command flags and global manager state between tests.

**Steps:**

- [ ] Write the fake client first; it must record prompt count and prompt text and return deterministic JSON for brief, portfolio brief, resume, focus, and why.
- [ ] Build fixture helpers for: active feature work, stale repository, plan-heavy repository, no-plan repository, empty repository, renamed file, ambiguous repository names, and secret-bearing diff.
- [ ] Make tests use `DEVPULSE_CONFIG` and temporary workspace paths rather than the real user home or keychain.
- [ ] Ensure each test restores global provider, dry-run, redaction, color, `since`, manager, and client override state with `t.Cleanup`.
- [ ] Add golden-file comparison helpers that normalize only platform line endings; do not strip meaningful content or ANSI codes unless the test explicitly requests `--no-color`.
- [ ] Run `go test ./cmd -run TestFixture -v` repeatedly to prove tests are isolated and order-independent.
- [ ] Commit as `test: add deterministic CLI fixture harness`.

**Acceptance criteria:**
- [ ] Command tests never use a real provider, network, real home directory, or real keychain.
- [ ] The fixture corpus reproduces every high-risk audit scenario locally.
- [ ] Tests pass when run individually and as a full package.

**Dependencies:** R1-T1 through R3-T4.

**Estimated scope:** Large; keep fixture construction in test-only files.

### R4-T3: Protect the public command output with golden tests

**Files:**
- Create: `cmd/brief_test.go` if not completed in Task 4
- Create: `cmd/resume_test.go`
- Create: `cmd/focus_test.go`
- Modify: `cmd/health_test.go`
- Create: `cmd/why_test.go`
- Create: `cmd/commit_test.go`
- Extend: `cmd/doctor_test.go`
- Create: `cmd/testdata/brief-cross-repo.golden`
- Create: `cmd/testdata/brief-focused.golden`
- Create: `cmd/testdata/resume.golden`
- Create: `cmd/testdata/focus.golden`
- Create: `cmd/testdata/health.golden`
- Create: `cmd/testdata/why.golden`
- Create: `cmd/testdata/commit.golden`
- Create: `cmd/testdata/doctor.golden`

**Interfaces:**
- Tests exercise the existing command runners and renderer functions; no production output format is changed merely to make a test easier.
- Golden fixtures use stable timestamps, fixed repository names, `--no-color`, and fake provider responses.

**Steps:**

- [ ] Add a focused brief golden test that asserts repository order, current focus, blockers, and next steps.
- [ ] Add a portfolio brief golden test that asserts all fixture repositories appear once and one provider call is made.
- [ ] Add resume, focus, health, why, commit, and doctor golden tests for the documented headings and failure/warning output.
- [ ] Add explicit behavior tests for brief zero/one/ambiguous/invalid arguments and doctor failure exit status.
- [ ] Add cache tests around the command harness showing the second run is a cache hit and every prompt-input mutation causes a miss.
- [ ] Review each golden file manually for accidental paths, timestamps, secrets, provider payloads, or platform-specific noise.
- [ ] Run `go test ./cmd -v` and update goldens only when the public contract intentionally changes.
- [ ] Commit as `test: cover public command output contracts`.

**Acceptance criteria:**
- [ ] Every public V1 command has a regression test for its successful output and its highest-risk error path.
- [ ] Golden output is deterministic on Windows and POSIX runners.
- [ ] A future command-contract regression fails a focused test before release.

**Dependencies:** R4-T2.

**Estimated scope:** Large.

**R4 exit gate — `v0.6.0`:**
- [ ] README, SECURITY.md, CONTRIBUTING.md, Cobra help, defaults, and provider behavior agree.
- [ ] Command tests do not use a real provider, network, real home directory, or real keychain.
- [ ] `brief`, `resume`, `focus`, `health`, `why`, `commit`, and `doctor` each have success and high-risk error coverage.
- [ ] Golden output is deterministic on Windows and POSIX runners.
- [ ] `go test ./...` passes with the complete fixture suite.

### Release R5 — `v0.7.0` Release Candidate

**Release goal:** Convert the tested CLI into a reproducible, cross-platform release candidate with required CI and artifact checks.

**Release deliverables:**
- Required Linux and Windows CI jobs, with Linux race checks.
- Go module verification and dependency review step/documentation.
- GoReleaser archive/checksum verification.
- Technical release checklist in `docs/RELEASE.md`.

### R5-T1: Add Windows CI, race checks, and release verification

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `.goreleaser.yaml`
- Create: `docs/RELEASE.md`

**Interfaces:**
- CI must run the same Go version on `ubuntu-latest` and `windows-latest`.
- Release automation continues to publish Linux, Windows, and macOS archives with `checksums.txt`.
- `docs/RELEASE.md` is the technical checklist for building, verifying, and rolling back a V1 artifact; it contains no launch or marketing plan.

**Steps:**

- [ ] Add a Windows test job covering formatting, vet, tests, build, config path handling, registration, and doctor diagnostics.
- [ ] Keep Linux checks for `gofmt`, `go vet`, `go test`, and `go build`; add `go test -race ./...` on Linux where supported.
- [ ] Add `go mod verify` and a dependency vulnerability review step or documented pre-release command.
- [ ] Confirm release GoReleaser targets and archive names match README instructions for Linux, Windows, and macOS; retain SHA-256 checksums.
- [ ] Add a release smoke job or documented local procedure that extracts each archive, runs `devpulse version`, `devpulse --help`, and the clean-workspace test without a provider key.
- [ ] Document the V1 release sequence: clean checkout, full checks, `goreleaser check`, snapshot build, archive inspection, checksum verification, tag from `main`, and rollback by removing the release/tag and publishing a corrected patch release.
- [ ] Run the workflow commands locally as far as the installed toolchain allows and verify YAML syntax.
- [ ] Commit as `ci: add cross-platform v1 release gates`.

**Acceptance criteria:**
- [ ] Windows is a required CI target rather than an untested release destination.
- [ ] Published archives contain the documented binary and required license/security/contributing files.
- [ ] Checksums can be verified on POSIX and PowerShell using the documented commands.

**Dependencies:** R1-T1, R4-T1, and R4-T3.

**Estimated scope:** Medium.

**R5 exit gate — `v0.7.0`:**
- [ ] Linux and Windows CI pass on the same Go version.
- [ ] Linux race tests, vet, build, module verification, and formatting checks pass.
- [ ] Snapshot archives contain the documented binary and required project files.
- [ ] Checksums verify on POSIX and PowerShell.
- [ ] The extracted binaries run `version`, `help`, and the clean-workspace smoke path.

### Release R6 — `v1.0.0` Technical V1 Release

**Release goal:** Run the final technical gate against the completed release candidate and hand off a tag-ready repository.

**Release deliverables:**
- Complete automated verification evidence.
- Clean-install and failure-path evidence.
- A final `docs/RELEASE.md` record and maintainer-ready tag proposal.

### R6-T1: Run the technical V1 readiness gate

**Files:**
- Modify: `docs/RELEASE.md` with the final evidence links and command output summary.
- Modify: `CHANGELOG.md` only if the repository introduces one during release preparation.

**Steps:**

- [ ] Start from a clean checkout on `main` containing the completed V1 commits.
- [ ] Run `gofmt -l .` and require empty output.
- [ ] Run `go vet ./...`, `go test ./...`, `go test -race ./...`, `go build ./...`, and `go mod verify`.
- [ ] Run the fixture/golden suite with `go test ./cmd ./internal/... -v` and inspect failures rather than weakening assertions.
- [ ] Build a GoReleaser snapshot, inspect archive contents, run the binary from an extracted directory, and verify `checksums.txt`.
- [ ] Exercise the clean technical path: create a temporary workspace, initialize goals, register two fixture repositories, run `brief --dry-run`, `brief`, `resume`, `focus`, `health`, `why`, `commit`, and `doctor` with fake or configured provider behavior as appropriate.
- [ ] Exercise the failure path: missing provider key, invalid provider, deleted repo, malformed cache, redacted secret, prompt-injection fixture, and failing doctor check.
- [ ] Confirm no V1 blocker remains in the audit list and no required command/documentation contract is inconsistent.
- [ ] Record the evidence in `docs/RELEASE.md` and propose the tag `v1.0.0` from `main`.

**Acceptance criteria:**
- [ ] All automated and manual technical gates pass.
- [ ] No P0/P1 technical issue remains open for the documented V1 scope.
- [ ] The repository is ready for the maintainer to tag and publish `v1.0.0`.

**Dependencies:** R1-T1 through R5-T1.

**Estimated scope:** Medium.

**R6 exit gate — `v1.0.0`:**
- [ ] All automated and manual technical gates pass from a clean checkout on `main`.
- [ ] No P0/P1 technical issue remains for the documented V1 scope.
- [ ] `docs/RELEASE.md` records the evidence and rollback procedure.
- [ ] The maintainer has explicitly approved tagging `v1.0.0`.

## Checkpoints

### Checkpoint A — after `v0.3.0` and `v0.4.0`

- [ ] Workspace tests pass on the development host.
- [ ] Path normalization covers Windows separators and renamed files.
- [ ] `brief` supports both zero and one repository argument.
- [ ] Focused AI/parser tests pass.

### Checkpoint B — after `v0.5.0`

- [ ] Cache mutation tests pass for every prompt input.
- [ ] Focus validation rejects invalid model output and urgency is deterministic.
- [ ] `doctor` returns non-zero on failures.
- [ ] Dry-run, redaction, secret scanning, and prompt-injection tests pass.

### Checkpoint C — after `v0.6.0`

- [ ] Documentation matches help and defaults.
- [ ] Fixture tests are isolated and network-free.
- [ ] Golden output tests cover the public V1 command contract.

### Checkpoint D — after `v0.7.0` and `v1.0.0`

- [ ] Linux and Windows CI pass.
- [ ] Release archives and checksums verify.
- [ ] Technical V1 readiness evidence is recorded.

## Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Reusing reverted fixes without understanding their failure mode | High | Re-implement from the acceptance criteria, run focused regression tests, and inspect the diff before committing. |
| Global Cobra state makes command tests order-dependent | High | Add a test-only command harness that resets flags, manager, provider, and client overrides with `t.Cleanup`. |
| Model output changes invalidate brittle tests | Medium | Validate typed contracts, use fake clients for deterministic tests, and keep golden files focused on the user-facing contract. |
| Windows filesystem semantics differ from POSIX mode bits | High | Use build-tagged permission helpers and platform-appropriate assertions. |
| Cache fingerprints grow too large or accidentally include secrets | Medium | Hash structured inputs; never store raw prompt text or API keys in cache keys or payloads. |
| V1 scope expands into the autonomous-platform roadmap | High | Treat the explicit non-goals as a release gate; reject new commands and integrations unless they are required to close a listed technical blocker. |

## Explicitly deferred after V1 development

The following remain outside this implementation plan: alpha recruitment and retention measurement, market/competitor work, launch communications, hosted services, team features, billing, telemetry, IDE/UI clients, MCP integrations, background monitoring, persistent semantic memory, tool execution, multi-agent orchestration, and autonomous code modification.

## Plan self-review

- [x] Every task has files, interfaces or boundaries, acceptance criteria, verification steps, and dependencies.
- [x] High-risk platform, cache, contract, validation, privacy, and diagnostic work occurs before release polish.
- [x] The broad V2 roadmap is represented only as explicit non-goals; no V2 feature is silently required for V1.
- [x] No market validation or launch work is mixed into the development task list.
- [x] No task requires a real provider or network during automated tests.
- [x] The plan uses the current `internal/ai` provider abstraction rather than introducing a Gemini-only package.

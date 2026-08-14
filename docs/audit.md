# DevPulse Full Audit

**Audit date:** 2026-08-14  
**Audited checkout:** `mvp` at `65979368720cf1f7ad845d3d4c46b69f8fb7790c`  
**Release-line comparison:** `main` / `v0.2.0` at `f63a56e59b1f7c27927f822b81770a5b9dd8ea0b`

## Executive verdict

DevPulse has a coherent MVP: a small Go CLI, clear package boundaries, local repository collection, provider abstraction, prompt redaction, OS keychain storage, caching, and useful command-level tests. It is not yet a credible `v1.0.0` release from the audited checkout.

The immediate blockers are cross-platform workspace behavior, incomplete CI coverage, a user-facing `brief` contract mismatch, stale-cache paths, and diagnostics that do not communicate failure through the process exit code. These are fixable without changing the product thesis. The larger product risk is not technical: major coding assistants and adjacent open-source tools are converging on repository memory and session recovery, so DevPulse must prove that its cross-repository Git/plan workflow is more useful and more trustworthy than a built-in feature.

## Verification baseline

| Check | Result | Evidence |
|---|---|---|
| `go test -cover ./...` on the audited Windows checkout | **Fail** | Five failures in `internal/config`; workspace `chmod` errors, home-directory test isolation, and POSIX permission expectation |
| `go vet ./...` | Pass | Exit code 0 |
| `go build ./...` | Pass with environment warning | Exit code 0; Go module-cache stat write reported `Access is denied` outside the repository |
| `go mod verify` | Pass | `all modules verified` |
| `git diff --check` | Pass | No whitespace errors |
| Coverage | Uneven | `compressor` 95.4%, `security` 93.5%, `cmd` 14.5%, `config` 20.8% |

The checked-out `mvp` branch is behind `main` by one test-only commit. That commit makes the config tests more Windows-aware, but does not address the production `ensureWorkspace` permission behavior observed during this audit.

## Findings

### A-001 — High: Windows workspace initialization can fail on an existing workspace

**Evidence:** `internal/config/config.go:188-203`; observed error: `set directory permissions ... chmod ...: Access is denied`.

`ensureWorkspace` treats `os.Chmod` as mandatory for every workspace directory. POSIX mode bits are meaningful on Unix, but Windows permission behavior is different and existing directories may reject this operation. A normal command can therefore fail before loading configuration.

**Impact:** Windows users can be blocked from every command after installation or migration.

**Required action:** Make permission enforcement platform-aware. Preserve restrictive permissions where the platform supports them; on Windows, create directories with the strongest supported behavior and do not fail solely because a redundant `Chmod` cannot be applied. Add clean-workspace and existing-workspace tests.

### A-002 — High: CI does not execute the advertised Windows target

**Evidence:** `.github/workflows/ci.yml:14-46` runs only `ubuntu-latest`, while GoReleaser publishes Windows archives.

The repository can build Windows binaries without testing Windows filesystem, keychain, path, terminal, or permission behavior.

**Impact:** Release regressions can pass CI and reach Windows users.

**Required action:** Add a Windows test job. Keep Linux coverage, and add at least Windows unit/integration tests for config, repository registration, path handling, and CLI diagnostics.

### A-003 — High: The documented `brief` contract does not match the implementation

**Evidence:** `README.md:112-116` and `README.md:122-125` show `devpulse brief` as a cross-repository default; `cmd/brief.go:18-21` requires exactly one repository and executes a single-repository brief.

**Impact:** First-time users following the README receive an argument error. The flagship product promise is therefore broken at activation.

**Required action:** Make `brief` dual-mode: no argument produces the cross-repository standup; an optional repository produces a focused brief. Update Cobra help, completion, tests, and README together.

### A-004 — High: Cache keys omit prompt inputs

**Evidence:** `cmd/brief.go:68-81` includes `HeadSHA` and plan summary but not repository notes; `cmd/focus.go` keys only on repository names and HEAD SHAs. `internal/collector/collector.go:48-62` loads plan summaries and notes into prompts.

**Impact:** Editing a note or plan can leave users looking at an older AI response until the TTL expires. `focus` can remain stale even when planning context changes.

**Required action:** Build cache keys from every prompt input: collected repository data, plan summary, notes, goals, provider, model, redaction mode, and command-specific options. Add mutation-based cache invalidation tests.

### A-005 — Medium: AI response validation is too permissive for ranking output

**Evidence:** `internal/ai/parser.go` validates only that `ranked` is non-empty; `internal/ai/schema.go` defines `ProximityScore` and `RepoName`, but there is no range or repository-membership validation before rendering in `cmd/focus.go`.

**Impact:** The model can return an unknown repository, duplicate repositories, or scores outside 1–5. The output can look authoritative while violating the command contract.

**Required action:** Validate score range, non-empty names, uniqueness, and membership in the collected repository set. Compute deadline urgency deterministically from parsed goals instead of trusting the model alone.

### A-006 — Medium: `doctor` prints failures but exits successfully

**Evidence:** `cmd/doctor.go:153-162` renders a failure summary and returns the writer error, not an error representing `failCount > 0`.

**Impact:** Shell scripts and CI cannot reliably gate on diagnostics.

**Required action:** Return a typed or stable error when one or more checks fail, while preserving the human-readable report. Add a CLI-level exit-code test.

### A-007 — Medium: Documentation contains stale model and release claims

**Evidence:** `README.md:156-161` documents `gemini-2.0-flash`, while `internal/ai/models.go` defaults to `gemini-2.5-flash-lite` and `gemini-2.5-flash`. README also states that downloads are unsigned, with no release-signing or provenance workflow.

**Impact:** Users configure obsolete examples and receive weaker supply-chain assurance than expected for a security-sensitive local tool.

**Required action:** Generate or manually verify model documentation from the current defaults. For v1, publish checksums and provenance; decide explicitly whether code signing is v1 or v1.1 scope.

### A-008 — Medium: The core command layer has low test coverage

**Evidence:** Coverage from the baseline run: `cmd` 14.5%, `config` 20.8%, while behavior-heavy command paths are largely untested with real managers, fake AI clients, and process exit assertions.

**Impact:** Regressions in CLI contracts, cache wiring, error handling, and output behavior are likely to escape unit tests.

**Required action:** Add fixture-repository CLI tests, fake-client pipeline tests for every command, cross-platform config tests, and golden output tests for the public command contract.

## Strengths worth preserving

- The collector, compressor, AI, cache, security, and output packages provide reasonable seams for testing.
- AI calls are lazy on cache hits, bounded by a prompt-size guard, retried, and parsed into typed responses.
- Repository-derived prompt content is explicitly treated as untrusted, and the scanner covers several common credential families.
- API keys are kept out of the config file and use OS keychain/environment-variable lookup.
- CI already enforces formatting, vet, tests, and builds on Linux.

## Security and privacy assessment

The local-first claim is directionally credible, but the release bar should be higher because repository diffs and plans are sensitive. The next security pass must test redaction against multiline PEM material, provider-specific keys, connection strings, high-entropy values, prompt injection in commit messages, and secrets returned by the model. It must also verify that cache files, logs, dry-run output, and error messages do not reintroduce redacted content.

No destructive command execution or server-side URL fetch was found in the audited surface. The primary trust boundary is local repository content entering an external LLM provider, not an HTTP application boundary.

## Release recommendation

Do not tag `v1.0.0` from the audited checkout. First close A-001 through A-006, add Windows CI, reconcile `mvp` with `main`, and run the alpha validation described in [`docs/market-viability.md`](market-viability.md). A v1 release is technically realistic after those changes because the required work is stabilization and contract alignment rather than a rewrite.

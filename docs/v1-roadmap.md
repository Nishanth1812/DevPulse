# DevPulse v1.0.0 Roadmap

**Target horizon:** 8–12 weeks
**Release baseline:** reconcile `mvp` with `main` / `v0.2.0` before stabilization
**Primary user:** solo developer with multiple repositories
**Release model:** open-source, local-first, provider-agnostic

## v1 definition

DevPulse v1 is a dependable terminal tool that helps one developer recover and prioritize context across personal Git repositories. It must be safe to trust with repository metadata, quick to install, explicit about data sent to providers, and consistent on Windows, macOS, and Linux.

### Public command contract

- `devpulse brief` — cross-repository standup covering active, stale, blocked, and next work.
- `devpulse brief <partial-name>` — focused brief for one registered repository.
- `devpulse resume <partial-name>` — deep context recovery for one repository.
- `devpulse focus` — ranked cross-repository prioritization.
- `devpulse health` — deterministic repository hygiene checks.
- `devpulse why <partial-name> <file>` — file-level Git archaeology.
- `devpulse commit` — Conventional Commit suggestion from staged changes.
- Existing setup, notes, registration, configuration, diagnostic, and debug commands remain compatible.

## Phase 0 — Baseline and product contract

**Timing:** week 1

- Reconcile the checked-out `mvp` branch with `main` without losing useful changes.
- Record the release baseline SHA and update the changelog strategy.
- Implement and document the dual-mode `brief` contract.
- Add a release checklist covering tests, supported platforms, artifacts, checksums, documentation, and rollback.

**Exit criteria:** README, Cobra help, completion behavior, tests, and roadmap agree on the same command contract.

## Phase 1 — Trust and reliability

**Timing:** weeks 1–3

- Make workspace directory and file permission handling platform-aware.
- Make config tests hermetic on Windows and POSIX systems.
- Add Windows CI alongside Linux CI; retain macOS build coverage through release smoke tests.
- Normalize repository-relative paths before Git history queries.
- Include all prompt inputs in cache keys, including notes, plans, goals, provider, model, redaction mode, and command options.
- Validate AI responses for required fields, score ranges, repository membership, uniqueness, and bounded output size.
- Compute deadline urgency from parsed goals deterministically.
- Make `doctor` return a non-zero process status when a check fails.
- Test redaction, prompt injection, cache invalidation, retry behavior, cancellation, and oversized prompts.

**Exit criteria:** no known high-severity audit finding remains; Windows and Linux test jobs pass; cache and diagnostic behavior are covered by regression tests.

## Phase 2 — Activation and UX

**Timing:** weeks 3–5

- Make the first-run path clear: install, authenticate or configure a provider, register a repository, and run the first brief.
- Improve errors for missing keys, empty repositories, invalid providers, stale registrations, unavailable keychains, and unreadable workspaces.
- Align model examples, security claims, command help, and release instructions with the implementation.
- Add golden output tests for brief, resume, focus, health, why, and doctor.
- Verify `--dry-run` and `--redact-diff` are understandable without prior product knowledge.

**Exit criteria:** a fresh user can reach a useful first brief in five minutes using documented steps, with no undocumented prerequisites.

## Phase 3 — Alpha validation

**Timing:** weeks 5–8

- Recruit 10–15 solo developers with multiple active repositories.
- Run the two-week validation experiment defined in [`docs/market-viability.md`](market-viability.md).
- Collect opt-in feedback on activation, accuracy, retention, trust, latency, cost, and missing workflows.
- Create a small fixture corpus representing common repo shapes: active feature work, stale repo, plan-heavy repo, no-plan repo, empty repo, renamed file, and secret-bearing diff.
- Fix only issues that block activation, trust, or repeat use.

**Exit criteria:** activation, retention, value, trust, and reliability gates are measured. Any failed gate produces a product decision before release-candidate work continues.

## Phase 4 — Release candidate

**Timing:** weeks 8–10

- Freeze v1 scope and stop adding integrations.
- Run full unit, integration, CLI, race, security, dependency, and cross-platform checks.
- Validate GoReleaser archives on Linux, Windows, and macOS smoke environments.
- Publish SHA-256 checksums and artifact provenance; document the remaining signing decision if code signing is deferred.
- Review GPL-3.0 notices, changelog, security policy, contributing guide, installation instructions, and examples.
- Test upgrade behavior from the latest pre-v1 release and recovery from invalid config/cache files.

**Exit criteria:** the release candidate installs cleanly on a fresh machine, produces a first brief, and has no open P0/P1 issue.

## Phase 5 — v1.0.0 release

**Timing:** weeks 10–12

- Tag `v1.0.0` from `main` only after all release gates pass.
- Publish binaries, checksums, release notes, and supported-platform documentation.
- Announce the focused use case and the privacy/data-flow model.
- Track installation failures, provider failures, misleading summaries, and user retention through public issues and opt-in feedback.
- Schedule a post-release review after two weeks; decide whether the next investment is usability, integrations, or a different product position.

## Testing matrix

### Required automated checks

- `gofmt -l .`
- `go vet ./...`
- `go test ./...`
- `go test -race ./...` where supported
- `go mod verify`
- Dependency vulnerability review before release
- Linux and Windows CI; macOS release smoke test

### Required behavioral scenarios

- Clean first run with no config, no goals, and no API key.
- Existing workspace on Windows and POSIX systems.
- Config save, atomic replacement, invalid config, and read-only workspace.
- Repository registration, missing path, empty Git repository, renamed file, and Windows path separators.
- Cache hits and invalidation after HEAD, plan, notes, goals, provider, model, and redaction changes.
- Provider timeout, retry exhaustion, malformed JSON, oversized response, and secret-bearing response.
- Prompt injection embedded in a commit message, plan file, note, or diff.
- `brief` with zero, one, ambiguous, and invalid repository arguments.
- `doctor` human output and process exit status.
- Fresh-machine installation and first useful brief from every published archive family.

## Explicit non-goals

V1 does not include team accounts, hosted storage, billing, background session tracking, IDE plugins, mandatory telemetry, full project-management integrations, or autonomous code changes. Those are post-validation options, not release requirements.

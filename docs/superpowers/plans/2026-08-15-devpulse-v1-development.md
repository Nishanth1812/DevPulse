# DevPulse V1 Development Plan

> **For agentic workers:** The detailed task-by-task plan is maintained at [`tasks/plan.md`](../../../tasks/plan.md); use that file as the implementation source of truth.

**Goal:** Make DevPulse technically ready for a V1 release through code, tests, CI, technical documentation, and release verification.

**Architecture:** Preserve the existing Go CLI and its `cmd/` plus `internal/` package boundaries. Strengthen cross-platform config, command contracts, cache fingerprints, response validation, privacy boundaries, test seams, and release automation without adding hosted or autonomous features.

**Tech Stack:** Go 1.25.12, Cobra, go-git, Groq/Gemini clients, GoReleaser, GitHub Actions.

## Global Constraints

- Development scope only: implementation, tests, CI, technical documentation, and release readiness.
- No market validation, launch planning, hosted services, team features, telemetry, IDE clients, MCP integrations, or autonomous code changes.
- `brief` must support zero arguments for a cross-repository briefing and one optional repository argument for a focused briefing.
- All provider-bound prompts are redacted and all repository-derived text is treated as untrusted data.
- All cache fingerprints include every prompt input and command option.

---

Use [`tasks/plan.md`](../../../tasks/plan.md) for the full ordered release train, release-specific task IDs (`R1-T1` through `R6-T1`), interfaces, acceptance criteria, checkpoints, risks, and V1 readiness gate. Track progress in [`tasks/todo.md`](../../../tasks/todo.md).

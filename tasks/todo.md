# DevPulse V1 Development Task Checklist

Source of truth: [tasks/plan.md](plan.md)

## Release R1 — `v0.3.0` Platform Stability

- [x] `R1-T1` — Make workspace and config handling portable
- [x] `R1-T2` — Normalize repository and file paths before Git queries

### R1 exit gate

- [x] Cross-platform workspace and path tests pass
- [x] Stable absolute repository paths are stored
- [x] File-history queries handle platform separators safely
- [x] `v0.3.0` engineering checkpoint is approved

## Release R2 — `v0.4.0` Portfolio Brief

- [x] `R2-T1` — Define and test the portfolio brief response contract
- [x] `R2-T2` — Wire `brief` into dual-mode command behavior

### R2 exit gate

- [x] `brief` works with zero and one repository argument
- [x] Cross-repository brief uses one provider call
- [x] Argument, completion, and output contract tests pass
- [ ] `v0.4.0` engineering checkpoint is approved

## Release R3 — `v0.5.0` Trust and Correctness

- [ ] `R3-T1` — Create complete, deterministic cache fingerprints
- [ ] `R3-T2` — Validate model output and compute urgency in code
- [ ] `R3-T3` — Make `doctor` communicate failures through the process status
- [ ] `R3-T4` — Harden privacy and provider-boundary behavior

### R3 exit gate

- [ ] Cache invalidation covers all prompt inputs
- [ ] Invalid focus output is rejected and urgency is deterministic
- [ ] Doctor exit status and privacy regression tests pass
- [ ] `v0.5.0` engineering checkpoint is approved

## Release R4 — `v0.6.0` Testable CLI

- [ ] `R4-T1` — Improve documented error paths and technical documentation
- [ ] `R4-T2` — Add deterministic fixture repositories and command test seams
- [ ] `R4-T3` — Protect public command output with golden tests

### R4 exit gate

- [ ] Documentation matches implementation and help output
- [ ] Fixture tests are network-free and isolated
- [ ] Public command golden tests pass
- [ ] `v0.6.0` engineering checkpoint is approved

## Release R5 — `v0.7.0` Release Candidate

- [ ] `R5-T1` — Add Windows CI, race checks, and release verification

### R5 exit gate

- [ ] Linux and Windows CI pass
- [ ] Release archives and checksums verify
- [ ] Extracted binaries pass smoke checks
- [ ] `v0.7.0` release candidate is approved

## Release R6 — `v1.0.0` Technical V1 Release

- [ ] `R6-T1` — Run the technical V1 readiness gate

### R6 exit gate

- [ ] All automated and manual technical gates pass
- [ ] Technical evidence is recorded in `docs/RELEASE.md`
- [ ] No P0/P1 technical issue remains in V1 scope
- [ ] Maintainer approval obtained before tagging `v1.0.0`

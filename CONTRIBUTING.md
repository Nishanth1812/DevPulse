# Contributing to DevPulse

First off, thanks for taking the time to contribute! 🎉

The following is a set of guidelines for contributing to DevPulse. These are just guidelines, not hard rules — use your best judgment and feel free to propose changes to this document via a pull request.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [What we're looking for](#what-were-looking-for)
- [Getting started](#getting-started)
- [Finding / creating an issue](#finding--creating-an-issue)
- [Making changes](#making-changes)
- [Code style](#code-style)
- [Commit conventions](#commit-conventions)
- [Submitting a pull request](#submitting-a-pull-request)
- [Reviewing](#reviewing)

---

## Code of Conduct

This project and everyone participating in it is governed by the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behaviour.

## What we're looking for

- Bug fixes with clear reproduction steps
- Documentation improvements and typo fixes
- New commands or features — please open an issue first to discuss
- Performance improvements, especially around token usage and cache behaviour
- Tests — the codebase currently has limited coverage and any new tests are welcome

## Getting started

### Prerequisites

- Go `1.25` or later
- `git`
- Optionally, a Groq or Gemini API key for running LLM-powered commands

### Clone and build

```sh
git clone https://github.com/Nishanth1812/devpulse.git
cd devpulse
go build ./...
go test ./...
go vet ./...
go mod verify
```

Before opening an integration change, also run:

```sh
gofmt -l .
go test -race ./...   # Linux or another supported race-detector runner
```

Command tests must use temporary fixtures and fake providers. They must not
require a real provider, network, home directory, or keychain.

## Finding / creating an issue

- Search [existing issues](https://github.com/Nishanth1812/devpulse/issues) to see if the problem or feature has already been reported.
- If open an issue, please use the provided templates and include:
  - The version of DevPulse (`devpulse version`)
  - Your OS and architecture
  - Steps to reproduce (for bugs)
  - The expected and actual behaviour

## Making changes

Please work against the `mvp` branch for development work. The `dev` and
`main` branches are integration and release branches.

```sh
git checkout mvp
git pull --ff-only origin mvp
git checkout -b fix/your-fix mvp
# ... make changes ...
```

Keep changes small and focused. A single PR should address a single concern.

## Code style

- Run `gofmt` / `go fmt` before committing.
- Run `go vet ./...` — pull requests are expected to be clean.
- Follow the existing package layout: `cmd/` for CLI commands, `internal/` for library code.
- Prefer small, focused functions over large ones.
- Avoid adding dependencies unless they provide significant value — prefer the standard library and existing project dependencies.
- Do not commit generated binaries or API keys.

## Commit conventions

DevPulse uses [Conventional Commits](https://www.conventionalcommits.org/). The changelog in releases is grouped automatically from these prefixes:

- `feat:` — new feature
- `fix:` — bug fix
- `security:` — security hardening
- `perf:` — performance improvement
- `refactor:` — code change that neither fixes a bug nor adds a feature
- `ci:` / `build:` — build and CI changes
- `chore:` — maintenance tasks
- `docs:` — documentation only (excluded from release changelogs)

Example:

```
feat: add --since flag to the resume command
```

## Submitting a pull request

1. Push your branch and open a pull request against `mvp`.
2. Fill in the PR template — describe *what* changed and *why*.
3. Make sure CI (format + build + vet + test) passes.
4. If your PR addresses an issue, link it in the description (e.g. "Closes #123").

> Releases are published from Git tags on `main`. Merging to `mvp` does **not** create a release; changes still flow through `dev` and an approved `main` merge. To release, tag a version from `main`:

```sh
git checkout main
git pull origin main
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z  # triggers GitHub Actions to build and release
```

Verify a Windows archive with SHA-256 before extracting it:

```powershell
Get-FileHash .\devpulse_Windows_x86_64.zip -Algorithm SHA256
Select-String .\checksums.txt devpulse_Windows_x86_64.zip
```

## Reviewing

- Be respectful and constructive. The goal is better code, not winning arguments.
- Suggest, don't demand — unless it's a correctness, security, or licensing issue.
- Thank contributors for their work.

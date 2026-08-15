# DevPulse

Regain the context you lose between coding sessions.

DevPulse is a fast, local-first CLI that reads your Git history and your own plans, and uses an LLM to bring you back up to speed across all your repositories. It turns the first fifteen minutes of "where was I?" into an instant briefing — grounded in your actual code history and stated intentions, not generic productivity advice.

[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/Nishanth1812/devpulse?filename=go.mod)](go.mod)
[![Release](https://img.shields.io/github/v/release/Nishanth1812/devpulse)](https://github.com/Nishanth1812/devpulse/releases/latest)

---

## Why DevPulse

Jumping between multiple repositories means re-establishing context every time you come back. Old commits, WIP messages, half-finished features, plan files that never got connected to what you actually shipped — it all costs time and mental energy.

DevPulse fixes that by answering three questions about your repos:

- **What is in flight right now?**
- **What did I start but not finish?**
- **What should I work on next?**

Every suggestion is grounded in your commit diffs, your `PLAN.md`/`ROADMAP.md` files, and your personal goals file.

---

## Installation

### From source (Go 1.25+)

```sh
go install github.com/Nishanth1812/devpulse@latest
```

### Prebuilt binaries

Download the archive for your platform from the [Releases](https://github.com/Nishanth1812/devpulse/releases/latest) page.

| Platform | Arch | File |
|---|---|---|
| Linux | x86_64 / arm64 / i386 | `devpulse_Linux_*.tar.gz` |
| macOS | Universal / arm64 / x86_64 | `devpulse_Darwin_all.tar.gz` |
| Windows | x86_64 / arm64 / i386 | `devpulse_Windows_*.zip` |

Verify a download against the published checksums:

```sh
sha256sum -c checksums.txt
```

> **Note:** Binary downloads are unsigned. See [SECURITY.md](SECURITY.md) for details.

### Shell completion

```sh
# bash
devpulse completion bash > /etc/bash_completion.d/devpulse

# zsh
devpulse completion zsh > "${fpath[1]}/_devpulse"

# fish
devpulse completion fish > ~/.config/fish/completions/devpulse.fish

# powershell
devpulse completion powershell | Out-String | Invoke-Expression
```

---

## Getting Started

### 1. Add an API key

DevPulse needs either a [Groq](https://console.groq.com) or [Gemini](https://aistudio.google.com) API key to power the LLM commands. Keys are stored in your OS keychain, never in a config file.

```sh
devpulse auth                    # store a Groq API key
devpulse auth -p gemini          # store a Gemini API key
```

You can also set `GROQ_API_KEY` or `GEMINI_API_KEY` as environment variables — useful in CI.

### 2. Register repositories

```sh
devpulse register /path/to/my-repo
```

### 3. Create your goals file

```sh
devpulse init
```

This creates `~/.devpulse/goals.md` with four optional sections that DevPulse reads to prioritise your work:

```
## Now
Things you are actively trying to finish this sprint or week.

## Next
Things that become active once the Now items are done.

## Deadlines
Hard dates. Format: YYYY-MM-DD — description.

## Someday
Projects or ideas you are not touching now but do not want to forget.
```

### 4. Run your first briefing

```sh
devpulse brief
```

---

## Commands

| Command | Description |
|---|---|
| `devpulse brief` | Cross-repo summary of what's in-flight, blocked, and stale (default command) |
| `devpulse resume <repo>` | Deep context recovery for one repo — what you built, what's incomplete, next step |
| `devpulse focus` | Ranks all repos by completion proximity, weighted against your goals |
| `devpulse health` | Per-repo hygiene scan (rule-based, no LLM) |
| `devpulse why <repo> <file>` | File-level archaeology — narrate every significant decision in a file |
| `devpulse commit` | Generate a Conventional Commit message from staged changes |
| `devpulse note add/list` | Attach markdown notes to registered repositories |
| `devpulse register` | Register a repository path |
| `devpulse unregister` | Remove a repository |
| `devpulse list` | List all registered repositories |
| `devpulse init` | Create the goals file scaffold |
| `devpulse auth` | Store an API key in the OS keychain |
| `devpulse config get/set` | Read and update configuration |
| `devpulse doctor` | Run environment diagnostics |
| `devpulse debug collect` | Dump collected repo data as JSON |
| `devpulse version` | Print version and build metadata |

### Repo name matching

Commands that take a repo argument accept **partial names**:

```sh
devpulse resume acm    # resolves to ACM-APP-BACKEND
devpulse resume nut    # resolves to nutrition-tracker
```

If multiple repos match, DevPulse prints a numbered list. Tab-completion is supported for all repo-argument commands.

---

## Configuration

Config lives at `~/.devpulse/config.toml`. It stores only non-sensitive settings — registered repository paths, model preference, and cache duration. API keys are never stored here.

```sh
devpulse config get model.fast
devpulse config set model.fast gemini-2.5-flash-lite
devpulse config set fuzzy.threshold 50
```

The fast defaults are Groq `openai/gpt-oss-20b` and Gemini
`gemini-2.5-flash-lite`; the deep Gemini default is `gemini-2.5-flash`.
Cached responses expire after 24 hours by default.

`brief` accepts zero or one repository argument:

```sh
devpulse brief             # portfolio result for all registered repositories
devpulse brief acm         # focused result for one partial repository name
devpulse brief --help
```

---

## Privacy & Security

DevPulse is **local-first and telemetry-free**:

- Only commit messages, compressed diffs, plan summaries, and your goals file are sent to the AI provider — never full file contents.
- API keys stay in your OS keychain.
- All prompts are scanned for secrets (API keys, tokens, JWTs, PEM keys) and redacted before sending.
- `--dry-run` prints exactly what would be sent without calling any API.
- `--redact-diff` strips all diff content for low-trust codebases.

See [SECURITY.md](SECURITY.md) for the full data-flow and secret-handling documentation.

---

## Development

Get the source and set up the toolchain:

```sh
git clone https://github.com/Nishanth1812/devpulse.git
cd devpulse
go build ./...
```

Run the test suite and vet:

```sh
gofmt -l .                 # must print nothing
go test ./...
go vet ./...
go build ./...
go mod verify
```

Command tests use temporary fixture repositories and fake providers; they do
not require a real API key, network access, home directory, or keychain.
Supported Linux runners also run `go test -race ./...`; Windows CI covers the
same formatting, vet, test, build, and clean-workspace checks.

DevPulse uses [Conventional Commits](https://www.conventionalcommits.org/) to drive the automatically-grouped [release changelog](.goreleaser.yaml).

---

## Contributing

Contributions are welcome — bug fixes, docs, features, ideas. See [CONTRIBUTING.md](CONTRIBUTING.md) for the setup guide, issue triage, and PR process.

Please review our [Code of Conduct](CODE_OF_CONDUCT.md) and report security issues via [SECURITY.md](SECURITY.md) rather than public issues.

---

## License

This project is licensed under the **GNU General Public License v3.0**. See [LICENSE](LICENSE) for the full text.

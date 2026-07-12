# Security

## Data That Leaves Your Machine

Only the following data is sent to the AI provider (Groq or Gemini):

- **Commit messages** — subject lines and bodies from recent commits
- **Compressed diffs** — changed lines only, capped at 80 lines per file, with blank lines, whitespace-only changes, and import-only changes stripped; binary files replaced with a placeholder
- **Plan file summaries** — contents of PLAN.md, ROADMAP.md, TODO.md, etc., with completed checkboxes stripped and capped at 300 lines
- **Goals file** — the Now, Next, Deadlines, and Someday sections from `~/.devpulse/goals.md`

Full file contents are never sent. Only lines that actually changed (in diffs) or headings and active items (in plan files) are included.

## Data That Stays On Your Machine

- Repository paths and names — stored only in `~/.devpulse/config.toml`
- API keys — stored in your OS keychain (see API Key Storage below), never written to any file on disk
- Cached responses — stored in `~/.devpulse/cache/` when cache is enabled
- Logs — written to `~/.devpulse/logs/`, auto-purged after 7 days
- Notes — stored in `~/.devpulse/notes/` as plain markdown files

## No Telemetry

DevPulse never phones home, checks for updates silently, or sends usage data anywhere. The only outbound network calls are:

1. To the AI provider (Groq or Gemini) when an LLM-powered command is run
2. Optionally to the GitHub API, if configured

## API Key Storage

API keys are stored in your operating system's credential manager via `zalando/go-keyring`:

- **Windows**: Windows Credential Manager
- **macOS**: Keychain
- **Linux**: Secret Service (D-Bus)

### Setting a key

    devpulse auth                  # store a Groq API key
    devpulse auth --provider gemini   # store a Gemini API key

You will be prompted to enter the key. It is not echoed to the screen.

### Environment variable fallback

If a key is not found in the keychain, DevPulse checks the following environment variables before falling back to an error:

- `GROQ_API_KEY` — for Groq
- `GEMINI_API_KEY` — for Gemini

This is useful for CI/CD environments where interactive keychain access is not available.

### Key rotation

To rotate a key, simply run the corresponding auth command again:

    devpulse auth
    devpulse auth --provider gemini

This overwrites the stored key in your keychain.

### Revoking access

Remove the stored key from your keychain using your operating system's credential manager, or overwrite it with an invalid value:

    # On macOS
    security delete-generic-password -s devpulse -a groq-api-key

    # On Windows (PowerShell as admin)
    [CredentialManager]::RemoveCredential("devpulse/groq-api-key")

## Prompt Inspection

### --dry-run

Before sending any data to an AI provider, you can inspect exactly what will be sent:

    devpulse brief my-repo --dry-run
    devpulse commit --dry-run

This prints the full prompt and an estimated token count without making any API call.

### --redact-diff

If you want to use the LLM-powered features of DevPulse but do not want diff content to leave your machine:

    devpulse brief my-repo --redact-diff

In this mode, only commit messages and plan file summaries are sent. Diffs are not collected at all. This mode cannot be used with `devpulse commit`, since commit message generation inherently requires seeing the staged diff.

## Sensitive Content Redaction

Before any prompt is sent to an AI provider, DevPulse scans it for patterns that look like secrets:

- PEM private key headers (`-----BEGIN RSA PRIVATE KEY-----`)
- GitHub tokens (`ghp_`, `gho_`, `ghu_`, `ghs_`, `ghr_`)
- OpenAI/Groq API keys (`sk-...`)
- AWS access keys (`AKIA...`)
- JWTs (`eyJ...`)
- Slack tokens (`xox*-...`)
- Inline credentials (`api_key: "..."`, `secret = "..."`)

If any of these patterns are detected, the matching content is replaced with a redaction placeholder before the prompt is sent, and a warning is printed to stderr.

## Log Sanitization

Log files written to `~/.devpulse/logs/` are sanitized to remove the same secret patterns before they are written to disk. Log files have `0600` permissions and are automatically purged after 7 days.

## File Permissions

| Path | Permission |
|---|---|
| `~/.devpulse/` | 0700 (owner only) |
| `~/.devpulse/config.toml` | 0600 (owner only) |
| `~/.devpulse/cache/` | 0700 (owner only) |
| `~/.devpulse/logs/` | 0700 (owner only) |
| `~/.devpulse/notes/` | 0700 (owner only) |

## Reporting a Vulnerability

If you discover a security vulnerability in DevPulse, please open a GitHub Issue at https://github.com/Nishanth1812/devpulse/issues.

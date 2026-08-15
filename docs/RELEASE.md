# Technical Release Checklist

This document covers the reproducible technical release path for DevPulse. It
does not cover launch communications, marketing, or product announcements.

## Required starting state

- Work from a clean checkout of `main` after the approved `mvp` and `dev` merges.
- Confirm the intended version is not already tagged.
- Do not include API keys, local configuration, cache files, logs, or generated
  binaries in the commit.

## Local verification

Run the following from the repository root:

```sh
gofmt -l .                 # empty output required
go vet ./...
go test ./...
go test -race ./...
go build ./...
go mod verify
```

Run the deterministic command suite explicitly:

```sh
go test ./cmd ./internal/... -v
```

The fixture and golden tests must remain network-free and must not use a real
home directory, keychain, provider, or API key.

## GoReleaser verification

Validate the configuration and create a clean snapshot:

```sh
goreleaser check
goreleaser release --snapshot --clean
```

Inspect every archive in `dist/`. Each archive must contain the `devpulse`
binary (`devpulse.exe` on Windows) plus `README.md`, `SECURITY.md`,
`CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, and the applicable license files.

Verify checksums on POSIX:

```sh
cd dist
sha256sum -c checksums.txt
```

Verify a Windows archive in PowerShell:

```powershell
Get-FileHash .\devpulse_Windows_x86_64.zip -Algorithm SHA256
Select-String .\checksums.txt devpulse_Windows_x86_64.zip
```

Extract each platform archive into a clean temporary directory and run:

```sh
devpulse version
devpulse --help
devpulse doctor --no-color
```

The clean-workspace path must not require a provider key. Exercise `brief
--dry-run` with a fixture repository to confirm prompt construction without a
network call.

## CI and publication sequence

1. Push the verified development branch and wait for required Linux and
   Windows CI checks.
2. Merge to `dev`, rerun the release gate, and merge the approved commit to
   `main`.
3. Confirm the tag commit equals `origin/main`.
4. Create and push an annotated semantic-version tag:

   ```sh
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

5. The release workflow verifies the tag-on-main guard, formatting, vet, tests,
   module integrity, GoReleaser configuration, archives, and checksums.
6. Inspect the published release assets and confirm the release is not a draft
   or prerelease.

## Failure and rollback

Before publication, fix the branch and create a new candidate tag. If a tag or
release has already been published, remove the bad release/tag only after
confirming the exact target, then publish a corrected patch version from a
clean `main` commit. Never move an existing public tag silently.

Record the final workflow URL, commit SHA, tag, artifact list, checksum result,
and any environment limitations in the release evidence for the version.

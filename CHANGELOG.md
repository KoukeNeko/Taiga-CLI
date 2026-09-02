# Changelog

***English** · [繁體中文](CHANGELOG.zh-TW.md)*

Notable changes to this project are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The release workflow publishes the section matching the tag as the GitHub Release notes, alongside the same section from [CHANGELOG.zh-TW.md](CHANGELOG.zh-TW.md), so both files must carry every released version.

Release notes render with hard line breaks, so each paragraph and bullet stays on one line here even when that line is long. Wrapping them would show up as broken sentences on the release page.

## [Unreleased]

## [0.2.3] - 2026-09-02

Tooling and documentation only. The `aihki` binary is unchanged from 0.2.2.

### Added

- The install script now writes shell completions instead of only printing the command that generates them. It writes into a standard directory that already exists and names each file it wrote; it never creates such a directory, since that would leave a stray folder in the home directory of someone who does not use that shell. Windows still prints the hint, because installing PowerShell completion means editing a profile rather than dropping in a file.
- The uninstaller removes exactly those files, skipping any that does not look generated so a hand-written completion of the same name survives.

### Fixed

- CI now runs the installer and uninstaller as a round trip on all three operating systems. An earlier release claimed this coverage, but the change that was meant to add it silently matched nothing, so the job had only ever run the installer.

## [0.2.2] - 2026-09-02

### Fixed

- Errors that tell you what to run next named the pre-rename command, so a missing credential suggested `taiga auth login` and `TAIGA_TOKEN` rather than `aihki auth login` and `AIHKI_TOKEN`. Seventeen such messages across eleven files now name commands that exist.

### Changed

- The end-to-end suite authenticates with `AIHKI_TOKEN`, which it had never exercised, and gained a case proving `TAIGA_TOKEN` still authenticates a real binary. A fallback covered only by a unit test is a fallback nobody has run.

## [0.2.1] - 2026-09-02

Tooling and documentation only. The `aihki` binary is functionally unchanged from 0.2.0.

### Added

- Uninstall scripts for macOS, Linux and Windows, covering the binary, the PATH entry on Windows, and optionally the configuration directory and the credential in the OS keyring. Removal is conservative by default because uninstalling is often one step of an upgrade: `--purge` (`-Purge` on Windows) opts into removing settings and credentials, including whatever the pre-rename name left behind, and `--dry-run` reports what would go without changing anything.
- The POSIX uninstaller detects a Homebrew installation and hands you back to `brew uninstall` rather than deleting files Homebrew tracks. Both scripts print where a `project use --local` pin lives, since no uninstaller can find one from outside the repository holding it.
- A hero image on the repository front page.

### Changed

- INSTALL.md documents uninstalling in both languages.
- CI runs the installer and uninstaller as a round trip on all three operating systems, so the uninstaller has to remove exactly what the installer wrote.

## [0.2.0] - 2026-09-02

### Renamed to Aihki

The project is now **Aihki**, and the binary is `aihki`. Taiga ships under MPL-2.0, whose section 2.3 grants no rights in its trademarks or logos, and the Taiga maintainers have previously required a spin-off to drop the name. Leading with someone else's mark as this project's own identifier was not something the licence ever supported, so the name is now used only to describe what this client talks to.

Nothing about the Taiga integration changes. What changes is the identity of this tool.

**Existing installations keep working.** Every stored setting is migrated on first use rather than being abandoned:

- A credential in the old `taiga-cli` keyring service is adopted into `aihki` the first time it is read, so you are not logged out.
- A config file under `taiga-cli/` is read when `aihki/` has none, and the next save writes to the new location.
- A repository pinned with `taiga.profile` or `taiga.project` in `.git/config` still resolves; `aihki.*` takes precedence when both exist.
- `TAIGA_TOKEN`, `TAIGA_PROFILE`, `TAIGA_API_URL` and `TAIGA_PROJECT` are still honoured. `AIHKI_*` wins when both are set.

**What you do need to change:** `brew install koukeneko/tap/aihki` replaces the old formula, release archives are now named `aihki_<version>_<os>_<arch>`, and completion files are `aihki.bash`, `_aihki`, `aihki.fish` and `aihki.ps1`.

## [0.1.0] - 2026-09-02

First stable release. A Taiga 6 command line tool written in Go, offering readable terminal output and a stable JSON contract for shells, CI, and agents.

### Install

```sh
brew install koukeneko/tap/aihki
```

Or download the archive for your platform below, verify it against `SHA256SUMS`, and extract it.

### Features

- Full day-to-day operations for projects, epics, user stories, tasks, issues, sprints, and wiki pages.
- Members and role permissions, webhooks, custom fields, eight families of workflow metadata, swimlanes, tags, and due-date presets.
- Cross-project epic ↔ story links, plus watching, voting, comment editing, and history.
- Project export and import, ownership transfer, CSV export, and a third-party importer gateway.
- Search, timeline, and statistics for backlog velocity, issue trends, member contributions, and sprint burndown.
- Streaming attachment upload and download, with size and SHA-1 verified on the way in.
- Bulk creation for epics, stories, tasks, and issues, and bulk move and reorder for stories, tasks, and issues.

### Automation surface

- `--json` emits an envelope carrying `meta.contract`, currently contract 1.
- `--fields` selects fields, and `aihki schema <command>` returns JSON Schema with safety and idempotency annotations.
- Exit codes are partitioned by failure kind, and `--dry-run` resolves fully while guaranteeing that no write request is sent.
- Completions for Bash, Zsh, Fish, and PowerShell, with a five-minute cache and stale-on-error fallback when offline.

### Security

- Passwords are never accepted on the command line, and tokens live in the OS keyring rather than a config file.
- Webhook secrets, application token auth codes, and ownership transfer tokens never appear in any output.
- Only idempotent GETs are retried; a write whose outcome is unknown reports `ambiguous_commit` and asks for confirmation.
- Optimistic concurrency conflicts stop the command instead of merging or overwriting.
- Attachment downloads never send the API bearer token to the media URL.

### Release artifacts

- macOS binaries are signed with a Developer ID certificate and notarized by Apple, so a download runs without extra steps.
- Linux and Windows archives are byte-for-byte reproducible: the same source, toolchain, version, commit, and epoch produce identical bytes. macOS cannot reproduce because notarization requires a secure timestamp.
- Every release ships `SHA256SUMS` and an SPDX 2.3 SBOM.

### Known limitations

- Taiga 6's REST API exposes `archived_code` as read-only and offers no archive action, so `project archive|unarchive` reports `unsupported_capability` and points at a site administrator.
- A notarization ticket cannot be stapled to a bare executable, so Gatekeeper resolves it online and a fully offline first run can still be blocked.
- `stats system` exists only when a site enables Taiga's `STATS_ENABLED`; otherwise it reports `not_found`.

### Compatibility

Verified against Taiga 6.10.2 through a full Docker E2E run against a pinned image digest. Supports macOS, Linux, and Windows on `amd64` and `arm64`.

[Unreleased]: https://github.com/KoukeNeko/aihki/compare/v0.2.3...HEAD
[0.2.3]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.3
[0.2.2]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.2
[0.2.1]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.1
[0.2.0]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.0
[0.1.0]: https://github.com/KoukeNeko/aihki/releases/tag/v0.1.0

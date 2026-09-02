# Changelog

***English** · [繁體中文](CHANGELOG.zh-TW.md)*

Notable changes to this project are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The release workflow publishes the section matching the tag as the GitHub Release notes, alongside the same section from [CHANGELOG.zh-TW.md](CHANGELOG.zh-TW.md), so both files must carry every released version.

Release notes render with hard line breaks, so each paragraph and bullet stays on one line here even when that line is long. Wrapping them would show up as broken sentences on the release page.

## [Unreleased]

## [0.1.0] - 2026-09-02

First stable release. A Taiga 6 command line tool written in Go, offering readable terminal output and a stable JSON contract for shells, CI, and agents.

### Install

```sh
brew install koukeneko/tap/taiga
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
- `--fields` selects fields, and `taiga schema <command>` returns JSON Schema with safety and idempotency annotations.
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

[Unreleased]: https://github.com/KoukeNeko/Taiga-CLI/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/KoukeNeko/Taiga-CLI/releases/tag/v0.1.0

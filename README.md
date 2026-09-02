<h1 align="center">Taiga CLI</h1>

<p align="center">
  <strong>Bring Taiga to the command line.</strong><br>
  Readable terminal output for people, and a stable JSON contract for shells, CI, and agents.
</p>

<p align="center">
  <a href="https://github.com/KoukeNeko/Taiga-CLI/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/KoukeNeko/Taiga-CLI?style=for-the-badge&logo=github&label=RELEASE&color=2196F3"></a>
  <a href="https://github.com/KoukeNeko/Taiga-CLI/releases"><img alt="Release downloads" src="https://img.shields.io/github/downloads/KoukeNeko/Taiga-CLI/total?style=for-the-badge&logo=github&label=DOWNLOADS&color=4CAF50"></a>
  <a href="https://github.com/KoukeNeko/Taiga-CLI/actions/workflows/ci.yml"><img alt="CI status" src="https://img.shields.io/github/actions/workflow/status/KoukeNeko/Taiga-CLI/ci.yml?branch=main&style=for-the-badge&logo=githubactions&logoColor=white&label=CI"></a>
  <a href="COMPATIBILITY.md"><img alt="Verified against Taiga 6.10.2" src="https://img.shields.io/badge/TAIGA-6.10.2_VERIFIED-00A5A5?style=for-the-badge"></a>
  <a href="LICENSE"><img alt="MIT licence" src="https://img.shields.io/badge/LICENSE-MIT-4CAF50?style=for-the-badge&logo=github"></a>
</p>

<p align="center">
  <strong>English</strong> · <a href="README.zh-TW.md">繁體中文</a>
</p>

<p align="center">
  <a href="INSTALL.md">Install</a>
  · <a href="#getting-started">Getting started</a>
  · <a href="https://github.com/KoukeNeko/Taiga-CLI/wiki">Handbook</a>
  · <a href="CHANGELOG.md">Changelog</a>
  · <a href="COMPATIBILITY.md">Compatibility</a>
</p>

```sh
taiga issue list
taiga issue create --subject "Fix token refresh" --type Bug
taiga issue close 42 --status Closed

# The same data, for a script, a CI job, or an agent
taiga issue view 42 --json --fields ref,subject,status,version
```

```json
{"data":{"ref":42,"status":"Closed","subject":"Fix token refresh","version":8},"meta":{"contract":1}}
```

Taiga CLI lets you drive [Taiga 6](https://taiga.io/) projects, agile workflows, and wikis without
leaving the terminal. It discovers the API from the frontend's `conf.json`, including sites deployed
under a `/taiga/` subpath, and hands the token to your operating system keyring instead of writing it
into a config file.

**One command serves both people and programs.** Run it directly and you get aligned tables; add
`--json` and you get a versioned contract, backed by fixed exit codes and JSON Schema descriptors, so
a shell script, a CI job, or an LLM agent can drive it safely. The output format never changes just
because it was piped — if you want JSON, you ask for it.

**Writes behave predictably.** Every mutation goes through Taiga's optimistic concurrency control and
stops on conflict instead of overwriting. Only idempotent GETs are retried. If a connection drops
mid-write, the CLI reports `ambiguous_commit` and asks you to verify rather than blindly resending.

## What it does

### The whole Taiga workflow

Day-to-day operations for projects, epics, user stories, tasks, issues, sprints, and wiki pages are
all here: list, view, create, edit, assign, comment, close, delete. Plus members and role
permissions, webhooks, custom fields, eight families of workflow metadata, swimlanes, tags, due-date
presets, and cross-project epic ↔ story links.

Work items accept a bare ref, a `project#ref` pair, or a pasted Taiga URL — all three work:

```text
42
example-project#42
https://taiga.example.com/taiga/project/example-project/issue/42
```

### A stable surface for automation

`--json` emits a `meta.contract` version, `--fields` selects columns, and `taiga schema <command>`
returns that command's input and output JSON Schema along with `safety` and `idempotency`
annotations — enough for an agent to decide whether a command may run unattended. Exit codes are
partitioned by failure kind, and `--dry-run` resolves and displays the mutation it would send while
guaranteeing that no write request leaves the process.

### Hard to break things with

Deleting work items and metadata is verified by reading the target back. Attachment and CSV downloads
stream to a `0600` temporary file, verify their digests, and land atomically without clobbering an
existing file. Destructive actions require an explicit `--yes` when there is no terminal. Webhook
secrets, application-token auth codes, and ownership-transfer tokens never appear in any output,
including dry runs.

### Many sites, many projects

Profiles switch between Taiga sites, each remembering its own API URL and default project. You can
also pin a profile and project to a single Git repository, stored in `.git/config` so it is never
committed:

```sh
taiga project use example-project --local
```

### Diagnosable when something breaks

`taiga doctor` checks frontend discovery, the API, authentication, and the default project one by
one. When you need help, `taiga doctor bundle` produces a report you can share without worrying:
version information, presence booleans, and status codes only — no URLs, usernames, project names,
or credentials — created locally and never uploaded.

## Getting started

1. **Install.** Homebrew, on macOS and Linux:

   ```sh
   brew install koukeneko/tap/taiga
   ```

   Or the install script, which verifies the download against the release checksums before it
   installs anything:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/KoukeNeko/Taiga-CLI/main/scripts/install.sh | sh
   ```

   ```powershell
   irm https://raw.githubusercontent.com/KoukeNeko/Taiga-CLI/main/scripts/install.ps1 | iex
   ```

   Release archives, manual checksum verification, and building from source are covered in
   [INSTALL.md](INSTALL.md). On Windows, open a new terminal afterwards to pick up the PATH change.

2. **Log in.** The token goes to the OS keyring:

   ```sh
   taiga auth login --host https://taiga.example.com/taiga/ --profile company
   ```

3. **Pick a project:**

   ```sh
   taiga project list
   taiga project use example-project
   ```

4. **Start working:**

   ```sh
   taiga issue list
   taiga issue create --subject "Fix token refresh" --type Bug
   taiga issue assign 42 --to alice
   taiga issue close 42 --status Closed
   ```

5. **Wire up automation:**

   ```sh
   taiga issue view 42 --json --fields id,ref,subject,status,version --no-input
   ```

The full command reference, flag documentation, and per-subsystem behaviour live in the
[handbook wiki](https://github.com/KoukeNeko/Taiga-CLI/wiki).

## Compatibility

- Taiga 6.10.2, verified by Docker E2E against a pinned image digest
- macOS, Linux, and Windows on `amd64` and `arm64`, built as pure Go (`CGO_ENABLED=0`)
- Password login, existing bearer tokens, and refresh-token rotation

The detailed matrix and known limits are in [COMPATIBILITY.md](COMPATIBILITY.md).

---

## Technical reference

### Setting resolution

General settings live in the operating system's user config directory. Tokens are never written
there:

```toml
current_profile = "company"

[profiles.company]
api_url = "https://taiga.example.com/taiga/api/v1/"
project = "example-project"
```

Resolution order, highest first:

```text
command flag
→ TAIGA_PROFILE / TAIGA_API_URL / TAIGA_PROJECT / TAIGA_TOKEN
→ Git-local taiga.profile / taiga.project
→ current profile
→ safe defaults
```

### JSON contract

Successful data goes to stdout and errors go to stderr, never mixed into one stream. Single records
use `data`, lists use `items` and `page`, and both carry `meta.contract`:

```json
{
  "data": { "id": 123, "ref": 42, "subject": "Fix token refresh", "version": 7 },
  "meta": { "contract": 1 }
}
```

Within one contract version only optional fields are added. Removing a field, renaming it, or
changing the type of an existing one requires a version bump and migration notes in the release.

| Exit code | Meaning |
| ---: | --- |
| 0 | success |
| 2 | usage / schema |
| 3 | authentication |
| 4 | forbidden |
| 5 | not found |
| 6 | OCC conflict |
| 7 | validation / ambiguity |
| 8 | throttled |
| 9 | transport / upstream |
| 10 | confirmation required |
| 11 | ambiguous commit |

### Security principles

- Passwords are never accepted on the command line
- Authorization headers, passwords, and tokens never appear in verbose logs
- Only GETs are retried automatically, with a bounded count; POST and PATCH are never resent blindly
- A write whose outcome is unknown reports `ambiguous_commit` instead of retrying
- OCC conflicts are never auto-merged or overwritten
- Attachment downloads never send the API bearer token to the media URL
- TLS verification is always on

### Development and testing

The fast loop, no Docker required:

```sh
make test
make test-race
make lint
```

Integration tests against a real Taiga server:

```sh
make test-integration
```

The harness uses a dedicated `taiga-cli-e2e` Compose project on `localhost:19000`, creates its own
throwaway account, project, and issues, and tears down only its own containers and volumes — it never
touches a Taiga instance you use day to day.

Rebuilding cross-platform release artifacts:

```sh
make release \
  VERSION=v0.1.0 \
  COMMIT="$(git rev-parse HEAD)" \
  SOURCE_DATE_EPOCH="$(git show -s --format=%ct HEAD)"
```

The same source, Go toolchain, version, commit, and epoch produce byte-identical Linux and Windows
archives. The maintainer release process is in [RELEASING.md](RELEASING.md). macOS archives are the exception: notarization requires a secure timestamp from Apple, so a
Developer ID signature can never reproduce. Their contents are otherwise built identically.

<p>
  <img alt="Cobra" src="https://img.shields.io/badge/COBRA-CLI-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img alt="Reproducible builds" src="https://img.shields.io/badge/BUILDS-REPRODUCIBLE-4CAF50?style=for-the-badge">
  <img alt="SPDX 2.3 SBOM" src="https://img.shields.io/badge/SBOM-SPDX_2.3-2196F3?style=for-the-badge">
</p>

## License

[MIT](LICENSE) © KoukeNeko

# Installing and upgrading

***English** · [繁體中文](INSTALL.zh-TW.md)*

## Install script

macOS and Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/KoukeNeko/Taiga-CLI/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/KoukeNeko/Taiga-CLI/main/scripts/install.ps1 | iex
```

The script detects your platform, resolves the latest **stable** release, downloads `SHA256SUMS`, and
**verifies the digest before installing**. A mismatch, or an archive missing from `SHA256SUMS`, aborts
and leaves any existing installation untouched. The default location is `~/.local/bin`, or
`%LOCALAPPDATA%\Programs\taiga` on Windows, where the directory is added to your user PATH.

On Windows, **open a new terminal** afterwards for the PATH change to take effect.

Choosing a version or location:

```sh
TAIGA_VERSION=v0.1.0 TAIGA_INSTALL_DIR=/usr/local/bin sh install.sh
```

Passing parameters on Windows means saving the script to a file first, and the PowerShell execution
policy blocks a `.ps1` downloaded from the internet by default. The `irm | iex` form above is
unaffected because it runs a string rather than a file, but a saved script needs an explicit
exemption:

```powershell
irm https://raw.githubusercontent.com/KoukeNeko/Taiga-CLI/main/scripts/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Version v0.1.0 -InstallDir C:\Tools\taiga
```

To avoid repeating `-ExecutionPolicy Bypass`, run `Unblock-File .\install.ps1` once to clear the
download mark.

## Homebrew (macOS and Linux)

```sh
brew install koukeneko/tap/taiga
```

Upgrading:

```sh
brew upgrade taiga
```

The tap tracks **stable releases only** and never installs a pre-release. The formula installs the
binary from the release archive, so macOS users get the signed and notarized executable, along with
Bash, Zsh, and Fish completions.

To try a pre-release, download the archive manually as described in the next section.

## Official release archives

Download the archive for your platform from the GitHub Release, together with `SHA256SUMS` from the
same version:

| Operating system | Architecture | Archive |
| --- | --- | --- |
| macOS | Intel | `taiga_<version>_darwin_amd64.tar.gz` |
| macOS | Apple silicon | `taiga_<version>_darwin_arm64.tar.gz` |
| Linux | x86-64 | `taiga_<version>_linux_amd64.tar.gz` |
| Linux | ARM64 | `taiga_<version>_linux_arm64.tar.gz` |
| Windows | x86-64 | `taiga_<version>_windows_amd64.zip` |
| Windows | ARM64 | `taiga_<version>_windows_arm64.zip` |

Verifying on Linux:

```sh
sha256sum --check SHA256SUMS
```

Verifying on macOS:

```sh
shasum -a 256 --check SHA256SUMS
```

On Windows PowerShell, use `Get-FileHash -Algorithm SHA256 <archive>` and compare against
`SHA256SUMS`. Once verified, extract the archive and move `taiga` (`taiga.exe` on Windows) into a
directory on your `PATH`. Every archive also carries the READMEs, the compatibility matrix, an SPDX
SBOM, and completions for four shells.

## macOS Gatekeeper

Released macOS binaries are signed with a Developer ID certificate and notarized by Apple, so they run
without any extra step. You can confirm that yourself:

```sh
codesign --verify --strict --verbose=2 ./taiga
spctl -a -vvv -t install ./taiga
```

`spctl` reporting `accepted` with `source=Notarized Developer ID` is the expected result.

A notarization ticket cannot be stapled to a bare executable, because `stapler` supports only `.app`,
`.dmg`, and `.pkg`. Gatekeeper therefore resolves it **online**. A first run with no network at all
can still be blocked; reconnect and run it again, or clear the quarantine attribute:

```sh
xattr -d com.apple.quarantine ./taiga
```

Files fetched with `curl` or `wget` are never quarantined, so this does not arise there. Either way,
verify the download against `SHA256SUMS` before running it.

## Shell completion

The `completions/` directory in each archive contains:

- Bash: `taiga.bash`
- Zsh: `_taiga`
- Fish: `taiga.fish`
- PowerShell: `taiga.ps1`

They can also be generated after installation:

```sh
taiga completion bash
taiga completion zsh
taiga completion fish
taiga completion powershell
```

## Upgrading

1. Read that version's release notes and [COMPATIBILITY.md](COMPATIBILITY.md).
2. Download and verify the new archive.
3. Replace the old binary. Config files and OS keyring credentials do not need to move.
4. Run `taiga version --json` to confirm the version, commit, and platform.
5. Run `taiga doctor --json` to confirm the API, authentication, and default project.

Downgrading is the same in reverse: put back a verified older binary. If the release notes mention a
configuration migration, back up the Taiga CLI config file in your operating system's user config
directory first.

## Installing from source

Requires Go 1.25 or newer:

```sh
make install PREFIX="$HOME/.local"
```

A source installation reports version `dev`. Real release metadata is injected only by the
reproducible packaging pipeline.

# 安裝與升級

*[English](INSTALL.md) · **繁體中文***

## 安裝腳本

macOS 與 Linux：

```sh
curl -fsSL https://raw.githubusercontent.com/KoukeNeko/aihki/main/scripts/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/KoukeNeko/aihki/main/scripts/install.ps1 | iex
```

腳本會偵測平台、抓取最新**正式版**、下載 `SHA256SUMS` 並**核對雜湊後才安裝** —— 雜湊不符或檔案未列於
`SHA256SUMS` 都會中止並保留原有安裝。預設安裝位置為 `~/.local/bin`（Windows 為
`%LOCALAPPDATA%\Programs\aihki`，並自動加入使用者 PATH）。

Windows 安裝後需要**開一個新的終端機**，使用者 PATH 的變更才會生效。

指定版本或安裝位置：

```sh
AIHKI_VERSION=v0.1.0 AIHKI_INSTALL_DIR=/usr/local/bin sh install.sh
```

Windows 要傳參數就必須先把腳本存成檔案，而 PowerShell 的執行原則預設會封鎖從網路下載的 `.ps1`。上面
`irm | iex` 的寫法不受影響（它執行的是字串而非檔案），但存檔後執行需要明確放行：

```powershell
irm https://raw.githubusercontent.com/KoukeNeko/aihki/main/scripts/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Version v0.1.0 -InstallDir C:\Tools\aihki
```

若不想每次都加 `-ExecutionPolicy Bypass`，也可以先用 `Unblock-File .\install.ps1` 移除下載標記。

## Homebrew（macOS 與 Linux）

```sh
brew install koukeneko/tap/aihki
```

升級：

```sh
brew upgrade aihki
```

Tap 只追蹤**正式版**，不會安裝 pre-release。Formula 安裝的是 release archive 中的 binary，因此 macOS
使用者得到的就是已簽署並 notarize 的執行檔，同時會一併安裝 Bash、Zsh 與 Fish 的 completion。

要試用 pre-release 請依下一節手動下載 archive。

## 官方 release archive

從 GitHub Release 下載符合平台的 archive，以及同一版本的 `SHA256SUMS`：

| 作業系統 | 架構 | Archive |
| --- | --- | --- |
| macOS | Intel | `aihki_<version>_darwin_amd64.tar.gz` |
| macOS | Apple silicon | `aihki_<version>_darwin_arm64.tar.gz` |
| Linux | x86-64 | `aihki_<version>_linux_amd64.tar.gz` |
| Linux | ARM64 | `aihki_<version>_linux_arm64.tar.gz` |
| Windows | x86-64 | `aihki_<version>_windows_amd64.zip` |
| Windows | ARM64 | `aihki_<version>_windows_arm64.zip` |

Linux 驗證：

```sh
sha256sum --check SHA256SUMS
```

macOS 驗證：

```sh
shasum -a 256 --check SHA256SUMS
```

Windows PowerShell 可用 `Get-FileHash -Algorithm SHA256 <archive>`，並與 `SHA256SUMS` 對照。驗證後解壓縮，將 `aihki`（Windows 為 `aihki.exe`）移到 `PATH` 中的目錄。每個 archive 也包含 README、相容性文件、SPDX SBOM 與四種 shell completion。

## macOS Gatekeeper

Release 的 macOS binary 已用 Developer ID 憑證簽署並通過 Apple notarization，正常情況下直接執行即可，
不需要任何額外步驟。可自行確認：

```sh
codesign --verify --strict --verbose=2 ./aihki
spctl -a -vvv -t install ./aihki
```

`spctl` 顯示 `accepted` 且 `source=Notarized Developer ID` 即為正常。

Notarization ticket 無法 staple 到裸執行檔（`stapler` 只支援 `.app`、`.dmg`、`.pkg`），因此 Gatekeeper 會
**線上**查驗。若首次執行時完全沒有網路，仍可能被擋；連上網路後再執行一次即可，或移除隔離屬性：

```sh
xattr -d com.apple.quarantine ./aihki
```

以 `curl` 或 `wget` 下載的檔案不會被加上隔離屬性，本來就不會遇到這個情況。無論哪種方式，都應先用
`SHA256SUMS` 驗證檔案完整性再執行。

## Shell completion

Archive 的 `completions/` 包含：

- Bash：`aihki.bash`
- Zsh：`_aihki`
- Fish：`aihki.fish`
- PowerShell：`aihki.ps1`

也可在安裝後動態產生：

```sh
aihki completion bash
aihki completion zsh
aihki completion fish
aihki completion powershell
```

## 升級

1. 先閱讀該版本 Release Notes 與 [COMPATIBILITY.zh-TW.md](COMPATIBILITY.zh-TW.md)。
2. 下載並驗證新 archive。
3. 以新 binary 取代舊 binary；設定檔與 OS keyring credential 不需搬移。
4. 執行 `aihki version --json` 確認版本、commit 與平台。
5. 執行 `aihki doctor --json` 確認 API、authentication 與預設 Project。

降級時同樣只需換回已驗證的舊 binary。若 Release Notes 標示設定 migration，應先備份作業系統使用者設定目錄中的 Aihki 設定檔。

## 從原始碼安裝

需要 Go 1.25 或更新版本：

```sh
make install PREFIX="$HOME/.local"
```

原始碼安裝預設顯示 `dev` 版本。正式 release metadata 只由可重現 packaging 流程注入。

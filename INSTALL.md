# 安裝與升級

## 官方 release archive

從 GitHub Release 下載符合平台的 archive，以及同一版本的 `SHA256SUMS`：

| 作業系統 | 架構 | Archive |
| --- | --- | --- |
| macOS | Intel | `taiga_<version>_darwin_amd64.tar.gz` |
| macOS | Apple silicon | `taiga_<version>_darwin_arm64.tar.gz` |
| Linux | x86-64 | `taiga_<version>_linux_amd64.tar.gz` |
| Linux | ARM64 | `taiga_<version>_linux_arm64.tar.gz` |
| Windows | x86-64 | `taiga_<version>_windows_amd64.zip` |
| Windows | ARM64 | `taiga_<version>_windows_arm64.zip` |

Linux 驗證：

```sh
sha256sum --check SHA256SUMS
```

macOS 驗證：

```sh
shasum -a 256 --check SHA256SUMS
```

Windows PowerShell 可用 `Get-FileHash -Algorithm SHA256 <archive>`，並與 `SHA256SUMS` 對照。驗證後解壓縮，將 `taiga`（Windows 為 `taiga.exe`）移到 `PATH` 中的目錄。每個 archive 也包含 README、相容性文件、SPDX SBOM 與四種 shell completion。

## macOS Gatekeeper

Release binary 目前**未經 Apple 簽署與 notarization**。以瀏覽器下載 archive 時，macOS 會加上
`com.apple.quarantine` 屬性，解壓後執行會被 Gatekeeper 擋下並顯示無法驗證開發者。

移除隔離屬性後即可執行：

```sh
xattr -d com.apple.quarantine ./taiga
```

以 `curl` 或 `wget` 下載則不會被加上隔離屬性，不需要這個步驟：

```sh
curl -fLO https://github.com/KoukeNeko/Taiga-CLI/releases/download/<version>/taiga_<version>_darwin_arm64.tar.gz
```

無論哪種方式，都應先用 `SHA256SUMS` 驗證檔案完整性再執行。

## Shell completion

Archive 的 `completions/` 包含：

- Bash：`taiga.bash`
- Zsh：`_taiga`
- Fish：`taiga.fish`
- PowerShell：`taiga.ps1`

也可在安裝後動態產生：

```sh
taiga completion bash
taiga completion zsh
taiga completion fish
taiga completion powershell
```

## 升級

1. 先閱讀該版本 Release Notes 與 [COMPATIBILITY.md](COMPATIBILITY.md)。
2. 下載並驗證新 archive。
3. 以新 binary 取代舊 binary；設定檔與 OS keyring credential 不需搬移。
4. 執行 `taiga version --json` 確認版本、commit 與平台。
5. 執行 `taiga doctor --json` 確認 API、authentication 與預設 Project。

降級時同樣只需換回已驗證的舊 binary。若 Release Notes 標示設定 migration，應先備份作業系統使用者設定目錄中的 Taiga CLI 設定檔。

## 從原始碼安裝

需要 Go 1.25 或更新版本：

```sh
make install PREFIX="$HOME/.local"
```

原始碼安裝預設顯示 `dev` 版本。正式 release metadata 只由可重現 packaging 流程注入。

# Changelog

*[English](CHANGELOG.md) · **繁體中文***

本檔案記錄每個版本的重要變更。格式參考 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)，版本號遵循 [Semantic Versioning](https://semver.org/spec/v2.0.0.html)。

Release workflow 會把對應版本的段落與英文版 [CHANGELOG.md](CHANGELOG.md) 的同一段落一起發布為 GitHub Release 說明，因此兩份都必須記錄每個已發布的版本。

Release notes 以 hard line break 渲染，所以這裡每個段落與項目都維持單行，即使該行很長。若在中途換行，發布頁上會出現斷掉的句子。

## [Unreleased]

## [0.2.3] - 2026-09-02

僅工具與文件變更，`aihki` 執行檔與 0.2.2 相同。

### 新增

- 安裝腳本現在會實際寫入 shell completion，不再只是印出產生指令。它只寫入**已存在**的標準目錄，並逐一告知寫到哪裡；不會建立目錄 —— 替沒在用該 shell 的人建空資料夾只是製造垃圾。Windows 維持只印提示，因為安裝 PowerShell completion 需要修改使用者的 profile，侵入性遠高於放一個檔案。
- 解除安裝會精準移除這些檔案，並跳過看起來不是自動產生的檔案，避免誤刪同名的手寫 completion。

### 修正

- CI 現在會在三個作業系統上以「安裝後立即解除」的往返方式測試。先前版本宣稱有這項覆蓋，但當時的修改靜默地沒有套用，實際上該 job 從未執行過解除安裝。

## [0.2.2] - 2026-09-02

### 修正

- 告訴使用者「下一步該執行什麼」的錯誤訊息仍使用改名前的指令名 —— 缺少憑證時會建議 `taiga auth login` 與 `TAIGA_TOKEN`，而非 `aihki auth login` 與 `AIHKI_TOKEN`。11 個檔案中的 17 處訊息現在指向真實存在的指令。

### 變更

- 端對端測試改用 `AIHKI_TOKEN` 認證（先前從未被測到），並新增一個案例證明 `TAIGA_TOKEN` 在真實執行檔中仍可認證。只有單元測試覆蓋的回退路徑，等於沒有人真的跑過。

## [0.2.1] - 2026-09-02

僅工具與文件變更，`aihki` 執行檔的功能與 0.2.0 相同。

### 新增

- macOS、Linux 與 Windows 的解除安裝腳本，涵蓋執行檔、Windows 的 PATH 項目，以及選擇性的設定目錄與 OS keyring 憑證。預設保守，因為解除安裝常常只是升級的其中一步：`--purge`（Windows 為 `-Purge`）才會一併移除設定與憑證，含改名前舊名稱留下的資料；`--dry-run` 只列出將移除的項目而不做變更。
- POSIX 解除安裝腳本會偵測 Homebrew 安裝並引導改用 `brew uninstall`，不直接刪除 Homebrew 管理的檔案。兩個腳本都會說明 `project use --local` 綁定存放在哪裡 —— 任何解除安裝程式都無法從該 repository 之外找到它。
- Repository 首頁的 hero image。

### 變更

- INSTALL.md 補上中英雙語的解除安裝說明。
- CI 在三個作業系統上以「安裝後立即解除」的往返方式測試，確保解除安裝移除的正是安裝寫入的內容。

## [0.2.0] - 2026-09-02

### 更名為 Aihki

專案更名為 **Aihki**，執行檔為 `aihki`。Taiga 以 MPL-2.0 釋出，其第 2.3 節明文不授予商標與 logo 權利，而 Taiga 的維護者先前也曾要求分家專案放棄該名稱。把他人的商標當作本專案自己的識別，從來就不是那份授權支持的事；因此該名稱現在只用於描述本客戶端所搭配的軟體。

與 Taiga 的整合沒有任何改變，改變的是這個工具本身的識別。

**既有安裝可以繼續使用。** 所有已儲存的狀態都會在首次使用時遷移，而不是被丟棄：

- 舊 `taiga-cli` keyring service 中的憑證，首次讀取時會被接收到 `aihki`，不會被登出。
- `aihki/` 沒有設定檔時會讀取 `taiga-cli/` 下的舊檔，下次儲存時寫入新位置。
- 以 `taiga.profile` 或 `taiga.project` 綁定的 repository 仍然有效；兩者並存時 `aihki.*` 優先。
- `TAIGA_TOKEN`、`TAIGA_PROFILE`、`TAIGA_API_URL`、`TAIGA_PROJECT` 仍然被接受，`AIHKI_*` 優先。

**你需要調整的部分**：改用 `brew install koukeneko/tap/aihki`；release archive 更名為 `aihki_<version>_<os>_<arch>`；completion 檔名為 `aihki.bash`、`_aihki`、`aihki.fish`、`aihki.ps1`。

## [0.1.0] - 2026-09-02

首個正式版本。以 Go 實作的 Taiga 6 命令列工具，提供人類可讀的終端輸出，以及給 Shell、CI 與 Agent 使用的穩定 JSON contract。

### 安裝

```sh
brew install koukeneko/tap/aihki
```

或從下方下載對應平台的 archive，並以 `SHA256SUMS` 驗證後解壓縮。

### 功能

- Project、Epic、User Story、Task、Issue、Sprint 與 Wiki 的完整日常操作。
- 成員與 Role 權限、Webhook、Custom field、八類 workflow metadata、Swimlane、Tag、Due-date preset。
- 跨專案的 Epic ↔ Story 關聯，以及 watch、vote、comment 編輯與歷史。
- Project 匯出／匯入、ownership transfer、CSV export 與第三方 importer gateway。
- Search、Timeline，以及 backlog velocity、Issue 趨勢、成員貢獻與 Sprint 燃盡統計。
- 附件的 streaming 上傳／下載，下載時核對 SHA-1 與大小。
- Epic、Story、Task、Issue 的批次建立，以及 Story／Task／Issue 的批次搬移與排序。

### 自動化介面

- `--json` 輸出帶 `meta.contract` 版本號的封包，目前為 contract 1。
- `--fields` 挑選欄位，`aihki schema <command>` 提供 JSON Schema 與 safety／idempotency 標註。
- 依錯誤種類分流的固定 exit code；`--dry-run` 完整解析但保證不送出寫入請求。
- Bash、Zsh、Fish、PowerShell 四種 shell completion，含 5 分鐘快取與離線 stale-on-error。

### 安全性

- 密碼不接受來自命令列；token 存於 OS keyring，不寫入設定檔。
- Webhook secret、application token auth code 與 ownership transfer token 不出現在任何輸出。
- 只有 idempotent 的 GET 會自動重試；寫入結果不明時回報 `ambiguous_commit` 要求人工確認。
- 樂觀鎖衝突時停止，不自動 merge 或覆寫。
- 附件下載不會把 API bearer token 送往 media URL。

### 發布產物

- macOS binary 以 Developer ID 憑證簽署並經 Apple notarization，下載後可直接執行。
- Linux 與 Windows archive 具位元可重現性；相同 source、toolchain、version、commit 與 epoch 會產生相同的 bytes。macOS 因 notarization 要求安全時間戳而不可重現。
- 每個 release 附 `SHA256SUMS` 與 SPDX 2.3 SBOM。

### 已知限制

- Taiga 6 REST API 將 `archived_code` 設為唯讀且無 archive action，因此 `project archive|unarchive` 會回報 `unsupported_capability`，需由站台管理者處理。
- Notarization ticket 無法 staple 到裸執行檔，Gatekeeper 需線上查驗；完全離線的首次執行仍可能被擋。
- `stats system` 僅在站台啟用 Taiga `STATS_ENABLED` 時存在，否則回報 `not_found`。

### 相容性

已針對 Taiga 6.10.2 以固定 image digest 執行完整 Docker E2E 驗證。支援 macOS、Linux、Windows 的 `amd64` 與 `arm64`。

[Unreleased]: https://github.com/KoukeNeko/aihki/compare/v0.2.3...HEAD
[0.2.3]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.3
[0.2.2]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.2
[0.2.1]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.1
[0.2.0]: https://github.com/KoukeNeko/aihki/releases/tag/v0.2.0
[0.1.0]: https://github.com/KoukeNeko/aihki/releases/tag/v0.1.0

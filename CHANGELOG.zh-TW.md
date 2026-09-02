# Changelog

*[English](CHANGELOG.md) · **繁體中文***

本檔案記錄每個版本的重要變更。格式參考 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)，版本號遵循 [Semantic Versioning](https://semver.org/spec/v2.0.0.html)。

Release workflow 會把對應版本的段落與英文版 [CHANGELOG.md](CHANGELOG.md) 的同一段落一起發布為 GitHub Release 說明，因此兩份都必須記錄每個已發布的版本。

Release notes 以 hard line break 渲染，所以這裡每個段落與項目都維持單行，即使該行很長。若在中途換行，發布頁上會出現斷掉的句子。

## [Unreleased]

## [0.1.0] - 2026-09-02

首個正式版本。以 Go 實作的 Taiga 6 命令列工具，提供人類可讀的終端輸出，以及給 Shell、CI 與 Agent 使用的穩定 JSON contract。

### 安裝

```sh
brew install koukeneko/tap/taiga
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
- `--fields` 挑選欄位，`taiga schema <command>` 提供 JSON Schema 與 safety／idempotency 標註。
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

[Unreleased]: https://github.com/KoukeNeko/Taiga-CLI/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/KoukeNeko/Taiga-CLI/releases/tag/v0.1.0

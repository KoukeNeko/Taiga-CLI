<h1 align="center">Taiga CLI</h1>

<p align="center">
  <strong>把 Taiga 帶到命令列。</strong><br>
  給人閱讀的終端輸出，以及給 Shell、CI 與 Agent 使用的穩定 JSON contract。
</p>

<p align="center">
  <img alt="Go 1.25+" src="https://img.shields.io/badge/GO-1.25%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img alt="Taiga 6.10.2 verified" src="https://img.shields.io/badge/TAIGA-6.10.2_VERIFIED-00A5A5?style=for-the-badge">
  <img alt="macOS, Linux, Windows" src="https://img.shields.io/badge/PLATFORMS-MACOS_·_LINUX_·_WINDOWS-4CAF50?style=for-the-badge">
  <img alt="JSON contract v1" src="https://img.shields.io/badge/JSON_CONTRACT-V1-2196F3?style=for-the-badge">
</p>

<p align="center">
  <a href="README.md">English</a> · <strong>繁體中文</strong>
</p>

<p align="center">
  <a href="INSTALL.md">安裝</a>
  · <a href="#快速開始">快速開始</a>
  · <a href="https://github.com/KoukeNeko/Taiga-CLI/wiki">使用手冊</a>
  · <a href="CHANGELOG.md">版本紀錄</a>
  · <a href="COMPATIBILITY.md">相容性</a>
</p>

Taiga CLI 讓你不必離開終端機就能操作 [Taiga 6](https://taiga.io/) 的專案、敏捷流程與 Wiki。它會自動從
frontend 的 `conf.json` 找出 API 位置，包括部署在 `/taiga/` 子路徑的站台，登入後把 token 交給作業系統
keyring 保管，不寫進設定檔。

**同一個指令同時服務人與程式。** 直接執行時輸出對齊的表格；加上 `--json` 就得到帶版本號的 contract，
搭配固定 exit code 與 JSON Schema descriptor，可以放心讓 Shell script、CI job 或 LLM agent 驅動。輸出格式
不會因為被 pipe 就偷偷改變 —— 要 JSON 就得明講。

**寫入行為可預測。** 所有 mutation 都走 Taiga 的樂觀鎖，衝突時停下來而不是覆蓋；只有 idempotent 的 GET
會自動重試；連線在寫入途中斷掉時回報 `ambiguous_commit`，要求你先確認，而不是盲目重送。

## 能做什麼

### 完整的 Taiga 工作流

Project、Epic、User Story、Task、Issue、Sprint 與 Wiki 的日常操作都在裡面：列表、檢視、建立、編輯、
指派、留言、關閉、刪除。加上成員與角色權限、Webhook、Custom field、八類 workflow metadata、Swimlane、
Tag、Due-date preset，以及跨專案的 Epic ↔ Story 關聯。

工作項目可以用裸 ref、`project#ref` 或直接貼 Taiga 網址來指定，三種寫法都通：

```text
42
example-project#42
https://taiga.example.com/taiga/project/example-project/issue/42
```

### 給自動化的穩定介面

`--json` 輸出 `meta.contract` 版本號，`--fields` 挑選欄位，`taiga schema <command>` 給出該指令的
input/output JSON Schema 與 safety/idempotency 標註 —— agent 可以據此判斷一個指令能不能自動執行。
Exit code 依錯誤種類固定分流，`--dry-run` 會完整解析並顯示將送出的變更，但保證不發出任何寫入請求。

### 不會意外破壞資料

刪除工作項目與 metadata 後會回讀確認；附件與 CSV 下載走 streaming、核對雜湊、以 `0600` 暫存檔原子落盤，
且預設不覆寫既有檔案。非互動模式下的破壞性操作一律要求明確的 `--yes`。Webhook secret、application
token 的 auth code 與 ownership transfer token 都不會出現在任何輸出或 dry-run 裡。

### 多站台與多專案

Profile 讓你在不同 Taiga 站台之間切換，各自記住 API URL 與預設專案。也可以把 profile 與 project 綁在
單一 Git repository 上，存進 `.git/config` 而不會被 commit：

```sh
taiga project use example-project --local
```

### 出事的時候查得出來

`taiga doctor` 逐項檢查 frontend discovery、API、authentication 與預設專案。需要求助時，
`taiga doctor bundle` 產生一份可以安心分享的診斷包 —— 只有版本資訊、設定「是否存在」的布林值與
狀態碼，不含任何 URL、使用者名稱、專案名稱或憑證，而且只在本機建立、不會自動上傳。

## 快速開始

1. **建置**（需要 Go 1.25 以上）：

   ```sh
   make build
   ./bin/taiga version
   ```

2. **登入**，token 會存進 OS keyring：

   ```sh
   taiga auth login --host https://taiga.example.com/taiga/ --profile company
   ```

3. **選定專案**：

   ```sh
   taiga project list
   taiga project use example-project
   ```

4. **開始操作**：

   ```sh
   taiga issue list
   taiga issue create --subject "Fix token refresh" --type Bug
   taiga issue assign 42 --to alice
   taiga issue close 42 --status Closed
   ```

5. **接上自動化**：

   ```sh
   taiga issue view 42 --json --fields id,ref,subject,status,version --no-input
   ```

完整的指令參考、旗標說明與各子系統的行為細節，見
[使用手冊 Wiki](https://github.com/KoukeNeko/Taiga-CLI/wiki)。

## 相容性

- Taiga 6.10.2 已透過固定 image digest 的 Docker E2E 驗證
- macOS、Linux、Windows 的 `amd64` 與 `arm64`，純 Go 建置（`CGO_ENABLED=0`）
- 帳密登入、既有 bearer token 與 refresh token rotation

詳細矩陣與已知限制見 [COMPATIBILITY.md](COMPATIBILITY.md)。

## 取得工具

macOS 與 Linux 上最簡單的方式是 Homebrew（正式版釋出後可用）：

```sh
brew install koukeneko/tap/taiga
```

Tap 只追蹤正式版。各平台的 archive、checksum 與 SBOM 都在
[Releases 頁面](https://github.com/KoukeNeko/Taiga-CLI/releases)。

從原始碼安裝到 `~/.local/bin`：

```sh
make install
```

或自訂位置：

```sh
make install PREFIX=/usr/local
```

Release archive 的下載、checksum 驗證、shell completion 安裝與升級方式見 [INSTALL.md](INSTALL.md)；
維護者的發布流程見 [RELEASING.md](RELEASING.md)。

---

## Technical reference

### 設定優先序

一般設定放在作業系統的使用者設定目錄，token 不會寫入設定檔：

```toml
current_profile = "company"

[profiles.company]
api_url = "https://taiga.example.com/taiga/api/v1/"
project = "example-project"
```

解析順序由高到低：

```text
command flag
→ TAIGA_PROFILE / TAIGA_API_URL / TAIGA_PROJECT / TAIGA_TOKEN
→ Git-local taiga.profile / taiga.project
→ current profile
→ safe defaults
```

### JSON contract

成功資料只寫 stdout，錯誤只寫 stderr。單筆用 `data`，列表用 `items` 與 `page`，兩者都帶 `meta.contract`：

```json
{
  "data": { "id": 123, "ref": 42, "subject": "Fix token refresh", "version": 7 },
  "meta": { "contract": 1 }
}
```

同一個 contract 版本內只會新增 optional 欄位；移除、改名或改變既有欄位型別必須提升版本並附遷移說明。

| Exit code | 意義 |
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

### 安全原則

- 不接受 command-line password
- Authorization、password、token 不會出現在 verbose log
- 只有 GET 會進行有上限的自動重試，POST／PATCH 不會盲目重送
- 寫入途中連線中斷且結果不明時回報 `ambiguous_commit`
- OCC conflict 不會自動 merge 或覆寫
- 附件下載不會把 API bearer token 送往 media URL
- TLS verification 預設永遠開啟

### 開發與測試

不需要 Docker 的快速迴圈：

```sh
make test
make test-race
make lint
```

對真實 Taiga server 的 integration test：

```sh
make test-integration
```

Integration harness 使用獨立的 `taiga-cli-e2e` Compose project 與 `localhost:19000`，自行建立臨時帳號、
專案與 Issue，結束後只清除自己的 container 與 volume，不會動到日常使用的 Taiga 實例。

重建跨平台 release artifacts：

```sh
make release \
  VERSION=v0.1.0 \
  COMMIT="$(git rev-parse HEAD)" \
  SOURCE_DATE_EPOCH="$(git show -s --format=%ct HEAD)"
```

相同的 source、Go toolchain、version、commit 與 epoch 會產生位元完全相同的 Linux 與 Windows archive。
macOS 是例外：notarization 要求 Apple 簽發的安全時間戳，因此 Developer ID 簽章本質上無法重現；除簽章外
其餘內容的建置方式完全相同。

<p>
  <img alt="Cobra" src="https://img.shields.io/badge/COBRA-CLI-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img alt="Reproducible builds" src="https://img.shields.io/badge/BUILDS-REPRODUCIBLE-4CAF50?style=for-the-badge">
  <img alt="SPDX 2.3 SBOM" src="https://img.shields.io/badge/SBOM-SPDX_2.3-2196F3?style=for-the-badge">
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/LICENSE-MIT-4CAF50?style=for-the-badge&logo=github"></a>
</p>

## License

[MIT](LICENSE) © KoukeNeko
